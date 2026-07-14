// Package queue nyiapin producer (client) dan consumer (server) Asynq.
// Koneksi Redis-nya dibangun dari REDIS_URL yang sama kayak cache dan rate limiter.
package queue

import (
	"fmt"

	"github.com/hibiken/asynq"
)

// redisOpt mustin parsing REDIS_URL jadi opsi koneksi Asynq di satu tempat. Dipakai
// bareng sama producer dan consumer biar dua-duanya dijamin nunjuk ke Redis yang sama,
// dan format URL-nya cuma diurai sekali di satu tempat — jadi nggak ada duplikasi atau drift.
func redisOpt(url string) (asynq.RedisConnOpt, error) {
	opt, err := asynq.ParseRedisURI(url)
	if err != nil {
		return nil, fmt.Errorf("parse redis uri untuk queue: %w", err)
	}
	return opt, nil
}

// NewClient bikin producer buat enqueue task. Jangan lupa Close pas shutdown biar
// koneksi Redis-nya nggak bocor.
func NewClient(url string) (*asynq.Client, error) {
	opt, err := redisOpt(url)
	if err != nil {
		return nil, err
	}
	return asynq.NewClient(opt), nil
}

// NewServer bikin consumer yang mroses task. concurrency nentuin berapa banyak task
// yang diproses paralel sama satu instance server — ini knob utama buat ngatur
// throughput vs beban ke DB/downstream.
func NewServer(url string, concurrency int) (*asynq.Server, error) {
	opt, err := redisOpt(url)
	if err != nil {
		return nil, err
	}
	return asynq.NewServer(opt, asynq.Config{Concurrency: concurrency}), nil
}
