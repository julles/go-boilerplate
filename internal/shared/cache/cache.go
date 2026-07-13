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
)

// NewClient membuat client Redis dan memastikan bisa dijangkau (fail-fast).
func NewClient(ctx context.Context, url string) (*redis.Client, error) {
	opt, err := redis.ParseURL(url)
	if err != nil {
		return nil, fmt.Errorf("parse redis url: %w", err)
	}
	rdb := redis.NewClient(opt)
	if err := rdb.Ping(ctx).Err(); err != nil {
		rdb.Close()
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
		if !errors.Is(err, redis.Nil) {
			slog.WarnContext(ctx, "cache get gagal", "key", key, "error", err)
		}
		return false
	}
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
	if err := c.rdb.Set(ctx, key, b, ttl).Err(); err != nil {
		slog.WarnContext(ctx, "cache set gagal", "key", key, "error", err)
	}
}
