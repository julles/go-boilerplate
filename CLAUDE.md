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
3. **KISS.** Kode sesimple mungkin sampai junior backend developer paham. Jangan overengineer: no abstraksi spekulatif, no interface untuk satu implementasi, no config untuk nilai yang tak pernah berubah. Tambah kompleksitas hanya saat benar-benar dibutuhkan.

Saat security/performance bentrok dengan "simpel", security dan performance menang — tapi cari cara paling simpel yang tetap aman & cepat.
