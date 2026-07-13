package main

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"os"

	"github.com/labstack/echo/v5"
	"github.com/labstack/echo/v5/middleware"
	echootel "github.com/labstack/echo-opentelemetry"
	echoprometheus "github.com/labstack/echo-prometheus"

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
	observability.SetupLogger()

	if err := run(); err != nil {
		slog.Error("aplikasi berhenti karena error", "error", err)
		os.Exit(1)
	}
}

func run() error {
	ctx := context.Background()

	cfg, err := config.Load()
	if err != nil {
		return err
	}

	// Dependency: database, redis, tracer.
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

	qClient, err := queue.NewClient(cfg.RedisURL)
	if err != nil {
		return err
	}
	defer qClient.Close()

	shutdownTracer := func(context.Context) error { return nil }
	if cfg.OtelEnabled {
		shutdownTracer, err = observability.InitTracer(ctx, cfg.ServerName, cfg.OTLPEndpoint)
		if err != nil {
			return err
		}
	}
	defer func() { _ = shutdownTracer(context.Background()) }()

	// HTTP server + middleware bersama.
	e := echo.New()
	e.HTTPErrorHandler = httpx.ErrorHandler // error handler yang tidak membocorkan detail internal
	e.Validator = httpx.NewValidator()      // validasi DTO via struct tag `validate:"..."`
	e.Use(middleware.Recover())
	e.Use(middleware.RequestLogger())
	if cfg.OtelEnabled {
		e.Use(echootel.NewMiddleware(cfg.ServerName)) // tracing
	}
	e.Use(echoprometheus.NewMiddleware(cfg.ServerName)) // metrics
	if cfg.RateEnabled {
		e.Use(appmw.RedisRateLimiter(rdb, cfg.RateLimit, cfg.RateWindow))
	}
	e.GET("/metrics", echoprometheus.NewHandler())

	// Health check: pastikan dependency (DB, Redis) sehat.
	e.GET("/health", func(c *echo.Context) error {
		ctx := c.Request().Context()
		if err := pool.Ping(ctx); err != nil {
			return echo.NewHTTPError(http.StatusServiceUnavailable, "database unhealthy")
		}
		if err := rdb.Ping(ctx).Err(); err != nil {
			return echo.NewHTTPError(http.StatusServiceUnavailable, "redis unhealthy")
		}
		return c.JSON(http.StatusOK, httpx.OK(map[string]string{"status": "ok"}))
	})

	// Registrasi modul (satu baris per modul).
	example.RegisterRoutes(e, pool, cache.New(rdb), qClient)

	// Start memblokir sampai SIGINT/SIGTERM, lalu graceful shutdown HTTP.
	slog.Info("server dimulai", "port", cfg.HTTPPort)
	if err := e.Start(cfg.HTTPPort); err != nil && !errors.Is(err, http.ErrServerClosed) {
		return err
	}
	return nil
}
