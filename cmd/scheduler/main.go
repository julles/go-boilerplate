package main

import (
	"context"
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	"github.com/julles/go-boilerplate/internal/example"
	"github.com/julles/go-boilerplate/internal/shared/cache"
	"github.com/julles/go-boilerplate/internal/shared/config"
	"github.com/julles/go-boilerplate/internal/shared/database"
	"github.com/julles/go-boilerplate/internal/shared/observability"
	"github.com/julles/go-boilerplate/internal/shared/scheduler"
)

func main() {
	// Set up logger global duluan biar error waktu startup pun formatnya konsisten.
	observability.SetupLogger()

	// Alur sebenarnya ada di run(); os.Exit cuma dipanggil di sini setelah run() selesai,
	// supaya semua `defer` (nutup DB/Redis) sempat kejalan — os.Exit itu ngelewatin defer.
	if err := run(); err != nil {
		slog.Error("scheduler berhenti karena error", "error", err)
		os.Exit(1)
	}
}

func run() error {
	// Context buat fase startup: buka connection pool DB, konek Redis.
	ctx := context.Background()

	// Load config paling awal; berhenti kalau invalid sebelum buka koneksi apa pun.
	cfg, err := config.Load()
	if err != nil {
		return err
	}

	// Job terjadwal umumnya ngolah atau agregasi data, jadi scheduler tetap butuh akses
	// DB. Pola pool + defer Close di sini sama persis kayak di API server dan worker.
	pool, err := database.NewPool(ctx, cfg.DatabaseURL, cfg.DBPool)
	if err != nil {
		return err
	}
	defer pool.Close()

	// Redis buat cache yang dipakai service pas job lagi jalan.
	rdb, err := cache.NewClient(ctx, cfg.RedisURL, cfg.RedisPool)
	if err != nil {
		return err
	}
	defer rdb.Close()

	// Rakit service, lalu daftarin jadwal tiap modul ke scheduler. RegisterSchedule bakal
	// balikin error — misalnya ekspresi cron-nya salah — dan kita berhenti lebih awal kalau
	// gagal, biar nggak sampai menjalankan scheduler dengan konfigurasi jadwal yang cacat.
	// Registrasi jadwal tiap modul, satu baris per modul.
	svc := example.NewService(example.NewRepository(pool), cache.New(rdb))
	sch := scheduler.New()
	if err := example.RegisterSchedule(sch, svc); err != nil {
		return err
	}

	// Start menjalankan cron di background goroutine-nya sendiri lalu langsung balik alias
	// non-blocking. Makanya kita perlu nahan main goroutine secara eksplisit di bawah;
	// kalau nggak, prosesnya bakal langsung keluar dan nggak ada satu job pun yang jalan.
	sch.Start()
	slog.Info("scheduler dimulai")

	// Beda sama API/worker yang blocking sendiri, di sini kita pasang penangkap sinyal
	// manual. NotifyContext ngasih context yang otomatis dibatalkan begitu SIGINT/SIGTERM
	// datang; `<-sigCtx.Done()` nge-block main goroutine sampai sinyal itu tiba. stop()
	// lewat defer tugasnya nglepas pendaftaran sinyal pas fungsinya selesai.
	// Tunggu SIGINT/SIGTERM, baru hentikan cron dengan rapi.
	sigCtx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	<-sigCtx.Done()

	// Begitu sinyal diterima, hentikan cron dengan rapi: Stop nyegah job baru dipicu. Baru
	// setelah ini fungsinya balik, jadi defer di atas nutup Redis dulu baru DB (LIFO).
	slog.Info("scheduler shutdown")
	sch.Stop()
	return nil
}
