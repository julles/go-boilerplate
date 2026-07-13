// Package middleware berisi middleware bersama (rate limiter berbasis Redis).
package middleware

import (
	"log/slog"
	"net/http"
	"time"

	"github.com/labstack/echo/v5"
	"github.com/redis/go-redis/v9"
)

// RedisRateLimiter membatasi request per klien (berdasarkan IP) memakai fixed-window
// counter di Redis, sehingga batas berlaku global lintas replica.
//
// Bila Redis tidak dapat dijangkau, middleware fail-open (izinkan request) + log warning
// agar ketersediaan layanan tetap terjaga.
//
// ponytail: fixed window sederhana; ganti ke sliding window kalau butuh presisi di batas window.
func RedisRateLimiter(rdb *redis.Client, limit int, window time.Duration) echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c *echo.Context) error {
			// Endpoint infra (health probe, scrape metrics) tidak di-rate-limit.
			switch c.Path() {
			case "/health", "/metrics":
				return next(c)
			}

			ctx := c.Request().Context()
			key := "ratelimit:" + c.RealIP()

			count, err := rdb.Incr(ctx, key).Result()
			if err != nil {
				slog.WarnContext(ctx, "rate limiter: redis error, fail-open", "error", err)
				return next(c)
			}
			if count == 1 {
				// set expiry hanya saat window baru dibuat
				rdb.Expire(ctx, key, window)
			}
			if count > int64(limit) {
				return echo.NewHTTPError(http.StatusTooManyRequests, "terlalu banyak permintaan")
			}
			return next(c)
		}
	}
}
