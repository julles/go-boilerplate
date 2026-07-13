package example

import (
	"context"
	"time"

	"github.com/julles/go-boilerplate/internal/example/dto"
	"github.com/julles/go-boilerplate/internal/shared/cache"
)

const merchantCacheTTL = 5 * time.Minute

// Service berisi logika bisnis merchant. Struct konkret, tanpa interface (KISS).
type Service struct {
	repo  *Repository
	cache *cache.Cache
}

func NewService(repo *Repository, c *cache.Cache) *Service {
	return &Service{repo: repo, cache: c}
}

func (s *Service) Create(ctx context.Context, code string) (dto.MerchantResponse, error) {
	m, err := s.repo.Create(ctx, code)
	if err != nil {
		return dto.MerchantResponse{}, err
	}
	return toResponse(m), nil
}

// GetByID memakai pola cache-aside: cek cache dulu, baru database.
func (s *Service) GetByID(ctx context.Context, id string) (dto.MerchantResponse, error) {
	key := "merchant:" + id

	var cached dto.MerchantResponse
	if s.cache.Get(ctx, key, &cached) {
		return cached, nil
	}

	m, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return dto.MerchantResponse{}, err
	}
	resp := toResponse(m)
	s.cache.Set(ctx, key, resp, merchantCacheTTL)
	return resp, nil
}

func (s *Service) List(ctx context.Context, p dto.QueryParams) ([]dto.MerchantResponse, error) {
	merchants, err := s.repo.List(ctx, p)
	if err != nil {
		return nil, err
	}
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

	merchants, err := s.repo.ListRecent(ctx, from, to)
	if err != nil {
		return 0, err
	}
	for range merchants {
		// tempat logika pemrosesan per baris (di sini no-op sebagai teladan)
	}
	return len(merchants), nil
}

func toResponse(m Merchant) dto.MerchantResponse {
	return dto.MerchantResponse{
		ID:        m.ID,
		Code:      m.Code,
		Status:    m.Status,
		CreatedAt: m.CreatedAt,
	}
}
