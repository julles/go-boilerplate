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
	h := NewHandler(NewService(NewRepository(db), c), q)

	g := e.Group("/example")
	g.POST("", h.Create)
	g.GET("", h.List)
	g.GET("/:id", h.Get)
	g.POST("/produce", h.Produce)
}
