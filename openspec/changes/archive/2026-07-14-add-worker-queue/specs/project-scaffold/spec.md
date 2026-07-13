## ADDED Requirements

### Requirement: Entrypoint worker dan scheduler
Struktur proyek SHALL menyediakan entrypoint terpisah `cmd/worker/main.go` (consumer queue) dan `cmd/scheduler/main.go` (cron), di samping `cmd/api/main.go`. Ketiganya SHALL berbagi kode `internal/` yang sama.

#### Scenario: Tiga binary dari satu codebase
- **WHEN** developer menjalankan `go build ./cmd/api`, `./cmd/worker`, dan `./cmd/scheduler`
- **THEN** ketiganya berhasil di-build dari `internal/` yang sama

### Requirement: File tasks.go dan schedule.go per modul
Tiap modul fitur SHALL boleh menambahkan `tasks.go` (handler consumer, dipakai worker) dan `schedule.go` (entry cron, dipakai scheduler), mengikuti pola satu folder per fitur. Nama file tetap tanpa prefix nama fitur.

#### Scenario: Menambah sisi worker/scheduler pada modul
- **WHEN** developer menambahkan `tasks.go` dan `schedule.go` di folder modul
- **THEN** worker dan scheduler memakainya lewat `RegisterTasks`/`RegisterSchedule` tanpa perubahan di luar folder modul
