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
	// Set up logger global (slog) duluan supaya semua log — termasuk error waktu
	// startup gagal — punya format yang konsisten sejak baris pertama dieksekusi.
	observability.SetupLogger()

	// Semua alur sebenarnya kita taruh di run() yang balikin error. Kenapa dipisah begini?
	// Supaya semua `defer` (nutup koneksi DB/Redis/tracer) tetap kejalan. os.Exit() itu
	// nggak menjalankan defer, jadi Exit cuma boleh dipanggil di sini — setelah run()
	// benar-benar selesai dan defer-nya udah sempat jalan.
	if err := run(); err != nil {
		slog.Error("aplikasi berhenti karena error", "error", err)
		os.Exit(1)
	}
}

func run() error {
	// ctx ini cuma dipakai buat fase startup: buka connection pool DB, konek Redis, init
	// tracer. Background() udah cukup di sini karena tiap request HTTP nanti bawa context
	// sendiri-sendiri.
	ctx := context.Background()

	// Load konfigurasi (dari env/file) paling awal. Kalau config-nya invalid, langsung
	// berhenti sebelum nyentuh dependency apa pun — percuma buka koneksi kalau confignya
	// aja udah salah.
	cfg, err := config.Load()
	if err != nil {
		return err
	}

	// Dependency dibikin berurutan dari yang paling mendasar dulu: DB, Redis, queue,
	// tracer. Tiap dependency langsung dipasangi `defer Close()` begitu berhasil dibuat,
	// jadi urutan nutupnya otomatis kebalik dari urutan bikinnya (LIFO) dan nggak ada
	// resource yang bocor kalau salah satu langkah berikutnya gagal di tengah jalan.

	// Connection pool PostgreSQL. Kita pakai pool, bukan koneksi tunggal, biar banyak
	// request HTTP bisa berbagi koneksi secara efisien tanpa reconnect tiap kali.
	pool, err := database.NewPool(ctx, cfg.DatabaseURL, cfg.DBPool)
	if err != nil {
		return err
	}
	defer pool.Close()

	// Client Redis buat cache sekaligus rate limiter. Dibikin setelah DB karena beberapa
	// alur — misalnya rate limiter — baru dipasang belakangan dan nyandar ke rdb ini.
	rdb, err := cache.NewClient(ctx, cfg.RedisURL, cfg.RedisPool)
	if err != nil {
		return err
	}
	defer rdb.Close()

	// Client queue (asynq) buat kirim/enqueue task ke worker. API server di sini cuma
	// jadi produsen task; yang benar-benar mengeksekusinya adalah proses worker terpisah.
	qClient, err := queue.NewClient(cfg.RedisURL)
	if err != nil {
		return err
	}
	defer qClient.Close()

	// Kita kasih default no-op dulu biar `defer shutdownTracer(...)` di bawah selalu aman
	// dipanggil walaupun OpenTelemetry-nya nggak diaktifkan. Jadi pas shutdown nggak usah
	// ada cek nil segala.
	shutdownTracer := func(context.Context) error { return nil }
	if cfg.OtelEnabled {
		// Tracer baru di-init kalau tracing memang dinyalakan lewat config, supaya
		// deployment yang nggak butuh observability nggak ikut nanggung overhead-nya.
		shutdownTracer, err = observability.InitTracer(ctx, cfg.ServerName, cfg.OTLPEndpoint)
		if err != nil {
			return err
		}
	}
	// Sengaja pakai context baru, bukan ctx startup, biar flush trace terakhir tetap bisa
	// dikirim pas shutdown walaupun ctx startup-nya udah nggak relevan. Error-nya kita
	// abaikan karena ini cuma best-effort cleanup dan prosesnya juga lagi mau berhenti.
	defer func() { _ = shutdownTracer(context.Background()) }()

	// HTTP server + middleware bersama.
	e := echo.New()
	e.HTTPErrorHandler = httpx.ErrorHandler // error handler yang nggak bocorin detail internal
	e.Validator = httpx.NewValidator()      // validasi DTO lewat struct tag `validate:"..."`

	// Urutan daftar middleware = urutan jalannya per request. Recover() dipasang paling
	// awal biar jadi lapisan paling luar: kalau handler atau middleware lain panic, dia yang
	// nangkep dan ngubah jadi HTTP 500, jadi satu request bermasalah nggak sampai
	// menjatuhkan seluruh server.
	e.Use(middleware.Recover())
	// Logger dipasang setelah Recover biar tiap request tetap kecatat.
	e.Use(middleware.RequestLogger())
	if cfg.OtelEnabled {
		// Middleware tracing cuma dipasang kalau tracer di atas beneran ke-init, biar
		// nggak ngirim span ke tracer no-op.
		e.Use(echootel.NewMiddleware(cfg.ServerName)) // tracing
	}
	e.Use(echoprometheus.NewMiddleware(cfg.ServerName)) // metrics
	if cfg.RateEnabled {
		// Rate limiter berbasis Redis. Sengaja dibikin opsional lewat config biar gampang
		// dimatiin di lingkungan dev/test. Kenapa Redis, bukan memori lokal? Biar batasnya
		// tetap konsisten waktu API di-scale ke banyak instance.
		e.Use(appmw.RedisRateLimiter(rdb, cfg.RateLimit, cfg.RateWindow))
	}
	// Endpoint /metrics dibuka biar bisa di-scrape Prometheus.
	e.GET("/metrics", echoprometheus.NewHandler())

	// Health check ini dipakai load balancer atau orchestrator seperti Kubernetes buat
	// nentuin apakah instance ini masih layak nerima traffic. Makanya dia beneran nge-Ping
	// dependency inti, bukan cuma balas "ok" doang: kalau DB atau Redis-nya mati, instance
	// harus dilaporin nggak sehat (503) biar traffic-nya dialihin ke instance lain.
	e.GET("/health", func(c *echo.Context) error {
		// Pakai context milik request biar Ping-nya ikut dibatalkan kalau klien mutus
		// koneksi atau kena timeout, jadi nggak menggantung sia-sia.
		ctx := c.Request().Context()
		if err := pool.Ping(ctx); err != nil {
			return echo.NewHTTPError(http.StatusServiceUnavailable, "database unhealthy")
		}
		if err := rdb.Ping(ctx).Err(); err != nil {
			return echo.NewHTTPError(http.StatusServiceUnavailable, "redis unhealthy")
		}
		return c.JSON(http.StatusOK, httpx.OK(map[string]string{"status": "ok"}))
	})

	// Registrasi modul, satu baris per modul. Tiap modul nerima dependency yang dia butuh
	// — pool DB, cache, queue — lewat injeksi, bukan ngambil dari variabel global. Cara ini
	// bikin modul gampang dites dan batas antar modul tetap jelas. cache.New(rdb) di sini
	// tugasnya membungkus client Redis mentah jadi abstraksi cache yang dipakai modul.
	example.RegisterRoutes(e, pool, cache.New(rdb), qClient)

	// Start bakal blocking sampai ada SIGINT/SIGTERM, lalu graceful shutdown HTTP-nya.
	slog.Info("server dimulai", "port", cfg.HTTPPort)
	// e.Start nge-block selama server hidup. Pas shutdown normal, dia balikin
	// http.ErrServerClosed — itu bukan kegagalan, jadi sengaja kita abaikan. Error lain,
	// misalnya port udah kepakai, diterusin ke atas biar prosesnya keluar dengan status gagal.
	if err := e.Start(cfg.HTTPPort); err != nil && !errors.Is(err, http.ErrServerClosed) {
		return err
	}
	return nil
}
