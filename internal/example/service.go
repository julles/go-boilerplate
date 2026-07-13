package example

import (
	"context"
	"time"

	"github.com/julles/go-boilerplate/internal/example/dto"
	"github.com/julles/go-boilerplate/internal/shared/cache"
)

// merchantCacheTTL membatasi umur entri cache. TTL pendek dipilih agar data basi
// (stale) otomatis kedaluwarsa; ini kompromi umum cache-aside: kita menerima data
// bisa "usang" maksimal selama TTL demi mengurangi beban baca ke database.
const merchantCacheTTL = 5 * time.Minute

// Service berisi logika bisnis merchant. Struct konkret, tanpa interface (KISS).
// Service tidak tahu apa-apa soal HTTP/queue; ia hanya menerima tipe primitif/DTO
// sehingga bisa dipanggil dari handler API, worker, maupun scheduler.
type Service struct {
	repo  *Repository
	cache *cache.Cache
}

func NewService(repo *Repository, c *cache.Cache) *Service {
	return &Service{repo: repo, cache: c}
}

// Create membuat merchant lalu memetakan entity DB (Merchant) ke DTO response.
// Mapping ke DTO penting agar bentuk internal tabel tidak bocor ke API dan kita
// bebas mengubah skema DB tanpa memecahkan kontrak response.
func (s *Service) Create(ctx context.Context, code string) (dto.MerchantResponse, error) {
	m, err := s.repo.Create(ctx, code)
	if err != nil {
		return dto.MerchantResponse{}, err
	}
	return toResponse(m), nil
}

// GetByID memakai pola cache-aside: cek cache dulu, baru database.
// Alur: baca cache -> jika ada (hit) langsung kembalikan; jika tidak (miss) baca DB,
// isi cache, lalu kembalikan. Pola ini menghemat query DB untuk data yang sering
// dibaca tetapi jarang berubah.
func (s *Service) GetByID(ctx context.Context, id string) (dto.MerchantResponse, error) {
	// Key cache diberi prefix "merchant:" sebagai namespace agar tidak bentrok
	// dengan key modul/entity lain yang berbagi store cache yang sama.
	key := "merchant:" + id

	// Cache hit: kembalikan hasil deserialisasi tanpa menyentuh database sama sekali.
	var cached dto.MerchantResponse
	if s.cache.Get(ctx, key, &cached) {
		return cached, nil
	}

	// Cache miss: ambil dari sumber kebenaran (database).
	m, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return dto.MerchantResponse{}, err
	}
	// Isi cache dengan hasil terbaru (dalam bentuk DTO, sama seperti yang dikembalikan)
	// agar permintaan berikutnya untuk id ini bisa dilayani dari cache sampai TTL habis.
	resp := toResponse(m)
	s.cache.Set(ctx, key, resp, merchantCacheTTL)
	return resp, nil
}

// List mengembalikan daftar merchant sebagai DTO. Query paginasi dilakukan di repo;
// service hanya memetakan tiap entity ke DTO response.
func (s *Service) List(ctx context.Context, p dto.QueryParams) ([]dto.MerchantResponse, error) {
	merchants, err := s.repo.List(ctx, p)
	if err != nil {
		return nil, err
	}
	// Prealokasi slice dengan kapasitas = jumlah baris agar append tidak perlu
	// realokasi/menyalin ulang array di tengah loop (hemat alokasi).
	out := make([]dto.MerchantResponse, 0, len(merchants))
	for _, m := range merchants {
		out = append(out, toResponse(m))
	}
	return out, nil
}

// ScanRecent mengambil merchant yang dibuat dalam durasi terakhir lalu memproses tiap baris.
// Teladan pola scheduler "select rentang -> proses langsung". Mengembalikan jumlah baris.
func (s *Service) ScanRecent(ctx context.Context, since time.Duration) (int, error) {
	to := time.Now()
	from := to.Add(-since)

	// Ambil semua baris dalam rentang [from, to] dengan satu query (bukan per-item)
	// untuk menghindari masalah N+1 dan menjaga scan tetap efisien.
	merchants, err := s.repo.ListRecent(ctx, from, to)
	if err != nil {
		return 0, err
	}
	for range merchants {
		// tempat logika pemrosesan per baris (di sini no-op sebagai teladan)
	}
	// Kembalikan jumlah baris yang diproses supaya pemanggil (cron) bisa mencatatnya
	// di log/metrik untuk observabilitas.
	return len(merchants), nil
}

// toResponse memetakan entity DB (Merchant) ke DTO yang dipublikasikan ke luar.
// Fungsi tunggal ini menjadi satu-satunya tempat mapping sehingga konsisten dipakai
// oleh Create/GetByID/List dan hanya field yang aman/diperlukan yang diekspos.
func toResponse(m Merchant) dto.MerchantResponse {
	return dto.MerchantResponse{
		ID:        m.ID,
		Code:      m.Code,
		Status:    m.Status,
		CreatedAt: m.CreatedAt,
	}
}
