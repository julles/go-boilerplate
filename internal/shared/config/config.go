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
	}

	var err error
	if cfg.DatabaseURL, err = requireEnv("DATABASE_URL"); err != nil {
		return cfg, err
	}
	if cfg.RedisURL, err = requireEnv("REDIS_URL"); err != nil {
		return cfg, err
	}
	return cfg, nil
}

func requireEnv(key string) (string, error) {
	v := os.Getenv(key)
	if v == "" {
		return "", fmt.Errorf("env %s wajib diisi", key)
	}
	return v, nil
}

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func getEnvBool(key string, fallback bool) bool {
	if v := os.Getenv(key); v != "" {
		if b, err := strconv.ParseBool(v); err == nil {
			return b
		}
	}
	return fallback
}

func getEnvInt(key string, fallback int) int {
	if v := os.Getenv(key); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			return n
		}
	}
	return fallback
}

func getEnvDuration(key string, fallback time.Duration) time.Duration {
	if v := os.Getenv(key); v != "" {
		if d, err := time.ParseDuration(v); err == nil {
			return d
		}
	}
	return fallback
}
