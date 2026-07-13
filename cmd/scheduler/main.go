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
	observability.SetupLogger()

	if err := run(); err != nil {
		slog.Error("scheduler berhenti karena error", "error", err)
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

	// Registrasi jadwal tiap modul (satu baris per modul).
	svc := example.NewService(example.NewRepository(pool), cache.New(rdb))
	sch := scheduler.New()
	if err := example.RegisterSchedule(sch, svc); err != nil {
		return err
	}

	sch.Start()
	slog.Info("scheduler dimulai")

	// Tunggu SIGINT/SIGTERM, lalu hentikan cron dengan rapi.
	sigCtx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	<-sigCtx.Done()

	slog.Info("scheduler shutdown")
	sch.Stop()
	return nil
}
