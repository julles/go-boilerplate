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
			// Endpoint infra (health probe, scrape metrics) di-skip: keduanya dipanggil
			// otomatis dan berkala oleh Kubernetes/Prometheus dari IP yang sama. Bila
			// ikut dihitung, probe/scrape sendiri bisa menghabiskan kuota window dan
			// justru menolak health check — persis kebalikan dari yang kita mau.
			switch c.Path() {
			case "/health", "/metrics":
				return next(c)
			}

			ctx := c.Request().Context()
			// Key per-IP. Counter disimpan di Redis (bukan memori proses) supaya batas
			// berlaku global lintas replica; kalau di memori, tiap pod punya hitungan
			// sendiri dan batas efektif jadi limit x jumlah pod.
			key := "ratelimit:" + c.RealIP()

			// INCR bersifat atomik: naikkan counter dan baca nilainya dalam satu operasi,
			// jadi aman dari race meski banyak request paralel dari IP yang sama.
			count, err := rdb.Incr(ctx, key).Result()
			if err != nil {
				// Fail-open: kalau Redis mati, kita PILIH mengizinkan request daripada
				// menolak semua traffic. Rate limiter adalah pelindung, bukan gerbang
				// utama — memblokir seluruh layanan karena limiter down = melipatgandakan
				// kegagalan. Cukup log warning agar tetap terlihat di monitoring.
				slog.WarnContext(ctx, "rate limiter: redis error, fail-open", "error", err)
				return next(c)
			}
			if count == 1 {
				// Expiry di-set HANYA saat request pertama membuat window baru (count==1).
				// Kalau di-set tiap request, TTL akan terus diperpanjang (sliding) dan
				// window tak pernah reset — bikin fixed-window jadi tidak pernah kedaluwarsa.
				rdb.Expire(ctx, key, window)
			}
			if count > int64(limit) {
				// Melewati batas dalam window → 429. Klien seharusnya backoff dan mencoba
				// lagi setelah window habis.
				return echo.NewHTTPError(http.StatusTooManyRequests, "too many requests")
			}
			return next(c)
		}
	}
}
