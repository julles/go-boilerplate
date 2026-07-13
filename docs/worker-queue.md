# Worker & Queue

Boilerplate menghasilkan **tiga binary** dari `internal/` yang sama:

```
cmd/api        HTTP + producer  → enqueue task ke Redis (Asynq)
cmd/worker     consumer         → memproses task dari queue (Asynq)
cmd/scheduler  cron             → tugas terjadwal (robfig/cron), bisa select→proses langsung
```

- **Queue** pakai [Asynq](https://github.com/hibiken/asynq) (berbasis Redis, `go-redis/v9`). Murni producer/consumer.
- **Scheduler** pakai [robfig/cron](https://github.com/robfig/cron). Terpisah dari queue; tiap job fungsi Go bebas.
- Redis yang sama dipakai untuk cache, rate limiter, dan queue (`REDIS_URL`).

## Menjalankan

```bash
go run ./cmd/api        # HTTP server (+ bisa enqueue)
go run ./cmd/worker     # proses task
go run ./cmd/scheduler  # cron — jalankan SATU instance saja
```

Atau semuanya sekaligus: `docker compose up --build`.

> ⚠️ **Scheduler harus single-replica.** Kalau di-scale >1, cron akan nge-fire ganda.

## Alur produce/consume (contoh modul `example`)

```
POST /example/produce {"message":"halo"}
        │  api: NewMessageTask → qClient.Enqueue
        ▼
     ┌───────┐
     │ REDIS │
     └───────┘
        │  worker: mux.HandleFunc("example:message", HandleMessage)
        ▼
   log "memproses pesan: halo"
```

## Menambah task (worker) di sebuah modul

Di `internal/<modul>/tasks.go`:

```go
const TypeMessage = "example:message"

type MessagePayload struct{ Message string `json:"message"` }

func NewMessageTask(msg string) (*asynq.Task, error) {
	b, _ := json.Marshal(MessagePayload{Message: msg})
	return asynq.NewTask(TypeMessage, b), nil
}

func (h *TaskHandler) HandleMessage(ctx context.Context, t *asynq.Task) error {
	var p MessagePayload
	if err := json.Unmarshal(t.Payload(), &p); err != nil {
		return fmt.Errorf("payload rusak: %w: %w", err, asynq.SkipRetry) // non-retryable
	}
	slog.InfoContext(ctx, "memproses pesan", "message", p.Message)
	return nil // error non-nil → Asynq retry otomatis
}

func RegisterTasks(mux *asynq.ServeMux, svc *Service) {
	h := NewTaskHandler(svc)
	mux.HandleFunc(TypeMessage, h.HandleMessage)
}
```

Lalu daftarkan di `cmd/worker/main.go`: `example.RegisterTasks(mux, svc)`.

**Handler harus idempoten** — Asynq at-least-once, task bisa diproses ulang saat retry.

## Menambah cron (scheduler) di sebuah modul

Di `internal/<modul>/schedule.go` — pola "select rentang → proses langsung":

```go
func RegisterSchedule(s *scheduler.Scheduler, svc *Service) error {
	return s.Add("@every 5m", func() {
		ctx := context.Background()
		n, err := svc.ScanRecent(ctx, 24*time.Hour) // SELECT rentang created_at, parameterized
		if err != nil {
			slog.ErrorContext(ctx, "scan-recent gagal", "error", err)
			return
		}
		slog.InfoContext(ctx, "scan-recent selesai", "jumlah", n)
	})
}
```

Spec cron: `@every 5m`, `@daily`, atau format cron `0 2 * * *`. Daftarkan di `cmd/scheduler/main.go`: `example.RegisterSchedule(sch, svc)`.

Butuh cron yang mendistribusikan kerja? Entry cron boleh juga `qClient.Enqueue(...)` alih-alih memproses langsung — tinggal beri scheduler akses client queue.
