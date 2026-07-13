package main

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"os"

	echootel "github.com/labstack/echo-opentelemetry"
	echoprometheus "github.com/labstack/echo-prometheus"
	"github.com/labstack/echo/v5"
	"github.com/labstack/echo/v5/middleware"

	"github.com/julles/go-boilerplate/internal/example"
	"github.com/julles/go-boilerplate/internal/shared/cache"
	"github.com/julles/go-boilerplate/internal/shared/config"
	"github.com/julles/go-boilerplate/internal/shared/database"
	"github.com/julles/go-boilerplate/internal/shared/httpx"
	appmw "github.com/julles/go-boilerplate/internal/shared/middleware"
	"github.com/julles/go-boilerplate/internal/shared/observability"
	"github.com/julles/go-boilerplate/internal/shared/queue"
)

func main() {
	// Siapkan logger global (slog) lebih dulu agar semua log—termasuk error saat
	// startup gagal—punya format yang konsisten sejak baris pertama dieksekusi.
	observability.SetupLogger()

	// Seluruh alur nyata ditaruh di run() yang mengembalikan error. Pola ini dipakai
	// supaya semua `defer` (penutupan koneksi DB/Redis/tracer) tetap dieksekusi:
	// os.Exit() TIDAK menjalankan defer, jadi Exit hanya boleh dipanggil di sini,
	// setelah run() benar-benar selesai dan defer-nya sudah jalan.
	if err := run(); err != nil {
		slog.Error("aplikasi berhenti karena error", "error", err)
		os.Exit(1)
	}
}

