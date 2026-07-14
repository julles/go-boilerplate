package example

import (
	"github.com/hibiken/asynq"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/labstack/echo/v5"

	"github.com/julles/go-boilerplate/internal/shared/cache"
)

// RegisterRoutes ngerakit dependency modul (repo -> service -> handler) sekaligus
// mendaftarkan route-nya. Nambah modul baru cukup satu baris pemanggilan di main.go.
func RegisterRoutes(e *echo.Echo, db *pgxpool.Pool, c *cache.Cache, q *asynq.Client) {
	// Wiring dependency-nya manual (poor man's DI): dirakit dari lapisan terdalam ke
	// luar — Repository(db) -> Service(repo, cache) -> Handler(service, queue). Ditulis
	// eksplisit begini bikin alur dependency-nya gampang dibaca tanpa framework DI.
	h := NewHandler(NewService(NewRepository(db), c), q)

	// Group ngasih prefix "/example" ke semua route modul, biar routing-nya terisolasi
	// per modul dan nggak tabrakan sama modul lain.
	g := e.Group("/example")
	g.POST("", h.Create)          // bikin merchant
	g.GET("", h.List)             // daftar merchant (paginasi/search)
	g.GET("/:id", h.Get)          // ambil satu merchant by id
	g.POST("/produce", h.Produce) // enqueue pesan ke worker
}
