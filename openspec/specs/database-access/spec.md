# database access Specification

## Requirements

### Requirement: Koneksi pgxpool
Boilerplate SHALL membuat satu `*pgxpool.Pool` bersama saat startup dari konfigurasi env, dan membagikannya ke repository tiap modul. Aplikasi MUST gagal cepat (fail-fast) bila koneksi awal ke database gagal.

#### Scenario: Startup dengan DB sehat
- **WHEN** aplikasi start dan database dapat dijangkau
- **THEN** pgxpool terbentuk dan dipakai ulang oleh semua repository

#### Scenario: Database tidak dapat dijangkau
- **WHEN** aplikasi start tetapi koneksi awal ke database gagal
- **THEN** aplikasi berhenti dengan error yang jelas, tidak menyala dalam keadaan setengah jadi

### Requirement: Query manual dengan parameter
Repository SHALL menulis SQL secara manual menggunakan placeholder parameter (`$1, $2, ...`). Input dari user MUST NOT digabung ke string SQL secara langsung (mencegah SQL injection).

#### Scenario: Query dengan input user
- **WHEN** repository mengambil data berdasarkan input user
- **THEN** nilai input dikirim sebagai parameter query, bukan dirangkai ke string SQL

### Requirement: Hindari N+1 query
Repository SHALL mengambil data yang berulang dalam satu query (mis. `WHERE id = ANY($1)`) alih-alih melakukan query di dalam loop.

#### Scenario: Mengambil data untuk banyak entitas
- **WHEN** service butuh data untuk sekumpulan id
- **THEN** repository menjalankan satu query untuk semua id, bukan satu query per id

### Requirement: Konteks pada setiap query
Setiap operasi database SHALL menerima `context.Context` dari request agar mendukung cancellation, timeout, dan propagasi trace.

#### Scenario: Request dibatalkan
- **WHEN** client memutus koneksi saat query berjalan
- **THEN** context ter-cancel dan query dihentikan

### Requirement: Ukuran pool pgx dapat dikonfigurasi
Ukuran pool koneksi Postgres SHALL dapat disetel dari environment: `DB_MAX_CONNS`, `DB_MIN_CONNS`, `DB_MAX_CONN_LIFETIME`, `DB_MAX_CONN_IDLE_TIME`. Bila variabel tidak diset, dipakai default sensible (`DB_MAX_CONNS=10`, `DB_MIN_CONNS=2`, `DB_MAX_CONN_LIFETIME=1h`, `DB_MAX_CONN_IDLE_TIME=30m`). Konfigurasi ini MUST diambil dari env, bukan dari query string `DATABASE_URL`.

#### Scenario: Pool memakai nilai dari env
- **WHEN** `DB_MAX_CONNS` dan `DB_MIN_CONNS` diset lalu aplikasi start
- **THEN** pgxpool dibuat dengan nilai max/min tersebut

#### Scenario: Default saat env kosong
- **WHEN** variabel pool tidak diset
- **THEN** pool memakai default sensible (mis. max 10, min 2), bukan default pgx yang lebih kecil

#### Scenario: Konfigurasi tidak valid
- **WHEN** nilai pool tidak konsisten (mis. min lebih besar dari max)
- **THEN** aplikasi gagal cepat saat start dengan error yang jelas

### Requirement: Tuning pool per-binary
Karena api, worker, dan scheduler adalah proses terpisah, tiap binary SHALL dapat memakai ukuran pool berbeda hanya dengan menyetel environment masing-masing, tanpa perubahan kode.

#### Scenario: Worker dan api pool berbeda
- **WHEN** `DB_MAX_CONNS` diset berbeda pada deployment api dan worker
- **THEN** masing-masing binary membuat pool sesuai env-nya sendiri
