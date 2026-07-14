// Package middleware berisi middleware yang dipakai bersama (rate limiter berbasis Redis).
package middleware

import (
	"log/slog"
	"net/http"
	"time"

	"github.com/labstack/echo/v5"
	"github.com/redis/go-redis/v9"
)

// RedisRateLimiter batasin request per client (berdasarkan IP) pakai fixed-window
// counter di Redis, jadi limitnya berlaku global lintas replica.
//
// Kalau Redis lagi nggak bisa dijangkau, middleware-nya fail-open (request tetap
// diizinkan) plus log warning, biar ketersediaan layanan tetap kejaga.
//
// ponytail: fixed window sederhana; ganti ke sliding window kalau butuh presisi di batas window.
func RedisRateLimiter(rdb *redis.Client, limit int, window time.Duration) echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c *echo.Context) error {
			// Endpoint infra (health probe & scrape metrics) sengaja di-skip: dua-duanya
			// dipanggil otomatis dan berkala sama Kubernetes/Prometheus dari IP yang sama.
			// Kalau ikut dihitung, probe/scrape-nya sendiri malah bisa ngabisin kuota
			// window dan bikin health check-nya ketolak — kebalikan banget dari yang kita mau.
			switch c.Path() {
			case "/health", "/metrics":
				return next(c)
			}

			ctx := c.Request().Context()
			// Key-nya per-IP. Counter-nya kita simpen di Redis, bukan di memori proses,
			// biar limitnya berlaku global lintas replica. Kalau disimpen di memori, tiap
			// pod bakal punya hitungan sendiri dan limit efektifnya jadi limit x jumlah pod.
			key := "ratelimit:" + c.RealIP()

			// INCR itu atomik: naikin counter dan baca nilainya sekaligus dalam satu
			// operasi, jadi aman dari race walaupun banyak request paralel dari IP yang sama.
			count, err := rdb.Incr(ctx, key).Result()
			if err != nil {
				// Fail-open: kalau Redis-nya mati, kita PILIH ngizinin request daripada
				// nolak semua traffic. Rate limiter itu pelindung, bukan gerbang utama —
				// mblokir seluruh layanan gara-gara limiter-nya down malah bikin
				// kegagalannya berlipat. Cukup log warning aja biar tetap keliatan di monitoring.
				slog.WarnContext(ctx, "rate limiter: redis error, fail-open", "error", err)
				return next(c)
			}
			if count == 1 {
				// Expiry cuma di-set pas request pertama yang bikin window baru (count==1).
				// Kalau di-set tiap request, TTL-nya bakal terus keperpanjang (jadi sliding)
				// dan window-nya nggak pernah reset — bikin fixed-window kita nggak pernah expired.
				rdb.Expire(ctx, key, window)
			}
			if count > int64(limit) {
				// Udah lewat batas dalam satu window → balikin 429. Client harusnya backoff
				// dan nyoba lagi setelah window-nya habis.
				return echo.NewHTTPError(http.StatusTooManyRequests, "too many requests")
			}
			return next(c)
		}
	}
}
