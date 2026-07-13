## 1. Inisialisasi proyek

- [x] 1.1 `go mod init github.com/julles/go-boilerplate` (Go 1.26)
- [x] 1.2 Tambah dependency: `echo/v5`, `pgx/v5` (+pgxpool), `go-redis/v9`, `echo-prometheus`, `echo-opentelemetry`, OTel SDK (OTLP exporter) — pin versi eksak
- [x] 1.3 Buat struktur folder: `cmd/api/`, `internal/shared/`, `internal/example/`
- [x] 1.4 Buat `.env.example` (port, DATABASE_URL, REDIS_URL, OTLP endpoint, rate limit) dan `.gitignore` (`.env`, binary)

## 2. Konfigurasi (env)

- [x] 2.1 `shared/config`: struct config + loader dari environment
- [x] 2.2 Fail-fast: keluar dengan pesan jelas bila env wajib (DATABASE_URL) kosong
- [x] 2.3 Pastikan secret hanya dari env, tidak ada nilai hardcode

## 3. Database (pgxpool)

- [x] 3.1 `shared/database`: buat `*pgxpool.Pool` dari config, dengan tuning pool dasar
- [x] 3.2 Ping saat startup — fail-fast bila koneksi awal gagal
- [x] 3.3 Sediakan fungsi close untuk graceful shutdown

## 4. Redis (cache + client bersama)

- [x] 4.1 `shared/cache`: buat client `go-redis/v9` dari config + close
- [x] 4.2 Cache helper `Get/Set` JSON dengan TTL (pola cache-aside)
- [x] 4.3 Perlakukan error Redis sebagai cache miss + log warning (tidak menggagalkan request)

## 5. Observability

- [x] 5.1 `shared/observability`: init `TracerProvider` + OTLP exporter (no-op bila endpoint kosong)
- [x] 5.2 Setup `slog` JSON handler ke stdout; sisipkan `trace_id` dari context
- [x] 5.3 Pastikan secret/kredensial tidak ikut tercatat di log
- [x] 5.4 Sediakan fungsi shutdown/flush tracer

## 6. HTTP core (shared)

- [x] 6.1 `shared/httpx`: response envelope (sukses & error) yang seragam
- [x] 6.2 Custom error handler: pesan generik ke client, detail hanya ke log; mapping error validasi → 400
- [x] 6.3 `shared/middleware`: pasang recover, request logger, `echo-opentelemetry`, `echo-prometheus` (+ handler `/metrics`)
- [x] 6.4 `shared/middleware`: rate limiter berbasis Redis (limit/window dari config), fail-open + warning saat Redis down

## 7. Modul contoh: example

- [x] 7.1 `example/dto`: `create_request.go` (+ validasi input), `query_params.go`, `response.go`
- [x] 7.2 `example/repository.go`: SQL manual parameterized (`$1,$2`) ke tabel `merchant.merchants`, scan manual, context di tiap query, hindari N+1
- [x] 7.3 `example/service.go`: logic bisnis (struct konkret, tanpa interface); contoh cache-aside pakai helper Redis
- [x] 7.4 `example/handler.go`: parse + validasi DTO, panggil service, balas via response envelope
- [x] 7.5 `example/module.go`: wiring `NewRepository → NewService → NewHandler` + daftar route

## 8. Entrypoint & shutdown

- [x] 8.1 `cmd/api/main.go`: load config → pgxpool → redis → observability → Echo + middleware
- [x] 8.2 Registrasi modul (`example.RegisterRoutes(e, db, rdb)`) — satu baris per modul
- [x] 8.3 Graceful shutdown pada SIGINT/SIGTERM: stop terima request baru, selesaikan in-flight, tutup pgxpool/redis/tracer

## 9. Script rename & finalisasi

- [x] 9.1 `rename.sh <module-path-baru>`: ganti module path di `go.mod` + semua import `.go`, lalu `go mod tidy`; tampilkan usage bila argumen kosong
- [x] 9.2 `go build ./...` dan jalankan server: verifikasi start, `/metrics` terekspos, endpoint example jalan
- [x] 9.3 Uji `rename.sh` di salinan: pastikan `go build ./...` tetap berhasil setelah rename
- [x] 9.4 Update `CLAUDE.md` bila ada konvensi baru yang perlu dicatat
