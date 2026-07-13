// Package config memuat konfigurasi aplikasi dari environment variable.
// Secret tidak pernah di-hardcode; semua berasal dari env.
package config

import (
	"fmt"
	"os"
	"strconv"
	"time"

	"github.com/joho/godotenv"
)

type Config struct {
	ServerName   string        // dipakai sebagai service name di tracing/log
	HTTPPort     string        // contoh ":8080"
	DatabaseURL  string        // wajib
	RedisURL     string        // wajib
	OTLPEndpoint string        // kosong = tracing dimatikan (no-op)
	OtelEnabled  bool          // false = middleware & tracer OTel tidak dipasang
	RateLimit    int           // jumlah request per window
	RateWindow   time.Duration // panjang window rate limit
	RateEnabled  bool          // false = middleware rate limiter tidak dipasang

	WorkerConcurrency int // jumlah task yang diproses paralel oleh worker

	DBPool    DBPoolConfig    // tuning pool koneksi Postgres
	RedisPool RedisPoolConfig // tuning pool koneksi Redis
}

// DBPoolConfig menampung parameter pool pgx.
type DBPoolConfig struct {
	MaxConns        int32
	MinConns        int32
	MaxConnLifetime time.Duration
	MaxConnIdleTime time.Duration
}

// RedisPoolConfig menampung parameter pool go-redis.
type RedisPoolConfig struct {
	PoolSize     int
	MinIdleConns int
}

// Load membaca konfigurasi dari env. Return error yang menyebut variable yang
// kurang bila env wajib tidak diset (fail-fast).
func Load() (Config, error) {
	// Muat .env bila ada (untuk pengembangan lokal). Di production, env di-inject
	// runtime dan file .env tidak ada — error sengaja diabaikan. godotenv tidak
	// menimpa env yang sudah diset, jadi env asli tetap menang.
	_ = godotenv.Load()

	cfg := Config{
		ServerName:   getEnv("SERVER_NAME", "go-boilerplate"),
		HTTPPort:     getEnv("HTTP_PORT", ":8080"),
		OTLPEndpoint: os.Getenv("OTLP_ENDPOINT"),
		OtelEnabled:  getEnvBool("OTEL_ENABLED", true),
		RateLimit:    getEnvInt("RATE_LIMIT", 100),
		RateWindow:   getEnvDuration("RATE_WINDOW", time.Minute),
		RateEnabled:  getEnvBool("RATE_LIMIT_ENABLED", true),

		WorkerConcurrency: getEnvInt("WORKER_CONCURRENCY", 10),

		DBPool: DBPoolConfig{
			MaxConns:        int32(getEnvInt("DB_MAX_CONNS", 10)),
			MinConns:        int32(getEnvInt("DB_MIN_CONNS", 2)),
			MaxConnLifetime: getEnvDuration("DB_MAX_CONN_LIFETIME", time.Hour),
			MaxConnIdleTime: getEnvDuration("DB_MAX_CONN_IDLE_TIME", 30*time.Minute),
		},
		RedisPool: RedisPoolConfig{
			PoolSize:     getEnvInt("REDIS_POOL_SIZE", 10),
			MinIdleConns: getEnvInt("REDIS_MIN_IDLE_CONNS", 0),
		},
	}

	// Validasi env wajib dilakukan TERAKHIR, setelah semua default terisi. Dengan
	// begitu error yang dikembalikan hanya soal secret yang benar-benar hilang,
	// bukan tercampur dengan parsing nilai opsional. Fail-fast: begitu satu env
	// wajib kosong, kembalikan error dan biarkan proses berhenti saat startup,
	// daripada baru meledak saat request pertama menyentuh DB/Redis.
	var err error
	if cfg.DatabaseURL, err = requireEnv("DATABASE_URL"); err != nil {
		return cfg, err
	}
	if cfg.RedisURL, err = requireEnv("REDIS_URL"); err != nil {
		return cfg, err
	}
	return cfg, nil
}

// requireEnv mengambil env yang tidak boleh kosong. String kosong dianggap "tidak
// diset" (bukan hanya nil), karena env var yang di-export tapi kosong sama tidak
// bergunanya dengan yang absen — keduanya harus ditolak.
func requireEnv(key string) (string, error) {
	v := os.Getenv(key)
	if v == "" {
		return "", fmt.Errorf("env %s wajib diisi", key)
	}
	return v, nil
}

// getEnv: nilai opsional bertipe string. Kembali ke fallback bila env kosong.
func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

// getEnvBool: parsing gagal sengaja jatuh ke fallback (bukan error). Ini menjaga
// konfigurasi tetap toleran — salah ketik pada flag opsional tidak boleh
// menggagalkan boot; nilai default yang aman lebih baik daripada crash.
func getEnvBool(key string, fallback bool) bool {
	if v := os.Getenv(key); v != "" {
		if b, err := strconv.ParseBool(v); err == nil {
			return b
		}
	}
	return fallback
}

// getEnvInt: sama seperti getEnvBool, nilai non-numerik diabaikan dan pakai fallback.
func getEnvInt(key string, fallback int) int {
	if v := os.Getenv(key); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			return n
		}
	}
	return fallback
}

// getEnvDuration: menerima format durasi Go (mis. "5m", "1h30m", "500ms").
// Format tidak valid diabaikan dan memakai fallback.
func getEnvDuration(key string, fallback time.Duration) time.Duration {
	if v := os.Getenv(key); v != "" {
		if d, err := time.ParseDuration(v); err == nil {
			return d
		}
	}
	return fallback
}
