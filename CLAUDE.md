# CLAUDE.md

Go backend boilerplate, dipakai ulang untuk banyak project.

## Stack
- **Echo v5** — HTTP framework
- **pgx** (postgres) — driver DB langsung, bukan `database/sql`, untuk performa
- **go-redis/v9** — rate limiter global + cache helper
- **go-playground/validator** — validasi DTO via struct tag (`c.Validate`, cara resmi Echo)
- **OpenTelemetry** (`echo-opentelemetry` + OTLP), **Prometheus** (`echo-prometheus`, `/metrics`), **slog** JSON ke stdout (Loki via infra)

## Konvensi
- Semua config dari env (secret tak di-hardcode). `.env.example` = daftar semua env var yang dibutuhkan, tanpa secret asli; `.env` tidak di-commit.

## Struktur modul (WAJIB ikut `internal/example`)
- Setiap modul/fitur baru WAJIB mengikuti struktur & pola `internal/example` sebagai acuan:
  `dto/` · `handler.go` · `service.go` · `repository.go` · `module.go` (+ `tasks.go`/`schedule.go` bila butuh worker/cron).
- Alur tetap: `handler → service → repository`. Nama file tanpa prefix nama fitur (`handler.go`, bukan `merchant_handler.go`).
- Wiring lewat `RegisterRoutes`/`RegisterTasks`/`RegisterSchedule`, dipanggil satu baris di `cmd/*`.
- Detail langkah lihat `docs/add-new-module.md`.

## Komentar kode (WAJIB)
- Setiap kode yang di-generate WAJIB diberi komentar **detail**: per baris / per blok logika, jelaskan maksud & alurnya.
- Komentar pakai **Bahasa Indonesia**, ditujukan agar junior backend developer langsung paham.
- Jelaskan "kenapa"-nya, bukan cuma "apa"-nya. Berlaku di semua session & mesin.

## Bahasa response API (WAJIB)
- Semua **message yang dikirim ke client** WAJIB pakai **Bahasa Inggris** — pesan error validasi, penolakan, error generik, semuanya.
- Alasannya: service dari boilerplate ini dikonsumsi service lain dan bisa diteruskan ke integrator/partner, jadi harus netral bahasa.
- Yang **tetap Bahasa Indonesia**: komentar kode, pesan log internal, teks error Go yang tak pernah sampai ke client, dan dokumen OpenSpec.
- Pesan dari pihak ketiga (provider/upstream) diteruskan **apa adanya** — itu data, bukan pesan kita.

## Gaya bahasa tulisan (WAJIB)
Berlaku untuk semua yang ditulis Bahasa Indonesia: komentar kode, docs, spec OpenSpec, commit message, dan jawaban di chat.

**Tulis seperti developer ngobrol, bukan seperti dokumen hasil terjemahan.** Campur Bahasa Indonesia dengan istilah teknis English secara natural — struktur kalimatnya Indonesia, istilah teknisnya biarkan English.

**Istilah teknis JANGAN diterjemahkan** ke Bahasa Indonesia formal:

| jangan | pakai |
|---|---|
| kanal | channel |
| kredensial | credential |
| pemanggil | caller |
| tanda tangan | signature |
| berkas | file |
| kedaluwarsa | expired |
| pertukaran mentah | raw exchange |
| rujukan | referensi |
| diturunkan / derivasi | di-derive |
| masa berlaku | validity period |
| galat | error |
| tembolok | cache |
| peladen | server |
| unggah / unduh | upload / download |

**Tapi jangan over-correct.** Meng-Inggris-kan kata yang wajar dalam Bahasa Indonesia sama kagoknya: `meng-enforce`, `men-generate-kan`, `di-consider`. Kalau kata Indonesianya memang dipakai sehari-hari (nilai, kode, wajib, identitas, jalur, alur), pakai itu.

Uji cepatnya: baca keras-keras. Kalau terdengar seperti sesuatu yang tak pernah diucapkan orang di ruang kerja, ganti.

## Git commit
- JANGAN tambahkan trailer `Co-Authored-By` (mis. Claude/AI) di commit message. Commit hanya atas nama akun developer.
- Berlaku di semua session & mesin — jangan pernah menyisipkan co-author AI apa pun.

## Rules (urutan = prioritas, non-negotiable)
1. **Security — harga mati.** Validasi input di setiap trust boundary, query pakai parameterized (jangan string concat), jangan bocorkan error internal ke response, secret dari env bukan hardcode.
2. **Performance — harga mati.** Pilih pola yang efisien; pgx pooling, hindari N+1, jangan alokasi/query yang tak perlu.
3. **Concurrency-safe — harga mati.** Project ini high-traffic, banyak request jalan barengan. Setiap kode (dan setiap spec/design) WAJIB memperhitungkan race condition: shared state diproteksi (mutex/atomic, atau hindari shared state sama sekali), operasi read-modify-write ke DB pakai transaksi/locking yang tepat (mis. `SELECT ... FOR UPDATE`, optimistic lock via versi, atau `INSERT ... ON CONFLICT`), jangan andalkan "cek dulu baru tulis" tanpa proteksi (TOCTOU), dan pastikan idempotensi di jalur yang bisa dobel (retry queue, dobel-submit). Kalau ragu suatu kode aman dari data race, uji dengan `go test -race`.
4. **KISS.** Kode sesimple mungkin sampai junior backend developer paham. Jangan overengineer: no abstraksi spekulatif, no interface untuk satu implementasi, no config untuk nilai yang tak pernah berubah. Tambah kompleksitas hanya saat benar-benar dibutuhkan.

