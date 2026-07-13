## ADDED Requirements

### Requirement: Dockerfile per binary
Boilerplate SHALL menyediakan Dockerfile multi-stage terpisah untuk tiap binary (`Dockerfile.api`, `Dockerfile.worker`, `Dockerfile.scheduler`). Image akhir SHALL kecil dan berjalan sebagai user non-root.

#### Scenario: Build image api
- **WHEN** `docker build -f Dockerfile.api .` dijalankan
- **THEN** menghasilkan image yang menjalankan binary API sebagai non-root

#### Scenario: Build image worker & scheduler
- **WHEN** `Dockerfile.worker` dan `Dockerfile.scheduler` di-build
- **THEN** masing-masing menghasilkan image yang menjalankan binary worker dan scheduler

### Requirement: docker-compose untuk pengembangan lokal
Boilerplate SHALL menyediakan `docker-compose.yml` yang menjalankan api, worker, scheduler, Postgres, dan Redis sekaligus, membaca konfigurasi dari `.env`.

#### Scenario: Menjalankan seluruh stack
- **WHEN** developer menjalankan `docker compose up`
- **THEN** api, worker, scheduler, Postgres, dan Redis menyala dan saling terhubung

### Requirement: Build context ramping
Boilerplate SHALL menyertakan `.dockerignore` agar file yang tidak perlu (mis. `.env`, artefak build) tidak masuk ke build context image.

#### Scenario: Secret tidak masuk image
- **WHEN** image di-build
- **THEN** file `.env` tidak ikut ter-copy ke dalam image
