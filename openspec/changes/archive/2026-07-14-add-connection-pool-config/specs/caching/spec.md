## ADDED Requirements

### Requirement: Ukuran pool Redis dapat dikonfigurasi
Ukuran pool koneksi Redis SHALL dapat disetel dari environment: `REDIS_POOL_SIZE` dan `REDIS_MIN_IDLE_CONNS`. Bila tidak diset, dipakai default sensible (`REDIS_POOL_SIZE=10`, `REDIS_MIN_IDLE_CONNS=0`). Nilai ini diterapkan ke client Redis bersama saat startup.

#### Scenario: Pool Redis memakai nilai dari env
- **WHEN** `REDIS_POOL_SIZE` diset lalu aplikasi start
- **THEN** client Redis dibuat dengan pool size tersebut

#### Scenario: Default saat env kosong
- **WHEN** `REDIS_POOL_SIZE` tidak diset
- **THEN** client Redis memakai pool size default (10)