func run() error {
	// ctx dipakai hanya untuk fase startup (membuka pool DB, konek Redis, init tracer).
	// Background() cukup di sini karena tiap request HTTP nanti punya context sendiri.
	ctx := context.Background()

	// Muat konfigurasi (env/file) paling awal. Jika config invalid, langsung berhenti
	// sebelum menyentuh dependency apa pun—percuma buka koneksi kalau konfignya salah.
	cfg, err := config.Load()
	if err != nil {
		return err
	}

	// Dependency dibuat berurutan dari yang paling mendasar (DB, Redis, queue, tracer).
	// Tiap dependency langsung dipasangi `defer Close()` tepat setelah berhasil dibuat,
	// sehingga urutan penutupan otomatis kebalikan urutan pembuatan (LIFO) dan tidak
	// ada resource yang bocor bila salah satu langkah berikutnya gagal.

	// Pool koneksi PostgreSQL. Pakai pool (bukan koneksi tunggal) agar banyak request
	// HTTP bisa berbagi koneksi secara efisien tanpa reconnect tiap kali.
	pool, err := database.NewPool(ctx, cfg.DatabaseURL, cfg.DBPool)
	if err != nil {
		return err
	}
	defer pool.Close()

	// Client Redis untuk cache dan rate limiter. Dibuat setelah DB karena beberapa
	// alur (mis. rate limiter) baru dipasang belakangan dan bergantung pada rdb ini.
	rdb, err := cache.NewClient(ctx, cfg.RedisURL, cfg.RedisPool)
	if err != nil {
		return err
	}
	defer rdb.Close()

	// Client queue (asynq) untuk mengirim/enqueue task ke worker. API server hanya
	// memproduksi task; yang mengeksekusi adalah proses worker terpisah.
	qClient, err := queue.NewClient(cfg.RedisURL)
	if err != nil {
		return err
	}
	defer qClient.Close()

	// Default no-op agar `defer shutdownTracer(...)` di bawah selalu aman dipanggil,
	// walau OpenTelemetry tidak diaktifkan. Dengan begitu tidak perlu cek nil saat shutdown.
	shutdownTracer := func(context.Context) error { return nil }
	if cfg.OtelEnabled {
		// Tracer hanya diinisialisasi bila tracing dinyalakan lewat config, supaya
		// deployment yang tak butuh observability tidak menanggung overhead-nya.
		shutdownTracer, err = observability.InitTracer(ctx, cfg.ServerName, cfg.OTLPEndpoint)
		if err != nil {
			return err
		}
	}
	// Pakai context baru (bukan ctx startup) supaya flush trace terakhir tetap bisa
	// dikirim saat shutdown, meski ctx startup sudah tidak relevan. Error diabaikan
	// karena ini best-effort cleanup dan proses memang sedang berhenti.
	defer func() { _ = shutdownTracer(context.Background()) }()

	// HTTP server + middleware bersama.
	e := echo.New()
	e.HTTPErrorHandler = httpx.ErrorHandler // error handler yang tidak membocorkan detail internal
	e.Validator = httpx.NewValidator()      // validasi DTO via struct tag `validate:"..."`

	// Urutan pendaftaran middleware = urutan eksekusinya per request. Recover() dipasang
	// paling awal agar menjadi lapisan terluar: bila handler/middleware lain panic, ia
	// menangkapnya dan mengubahnya jadi HTTP 500, sehingga satu request bermasalah tidak
	// menjatuhkan seluruh server.
	e.Use(middleware.Recover())
	// Logger dipasang setelah Recover supaya setiap request tetap tercatat.
	e.Use(middleware.RequestLogger())
	if cfg.OtelEnabled {
		// Middleware tracing hanya dipasang bila tracer di atas benar-benar diinisialisasi,
		// agar tidak mengirim span ke tracer no-op.
		e.Use(echootel.NewMiddleware(cfg.ServerName)) // tracing
	}
	e.Use(echoprometheus.NewMiddleware(cfg.ServerName)) // metrics
	if cfg.RateEnabled {
		// Rate limiter berbasis Redis: dibuat opsional lewat config supaya bisa dimatikan
		// di lingkungan dev/test. Redis dipakai (bukan memori lokal) agar batas tetap
		// konsisten ketika API di-scale ke banyak instance.
		e.Use(appmw.RedisRateLimiter(rdb, cfg.RateLimit, cfg.RateWindow))
	}
	// Endpoint /metrics diekspos untuk di-scrape Prometheus.
	e.GET("/metrics", echoprometheus.NewHandler())

	// Health check dipakai oleh load balancer / orchestrator (mis. Kubernetes) untuk
	// menentukan apakah instance ini layak menerima traffic. Karena itu ia benar-benar
	// mem-Ping dependency inti, bukan sekadar balas "ok": kalau DB atau Redis mati,
	// instance harus dilaporkan tidak sehat (503) agar traffic dialihkan.
	e.GET("/health", func(c *echo.Context) error {
		// Pakai context milik request agar Ping ikut dibatalkan bila klien memutus koneksi
		// atau timeout, sehingga tidak menggantung.
		ctx := c.Request().Context()
		if err := pool.Ping(ctx); err != nil {
			return echo.NewHTTPError(http.StatusServiceUnavailable, "database unhealthy")
		}
		if err := rdb.Ping(ctx).Err(); err != nil {
			return echo.NewHTTPError(http.StatusServiceUnavailable, "redis unhealthy")
		}
		return c.JSON(http.StatusOK, httpx.OK(map[string]string{"status": "ok"}))
	})

	// Registrasi modul (satu baris per modul). Tiap modul menerima dependency yang
	// dibutuhkannya (pool DB, cache, queue) lewat injeksi—bukan mengambil variabel global—
	// agar mudah diuji dan batas antar modul tetap jelas. cache.New(rdb) membungkus client
	// Redis mentah jadi abstraksi cache yang dipakai modul.
	example.RegisterRoutes(e, pool, cache.New(rdb), qClient)

	// Start memblokir sampai SIGINT/SIGTERM, lalu graceful shutdown HTTP.
	slog.Info("server dimulai", "port", cfg.HTTPPort)
	// e.Start memblokir selama server hidup. Saat shutdown normal ia mengembalikan
	// http.ErrServerClosed—itu bukan kegagalan, jadi sengaja diabaikan. Error lain
	// (mis. port sudah terpakai) diteruskan ke atas agar proses keluar dengan status gagal.
	if err := e.Start(cfg.HTTPPort); err != nil && !errors.Is(err, http.ErrServerClosed) {
		return err
	}
	return nil
}
