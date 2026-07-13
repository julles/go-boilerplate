// Package cache menyediakan client Redis bersama dan helper cache-aside.
package cache

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"github.com/redis/go-redis/v9"

	"github.com/julles/go-boilerplate/internal/shared/config"
)

// NewClient membuat client Redis dengan tuning pool dari config dan memastikan
// bisa dijangkau (fail-fast).
func NewClient(ctx context.Context, url string, pool config.RedisPoolConfig) (*redis.Client, error) {
	opt, err := redis.ParseURL(url)
	if err != nil {
		return nil, fmt.Errorf("parse redis url: %w", err)
	}
	// Tuning pool ditimpa dari config (sumber kebenaran tunggal), bukan dari URL.
	opt.PoolSize = pool.PoolSize         // batas koneksi ke Redis
	opt.MinIdleConns = pool.MinIdleConns // koneksi idle yang disiapkan agar request awal tidak kena latensi dial
	rdb := redis.NewClient(opt)
	// Sama seperti pgx, go-redis membuka koneksi secara lazy. Ping memaksa koneksi
	// nyata supaya masalah kredensial/jaringan gagal-cepat saat startup.
	if err := rdb.Ping(ctx).Err(); err != nil {
		rdb.Close() // cegah kebocoran koneksi bila ping gagal
		return nil, fmt.Errorf("ping redis: %w", err)
	}
	return rdb, nil
}

// Cache adalah helper get/set JSON dengan TTL untuk pola cache-aside.
type Cache struct {
	rdb *redis.Client
}

func New(rdb *redis.Client) *Cache { return &Cache{rdb: rdb} }

// Get mengisi dest dari cache. Return true bila hit.
// Kegagalan Redis diperlakukan sebagai cache miss (tidak menggagalkan request).
func (c *Cache) Get(ctx context.Context, key string, dest any) bool {
	b, err := c.rdb.Get(ctx, key).Bytes()
	if err != nil {
		// redis.Nil = key tidak ada = miss normal, jadi TIDAK di-log agar tidak
		// membanjiri log. Error lain (jaringan/Redis down) di-log warning tapi tetap
		// diperlakukan sebagai miss: cache adalah optimasi, bukan sumber kebenaran,
		// sehingga kegagalannya tidak boleh menggagalkan request (fallback ke DB).
		if !errors.Is(err, redis.Nil) {
			slog.WarnContext(ctx, "cache get gagal", "key", key, "error", err)
		}
		return false
	}
	// Data di cache korup/format berubah → perlakukan sebagai miss, biar pemanggil
	// mengambil ulang dari sumber daripada memaksakan nilai yang rusak.
	if err := json.Unmarshal(b, dest); err != nil {
		slog.WarnContext(ctx, "cache unmarshal gagal", "key", key, "error", err)
		return false
	}
	return true
}

// Set menyimpan val (sebagai JSON) dengan TTL. Error hanya dicatat, tidak dikembalikan.
func (c *Cache) Set(ctx context.Context, key string, val any, ttl time.Duration) {
	b, err := json.Marshal(val)
	if err != nil {
		slog.WarnContext(ctx, "cache marshal gagal", "key", key, "error", err)
		return
	}
	// Gagal menulis cache hanya di-log, tidak dikembalikan: menyimpan cache adalah
	// best-effort. Jika Redis bermasalah, request tetap sukses tanpa cache dan data
	// akan ditulis ulang pada kesempatan berikutnya. TTL memastikan entri kedaluwarsa
	// sendiri sehingga cache tidak menyimpan data basi selamanya.
	if err := c.rdb.Set(ctx, key, b, ttl).Err(); err != nil {
		slog.WarnContext(ctx, "cache set gagal", "key", key, "error", err)
	}
}
