# Menambah Modul Baru

Panduan ini memakai modul `example` (`internal/example/`) sebagai contoh. Pola tiap modul sama: satu folder berisi `dto/`, `handler.go`, `service.go`, `repository.go`, `module.go`. Alur data: **handler → service → repository → database**.

Misal kita mau membuat modul `product`.

## 1. Salin struktur dari `example`

```
internal/product/
├── dto/
│   ├── create_request.go
│   ├── query_params.go
│   └── response.go
├── repository.go
├── service.go
├── handler.go
├── module.go
├── tasks.go       (opsional) handler queue → dipakai cmd/worker
└── schedule.go    (opsional) entry cron  → dipakai cmd/scheduler
```

`tasks.go` dan `schedule.go` opsional — tambahkan hanya bila modul butuh background job / tugas terjadwal. Detailnya di [worker-queue.md](worker-queue.md).

Nama file **tanpa** prefix nama fitur (`handler.go`, bukan `product_handler.go`) — package `product` sudah jadi namespace, jadi dari luar tetap terbaca `product.Handler`.

## 2. DTO — batas validasi input

Validasi di trust boundary memakai struct tag [go-playground/validator](https://pkg.go.dev/github.com/go-playground/validator/v10). Cukup tandai aturan di tag; handler memanggil `c.Validate(&req)`. Contoh dari `example/dto/create_request.go`:

```go
type CreateRequest struct {
	Name  string `json:"name" validate:"required,min=3,max=100"`
	// contoh aturan lain:
	// Email string `json:"email" validate:"required,email"`
	// Qty   int    `json:"qty"   validate:"gte=1,lte=99"`
}
```

Aturan umum: `required`, `min`/`max` (panjang string), `gte`/`lte` (angka), `email`, `oneof=a b c`. Validator sudah didaftarkan sekali di `main.go` (`e.Validator = httpx.NewValidator()`), jadi modul baru tidak perlu setup apa pun.

`response.go` = bentuk data yang dikirim ke client. `query_params.go` = parsing query string dengan default aman (mis. `limit` dibatasi agar tidak menarik terlalu banyak baris).

## 3. Repository — SQL manual, parameterized

Query ditulis tangan dengan placeholder `$1, $2` (aman dari SQL injection), scan manual, dan **selalu** membawa `context.Context`. Contoh dari `example/repository.go`:

```go
type Repository struct{ db *pgxpool.Pool }

func NewRepository(db *pgxpool.Pool) *Repository { return &Repository{db: db} }

func (r *Repository) GetByID(ctx context.Context, id string) (Product, error) {
	const q = `SELECT id::text, name, created_at FROM product.products WHERE id = $1`
	var p Product
	if err := r.db.QueryRow(ctx, q, id).Scan(&p.ID, &p.Name, &p.CreatedAt); err != nil {
		return Product{}, err
	}
	return p, nil
}
```

Untuk ambil banyak baris, gunakan **satu** query (`WHERE id = ANY($1)` / `LIMIT/OFFSET`) — hindari query di dalam loop (N+1).

## 4. Service — logika bisnis (struct konkret)

Tanpa interface selama satu implementasi (KISS). Boleh pakai cache-aside via `shared/cache`. Contoh dari `example/service.go`:

```go
type Service struct {
	repo  *Repository
	cache *cache.Cache
}

func NewService(repo *Repository, c *cache.Cache) *Service {
	return &Service{repo: repo, cache: c}
}

func (s *Service) GetByID(ctx context.Context, id string) (dto.ProductResponse, error) {
	key := "product:" + id
	var cached dto.ProductResponse
	if s.cache.Get(ctx, key, &cached) {
		return cached, nil // cache hit
	}
	p, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return dto.ProductResponse{}, err
	}
	resp := toResponse(p)
	s.cache.Set(ctx, key, resp, 5*time.Minute)
	return resp, nil
}
```

## 5. Handler — jembatan HTTP

Parse + validasi DTO (`c.Bind` lalu `c.Validate`), panggil service, balas via `httpx.OK` / error. Contoh dari `example/handler.go`:

```go
func (h *Handler) Create(c *echo.Context) error {
	var req dto.CreateRequest
	if err := c.Bind(&req); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "body tidak valid")
	}
	if err := c.Validate(&req); err != nil { // aturan dari struct tag
		return err
	}
	m, err := h.svc.Create(c.Request().Context(), req.Name)
	if err != nil {
		return err
	}
	return c.JSON(http.StatusCreated, httpx.OK(m))
}

func (h *Handler) Get(c *echo.Context) error {
	p, err := h.svc.GetByID(c.Request().Context(), c.Param("id"))
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return echo.NewHTTPError(http.StatusNotFound, "product tidak ditemukan")
		}
		return err // error internal ditangani ErrorHandler global, tidak bocor ke client
	}
	return c.JSON(http.StatusOK, httpx.OK(p))
}
```

## 6. `module.go` — wiring + route

Bangun dependency (repo → service → handler) dan daftarkan route. Contoh dari `example/module.go`:

```go
func RegisterRoutes(e *echo.Echo, db *pgxpool.Pool, c *cache.Cache) {
	h := NewHandler(NewService(NewRepository(db), c))
	g := e.Group("/products")
	g.POST("", h.Create)
	g.GET("", h.List)
	g.GET("/:id", h.Get)
}
```

## 7. Daftarkan di `main.go`

Cukup satu baris:

```go
product.RegisterRoutes(e, pool, cache.New(rdb))
```

Selesai. Jalankan `go run ./cmd/api` dan endpoint `/products` aktif.

## Checklist

- [ ] Validasi semua input via struct tag `validate:"..."` + `c.Validate(&req)` di handler.
- [ ] Query parameterized (`$1, $2`), bawa `context`, hindari N+1.
- [ ] Service struct konkret, tanpa interface bila satu implementasi.
- [ ] Handler tidak membocorkan error internal (balikan `error` mentah → ditangani `httpx.ErrorHandler`).
- [ ] Route didaftarkan lewat `RegisterRoutes` + satu baris di `main.go`.
