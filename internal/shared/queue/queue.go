// Package queue menyiapkan producer (client) dan consumer (server) Asynq.
// Koneksi Redis dibangun dari REDIS_URL yang sama dengan cache/rate limiter.
package queue

import (
	"fmt"

	"github.com/hibiken/asynq"
)

// redisOpt memusatkan parsing REDIS_URL ke opsi koneksi Asynq. Dipakai bersama oleh
// producer dan consumer supaya keduanya dijamin menunjuk ke Redis yang sama dan
// format URL hanya diurai di satu tempat (hindari duplikasi/drift).
func redisOpt(url string) (asynq.RedisConnOpt, error) {
	opt, err := asynq.ParseRedisURI(url)
	if err != nil {
		return nil, fmt.Errorf("parse redis uri untuk queue: %w", err)
	}
	return opt, nil
}

// NewClient membuat producer untuk enqueue task. Tutup dengan Close saat shutdown
// agar koneksi Redis tidak bocor.
func NewClient(url string) (*asynq.Client, error) {
	opt, err := redisOpt(url)
	if err != nil {
		return nil, err
	}
	return asynq.NewClient(opt), nil
}

// NewServer membuat consumer yang memproses task. concurrency menentukan berapa
// banyak task diproses paralel oleh satu instance server — knob utama untuk
// throughput vs. beban DB/downstream.
func NewServer(url string, concurrency int) (*asynq.Server, error) {
	opt, err := redisOpt(url)
	if err != nil {
		return nil, err
	}
	return asynq.NewServer(opt, asynq.Config{Concurrency: concurrency}), nil
}