Saat security/performance/concurrency-safety bentrok dengan "simpel", yang tiga menang — tapi cari cara paling simpel yang tetap aman, cepat, dan bebas race.

## Praktik wajib (checklist konkret dari 4 rules di atas)

**Security**
- Semua input dari luar divalidasi di DTO pakai tag `validate:"..."` lalu `c.Validate` — body, query param, path param, header. Tidak ada yang dipercaya mentah.
- Query selalu parameterized (`$1`, `$2`). Kalau nama kolom/arah sort datang dari input (mis. `?sort=`), pakai **whitelist** map di kode — jangan pernah interpolasi identifier dari user.
- Error ke client hanya lewat `httpx.Err` / `echo.NewHTTPError` dengan pesan aman; detail internal (query, stack, error driver) masuk log via `ErrorHandler`, tidak pernah ke response body.
- Jangan pernah nge-log secret/PII mentah: token, password, API key, nomor kartu/rekening, raw body yang mengandung itu. Redact atau log ID-nya saja.
- Endpoint publik wajib di belakang rate limiter, dan wajib ada batas ukuran body (`middleware.BodyLimit`) supaya request raksasa tak bikin memori jebol.
- Authorization dicek per-resource (ownership/tenant), bukan cuma "sudah login". Cek kepemilikan di service, bukan diserahkan ke client.
- Jangan tambah dependency baru untuk hal yang stdlib atau library existing sudah bisa — tiap dependency = permukaan serangan baru.

**Performance**
- `ctx` dari request selalu diteruskan ke semua call DB/Redis/HTTP. Kalau client mutus, kerjaannya ikut berhenti — bukan menggantung menghabiskan koneksi pool.
- `SELECT` kolom yang dipakai saja (jangan `SELECT *`), dan pastikan kolom yang di-filter/join/sort punya index. Kalau bikin query baru, sebutkan index yang menopangnya.
- List endpoint wajib pagination dengan **limit maksimum** yang di-clamp di server — jangan percaya `?limit=` dari client.
- Hindari N+1: ambil sekaligus pakai `WHERE id = ANY($1)` atau join. Bulk insert/update pakai `pgx.Batch` / `CopyFrom`, bukan loop query.
- Transaksi sependek mungkin. **Jangan** panggil API eksternal, sleep, atau nunggu I/O lain sambil pegang tx / koneksi pool.
- HTTP client ke upstream: satu instance di-reuse dengan `Timeout` eksplisit. Jangan bikin `http.Client` per request, jangan pakai `http.DefaultClient` (tanpa timeout = goroutine numpuk saat upstream lambat).
- Cache ditambahkan kalau ada bukti jalur itu panas, bukan karena "kayaknya perlu". Tiap cache wajib punya TTL dan strategi invalidasi yang jelas.

**Concurrency-safe**
- Goroutine yang dilepas dari handler **tidak boleh** pakai context request — context itu dibatalkan begitu response ditulis, kerjaannya bakal mati di tengah. Pakai context baru (`context.WithoutCancel` + timeout sendiri).
- Jangan spawn goroutine unbounded per request. Pakai worker pool atau `errgroup` dengan `SetLimit`.
- Semua task worker/queue wajib idempoten — Asynq bisa retry dan mengeksekusi dobel. Amankan dengan unique key + `INSERT ... ON CONFLICT DO NOTHING` atau status guard di transaksi.
- Scheduler jalan **satu instance**. Kalau terpaksa di-scale, wajib pakai lock (Redis/DB advisory lock) supaya cron tidak jalan dobel.
- Counter/saldo/stok: update atomik di DB (`UPDATE ... SET x = x - $1 WHERE x >= $1`) atau `SELECT ... FOR UPDATE`. Jangan read di Go, hitung di Go, lalu write.

**Selesai artinya sudah diverifikasi**
- Sebelum bilang selesai: `go build ./... && go vet ./... && go test -race ./...`.
- Kalau salah satu tidak dijalankan, bilang tidak dijalankan. Kalau gagal, laporkan gagal beserta outputnya — jangan diperhalus.

## Sikap saat menjawab / mengeksplorasi (WAJIB)
- **Selalu objektif.** Jawab berdasarkan fakta dari kode, dokumentasi, dan bukti nyata — bukan asumsi atau tebakan. Kalau belum yakin, bilang belum yakin dan cek dulu; jangan mengarang.
- **Angka, format, dan batas yang masuk kode/spec wajib bisa ditunjuk sumbernya** (baris kode, dokumen resmi, hasil tes). Kalau tak bisa ditunjuk, berarti belum diverifikasi — ambil buktinya dulu.
- **Jangan sekadar mengiyakan.** Kalau ide/pendekatan developer keliru atau ada trade-off berisiko (security/performance/concurrency), sampaikan terus terang beserta alasannya — termasuk saat itu berarti membantah keputusan yang sudah diambil, atau membantah pendapatmu sendiri yang ternyata salah setelah diuji.
- **Sebutkan efek samping sebelum melakukannya.** Aksi yang menyentuh data/service nyata (migrasi, kirim request keluar, hapus data) diberitahukan dulu, termasuk apakah bisa dibatalkan.
- **Berpikir keras saat developer mengeksplorasi sesuatu.** Telusuri alur end-to-end dulu, pertimbangkan edge case, race condition, dan implikasi security/performa sebelum menyimpulkan. Utamakan jawaban yang benar, bukan yang cepat menyenangkan.
