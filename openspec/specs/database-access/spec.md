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
