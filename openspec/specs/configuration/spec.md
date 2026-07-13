# configuration Specification

## Requirements

### Requirement: Konfigurasi dari environment
Boilerplate SHALL memuat seluruh konfigurasi (port HTTP, koneksi database, koneksi Redis, endpoint OTLP, parameter rate limit) dari environment variable saat startup. Secret MUST NOT di-hardcode di dalam kode.

#### Scenario: Memuat config dari env
- **WHEN** aplikasi start dengan env yang lengkap
- **THEN** semua nilai konfigurasi terbaca dari environment

### Requirement: Fail-fast bila config wajib kosong
Aplikasi SHALL berhenti dengan error yang jelas saat startup bila environment variable wajib (mis. koneksi database) tidak diset, alih-alih menyala dengan nilai default yang tidak aman.

#### Scenario: Env wajib hilang
- **WHEN** variable koneksi database tidak diset
- **THEN** aplikasi keluar dengan pesan yang menyebut variable mana yang kurang

### Requirement: File .env.example sebagai acuan
Boilerplate SHALL menyertakan `.env.example` yang mendaftar semua variable yang dibutuhkan dengan nilai contoh (tanpa secret asli). File `.env` asli MUST NOT di-commit.

#### Scenario: Developer menyiapkan service baru
- **WHEN** developer menyalin `.env.example` menjadi `.env`
- **THEN** ia melihat seluruh variable yang perlu diisi untuk menjalankan service

### Requirement: Konfigurasi worker dan queue
Konfigurasi SHALL mendukung parameter worker dari environment: konkurensi consumer (`WORKER_CONCURRENCY`). Koneksi queue SHALL memakai `REDIS_URL` yang sama dengan cache/rate limiter. Semua binary (api/worker/scheduler) SHALL memakai loader konfigurasi yang sama.

#### Scenario: Konkurensi worker dari env
- **WHEN** `WORKER_CONCURRENCY` diset
- **THEN** worker memproses hingga sejumlah task itu secara paralel; bila tidak diset dipakai default yang wajar

#### Scenario: Binary berbagi konfigurasi
- **WHEN** worker atau scheduler dijalankan
- **THEN** keduanya memuat konfigurasi (termasuk `.env`) dengan mekanisme yang sama seperti api

### Requirement: Konfigurasi pool koneksi dari environment
Loader konfigurasi SHALL membaca parameter pool dari environment dan menyediakannya ke pembuat koneksi: `DB_MAX_CONNS`, `DB_MIN_CONNS`, `DB_MAX_CONN_LIFETIME`, `DB_MAX_CONN_IDLE_TIME` (Postgres) dan `REDIS_POOL_SIZE`, `REDIS_MIN_IDLE_CONNS` (Redis). Tiap parameter SHALL punya default sensible bila env tidak diset. Ketiga binary (api/worker/scheduler) SHALL memakai loader yang sama.

#### Scenario: Config memuat parameter pool
- **WHEN** aplikasi start dengan variabel pool diset
- **THEN** nilai pool terbaca dan diteruskan ke pembuat pgxpool/redis client

#### Scenario: Default terpakai
- **WHEN** variabel pool tidak diset
- **THEN** config mengembalikan default sensible tanpa error
