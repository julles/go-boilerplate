## ADDED Requirements

### Requirement: Rate limiter berbasis Redis
Boilerplate SHALL menyediakan middleware rate limiter yang menyimpan hitungan di Redis, sehingga batas berlaku global lintas replica (bukan per-pod). Batas (limit dan window) SHALL dapat dikonfigurasi dari env.

#### Scenario: Permintaan dalam batas
- **WHEN** klien mengirim request di bawah batas dalam satu window
- **THEN** request diteruskan seperti biasa

#### Scenario: Permintaan melebihi batas
- **WHEN** klien melewati batas dalam satu window
- **THEN** server menolak dengan status 429 Too Many Requests

#### Scenario: Batas berlaku lintas replica
- **WHEN** dua instance service berjalan dan menerima request dari klien yang sama
- **THEN** hitungan digabung via Redis sehingga batas total tetap terjaga

### Requirement: Perilaku saat Redis tidak tersedia
Middleware SHALL memiliki perilaku fallback yang eksplisit ketika Redis tidak tersedia. Default: fail-open (izinkan request) dengan mencatat warning, agar ketersediaan layanan tetap terjaga.

#### Scenario: Redis down
- **WHEN** rate limiter tidak dapat menjangkau Redis
- **THEN** request tetap dilayani (fail-open) dan warning tercatat di log
