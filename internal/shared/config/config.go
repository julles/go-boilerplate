// Package config memuat konfigurasi aplikasi dari environment variable.
// Secret nggak pernah di-hardcode; semuanya diambil dari env.
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
	DatabaseURL  string        // wajib diisi
	RedisURL     string        // wajib diisi
	OTLPEndpoint string        // kalau kosong, tracing dimatikan (jadi no-op)
	OtelEnabled  bool          // false = middleware & tracer OTel nggak dipasang
	RateLimit    int           // jumlah request yang diizinkan per window
	RateWindow   time.Duration // panjang window rate limit
	RateEnabled  bool          // false = middleware rate limiter nggak dipasang

	WorkerConcurrency int // berapa task yang diproses paralel oleh worker

	DBPool    DBPoolConfig    // tuning connection pool Postgres
	RedisPool RedisPoolConfig // tuning connection pool Redis
}

// DBPoolConfig menampung parameter connection pool pgx.
type DBPoolConfig struct {
	MaxConns        int32
	MinConns        int32
	MaxConnLifetime time.Duration
	MaxConnIdleTime time.Duration
}

// RedisPoolConfig menampung parameter connection pool go-redis.
type RedisPoolConfig struct {
	PoolSize     int
	MinIdleConns int
}

// Load membaca konfigurasi dari env. Kalau ada env wajib yang belum diset, dia
// balikin error yang nyebut variable mana yang kurang (fail-fast).
func Load() (Config, error) {
	// Muat file .env kalau ada, ini buat kebutuhan development lokal. Di production
	// env-nya di-inject langsung saat runtime dan file .env memang nggak ada, jadi
	// error-nya sengaja kita abaikan. godotenv juga nggak akan nimpa env yang sudah
	// keset duluan, jadi env asli dari environment tetap menang.
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

	// Validasi env wajib sengaja ditaruh PALING AKHIR, setelah semua default terisi.
	// Dengan urutan ini, error yang balik cuma soal secret yang beneran hilang, nggak
	// kecampur sama urusan parsing nilai opsional. Prinsipnya fail-fast: begitu ada
	// satu env wajib yang kosong, langsung balikin error dan biarkan proses berhenti
	// pas startup, daripada baru meledak nanti pas request pertama nyentuh DB/Redis.
	var err error
	if cfg.DatabaseURL, err = requireEnv("DATABASE_URL"); err != nil {
		return cfg, err
	}
	if cfg.RedisURL, err = requireEnv("REDIS_URL"); err != nil {
		return cfg, err
	}
	return cfg, nil
}

// requireEnv ngambil env yang nggak boleh kosong. String kosong kita anggap "belum
// diset", bukan cuma yang nil, karena env var yang di-export tapi isinya kosong sama
// nggak bergunanya dengan yang memang absen — dua-duanya harus ditolak.
func requireEnv(key string) (string, error) {
	v := os.Getenv(key)
	if v == "" {
		return "", fmt.Errorf("env %s wajib diisi", key)
	}
	return v, nil
}

// getEnv: buat nilai opsional bertipe string. Balik ke fallback kalau env-nya kosong.
func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

// getEnvBool: kalau parsing-nya gagal, kita sengaja jatuh ke fallback, bukan bikin
// error. Tujuannya biar config tetap toleran — salah ketik di flag opsional nggak
// boleh sampai bikin boot gagal; mendingan pakai nilai default yang aman daripada crash.
func getEnvBool(key string, fallback bool) bool {
	if v := os.Getenv(key); v != "" {
		if b, err := strconv.ParseBool(v); err == nil {
			return b
		}
	}
	return fallback
}

// getEnvInt: sama kayak getEnvBool, nilai yang bukan angka diabaikan dan pakai fallback.
func getEnvInt(key string, fallback int) int {
	if v := os.Getenv(key); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			return n
		}
	}
	return fallback
}

// getEnvDuration: nerima format durasi ala Go (mis. "5m", "1h30m", "500ms").
// Kalau formatnya nggak valid, diabaikan dan balik ke fallback.
func getEnvDuration(key string, fallback time.Duration) time.Duration {
	if v := os.Getenv(key); v != "" {
		if d, err := time.ParseDuration(v); err == nil {
			return d
		}
	}
	return fallback
}
