# scheduling Specification

## Requirements

### Requirement: Binary scheduler terpisah
Penjadwalan SHALL dijalankan oleh binary tersendiri (`cmd/scheduler`) memakai `robfig/cron`. Scheduler MUST dijalankan sebagai satu replica agar tugas terjadwal tidak ganda.

#### Scenario: Scheduler menjalankan cron
- **WHEN** binary scheduler dijalankan
- **THEN** semua entry cron yang terdaftar aktif sesuai jadwalnya

#### Scenario: Worker tidak menjalankan cron
- **WHEN** binary worker dijalankan
- **THEN** tidak ada tugas terjadwal yang dieksekusi oleh worker (worker hanya memproses queue)

### Requirement: Entry cron berupa fungsi bebas
Tiap entry cron SHALL berupa fungsi Go yang boleh melakukan pekerjaan apa pun — termasuk query database dan memproses hasilnya langsung tanpa melalui queue.

#### Scenario: Select rentang lalu proses
- **WHEN** entry cron "scan" terpicu sesuai jadwalnya
- **THEN** ia menjalankan satu query rentang (mis. berdasarkan `created_at`) dan memproses baris hasilnya secara langsung

#### Scenario: Query rentang parameterized
- **WHEN** entry cron menjalankan query rentang
- **THEN** batas rentang dikirim sebagai parameter query (`$1,$2`), bukan dirangkai ke string SQL

### Requirement: Registrasi jadwal per modul
Tiap modul SHALL mengekspos fungsi `RegisterSchedule` yang mendaftarkan entry cron-nya, memakai `service` modul yang sama.

#### Scenario: Menambah jadwal
- **WHEN** developer menambahkan entry di `schedule.go` modul dan memanggilnya dari `RegisterSchedule`
- **THEN** scheduler menjalankan jadwal itu tanpa perubahan di luar folder modul
