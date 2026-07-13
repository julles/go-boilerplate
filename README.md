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
go run ./cmd/api
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

## Struktur

```
cmd/api/            entrypoint
internal/
  <fitur>/          1 folder = 1 modul (dto, handler, service, repository, module.go)
  shared/           config, database, cache, observability, httpx, middleware
```

## Menambah modul baru

Lihat [docs/add-new-module.md](docs/add-new-module.md) — pola lengkap memakai modul `example` sebagai contoh.

## Endpoint bawaan

- `GET /metrics` — metrik Prometheus
- `/merchants` (modul `example`) — contoh CRUD sebagian; hapus/ganti saat mulai service asli
