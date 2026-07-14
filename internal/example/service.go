package example

import (
	"context"
	"time"

	"github.com/julles/go-boilerplate/internal/example/dto"
	"github.com/julles/go-boilerplate/internal/shared/cache"
)

// merchantCacheTTL membatasi umur tiap entri cache. Kita sengaja pakai TTL pendek
// biar data yang sudah stale otomatis kedaluwarsa. Ini trade-off khas cache-aside:
// kita rela data-nya "usang" paling lama selama TTL, ditukar dengan beban baca ke
// database yang jauh lebih ringan.
const merchantCacheTTL = 5 * time.Minute

// Service menampung logika bisnis merchant. Sengaja struct konkret tanpa interface
// (KISS). Service nggak tahu apa-apa soal HTTP atau queue — dia cuma nerima tipe
// primitif/DTO, jadi bisa dipanggil dari handler API, worker, maupun scheduler.
type Service struct {
	repo  *Repository
	cache *cache.Cache
}

func NewService(repo *Repository, c *cache.Cache) *Service {
	return &Service{repo: repo, cache: c}
}

// Create bikin merchant baru lalu memetakan entity DB (Merchant) ke DTO response.
// Mapping ke DTO ini penting: bentuk internal tabel jadi nggak bocor ke API, dan
// kita bebas mengubah skema DB tanpa merusak kontrak response ke client.
func (s *Service) Create(ctx context.Context, code string) (dto.MerchantResponse, error) {
	m, err := s.repo.Create(ctx, code)
	if err != nil {
		return dto.MerchantResponse{}, err
	}
	return toResponse(m), nil
}

// GetByID pakai pola cache-aside: cek cache dulu, database belakangan.
// Alurnya: baca cache — kalau ada (cache hit) langsung balikin; kalau nggak ada
// (cache miss) baru baca DB, isi cache-nya, lalu balikin. Pola ini menghemat query
// DB untuk data yang sering dibaca tapi jarang berubah.
func (s *Service) GetByID(ctx context.Context, id string) (dto.MerchantResponse, error) {
	// Key cache dikasih prefix "merchant:" sebagai namespace, biar nggak bentrok
	// dengan key modul/entity lain yang berbagi store cache yang sama.
	key := "merchant:" + id

	// Cache hit: langsung balikin hasil deserialisasi, database nggak disentuh sama sekali.
	var cached dto.MerchantResponse
	if s.cache.Get(ctx, key, &cached) {
		return cached, nil
	}

	// Cache miss: ambil datanya dari database sebagai source of truth.
	m, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return dto.MerchantResponse{}, err
	}
	// Lalu isi cache dengan hasil terbaru (bentuk DTO, sama persis dengan yang
	// dikembalikan) supaya request berikutnya untuk id ini bisa dilayani langsung
	// dari cache sampai TTL-nya habis.
	resp := toResponse(m)
	s.cache.Set(ctx, key, resp, merchantCacheTTL)
	return resp, nil
}

// List mengembalikan daftar merchant dalam bentuk DTO. Query paginasi-nya dikerjakan
// di repo; service di sini cuma memetakan tiap entity ke DTO response.
func (s *Service) List(ctx context.Context, p dto.QueryParams) ([]dto.MerchantResponse, error) {
	merchants, err := s.repo.List(ctx, p)
	if err != nil {
		return nil, err
	}
	// Prealokasi slice dengan kapasitas = jumlah baris, biar append nggak perlu
	// realokasi dan nyalin ulang array di tengah loop — lumayan hemat alokasi.
	out := make([]dto.MerchantResponse, 0, len(merchants))
	for _, m := range merchants {
		out = append(out, toResponse(m))
	}
	return out, nil
}

// ScanRecent ambil merchant yang dibuat dalam durasi terakhir, lalu memproses tiap
// barisnya. Ini contoh pola scheduler "select rentang lalu proses langsung", dan
// mengembalikan jumlah baris yang diproses.
func (s *Service) ScanRecent(ctx context.Context, since time.Duration) (int, error) {
	to := time.Now()
	from := to.Add(-since)

	// Ambil semua baris dalam rentang [from, to] pakai satu query saja (bukan
	// query per-item), biar terhindar dari masalah N+1 dan scan-nya tetap efisien.
	merchants, err := s.repo.ListRecent(ctx, from, to)
	if err != nil {
		return 0, err
	}
	for range merchants {
		// di sini tempatnya logika pemrosesan per baris (sengaja no-op sebagai contoh)
	}
	// Balikin jumlah baris yang diproses supaya pemanggil (cron) bisa mencatatnya
	// di log/metrik buat keperluan observabilitas.
	return len(merchants), nil
}

// toResponse memetakan entity DB (Merchant) ke DTO yang dipublikasikan ke luar.
// Dengan menaruh mapping di satu fungsi ini, semua pemakainya (Create/GetByID/List)
// jadi konsisten, dan cuma field yang aman/perlu saja yang kita ekspos ke client.
func toResponse(m Merchant) dto.MerchantResponse {
	return dto.MerchantResponse{
		ID:        m.ID,
		Code:      m.Code,
		Status:    m.Status,
		CreatedAt: m.CreatedAt,
	}
}
