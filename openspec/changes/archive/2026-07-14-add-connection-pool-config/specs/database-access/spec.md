## ADDED Requirements

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
