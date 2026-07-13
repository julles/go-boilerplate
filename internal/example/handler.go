package example

import (
	"errors"
	"net/http"

	"github.com/jackc/pgx/v5"
	"github.com/labstack/echo/v5"

	"github.com/julles/go-boilerplate/internal/example/dto"
	"github.com/julles/go-boilerplate/internal/shared/httpx"
)

// Handler menerjemahkan HTTP <-> service. Parsing + validasi DTO terjadi di sini.
type Handler struct {
	svc *Service
}

func NewHandler(svc *Service) *Handler {
	return &Handler{svc: svc}
}

func (h *Handler) Create(c *echo.Context) error {
	var req dto.CreateRequest
	if err := c.Bind(&req); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "body tidak valid")
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
			return echo.NewHTTPError(http.StatusNotFound, "merchant tidak ditemukan")
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
