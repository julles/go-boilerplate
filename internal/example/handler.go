package example

import (
	"errors"
	"net/http"

	"github.com/hibiken/asynq"
	"github.com/jackc/pgx/v5"
	"github.com/labstack/echo/v5"

	"github.com/julles/go-boilerplate/internal/example/dto"
	"github.com/julles/go-boilerplate/internal/shared/httpx"
)

// Handler menerjemahkan HTTP <-> service. Parsing + validasi DTO terjadi di sini.
type Handler struct {
	svc *Service
	q   *asynq.Client // producer queue (enqueue)
}

func NewHandler(svc *Service, q *asynq.Client) *Handler {
	return &Handler{svc: svc, q: q}
}

func (h *Handler) Create(c *echo.Context) error {
	var req dto.CreateRequest
	if err := c.Bind(&req); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid request body")
	}
	if err := c.Validate(&req); err != nil {
		return err
	}

	m, err := h.svc.Create(c.Request().Context(), req.Code)
	if err != nil {
		return err
	}
	return c.JSON(http.StatusCreated, httpx.OK(m))
}

func (h *Handler) Get(c *echo.Context) error {
	m, err := h.svc.GetByID(c.Request().Context(), c.Param("id"))
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return echo.NewHTTPError(http.StatusNotFound, "merchant not found")
		}
		return err
	}
	return c.JSON(http.StatusOK, httpx.OK(m))
}

func (h *Handler) List(c *echo.Context) error {
	merchants, err := h.svc.List(c.Request().Context(), dto.ParseQueryParams(c))
	if err != nil {
		return err
	}
	return c.JSON(http.StatusOK, httpx.OK(merchants))
}

// Produce meng-enqueue satu pesan ke queue untuk diproses worker (producer).
func (h *Handler) Produce(c *echo.Context) error {
	var req dto.ProduceRequest
	if err := c.Bind(&req); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid request body")
	}
	if err := c.Validate(&req); err != nil {
		return err
	}

	task, err := NewMessageTask(req.Message)
	if err != nil {
		return err
	}
	if _, err := h.q.EnqueueContext(c.Request().Context(), task); err != nil {
		return err
	}
	return c.JSON(http.StatusAccepted, httpx.OK(map[string]string{"status": "enqueued"}))
}
