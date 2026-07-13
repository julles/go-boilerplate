## Why

Connection pooling sudah ada (pgxpool untuk Postgres, go-redis untuk Redis), tetapi nilainya masih memakai default library — dan default pgx `MaxConns` hanya `max(4, jumlah CPU)`, terlalu kecil untuk API throughput tinggi. Nilai pool saat ini tidak bisa disetel lewat konfigurasi. Karena performance adalah rule non-negotiable dan sekarang ada tiga binary (api/worker/scheduler) yang masing-masing membuka pool sendiri, ukuran pool perlu bisa disetel eksplisit per service agar total koneksi ke Postgres/Redis terkendali.

## What Changes

- Menambah **konfigurasi pool eksplisit dari environment** (bukan lewat query string URL) untuk Postgres dan Redis:
  - Postgres (pgxpool): `DB_MAX_CONNS`, `DB_MIN_CONNS`, `DB_MAX_CONN_LIFETIME`, `DB_MAX_CONN_IDLE_TIME`.
  - Redis (go-redis): `REDIS_POOL_SIZE`, `REDIS_MIN_IDLE_CONNS`.
- `database.NewPool` beralih dari `pgxpool.New(url)` ke `pgxpool.ParseConfig(url)` → set field pool dari config → `pgxpool.NewWithConfig`.
- `cache.NewClient` mengeset `redis.Options` (`PoolSize`, `MinIdleConns`) dari config setelah `redis.ParseURL`.
- Signature `NewPool`/`NewClient` menerima nilai pool dari config; ketiga binary memakai `config.Load` yang sama sehingga tiap binary bisa disetel beda lewat env-nya sendiri (tuning per-binary tanpa kode tambahan).
- Default sensible yang lebih tinggi dari default pgx: `DB_MAX_CONNS=10`, `DB_MIN_CONNS=2`, `DB_MAX_CONN_LIFETIME=1h`, `DB_MAX_CONN_IDLE_TIME=30m`, `REDIS_POOL_SIZE=10`, `REDIS_MIN_IDLE_CONNS=0`.
- Dokumentasi: `.env.example`, README, dan catatan agar total koneksi lintas binary/replica tidak melebihi `max_connections` Postgres.

Non-goals: knob lanjutan (MaxIdleConns, PoolTimeout, HealthCheckPeriod, dsb) — pakai default library; menambahkannya bila ada kebutuhan nyata.

## Capabilities

### Modified Capabilities
- `database-access`: Menambah requirement bahwa ukuran pool pgx dapat dikonfigurasi dari environment dengan default sensible.
- `caching`: Menambah requirement bahwa ukuran pool Redis dapat dikonfigurasi dari environment.
- `configuration`: Menambah env baru untuk parameter pool DB dan Redis.

## Impact

- **Kode berubah**: `internal/shared/config/config.go` (+6 field + loader), `internal/shared/database/database.go` (ParseConfig + set pool), `internal/shared/cache/cache.go` (set redis.Options), pemanggil `NewPool`/`NewClient` di `cmd/api`, `cmd/worker`, `cmd/scheduler`.
- **Konfigurasi**: `.env.example` bertambah 6 variabel; README/docs menjelaskan tuning per-binary.
- **Dependencies**: tidak ada dependency baru (memakai API pgxpool & go-redis yang sudah ada).
- **Risiko**: total koneksi = Σ(pool × replica) lintas 3 binary bisa melebihi `max_connections` Postgres bila disetel terlalu besar — didokumentasikan.
