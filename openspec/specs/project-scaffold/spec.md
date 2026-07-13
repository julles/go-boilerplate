# project scaffold Specification

## Requirements

### Requirement: Struktur folder per-fitur
Boilerplate SHALL menyusun kode fitur di `internal/<fitur>/`, di mana tiap fitur adalah satu package Go berisi `dto/`, `handler.go`, `service.go`, `repository.go`, dan `module.go`. Kode bersama SHALL berada di `internal/shared/`, dan entrypoint SHALL berada di `cmd/api/main.go`.

#### Scenario: Menambah fitur baru
- **WHEN** developer menambah fitur `order`
- **THEN** ia membuat folder `internal/order/` dengan `dto/`, `handler.go`, `service.go`, `repository.go`, `module.go` mengikuti pola modul contoh `example`

#### Scenario: Nama file tanpa prefix nama fitur
- **WHEN** membuat file di dalam package fitur
- **THEN** nama file SHALL tanpa prefix nama fitur (`handler.go`, bukan `example_handler.go`) karena package sudah menjadi namespace

### Requirement: Modul contoh sebagai teladan
Boilerplate SHALL menyertakan satu modul contoh `example` yang minimal namun lengkap (handler, service, repository, dto, module) sebagai acuan pola, bukan showcase fitur. Repository-nya query ke tabel `merchant.merchants` sebagai contoh SQL manual schema-qualified.

#### Scenario: Developer menyalin pola
- **WHEN** developer membaca `internal/example/`
- **THEN** ia memahami alur `handler → service → repository` dan bisa menirunya untuk fitur lain

### Requirement: Script rename modul
Boilerplate SHALL menyediakan `rename.sh` yang menerima satu argumen module path baru, lalu mengganti module path lama (`github.com/julles/go-boilerplate`) di `go.mod` dan seluruh import di file `.go`, kemudian menjalankan `go mod tidy`.

#### Scenario: Rename saat clone jadi service baru
- **WHEN** developer menjalankan `./rename.sh github.com/julles/order-service`
- **THEN** `go.mod` dan semua import berubah ke module path baru, dan `go build ./...` tetap berhasil

#### Scenario: Argumen kosong
- **WHEN** `rename.sh` dijalankan tanpa argumen
- **THEN** script SHALL menampilkan cara pakai dan keluar tanpa mengubah apa pun

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
