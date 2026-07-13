## Why

Kita butuh satu boilerplate backend Go yang bisa di-clone berulang untuk membuat service baru dengan cepat, tanpa menyusun ulang pondasi (HTTP, database, observability) tiap kali. Boilerplate ini mengunci tiga prinsip non-negotiable — **security**, **performance**, **KISS** — langsung ke dalam struktur kode, sehingga tiap service turunan otomatis aman, cepat, dan mudah dipahami junior developer.

## What Changes

- Menyiapkan proyek Go modul `github.com/julles/go-boilerplate` dengan Echo v5 sebagai HTTP framework dan pgx (pgxpool) sebagai driver Postgres.
- Menetapkan **struktur folder per-fitur ala NestJS**: tiap modul (`internal/<fitur>/`) berisi `dto/`, `handler.go`, `service.go`, `repository.go`, dan `module.go` (wiring + daftar route). Disertai satu modul contoh (`example`) yang query ke tabel `merchant.merchants`.
- Menyediakan pondasi bersama di `internal/shared/`: config (env), koneksi pgxpool, client Redis, observability, response envelope, dan middleware.
- **Observability lengkap**: OpenTelemetry traces (`echo-opentelemetry` + OTLP exporter), metrics Prometheus (`echo-prometheus` di `/metrics`), dan structured log JSON via `slog` ke stdout (di-scrape ke Loki oleh infra, bukan push dari app).
- **Redis** (`go-redis/v9`) sebagai backing rate limiter (limit global lintas replica) dan cache helper (get/set + TTL).
- Rate limiter, request logger, recover, dan error handler yang tidak membocorkan error internal.
- Query database **ditulis manual** (parameterized `$1,$2`) — tanpa ORM/generator, untuk kontrol penuh dan performa.
- Script `rename.sh <module-path-baru>` untuk mengganti seluruh module path saat boilerplate di-clone jadi service baru.
- Graceful shutdown untuk menutup pool DB, Redis, dan tracer dengan rapi.

Non-goals (sengaja dikecualikan): migration DB (service memakai DB/tabel existing), automated testing (dipikirkan kemudian), implementasi message queue (client Redis disiapkan, queue menyusul).

## Capabilities

### New Capabilities
- `project-scaffold`: Struktur folder proyek, konvensi penamaan modul per-fitur ala NestJS, entrypoint `cmd/api`, dan script `rename.sh`.
- `http-server`: Bootstrap Echo v5, wiring modul, response envelope konsisten, error handler yang aman, dan graceful shutdown.
- `database-access`: Koneksi pgxpool, konvensi repository dengan SQL manual parameterized, dan tuning pool.
- `caching`: Client Redis bersama plus cache helper (get/set JSON + TTL) dengan pola cache-aside.
- `rate-limiting`: Rate limiter berbasis Redis agar limit berlaku global lintas replica.
- `observability`: Tracing OpenTelemetry (OTLP), metrics Prometheus (`/metrics`), dan structured logging JSON via `slog` dengan korelasi `trace_id`.
- `configuration`: Pemuatan konfigurasi dari environment variable (secret tidak di-hardcode), dengan `.env.example` sebagai acuan.

### Modified Capabilities
<!-- Tidak ada. Ini proyek baru; belum ada spec existing. -->

## Impact

- **Kode baru**: seluruh struktur `cmd/`, `internal/<fitur>/`, `internal/shared/`, `rename.sh`, `.env.example`, `go.mod`.
- **Dependencies**: `github.com/labstack/echo/v5`, `github.com/jackc/pgx/v5`, `github.com/redis/go-redis/v9`, `github.com/labstack/echo-prometheus`, `github.com/labstack/echo-opentelemetry`, OpenTelemetry SDK (OTLP exporter), `log/slog` (stdlib).
- **Infra terkait (di luar app)**: OTel Collector/Tempo untuk traces, Prometheus untuk scrape metrics, Alloy/Promtail → Loki untuk logs, instance Redis, dan Postgres.
- **Risiko**: paket `echo-prometheus` & `echo-opentelemetry` masih versi awal (`v0.0.x`) — API bisa berubah; versi harus di-pin.
