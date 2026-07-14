// Package database menyediakan connection pool Postgres lewat pgx.
package database

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/julles/go-boilerplate/internal/shared/config"
)

// NewPool bikin pgxpool dengan tuning pool dari config, lalu memastikan database-nya
// beneran bisa dijangkau (fail-fast).
func NewPool(ctx context.Context, url string, pool config.DBPoolConfig) (*pgxpool.Pool, error) {
	// Guard eksplisit: pgxpool sendiri nggak nolak kalau min > max, jadi kita yang
	// fail-fast di sini biar salah konfigurasi langsung ketahuan.
	if pool.MaxConns < 1 {
		return nil, fmt.Errorf("DB_MAX_CONNS harus >= 1, dapat %d", pool.MaxConns)
	}
	if pool.MinConns < 0 || pool.MinConns > pool.MaxConns {
		return nil, fmt.Errorf("DB_MIN_CONNS (%d) harus antara 0 dan DB_MAX_CONNS (%d)", pool.MinConns, pool.MaxConns)
	}

	cfg, err := pgxpool.ParseConfig(url)
	if err != nil {
		return nil, fmt.Errorf("parse database url: %w", err)
	}
	// Nilai pool sengaja ditimpa dari config (env), bukan dari query string di URL.
	// Dengan begini config jadi satu-satunya source of truth buat tuning pool, jadi
	// tuning-nya nggak perlu kesebar dan keduplikasi di dalam DSN tiap environment.
	cfg.MaxConns = pool.MaxConns               // batas atas jumlah koneksi ke Postgres
	cfg.MinConns = pool.MinConns               // koneksi hangat yang dijaga biar tetap kebuka
	cfg.MaxConnLifetime = pool.MaxConnLifetime // umur maksimum koneksi; biar koneksi nggak jadi stale sekaligus bantu rebalancing pas failover
	cfg.MaxConnIdleTime = pool.MaxConnIdleTime // koneksi yang nganggur ditutup supaya resource server nggak ketahan percuma

	p, err := pgxpool.NewWithConfig(ctx, cfg)
	if err != nil {
		return nil, fmt.Errorf("membuat pgx pool: %w", err)
	}
	// pgxpool itu lazy — NewWithConfig belum benar-benar nyentuh server. Ping kita
	// panggil untuk maksa satu koneksi nyata kebuka, jadi kalau kredensial atau
	// jaringannya salah langsung ketahuan pas startup (fail-fast), bukan pas request
	// pertama dari user baru meledak.
	if err := p.Ping(ctx); err != nil {
		p.Close() // tutup pool biar nggak bocor kalau ping gagal
		return nil, fmt.Errorf("ping database: %w", err)
	}
	return p, nil
}
