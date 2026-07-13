## ADDED Requirements

### Requirement: Konfigurasi pool koneksi dari environment
Loader konfigurasi SHALL membaca parameter pool dari environment dan menyediakannya ke pembuat koneksi: `DB_MAX_CONNS`, `DB_MIN_CONNS`, `DB_MAX_CONN_LIFETIME`, `DB_MAX_CONN_IDLE_TIME` (Postgres) dan `REDIS_POOL_SIZE`, `REDIS_MIN_IDLE_CONNS` (Redis). Tiap parameter SHALL punya default sensible bila env tidak diset. Ketiga binary (api/worker/scheduler) SHALL memakai loader yang sama.

#### Scenario: Config memuat parameter pool
- **WHEN** aplikasi start dengan variabel pool diset
- **THEN** nilai pool terbaca dan diteruskan ke pembuat pgxpool/redis client

#### Scenario: Default terpakai
- **WHEN** variabel pool tidak diset
- **THEN** config mengembalikan default sensible tanpa error
