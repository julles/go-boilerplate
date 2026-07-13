## 1. Dependency & konfigurasi

- [x] 1.1 Tambah dependency: `github.com/hibiken/asynq`, `github.com/robfig/cron/v3` (pin versi)
- [x] 1.2 `shared/config`: tambah `WORKER_CONCURRENCY` (default wajar) + baca `REDIS_URL` untuk queue
- [x] 1.3 Update `.env.example` (WORKER_CONCURRENCY, catatan queue pakai REDIS_URL yang sama)

## 2. Shared queue (Asynq)

- [x] 2.1 `shared/queue`: helper `RedisOpt(url)` → `asynq.RedisClientOpt` dari `REDIS_URL`
- [x] 2.2 `shared/queue`: `NewClient(url)` (producer) + fungsi close
- [x] 2.3 `shared/queue`: `NewServer(url, concurrency)` (consumer) yang menerima `*asynq.ServeMux`

## 3. Shared scheduler (robfig/cron)

- [x] 3.1 `shared/scheduler`: wrapper tipis `robfig/cron` — buat, tambah entry, start/stop rapi

## 4. Modul example — producer (API)

- [x] 4.1 `example/dto/produce_request.go`: `ProduceRequest{ Message string }` + tag `validate:"required,..."`
- [x] 4.2 `example/tasks.go`: definisi type `example:message` + payload struct + helper `NewMessageTask(payload)`
- [x] 4.3 `example/handler.go`: `Produce` — bind + validate, enqueue via client queue
- [x] 4.4 `example/module.go`: `RegisterRoutes` terima `*asynq.Client`; daftar `POST /example/produce`
- [x] 4.5 `cmd/api/main.go`: buat `queue.NewClient`, inject ke `example.RegisterRoutes`, close saat shutdown

## 5. Modul example — consumer (worker)

- [x] 5.1 `example/tasks.go`: handler `HandleMessage(ctx, task)` — unmarshal payload, proses (log "memproses pesan: ..."), idempoten
- [x] 5.2 `example/tasks.go`: `RegisterTasks(mux, svc)` — daftarkan handler ke type `example:message`
- [x] 5.3 `cmd/worker/main.go`: load config → db → redis/queue server → daftar `RegisterTasks` → run → graceful shutdown

## 6. Modul example — scheduler (cron)

- [x] 6.1 `example/service.go`: `ScanRecent(ctx, from, to)` — SELECT rentang `merchant.merchants` by `created_at` (parameterized), kembalikan jumlah
- [x] 6.2 `example/schedule.go`: `RegisterSchedule(cron, svc)` — entry cron (mis. tiap 5m) panggil `ScanRecent` lalu log jumlah baris
- [x] 6.3 `cmd/scheduler/main.go`: load config → db → scheduler → daftar `RegisterSchedule` → start → graceful shutdown

## 7. Docker & compose

- [x] 7.1 `Dockerfile.api`: multi-stage, image kecil, non-root
- [x] 7.2 `Dockerfile.worker` dan `Dockerfile.scheduler`: pola sama, ganti binary target
- [x] 7.3 `.dockerignore`: kecualikan `.env`, artefak build, `.git`, dll
- [x] 7.4 `docker-compose.yml`: services api + worker + scheduler + postgres + redis, baca `.env`, depends_on

## 8. Dokumentasi & verifikasi

- [x] 8.1 `docs/worker-queue.md`: arsitektur 3 binary, cara enqueue, cara tambah task & cron
- [x] 8.2 `docs/add-new-module.md`: tambah bagian `tasks.go` (worker) dan `schedule.go` (scheduler)
- [x] 8.3 `README.md`: tambah cara jalankan worker/scheduler + `docker compose up`
- [x] 8.4 Verifikasi: `go build ./...`; jalankan api+worker, `POST /example/produce`, pastikan worker memproses; jalankan scheduler, pastikan cron `ScanRecent` jalan
