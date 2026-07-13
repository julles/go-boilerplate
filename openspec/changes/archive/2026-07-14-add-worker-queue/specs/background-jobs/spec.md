## ADDED Requirements

### Requirement: Producer enqueue message
API SHALL dapat memasukkan (enqueue) message ke queue berbasis Asynq melalui client bersama. Payload task SHALL berupa struct yang di-serialize (JSON), bukan string mentah.

#### Scenario: Enqueue dari endpoint produce
- **WHEN** klien mengirim `POST /example/produce` dengan body berisi message valid
- **THEN** sebuah task dimasukkan ke queue dan API membalas sukses tanpa menunggu pemrosesan

#### Scenario: Validasi payload produce
- **WHEN** body produce tidak valid (mis. message kosong)
- **THEN** API membalas 400 dan tidak ada task yang di-enqueue

### Requirement: Consumer memproses message
Binary worker (`cmd/worker`) SHALL menjalankan `asynq.Server` yang mengambil task dari queue dan memprosesnya melalui handler yang didaftarkan tiap modul.

#### Scenario: Worker memproses task
- **WHEN** worker berjalan dan ada task `example:message` di queue
- **THEN** handler modul memproses message tersebut dan menandai task selesai

#### Scenario: Konkurensi dari konfigurasi
- **WHEN** worker dijalankan dengan `WORKER_CONCURRENCY` tertentu
- **THEN** worker memproses hingga sejumlah task itu secara paralel

### Requirement: Retry saat handler gagal
Task yang handler-nya mengembalikan error SHALL dicoba ulang mengikuti kebijakan retry Asynq (dengan batas maksimum), bukan langsung hilang.

#### Scenario: Handler gagal sementara
- **WHEN** handler mengembalikan error pada percobaan pertama
- **THEN** task dijadwalkan ulang untuk dicoba kembali hingga batas retry

### Requirement: Registrasi task per modul
Tiap modul SHALL mengekspos fungsi `RegisterTasks` yang mendaftarkan handler task-nya ke mux worker, memakai `service` modul yang sama dengan API.

#### Scenario: Menambah handler task
- **WHEN** developer menambahkan handler di `tasks.go` modul dan memanggilnya dari `RegisterTasks`
- **THEN** worker memproses task tersebut tanpa perubahan di luar folder modul
