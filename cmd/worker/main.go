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
	observability.SetupLogger()

	if err := run(); err != nil {
		slog.Error("worker berhenti karena error", "error", err)
		os.Exit(1)
	}
}

func run() error {
	ctx := context.Background()

	cfg, err := config.Load()
	if err != nil {
		return err
	}

	pool, err := database.NewPool(ctx, cfg.DatabaseURL)
	if err != nil {
		return err
	}
	defer pool.Close()

	rdb, err := cache.NewClient(ctx, cfg.RedisURL)
	if err != nil {
		return err
	}
	defer rdb.Close()

	srv, err := queue.NewServer(cfg.RedisURL, cfg.WorkerConcurrency)
	if err != nil {
		return err
	}

	// Registrasi task tiap modul (satu baris per modul).
	svc := example.NewService(example.NewRepository(pool), cache.New(rdb))
	mux := asynq.NewServeMux()
	example.RegisterTasks(mux, svc)

	slog.Info("worker dimulai", "concurrency", cfg.WorkerConcurrency)
	// Run memblokir sampai SIGINT/SIGTERM, lalu menuntaskan task berjalan (graceful).
	return srv.Run(mux)
}
