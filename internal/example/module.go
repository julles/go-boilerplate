package example

import (
	"github.com/hibiken/asynq"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/labstack/echo/v5"

	"github.com/julles/go-boilerplate/internal/shared/cache"
)

// RegisterRoutes membangun dependency modul (repo -> service -> handler) dan
// mendaftarkan route-nya. Tambah modul baru = satu baris pemanggilan di main.go.
func RegisterRoutes(e *echo.Echo, db *pgxpool.Pool, c *cache.Cache, q *asynq.Client) {
	// Wiring dependency manual (poor man's DI): rakit dari lapisan terdalam ke luar --
	// Repository(db) -> Service(repo, cache) -> Handler(service, queue). Eksplisit
	// begini membuat alur dependency gampang dibaca tanpa framework DI.
	h := NewHandler(NewService(NewRepository(db), c), q)

	// Group memberi prefix "/example" pada semua route modul agar routing terisolasi
	// per modul dan tidak bertabrakan dengan modul lain.
	g := e.Group("/example")
	g.POST("", h.Create)          // buat merchant
	g.GET("", h.List)             // daftar merchant (paginasi/search)
	g.GET("/:id", h.Get)          // ambil satu merchant by id
	g.POST("/produce", h.Produce) // enqueue pesan ke worker
}
