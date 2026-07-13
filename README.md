# go-boilerplate

Boilerplate backend Go yang di-clone untuk membuat service baru. Fokus: **security**, **performance**, **KISS**.

Stack: Echo v5 · pgx (pgxpool) · Redis (go-redis) · OpenTelemetry · Prometheus · slog.

## Instalasi

Butuh Go 1.26+, Postgres, dan Redis.

```bash
# 1. Clone
git clone <repo-url> nama-service && cd nama-service

# 2. Ganti module path ke service baru
./rename.sh github.com/julles/nama-service

# 3. Siapkan environment
cp .env.example .env      # lalu isi DATABASE_URL, REDIS_URL, dll

# 4. Ambil dependency
go mod tidy

# 5. Jalankan
go run ./cmd/api        # HTTP server
go run ./cmd/worker     # consumer queue (background job)
go run ./cmd/scheduler  # cron (jalankan SATU instance)
```

Atau seluruh stack sekaligus (api + worker + scheduler + postgres + redis):

```bash
docker compose up --build
```

Env di-inject via runtime (12-factor), `.env` dipakai untuk tooling lokal (docker-compose `env_file`, direnv). Lihat `.env.example` untuk daftar variabel.

## Environment

| Variabel | Wajib | Default | Keterangan |
|---|---|---|---|
| `DATABASE_URL` | ✅ | — | Koneksi Postgres |
| `REDIS_URL` | ✅ | — | Koneksi Redis (rate limiter + cache) |
| `HTTP_PORT` | | `:8080` | Alamat listen |
| `SERVER_NAME` | | `go-boilerplate` | Nama service (tracing/log) |
| `OTLP_ENDPOINT` | | kosong | Endpoint OTLP; kosong = tracing mati |
| `RATE_LIMIT` | | `100` | Request per window |
| `RATE_WINDOW` | | `1m` | Panjang window |
| `WORKER_CONCURRENCY` | | `10` | Task paralel di worker |
| `DB_MAX_CONNS` | | `10` | Maks koneksi pool Postgres |
| `DB_MIN_CONNS` | | `2` | Min koneksi idle (warm) |
| `DB_MAX_CONN_LIFETIME` | | `1h` | Umur maks koneksi |
| `DB_MAX_CONN_IDLE_TIME` | | `30m` | Idle sebelum ditutup |
| `REDIS_POOL_SIZE` | | `10` | Ukuran pool Redis |
| `REDIS_MIN_IDLE_CONNS` | | `0` | Min koneksi idle Redis |

### Tuning pool per-binary

api, worker, dan scheduler proses terpisah — set ukuran pool berbeda lewat env masing-masing, tanpa ubah kode:

```
api:        DB_MAX_CONNS=20   # traffic tinggi
worker:     DB_MAX_CONNS=10   # ≈ WORKER_CONCURRENCY
scheduler:  DB_MAX_CONNS=2    # cron sesekali
```

> ⚠️ Jaga `Σ(DB_MAX_CONNS × replica)` lintas semua binary **≤ `max_connections`** Postgres.

## Struktur

```
cmd/api/            entrypoint
internal/
  <fitur>/          1 folder = 1 modul (dto, handler, service, repository, module.go)
  shared/           config, database, cache, observability, httpx, middleware
```

## Menambah modul baru

Lihat [docs/add-new-module.md](docs/add-new-module.md) — pola lengkap memakai modul `example` sebagai contoh.

## Worker & queue

Repo ini menghasilkan tiga binary (`cmd/api`, `cmd/worker`, `cmd/scheduler`) dari kode yang sama. Producer/consumer pakai Asynq (Redis), scheduler pakai `robfig/cron`. Detail: [docs/worker-queue.md](docs/worker-queue.md).

## Endpoint bawaan

- `GET /health` — health check (DB + Redis)
- `GET /metrics` — metrik Prometheus
- `/example` (modul `example`) — contoh CRUD + `POST /example/produce` (enqueue); hapus/ganti saat mulai service asli
