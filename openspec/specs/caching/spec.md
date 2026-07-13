# caching Specification

## Requirements

### Requirement: Client Redis bersama
Boilerplate SHALL membuat satu client Redis bersama (`go-redis/v9`) saat startup dari konfigurasi env, dan membagikannya ke modul yang membutuhkan.

#### Scenario: Startup dengan Redis
- **WHEN** aplikasi start dan Redis dapat dijangkau
- **THEN** client Redis terbentuk dan dipakai ulang oleh cache helper dan rate limiter

### Requirement: Cache helper get/set dengan TTL
Boilerplate SHALL menyediakan helper cache tipis untuk menyimpan dan mengambil nilai JSON dengan TTL, mendukung pola cache-aside.

#### Scenario: Cache hit
- **WHEN** kunci diminta dan masih ada di Redis (belum kadaluarsa)
- **THEN** helper mengembalikan nilai dari cache tanpa mengakses database

#### Scenario: Cache miss lalu isi
- **WHEN** kunci tidak ada di cache
- **THEN** pemanggil mengambil dari sumber asli lalu menyimpannya ke cache dengan TTL yang ditentukan

### Requirement: Kegagalan Redis tidak mematikan request
Kegagalan operasi cache (Redis tidak tersedia) MUST NOT menggagalkan request. Helper SHALL memperlakukan error cache seperti cache miss dan mencatat warning.

#### Scenario: Redis down saat baca cache
- **WHEN** operasi baca cache gagal karena Redis tidak tersedia
- **THEN** helper mengembalikan cache miss, request tetap dilayani dari sumber asli, dan warning tercatat di log

### Requirement: Ukuran pool Redis dapat dikonfigurasi
Ukuran pool koneksi Redis SHALL dapat disetel dari environment: `REDIS_POOL_SIZE` dan `REDIS_MIN_IDLE_CONNS`. Bila tidak diset, dipakai default sensible (`REDIS_POOL_SIZE=10`, `REDIS_MIN_IDLE_CONNS=0`). Nilai ini diterapkan ke client Redis bersama saat startup.

#### Scenario: Pool Redis memakai nilai dari env
- **WHEN** `REDIS_POOL_SIZE` diset lalu aplikasi start
- **THEN** client Redis dibuat dengan pool size tersebut

#### Scenario: Default saat env kosong
- **WHEN** `REDIS_POOL_SIZE` tidak diset
- **THEN** client Redis memakai pool size default (10)
