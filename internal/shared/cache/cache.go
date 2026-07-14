// Package cache menyediakan client Redis bersama plus helper untuk pola cache-aside.
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

// NewClient bikin client Redis dengan tuning pool dari config, lalu memastikan
// Redis-nya benar-benar bisa dijangkau (fail-fast saat startup).
func NewClient(ctx context.Context, url string, pool config.RedisPoolConfig) (*redis.Client, error) {
	opt, err := redis.ParseURL(url)
	if err != nil {
		return nil, fmt.Errorf("parse redis url: %w", err)
	}
	// Tuning pool sengaja diambil dari config sebagai satu-satunya source of truth,
	// bukan dari query string di URL.
	opt.PoolSize = pool.PoolSize         // batas jumlah koneksi ke Redis
	opt.MinIdleConns = pool.MinIdleConns // koneksi idle yang disiapkan lebih dulu supaya request awal nggak kena latensi buka koneksi
	rdb := redis.NewClient(opt)
	// Sama seperti pgx, go-redis buka koneksi secara lazy. Ping kita panggil untuk
	// memaksa satu koneksi nyata terbentuk, jadi kalau kredensial atau jaringannya
	// bermasalah langsung ketahuan pas startup, bukan pas request pertama masuk.
	if err := rdb.Ping(ctx).Err(); err != nil {
		rdb.Close() // tutup koneksi biar nggak bocor kalau ping gagal
		return nil, fmt.Errorf("ping redis: %w", err)
	}
	return rdb, nil
}

// Cache adalah helper get/set JSON dengan TTL untuk kebutuhan pola cache-aside.
type Cache struct {
	rdb *redis.Client
}

func New(rdb *redis.Client) *Cache { return &Cache{rdb: rdb} }

// Get mengisi dest dari cache dan balikin true kalau cache hit.
// Kalau Redis-nya error, kita anggap saja sebagai cache miss — request tetap jalan.
func (c *Cache) Get(ctx context.Context, key string, dest any) bool {
	b, err := c.rdb.Get(ctx, key).Bytes()
	if err != nil {
		// redis.Nil artinya key-nya memang nggak ada alias cache miss biasa, jadi
		// sengaja TIDAK di-log biar log nggak kebanjiran. Error lain (jaringan atau
		// Redis lagi down) di-log warning, tapi tetap kita anggap miss: cache itu cuma
		// optimasi, bukan source of truth, jadi kalau dia gagal jangan sampai bikin
		// request ikut gagal — cukup fallback ambil ke DB.
		if !errors.Is(err, redis.Nil) {
			slog.WarnContext(ctx, "cache get gagal", "key", key, "error", err)
		}
		return false
	}
	// Kalau data di cache korup atau formatnya berubah, kita perlakukan sebagai miss
	// juga, biar pemanggil ambil ulang dari sumbernya daripada dipaksa pakai nilai rusak.
	if err := json.Unmarshal(b, dest); err != nil {
		slog.WarnContext(ctx, "cache unmarshal gagal", "key", key, "error", err)
		return false
	}
	return true
}

// Set menyimpan val dalam bentuk JSON dengan TTL tertentu. Error-nya cuma di-log,
// nggak dikembalikan ke pemanggil.
func (c *Cache) Set(ctx context.Context, key string, val any, ttl time.Duration) {
	b, err := json.Marshal(val)
	if err != nil {
		slog.WarnContext(ctx, "cache marshal gagal", "key", key, "error", err)
		return
	}
	// Gagal nulis ke cache cukup di-log dan nggak dikembalikan, karena nyimpen ke
	// cache itu sifatnya best-effort. Kalau Redis lagi bermasalah, request tetap
	// sukses jalan tanpa cache dan datanya bakal ditulis ulang di kesempatan
	// berikutnya. TTL bikin tiap entri kedaluwarsa sendiri, jadi cache nggak akan
	// nyimpen data stale selamanya.
	if err := c.rdb.Set(ctx, key, b, ttl).Err(); err != nil {
		slog.WarnContext(ctx, "cache set gagal", "key", key, "error", err)
	}
}
