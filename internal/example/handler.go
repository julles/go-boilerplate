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
// Prinsipnya: handler tipis. Ia hanya mengurus hal-hal HTTP (bind body, validasi,
// pilih status code, format response) dan mendelegasikan seluruh logika bisnis ke
// Service. Dengan begitu logika bisnis tetap bisa dipakai ulang dari worker/cron
// tanpa bergantung pada objek HTTP.
type Handler struct {
	svc *Service
	q   *asynq.Client // producer queue (enqueue)
}

func NewHandler(svc *Service, q *asynq.Client) *Handler {
	return &Handler{svc: svc, q: q}
}

// Create menangani POST: membuat merchant baru.
func (h *Handler) Create(c *echo.Context) error {
	// Bind mengurai JSON body ke DTO. Bila body bukan JSON valid / tipe tidak cocok,
	// itu murni kesalahan klien sehingga langsung dibalas 400 (bukan 500).
	var req dto.CreateRequest
	if err := c.Bind(&req); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid request body")
	}
	// Validasi aturan field (mis. required, panjang, format) di batas masuk (trust
	// boundary) sebelum data menyentuh logika bisnis. Error validasi sudah membawa
	// status code sendiri, jadi cukup diteruskan apa adanya.
	if err := c.Validate(&req); err != nil {
		return err
	}

	// Delegasi ke service. Context dari request diteruskan agar cancel/timeout klien
	// ikut membatalkan query DB (hindari kerja sia-sia bila klien sudah pergi).
	m, err := h.svc.Create(c.Request().Context(), req.Code)
	if err != nil {
		return err
	}
	// 201 Created menandakan resource baru berhasil dibuat; body dibungkus httpx.OK
	// agar bentuk envelope response seragam di seluruh endpoint.
	return c.JSON(http.StatusCreated, httpx.OK(m))
}

// Get menangani GET /:id: ambil satu merchant.
func (h *Handler) Get(c *echo.Context) error {
	m, err := h.svc.GetByID(c.Request().Context(), c.Param("id"))
	if err != nil {
		// Bedakan "tidak ditemukan" dari error server lain: pgx.ErrNoRows dipetakan
		// ke 404 (kesalahan klien merujuk data yang tak ada), error lain tetap 500.
		if errors.Is(err, pgx.ErrNoRows) {
			return echo.NewHTTPError(http.StatusNotFound, "merchant not found")
		}
		return err
	}
	return c.JSON(http.StatusOK, httpx.OK(m))
}

// List menangani GET: daftar merchant dengan paginasi/pencarian.
func (h *Handler) List(c *echo.Context) error {
	// ParseQueryParams mengambil limit/offset/search dari query string dan memberi
	// nilai default yang aman, sehingga service tak perlu tahu detail parsing HTTP.
	merchants, err := h.svc.List(c.Request().Context(), dto.ParseQueryParams(c))
	if err != nil {
		return err
	}
	return c.JSON(http.StatusOK, httpx.OK(merchants))
}

// Produce meng-enqueue satu pesan ke queue untuk diproses worker (producer).
// Pola ini memisahkan penerimaan request (cepat) dari pemrosesan (asinkron):
// API cukup menaruh task di antrean lalu langsung membalas, worker yang bekerja.
func (h *Handler) Produce(c *echo.Context) error {
	var req dto.ProduceRequest
	if err := c.Bind(&req); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid request body")
	}
	if err := c.Validate(&req); err != nil {
		return err
	}

	// Bentuk task (serialisasi payload) memakai helper yang sama dengan sisi worker,
	// supaya format payload selalu konsisten antara producer dan consumer.
	task, err := NewMessageTask(req.Message)
	if err != nil {
		return err
	}
	// EnqueueContext menaruh task ke Redis/asynq. Context diteruskan agar enqueue
	// ikut terbatas oleh timeout request.
	if _, err := h.q.EnqueueContext(c.Request().Context(), task); err != nil {
		return err
	}
	// 202 Accepted: request diterima untuk diproses nanti, hasil belum tentu selesai
	// saat response dikirim -- inilah semantik yang tepat untuk kerja asinkron.
	return c.JSON(http.StatusAccepted, httpx.OK(map[string]string{"status": "enqueued"}))
}
