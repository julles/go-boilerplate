// Package queue menyiapkan producer (client) dan consumer (server) Asynq.
// Koneksi Redis dibangun dari REDIS_URL yang sama dengan cache/rate limiter.
package queue

import (
	"fmt"

	"github.com/hibiken/asynq"
)

func redisOpt(url string) (asynq.RedisConnOpt, error) {
	opt, err := asynq.ParseRedisURI(url)
	if err != nil {
		return nil, fmt.Errorf("parse redis uri untuk queue: %w", err)
	}
	return opt, nil
}

// NewClient membuat producer untuk enqueue task. Tutup dengan Close saat shutdown.
func NewClient(url string) (*asynq.Client, error) {
	opt, err := redisOpt(url)
	if err != nil {
		return nil, err
	}
	return asynq.NewClient(opt), nil
}

// NewServer membuat consumer yang memproses task dengan konkurensi tertentu.
func NewServer(url string, concurrency int) (*asynq.Server, error) {
	opt, err := redisOpt(url)
	if err != nil {
		return nil, err
	}
	return asynq.NewServer(opt, asynq.Config{Concurrency: concurrency}), nil
}
