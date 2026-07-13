## Context

Boilerplate menghasilkan satu binary API. Service nyata butuh background job (real-time, via queue) dan tugas terjadwal (cron). Kita tambah dua entrypoint yang berbagi `internal/` yang sama, dengan pemisahan tegas: Asynq untuk queue produce/consume, `robfig/cron` untuk penjadwalan. Redis sudah ada di stack (cache + rate limiter), dan Asynq memakai `go-redis/v9` yang sama — tidak menambah driver baru.

## Goals / Non-Goals

**Goals:**
- Producer/consumer berbasis Redis, sesimpel `client.Enqueue` → worker memproses.
- Scheduler yang fleksibel: tiap entry cron bisa enqueue ATAU select-rentang-lalu-proses langsung.
- Konsisten dengan pola per-modul: menambah job/cron = menambah `tasks.go`/`schedule.go` di folder modul, reuse `service`.
- Tiga binary yang jelas + Docker/Compose agar clone langsung jalan.

**Non-Goals:**
- Dashboard `asynqmon` (infra jalankan sendiri).
- Dead-letter/observability queue kustom (pakai retry & log bawaan Asynq).
- Broker selain Redis.
- Scheduler asynq (sengaja tidak dipakai; lihat Decision 3).

## Decisions

### 1. Asynq untuk queue produce/consume
`hibiken/asynq` — padanan BullMQ/Sidekiq di Go, berbasis Redis, retry & konkurensi bawaan.
- **Kenapa**: memakai `go-redis/v9` (stack kita), API minimal (`Enqueue` / `ServeMux` handler), populer & terawat.
- **Alternatif ditolak**: River (Postgres, bukan Redis); Watermill (framework pub/sub, abstraksi berlebih, lawan KISS); Machinery (lebih berat); DIY Redis Streams (harus bikin retry/konkurensi sendiri).

### 2. Tiga binary terpisah (api / worker / scheduler)
- `cmd/api`: HTTP + producer (enqueue on-demand).
- `cmd/worker`: `asynq.Server` memproses task (scale bebas).
- `cmd/scheduler`: `robfig/cron` (single replica).
- **Kenapa**: pemisahan peran jelas (producer / consumer / cron producer), tiap binary di-scale independen — pola deploy standar Go.
- **Alternatif ditolak**: satu binary + flag mode → mode logic bercampur; scheduler di dalam worker + flag `SCHEDULER_ENABLED` → footgun (harus true di tepat 1 replica; salah set = cron mati diam-diam atau dobel). Binary terpisah menghilangkan ambiguitas: scheduler = 1 deployment.

### 3. Scheduler pakai `robfig/cron/v3`, bukan `asynq.Scheduler`
`asynq.Scheduler` hanya bisa enqueue task statis ke Redis sesuai cron. Kebutuhan kita: entry cron berupa fungsi Go bebas.
- **Kenapa**: dengan `robfig/cron`, tiap jadwal bisa "select rentang data → proses langsung" TANPA lewat queue — pola batch/polling yang umum. Satu engine cron, dua pola (enqueue atau proses langsung). Scheduler diberi akses DB + service seperti worker.
- **Alternatif ditolak**: `asynq.Scheduler` (tak bisa proses langsung, hanya enqueue); mencampur `asynq.Scheduler` + `robfig/cron` (dua mekanisme jadwal, membingungkan).

### 4. Pola per-modul: `tasks.go` + `schedule.go`
Tiap modul mengekspos tiga sisi dari `service` yang sama:
```
module.go    RegisterRoutes(e, db, cache, qClient)  ← cmd/api      (produce)
tasks.go     RegisterTasks(mux, svc)                ← cmd/worker   (consume)
schedule.go  RegisterSchedule(cron, svc)            ← cmd/scheduler (cron)
```
- **Kenapa**: konsisten dengan gaya NestJS per-fitur yang sudah dipakai. Menambah background job/cron = tambah file di folder modul, bukan menyebar ke banyak tempat. `tasks.go` = apa yang diproses; `schedule.go` = kapan & pola apa.

### 5. Setup terpusat di `shared/queue` dan `shared/scheduler`
- `shared/queue`: membuat `*asynq.Client` (producer) dan `*asynq.Server` (consumer) dari `REDIS_URL` + konfigurasi konkurensi. Koneksi Redis Asynq dibangun dari URL yang sama (`asynq.RedisClientOpt`).
- `shared/scheduler`: wrapper tipis di atas `robfig/cron` untuk start/stop rapi.
- **Kenapa (KISS)**: modul hanya mendaftar task/jadwal; koneksi & lifecycle diurus di satu tempat.

### 6. Definisi task type-safe
Tiap task punya konstanta type (mis. `"example:message"`) dan payload struct yang di-`json.Marshal` ke `asynq.Task`. Enqueue lewat helper di `tasks.go` agar producer (api) dan consumer (worker) memakai bentuk payload yang sama.
- **Kenapa (security/performance)**: payload eksplisit & tervalidasi; hindari string mentah yang rawan salah.

### 7. Docker multi-stage + Compose
Tiga Dockerfile multi-stage (build → image minimal berbasis distroless/alpine, non-root). `docker-compose.yml` menyalakan api + worker + scheduler + postgres + redis dengan `.env`.
- **Kenapa (security)**: image kecil non-root memperkecil attack surface; Compose bikin dev environment sekali `up`.

### 8. Konfigurasi & lifecycle
Env baru: `WORKER_CONCURRENCY` (jumlah goroutine consumer), `QUEUE_ENABLED` opsional. Semua binary memakai `config.Load` yang sama (autoload `.env`). Tiap binary graceful shutdown: worker menuntaskan task berjalan sebelum berhenti; scheduler stop cron; keduanya menutup pool DB & Redis.
- **Kenapa (performance)**: konkurensi bisa disetel per service; shutdown rapi mencegah task terpotong.

## Risks / Trade-offs

- **Redis jadi titik pusat (cache + rate limiter + queue)** → beban & kegagalan terkonsentrasi. Mitigasi: Asynq memakai koneksi/pool sendiri; untuk skala besar bisa pisah instance Redis via URL berbeda (nanti, bila perlu).
- **Scheduler single-replica = single point of failure untuk cron** → bila mati, cron tak jalan. Mitigasi: deploy dengan restart policy; kliring pekerjaan yang terlewat bersifat idempoten (didorong di dokumentasi).
- **Task non-idempoten bisa dobel** (Asynq retry / at-least-once). Mitigasi: dokumentasikan agar handler idempoten; contoh `example` dibuat idempoten.
- **`robfig/cron` in-memory** (jadwal didaftar dari kode saat start) → tak persist, tapi karena didaftar di kode itu justru deterministik. Trade-off diterima (lebih simpel daripada persist ke Redis).
- **Tiga binary menambah permukaan build/deploy** → lebih banyak Dockerfile. Mitigasi: multi-stage seragam + Compose; `main.go` worker/scheduler tipis.
