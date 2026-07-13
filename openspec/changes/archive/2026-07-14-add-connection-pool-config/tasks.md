## 1. Konfigurasi

- [x] 1.1 `shared/config`: tambah field pool DB (`DBMaxConns`, `DBMinConns`, `DBMaxConnLifetime`, `DBMaxConnIdleTime`) + Redis (`RedisPoolSize`, `RedisMinIdleConns`)
- [x] 1.2 Baca dari env dengan default: DB max=10, min=2, lifetime=1h, idle=30m; Redis pool=10, min_idle=0
- [x] 1.3 Pakai helper `getEnvInt`/`getEnvDuration` yang sudah ada

## 2. Pool Postgres (pgxpool)

- [x] 2.1 `shared/database`: ganti `pgxpool.New` → `ParseConfig(url)` lalu set `MaxConns`, `MinConns`, `MaxConnLifetime`, `MaxConnIdleTime` dari config
- [x] 2.2 Buat pool via `pgxpool.NewWithConfig`, tetap ping fail-fast
- [x] 2.3 Sesuaikan signature `NewPool` untuk menerima nilai pool dari config

## 3. Pool Redis (go-redis)

- [x] 3.1 `shared/cache`: setelah `redis.ParseURL`, set `opt.PoolSize` dan `opt.MinIdleConns` dari config
- [x] 3.2 Sesuaikan signature `NewClient` untuk menerima nilai pool dari config

## 4. Pemanggil (3 binary)

- [x] 4.1 Update pemanggil `NewPool`/`NewClient` di `cmd/api`, `cmd/worker`, `cmd/scheduler` mengikuti signature baru

## 5. Dokumentasi & verifikasi

- [x] 5.1 `.env.example`: tambah 6 variabel pool + komentar singkat
- [x] 5.2 README/docs: jelaskan tuning per-binary + peringatan `Σ(pool × replica) ≤ max_connections`
- [x] 5.3 Verifikasi: `go build ./...`; jalankan dengan `DB_MAX_CONNS`/`REDIS_POOL_SIZE` diset, pastikan start OK; uji env tidak valid (min>max) → gagal cepat
