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
	// Set up logger global duluan biar error waktu startup pun formatnya konsisten.
	observability.SetupLogger()

	// Alur sebenarnya ada di run(); os.Exit cuma dipanggil di sini setelah run() selesai,
	// supaya semua `defer` (nutup DB/Redis) sempat kejalan — os.Exit itu ngelewatin defer.
	if err := run(); err != nil {
		slog.Error("worker berhenti karena error", "error", err)
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

	// Worker butuh akses DB karena handler task-nya ngolah data beneran, bukan cuma baca
	// pesan doang. Pola pool + defer Close di sini sama persis kayak di API server.
	pool, err := database.NewPool(ctx, cfg.DatabaseURL, cfg.DBPool)
	if err != nil {
		return err
	}
	defer pool.Close()

	// Redis di sini punya dua peran: jadi cache (lewat cache.New di bawah) sekaligus jadi
	// broker queue asynq. Khusus di baris ini kita cuma butuh sisi cache-nya.
	rdb, err := cache.NewClient(ctx, cfg.RedisURL, cfg.RedisPool)
	if err != nil {
		return err
	}
	defer rdb.Close()

	// Server queue (asynq) yang narik task dari Redis lalu menjalankannya. Concurrency dari
	// config nentuin berapa task yang diproses paralel — ini knob buat nyeimbangin
	// throughput sama beban DB/Redis.
	srv, err := queue.NewServer(cfg.RedisURL, cfg.WorkerConcurrency)
	if err != nil {
		return err
	}

	// Rakit service beserta dependency-nya (repository di atas pool DB + cache), lalu
	// daftarin handler task tiap modul ke mux. mux inilah yang memetakan tipe task ke
	// fungsi handler-nya — mirip kayak router di HTTP server.
	// Registrasi task tiap modul, satu baris per modul.
	svc := example.NewService(example.NewRepository(pool), cache.New(rdb))
	mux := asynq.NewServeMux()
	example.RegisterTasks(mux, svc)

	slog.Info("worker dimulai", "concurrency", cfg.WorkerConcurrency)
	// Run nge-block sampai ada SIGINT/SIGTERM. asynq nangani sinyalnya sendiri secara
	// internal: begitu sinyal datang dia berhenti narik task baru tapi tetap nuntasin task
	// yang lagi jalan (graceful shutdown) sebelum balik, jadi nggak ada kerjaan yang keputus.
	return srv.Run(mux)
}
