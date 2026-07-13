## Context

Pooling sudah aktif tapi memakai default library. Default pgx `MaxConns = max(4, jumlah CPU)` kecil untuk API sibuk, dan tak ada cara menyetelnya kecuali menyisipkan query param ke `DATABASE_URL`. Tim memilih konfigurasi via env eksplisit (bukan URL) agar gamblang dibaca junior dan konsisten dengan pola config yang sudah ada. Ada tiga binary (api/worker/scheduler); masing-masing proses terpisah dengan env sendiri.

## Goals / Non-Goals

**Goals:**
- Ukuran pool DB & Redis disetel lewat env eksplisit, dengan default sensible (> default pgx).
- Tuning per-binary tanpa kode tambahan (memanfaatkan env per-proses).
- Perubahan minimal, mudah dipahami.

**Non-Goals:**
- Knob lanjutan (MaxIdleConns, PoolTimeout, HealthCheckPeriod, ConnMaxLifetime Redis) — pakai default.
- Konfigurasi pool via query string URL (sengaja tidak dipakai).
- Pool terpisah per-fitur (satu pool DB & satu client Redis bersama tetap).

## Decisions

### 1. Enam knob env
```
DB_MAX_CONNS=10             MinConns pgx → MaxConns
DB_MIN_CONNS=2              koneksi idle minimum (warm)
DB_MAX_CONN_LIFETIME=1h     recycle koneksi
DB_MAX_CONN_IDLE_TIME=30m   tutup koneksi idle
REDIS_POOL_SIZE=10          go-redis PoolSize
REDIS_MIN_IDLE_CONNS=0      go-redis MinIdleConns
```
- **Kenapa**: mencakup parameter yang paling sering di-tune (max/min + lifetime/idle) tanpa berlebihan. Nilai duration pakai format Go (`time.ParseDuration`), sudah ada helper `getEnvDuration`.

### 2. pgxpool via ParseConfig → NewWithConfig
`database.NewPool` berubah:
```
cfg, _ := pgxpool.ParseConfig(url)   // tetap hormati DATABASE_URL
cfg.MaxConns        = maxConns
cfg.MinConns        = minConns
cfg.MaxConnLifetime = maxConnLifetime
cfg.MaxConnIdleTime = maxConnIdleTime
pool, _ := pgxpool.NewWithConfig(ctx, cfg)
```
- **Kenapa**: `ParseConfig` tetap membaca URL (host/cred/sslmode), lalu field pool ditimpa dari env. `HealthCheckPeriod` dibiarkan default.
- **Alternatif ditolak**: menyisipkan `pool_max_conns` ke URL (permintaan user: jangan via URL).

### 3. redis.Options di-set setelah ParseURL
`cache.NewClient` berubah:
```
opt, _ := redis.ParseURL(url)
opt.PoolSize     = poolSize
opt.MinIdleConns = minIdleConns
rdb := redis.NewClient(opt)
```
- **Kenapa**: `ParseURL` menyediakan koneksi dasar; field pool ditimpa dari env.

### 4. Nilai pool dibawa lewat config, signature berubah tipis
`NewPool`/`NewClient` menerima nilai pool dari `config.Config` (bukan baca env sendiri) agar konfigurasi terpusat dan mudah diuji. Contoh: `NewPool(ctx, cfg)` atau `NewPool(ctx, cfg.DatabaseURL, cfg.DBPool)`.
- **Kenapa (KISS)**: satu sumber konfigurasi (config), fungsi pembuat pool tetap fokus.

### 5. Tuning per-binary lewat env proses
Karena api/worker/scheduler proses terpisah, tiap deployment set env sendiri:
```
api:       DB_MAX_CONNS=20
worker:    DB_MAX_CONNS=10   (≈ WORKER_CONCURRENCY)
scheduler: DB_MAX_CONNS=2
```
- **Kenapa (performance)**: profil beban tiap binary beda; satu codebase, disetel per-deployment.

## Risks / Trade-offs

- **Total koneksi lintas binary/replica bisa melebihi `max_connections` Postgres** → Mitigasi: dokumentasikan rumus `Σ(pool × replica) ≤ max_connections`, default konservatif (10).
- **Default dinaikkan dari 4 → 10** → sedikit lebih banyak koneksi idle saat start (dengan MinConns=2). Diterima; lebih aman untuk throughput dan mudah diturunkan via env.
- **Salah isi env (mis. MIN > MAX)** → pgxpool TIDAK menolak sendiri (diverifikasi: server tetap start). Mitigasi: `NewPool` menambah guard eksplisit (MaxConns >= 1, 0 <= MinConns <= MaxConns) sehingga gagal cepat dengan error jelas saat boot.
