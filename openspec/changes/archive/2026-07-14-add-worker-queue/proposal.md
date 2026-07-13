## Why

Boilerplate saat ini hanya menghasilkan satu binary (API). Banyak service butuh **pekerjaan latar** (background job) dan **tugas terjadwal** (cron) — kirim notifikasi, rekap, pembersihan data — yang tidak cocok dijalankan di dalam request HTTP. Kita perlu pola producer/consumer berbasis Redis (padanan BullMQ di Node) plus scheduler, yang di-reuse konsisten lintas service dan tetap simpel untuk junior developer.

## What Changes

- Menambah **dua entrypoint baru**: `cmd/worker` (consumer Asynq) dan `cmd/scheduler` (cron `robfig/cron`), di samping `cmd/api` yang sudah ada. Total tiga binary dari `internal/` yang sama.
- **Queue producer/consumer** memakai **Asynq** (`hibiken/asynq`, berbasis `go-redis/v9` — sama dengan stack cache). Asynq dipakai murni untuk enqueue → consume real-time, **tanpa** scheduler bawaan Asynq.
- **Scheduler terpisah** memakai **`robfig/cron/v3`** di `cmd/scheduler`. Tiap entry cron adalah fungsi Go bebas: bisa "select rentang data → proses langsung" tanpa lewat queue. Dijalankan sebagai satu replica (bukan flag).
- Tiap modul fitur menambah dua file mengikuti pola per-modul: `tasks.go` (handler consumer, dipakai `cmd/worker`) dan `schedule.go` (entry cron, dipakai `cmd/scheduler`). `module.go` yang ada bertambah dependency client queue untuk enqueue.
- Modul contoh `example` mendapat:
  - `POST /example/produce` — enqueue satu message ke queue (producer).
  - Consumer yang memproses message tersebut (`cmd/worker`).
  - Contoh cron di scheduler: select rentang `merchant.merchants` berdasarkan `created_at` lalu proses (log jumlah baris).
- **Docker**: `Dockerfile.api`, `Dockerfile.worker`, `Dockerfile.scheduler` (multi-stage, image kecil, non-root) dan `docker-compose.yml` yang menyalakan api + worker + scheduler + postgres + redis untuk pengembangan lokal. Ditambah `.dockerignore`.
- Setup Asynq (client + server) di `internal/shared/queue`, dan wrapper `robfig/cron` di `internal/shared/scheduler`.
- Env baru untuk konkurensi worker dan enable/disable, ditambahkan ke `.env.example`.
- Dokumentasi `docs/worker-queue.md` dan penambahan bagian task/cron di `docs/add-new-module.md`.

Non-goals: dashboard `asynqmon` (infra bisa jalankan sendiri), dead-letter handling kustom (pakai retry bawaan Asynq), dan message queue selain Redis.

## Capabilities

### New Capabilities
- `background-jobs`: Producer/consumer berbasis Asynq — enqueue message dari API dan pemrosesan oleh worker, dengan retry dan konkurensi.
- `scheduling`: Tugas terjadwal berbasis `robfig/cron` di binary scheduler terpisah, mendukung pola "select rentang → proses langsung".
- `deployment`: Dockerfile per binary (api/worker/scheduler) dan `docker-compose.yml` untuk menjalankan seluruh service beserta Postgres & Redis.

### Modified Capabilities
- `project-scaffold`: Struktur bertambah `cmd/worker`, `cmd/scheduler`, serta konvensi file `tasks.go` dan `schedule.go` per modul.
- `configuration`: Env baru untuk worker (konkurensi, enable) dan scheduler.

## Impact

- **Kode baru**: `cmd/worker/`, `cmd/scheduler/`, `internal/shared/queue/`, `internal/shared/scheduler/`, `internal/example/tasks.go`, `internal/example/schedule.go`, `internal/example/dto/produce_request.go`, `Dockerfile.*`, `docker-compose.yml`, `.dockerignore`, `docs/worker-queue.md`.
- **Kode berubah**: `internal/example/module.go` & `handler.go` (endpoint produce + dependency client queue), `cmd/api/main.go` (inject client queue), `internal/shared/config` (env baru), `.env.example`, `README.md`, `docs/add-new-module.md`.
- **Dependencies**: `github.com/hibiken/asynq`, `github.com/robfig/cron/v3` (keduanya sudah memakai `go-redis/v9`).
- **Infra terkait**: instance Redis dipakai bersama untuk queue (selain cache & rate limiter). Scheduler harus dijalankan single-replica.
