# observability Specification

## Requirements

### Requirement: Tracing OpenTelemetry
Boilerplate SHALL menginisialisasi `TracerProvider` OpenTelemetry dengan OTLP exporter (endpoint dari env) dan memasang middleware `echo-opentelemetry` sehingga tiap request HTTP menghasilkan span.

#### Scenario: Request menghasilkan span
- **WHEN** sebuah request masuk
- **THEN** sebuah span terbentuk untuk request tersebut dan diekspor ke endpoint OTLP

#### Scenario: OTLP endpoint tidak diset
- **WHEN** endpoint OTLP tidak dikonfigurasi
- **THEN** aplikasi tetap berjalan (tracing non-fatal) dengan perilaku yang terdefinisi (mis. exporter no-op) dan mencatatnya

### Requirement: Metrics Prometheus
Boilerplate SHALL memasang middleware `echo-prometheus` untuk mengumpulkan metrik HTTP dan mengekspos endpoint `/metrics` untuk di-scrape Prometheus.

#### Scenario: Scrape metrics
- **WHEN** Prometheus melakukan GET `/metrics`
- **THEN** server mengembalikan metrik dalam format Prometheus, termasuk metrik request HTTP

### Requirement: Structured logging JSON via slog
Boilerplate SHALL memakai `log/slog` (stdlib) untuk menulis log terstruktur JSON ke stdout. Log MUST NOT dikirim langsung dari aplikasi ke Loki; pengumpulan ke Loki ditangani infra (Alloy/Promtail) via scrape stdout.

#### Scenario: Log ditulis sebagai JSON
- **WHEN** aplikasi mencatat sebuah event
- **THEN** baris log keluar ke stdout dalam format JSON

#### Scenario: Korelasi dengan trace
- **WHEN** log dibuat dalam konteks sebuah request ber-trace
- **THEN** baris log menyertakan `trace_id` agar bisa dikorelasikan dengan trace

### Requirement: Tidak membocorkan data sensitif di log
Log MUST NOT memuat secret (password, token, connection string berisi kredensial).

#### Scenario: Mencatat error koneksi
- **WHEN** aplikasi mencatat error terkait koneksi database
- **THEN** kredensial di connection string tidak ikut tercatat
