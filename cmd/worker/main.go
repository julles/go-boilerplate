package main

import (
	"context"
	"log/slog"
	"os"

	"github.com/hibiken/asynq"

	"github.com/julles/go-boilerplate/internal/example"
	"github.com/julles/go-boilerplate/internal/shared/cache"
	"github.com/julles/go-boilerplate/internal/shared/config"
	"github.com/julles/go-boilerplate/internal/shared/database"
	"github.com/julles/go-boilerplate/internal/shared/observability"
	"github.com/julles/go-boilerplate/internal/shared/queue"
)

func main() {
	// Siapkan logger global lebih dulu agar error startup pun terformat konsisten.
	observability.SetupLogger()

	// Alur nyata ada di run(); os.Exit hanya dipanggil di sini setelah run() selesai
	// supaya semua `defer` (penutupan DB/Redis) sempat dieksekusi—os.Exit melewati defer.
	if err := run(); err != nil {
		slog.Error("worker berhenti karena error", "error", err)
		os.Exit(1)
	}
}

func run() error {
	// Context untuk fase startup (buka pool DB, konek Redis).
	ctx := context.Background()

	// Muat config paling awal; berhenti bila invalid sebelum membuka koneksi apa pun.
	cfg, err := config.Load()
	if err != nil {
		return err
	}

	// Worker butuh akses DB karena handler task memproses data nyata, bukan sekadar
	// membaca pesan. Pool + defer Close mengikuti pola yang sama dengan API server.
	pool, err := database.NewPool(ctx, cfg.DatabaseURL, cfg.DBPool)
	if err != nil {
		return err
	}
	defer pool.Close()

	// Redis dipakai dua peran: sebagai cache (lewat cache.New di bawah) dan sebagai
	// broker antrian asynq. Di sini kita hanya butuh sisi cache-nya.
	rdb, err := cache.NewClient(ctx, cfg.RedisURL, cfg.RedisPool)
	if err != nil {
		return err
	}
	defer rdb.Close()

	// Server queue (asynq) yang menarik task dari Redis dan menjalankannya. Concurrency
	// dari config menentukan berapa task diproses paralel—knob untuk menyeimbangkan
	// throughput dengan beban DB/Redis.
	srv, err := queue.NewServer(cfg.RedisURL, cfg.WorkerConcurrency)
	if err != nil {
		return err
	}

	// Rakit service beserta dependency-nya (repository di atas pool DB + cache), lalu
	// daftarkan handler task tiap modul ke mux. mux inilah yang memetakan tipe task
	// ke fungsi handler-nya—analog dengan router pada HTTP server.
	// Registrasi task tiap modul (satu baris per modul).
	svc := example.NewService(example.NewRepository(pool), cache.New(rdb))
	mux := asynq.NewServeMux()
	example.RegisterTasks(mux, svc)

	slog.Info("worker dimulai", "concurrency", cfg.WorkerConcurrency)
	// Run memblokir sampai SIGINT/SIGTERM. asynq menangani sinyal secara internal:
	// saat sinyal datang ia berhenti menarik task baru namun menuntaskan task yang
	// sedang berjalan (graceful) sebelum kembali, sehingga tidak ada pekerjaan terputus.
	return srv.Run(mux)
}
