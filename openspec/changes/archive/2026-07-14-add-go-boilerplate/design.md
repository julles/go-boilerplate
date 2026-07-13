## Context

Boilerplate ini menjadi titik awal semua service backend Go kami. Sekali di-clone, developer langsung dapat pondasi HTTP, database, cache, dan observability yang sudah menerapkan tiga rule non-negotiable: **security**, **performance**, **KISS**. Tantangannya: menyeimbangkan kelengkapan (observability, rate limiting, cache) dengan kesederhanaan agar junior developer paham polanya dalam sekali baca.

Kondisi saat ini: repo kosong, hanya ada folder `openspec/`. Go 1.26, Echo v5 (v5.3.0) tersedia. Paket observability untuk Echo v5 (`echo-prometheus`, `echo-opentelemetry`) sudah rilis walau masih `v0.0.x`.

## Goals / Non-Goals

**Goals:**
- Struktur per-fitur ala NestJS yang konsisten dan mudah di-copy untuk menambah modul baru.
- Menambah fitur baru = copy satu folder + satu baris registrasi di `main.go`.
- Pondasi bersama (config, db, cache, observability, middleware) yang siap pakai lintas service.
- Semua keputusan default sudah aman & performant (parameterized query, pgxpool, log non-blocking, rate limit global).
- Clone → ganti nama modul via `rename.sh` → jalan.

**Non-Goals:**
- Migration DB (service memakai DB/tabel existing).
- Automated testing (menyusul; struktur struct konkret tidak menghalangi penambahan test nanti).
- Message queue (client Redis disiapkan; implementasi queue menyusul).
- Dependency injection framework (wire/fx) — wiring manual, eksplisit.
- Push log langsung dari app ke Loki (ditangani infra via scrape).

## Decisions

### 1. Struktur per-fitur ala NestJS di `internal/<fitur>/`
Tiap fitur satu package: `dto/`, `handler.go`, `service.go`, `repository.go`, `module.go`. Alur `handler → service → repository → pgxpool`.
- **Kenapa**: package Go sudah jadi namespace, jadi `example.Handler` cukup jelas tanpa prefix `example_` pada nama file. Satu folder = satu domain, gampang di-copy.
- **Alternatif ditolak**: layout by-layer (`handlers/`, `services/`, `repositories/`) — menyebar satu fitur ke banyak folder, susah dilacak.
- Diletakkan di `internal/` agar tidak bisa di-import project lain (proteksi bawaan Go).

### 2. Wiring manual di `module.go`, tanpa DI framework
Tiap `module.go` mengekspos `RegisterRoutes(e *echo.Echo, deps ...)` yang membangun `NewRepository → NewService → NewHandler` lalu mendaftarkan route. `main.go` cukup memanggil `example.RegisterRoutes(e, db, rdb)`.
- **Kenapa (KISS)**: junior langsung lihat alur konstruksi tanpa "magic". Nambah fitur = 1 baris di `main.go`.
- **Alternatif ditolak**: `google/wire` atau `uber/fx` — codegen/reflection yang menyembunyikan alur, melanggar KISS.

### 3. Struct konkret, tanpa interface
`Service` depend langsung ke `*Repository`, `Handler` ke `*Service`. Tidak ada interface selama hanya satu implementasi.
- **Kenapa (KISS)**: "no interface untuk satu implementasi". Lebih sedikit kode, lebih mudah dibaca.
- **Trade-off**: unit test dengan mock jadi kurang mudah. Diterima karena testing di-scope kemudian; saat butuh, interface bisa ditambahkan tepat di titik yang perlu.

### 4. pgx + pgxpool, SQL ditulis manual
Repository memakai `*pgxpool.Pool`, query ditulis tangan dengan placeholder `$1,$2`, scan manual.
- **Kenapa (performance + security)**: pgx lebih cepat dari `database/sql`; parameterized query mencegah SQL injection by default; developer melihat SQL apa adanya.
- **Alternatif ditolak**: ORM (GORM) — overhead & abstraksi berlebih; `sqlc` — menambah tool codegen (memilih zero-dependency demi kesederhanaan).

### 5. Observability: traces + metrics + logs terpisah
- **Traces**: `echo-opentelemetry` membuat span per request; `TracerProvider` + OTLP exporter di-setup di `shared/observability` (endpoint dari env).
- **Metrics**: `echo-prometheus` (`NewMiddleware` + `NewHandler` di `/metrics`) untuk di-scrape Prometheus.
- **Logs**: `slog` (stdlib) menulis JSON ke stdout dengan `trace_id` tersisip; Alloy/Promtail men-scrape ke Loki.
- **Kenapa (performance + KISS)**: app tidak push log ke Loki (menghindari blocking I/O di hot path & dependency ekstra); `slog` cukup, zero-dependency. Pemisahan traces/metrics/logs sesuai model observability standar.

### 6. Redis (`go-redis/v9`) untuk rate limiter global + cache helper
- **Rate limiter**: store berbasis Redis agar limit berlaku global lintas replica (bukan in-memory per-pod).
- **Cache**: helper tipis `Get/Set` JSON dengan TTL (pola cache-aside).
- **Kenapa (performance + security)**: rate limit yang benar hanya bisa dijaga secara global via store bersama. Client Redis juga menyiapkan jalan untuk queue di masa depan tanpa membangunnya sekarang.

### 7. Config dari environment, secret tidak di-hardcode
Semua konfigurasi (DB URL, Redis URL, OTLP endpoint, port, limit) dibaca dari env saat startup. `.env.example` sebagai acuan.
- **Kenapa (security)**: secret tidak masuk kode/VCS. Fail-fast bila env wajib kosong.

### 8. Error handler & response envelope
Satu custom error handler Echo mengubah error jadi respons JSON konsisten dan **tidak membocorkan detail internal** (stack/SQL) ke client; detail tetap masuk log. Response envelope seragam untuk sukses & error.
- **Kenapa (security + KISS)**: mencegah kebocoran informasi, format respons konsisten lintas service.

### 9. `rename.sh` untuk portabilitas modul
`./rename.sh <module-path-baru>` mengganti `github.com/julles/go-boilerplate` di `go.mod` dan seluruh import `.go`, lalu `go mod tidy`.
- **Kenapa**: clone → satu perintah → service baru siap. Import path pendek dipilih agar substitusi ringan.

## Risks / Trade-offs

- **`echo-prometheus` & `echo-opentelemetry` masih `v0.0.x`** → API bisa berubah antar rilis. Mitigasi: pin versi eksak di `go.mod`; kedua middleware tipis sehingga mudah diganti bila upstream berubah.
- **Struct konkret tanpa interface mempersulit unit test dengan mock** → Mitigasi: tambahkan interface hanya di titik yang benar-benar diuji saat testing mulai digarap; keputusan sadar, bukan kelalaian.
- **Rate limiter & cache bergantung pada Redis** → jika Redis down, perlu perilaku fallback yang jelas. Mitigasi: rate limiter fail-open atau fail-closed ditentukan eksplisit (default aman: batasi longgar/allow dengan log warning) — didetailkan di spec `rate-limiting`.
- **Tanpa migration di boilerplate** → skema DB diasumsikan sudah ada. Mitigasi: didokumentasikan sebagai non-goal; service existing yang memakainya sudah punya skema.
- **Boilerplate cenderung menumpuk fitur** → melanggar KISS seiring waktu. Mitigasi: modul contoh (`example`) dijaga minimal sebagai teladan pola, bukan showcase fitur. Repository-nya query ke tabel `merchant.merchants` sebagai contoh SQL manual schema-qualified.
