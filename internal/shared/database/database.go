// Package database menyediakan koneksi pool Postgres via pgx.
package database

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/julles/go-boilerplate/internal/shared/config"
)

// NewPool membuat pgxpool dengan tuning pool dari config dan memastikan database
// dapat dijangkau (fail-fast).
func NewPool(ctx context.Context, url string, pool config.DBPoolConfig) (*pgxpool.Pool, error) {
	// Guard eksplisit: pgxpool tidak menolak min>max, jadi kita fail-fast di sini.
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
	// Nilai pool ditimpa dari env (config), bukan dari query string URL. Ini
	// menjadikan config sebagai satu-satunya sumber kebenaran tuning pool, sehingga
	// tuning tak perlu tersebar/terduplikasi di dalam string DSN tiap environment.
	cfg.MaxConns = pool.MaxConns               // batas atas koneksi ke Postgres
	cfg.MinConns = pool.MinConns               // koneksi hangat yang dijaga tetap terbuka
	cfg.MaxConnLifetime = pool.MaxConnLifetime // umur maksimum koneksi; mencegah koneksi basi & bantu rebalancing saat failover
	cfg.MaxConnIdleTime = pool.MaxConnIdleTime // koneksi menganggur ditutup agar resource server tidak tertahan

	p, err := pgxpool.NewWithConfig(ctx, cfg)
	if err != nil {
		return nil, fmt.Errorf("membuat pgx pool: %w", err)
	}
	// pgxpool bersifat lazy — NewWithConfig tidak benar-benar menyentuh server.
	// Ping memaksa satu koneksi nyata dibuka supaya kredensial/jaringan yang salah
	// ketahuan saat startup (fail-fast), bukan pada request pertama pengguna.
	if err := p.Ping(ctx); err != nil {
		p.Close() // hindari kebocoran pool bila ping gagal
		return nil, fmt.Errorf("ping database: %w", err)
	}
	return p, nil
}
