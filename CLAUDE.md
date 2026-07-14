# CLAUDE.md

Go backend boilerplate, dipakai ulang untuk banyak project.

## Stack
- **Echo v5** — HTTP framework
- **pgx** (postgres) — driver DB langsung, bukan `database/sql`, untuk performa
- **go-redis/v9** — rate limiter global + cache helper
- **go-playground/validator** — validasi DTO via struct tag (`c.Validate`, cara resmi Echo)
- **OpenTelemetry** (`echo-opentelemetry` + OTLP), **Prometheus** (`echo-prometheus`, `/metrics`), **slog** JSON ke stdout (Loki via infra)

## Konvensi
- Semua config dari env (secret tak di-hardcode). `.env.example` = daftar semua env var yang dibutuhkan, tanpa secret asli; `.env` tidak di-commit.

## Struktur modul (WAJIB ikut `internal/example`)
- Setiap modul/fitur baru WAJIB mengikuti struktur & pola `internal/example` sebagai acuan:
  `dto/` · `handler.go` · `service.go` · `repository.go` · `module.go` (+ `tasks.go`/`schedule.go` bila butuh worker/cron).
- Alur tetap: `handler → service → repository`. Nama file tanpa prefix nama fitur (`handler.go`, bukan `merchant_handler.go`).
- Wiring lewat `RegisterRoutes`/`RegisterTasks`/`RegisterSchedule`, dipanggil satu baris di `cmd/*`.
- Detail langkah lihat `docs/add-new-module.md`.

## Komentar kode (WAJIB)
- Setiap kode yang di-generate WAJIB diberi komentar **detail**: per baris / per blok logika, jelaskan maksud & alurnya.
- Komentar pakai **Bahasa Indonesia**, ditujukan agar junior backend developer langsung paham.
- Jelaskan "kenapa"-nya, bukan cuma "apa"-nya. Berlaku di semua session & mesin.

## Git commit
- JANGAN tambahkan trailer `Co-Authored-By` (mis. Claude/AI) di commit message. Commit hanya atas nama akun developer.
- Berlaku di semua session & mesin — jangan pernah menyisipkan co-author AI apa pun.

## Rules (urutan = prioritas, non-negotiable)
1. **Security — harga mati.** Validasi input di setiap trust boundary, query pakai parameterized (jangan string concat), jangan bocorkan error internal ke response, secret dari env bukan hardcode.
2. **Performance — harga mati.** Pilih pola yang efisien; pgx pooling, hindari N+1, jangan alokasi/query yang tak perlu.
3. **Concurrency-safe — harga mati.** Project ini high-traffic, banyak request jalan barengan. Setiap kode (dan setiap spec/design) WAJIB memperhitungkan race condition: shared state diproteksi (mutex/atomic, atau hindari shared state sama sekali), operasi read-modify-write ke DB pakai transaksi/locking yang tepat (mis. `SELECT ... FOR UPDATE`, optimistic lock via versi, atau `INSERT ... ON CONFLICT`), jangan andalkan "cek dulu baru tulis" tanpa proteksi (TOCTOU), dan pastikan idempotensi di jalur yang bisa dobel (retry queue, dobel-submit). Kalau ragu suatu kode aman dari data race, uji dengan `go test -race`.
4. **KISS.** Kode sesimple mungkin sampai junior backend developer paham. Jangan overengineer: no abstraksi spekulatif, no interface untuk satu implementasi, no config untuk nilai yang tak pernah berubah. Tambah kompleksitas hanya saat benar-benar dibutuhkan.

Saat security/performance/concurrency-safety bentrok dengan "simpel", yang tiga menang — tapi cari cara paling simpel yang tetap aman, cepat, dan bebas race.
