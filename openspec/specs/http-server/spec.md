# http server Specification

## Requirements

### Requirement: Bootstrap server Echo v5
Boilerplate SHALL menginisialisasi server Echo v5 di `cmd/api/main.go`, memasang middleware bersama (recover, request logger, tracing, metrics, rate limiter), lalu mendaftarkan tiap modul fitur.

#### Scenario: Server menyala
- **WHEN** aplikasi dijalankan dengan env yang valid
- **THEN** server Echo listen pada port dari konfigurasi dan siap menerima request

### Requirement: Registrasi modul via module.go
Tiap modul fitur SHALL mengekspos fungsi registrasi (mis. `RegisterRoutes(e *echo.Echo, deps...)`) yang membangun dependency (`NewRepository → NewService → NewHandler`) dan mendaftarkan route-nya. `main.go` SHALL memanggil registrasi tersebut satu baris per modul, tanpa DI framework.

#### Scenario: Menambah modul ke server
- **WHEN** developer menambahkan `example.RegisterRoutes(e, db, rdb)` di `main.go`
- **THEN** seluruh endpoint example aktif tanpa perubahan lain

### Requirement: Response envelope konsisten
Server SHALL mengembalikan response JSON dengan format envelope yang seragam untuk kasus sukses maupun error di seluruh service.

#### Scenario: Response sukses
- **WHEN** sebuah endpoint berhasil
- **THEN** body mengikuti envelope standar berisi data hasil

### Requirement: Error handler tidak membocorkan detail internal
Server SHALL memakai custom error handler yang mengubah error menjadi response JSON konsisten. Detail internal (stack trace, pesan SQL, path file) MUST NOT dikirim ke client; detail tersebut hanya masuk ke log.

#### Scenario: Terjadi error internal
- **WHEN** handler mengembalikan error tak terduga
- **THEN** client menerima pesan generik dengan status yang sesuai, dan detail lengkap tercatat di log

#### Scenario: Error validasi input
- **WHEN** request gagal validasi
- **THEN** client menerima status 400 dengan pesan yang menjelaskan field yang salah, tanpa detail internal

### Requirement: Graceful shutdown
Server SHALL menutup koneksi dengan rapi saat menerima sinyal terminasi (SIGINT/SIGTERM): berhenti menerima request baru, menyelesaikan request berjalan, lalu menutup pool DB, client Redis, dan tracer.

#### Scenario: Menerima sinyal terminasi
- **WHEN** proses menerima SIGTERM
- **THEN** server menyelesaikan request in-flight, menutup pgxpool, Redis, dan flush tracer sebelum keluar
