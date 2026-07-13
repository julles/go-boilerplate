## ADDED Requirements

### Requirement: Konfigurasi worker dan queue
Konfigurasi SHALL mendukung parameter worker dari environment: konkurensi consumer (`WORKER_CONCURRENCY`). Koneksi queue SHALL memakai `REDIS_URL` yang sama dengan cache/rate limiter. Semua binary (api/worker/scheduler) SHALL memakai loader konfigurasi yang sama.

#### Scenario: Konkurensi worker dari env
- **WHEN** `WORKER_CONCURRENCY` diset
- **THEN** worker memproses hingga sejumlah task itu secara paralel; bila tidak diset dipakai default yang wajar

#### Scenario: Binary berbagi konfigurasi
- **WHEN** worker atau scheduler dijalankan
- **THEN** keduanya memuat konfigurasi (termasuk `.env`) dengan mekanisme yang sama seperti api
