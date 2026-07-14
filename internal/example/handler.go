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

// Handler jadi jembatan antara HTTP dan service. Parsing plus validasi DTO
// dikerjakan di sini. Prinsipnya: handler-nya tipis — dia cuma ngurus hal-hal
// HTTP (bind body, validasi, milih status code, format response) lalu melempar
// seluruh logika bisnis ke Service. Dengan begitu logika bisnis tetap bisa dipakai
// ulang dari worker/cron tanpa ikut nyangkut ke objek HTTP.
type Handler struct {
	svc *Service
	q   *asynq.Client // producer queue buat enqueue task
}

func NewHandler(svc *Service, q *asynq.Client) *Handler {
	return &Handler{svc: svc, q: q}
}

// Create menangani POST buat bikin merchant baru.
func (h *Handler) Create(c *echo.Context) error {
	// Bind nge-parse JSON body ke DTO. Kalau body-nya bukan JSON valid atau tipenya
	// nggak cocok, itu murni salah client — jadi langsung balas 400, bukan 500.
	var req dto.CreateRequest
	if err := c.Bind(&req); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid request body")
	}
	// Validasi aturan field (misal required, panjang, format) di trust boundary,
	// sebelum datanya nyampe ke logika bisnis. Error validasi udah bawa status code
	// sendiri, jadi tinggal diteruskan apa adanya.
	if err := c.Validate(&req); err != nil {
		return err
	}

	// Lempar ke service. Context dari request ikut diteruskan biar cancel/timeout dari
	// client juga membatalkan query DB — nggak ada gunanya kerja kalau client udah pergi.
	m, err := h.svc.Create(c.Request().Context(), req.Code)
	if err != nil {
		return err
	}
	// 201 Created nandain resource baru berhasil dibuat. Body-nya dibungkus httpx.OK
	// biar bentuk envelope response-nya seragam di semua endpoint.
	return c.JSON(http.StatusCreated, httpx.OK(m))
}

// Get menangani GET /:id buat ambil satu merchant.
func (h *Handler) Get(c *echo.Context) error {
	m, err := h.svc.GetByID(c.Request().Context(), c.Param("id"))
	if err != nil {
		// Bedakan kasus "nggak ketemu" dari error server lainnya: pgx.ErrNoRows kita
		// petakan ke 404 (client nyari data yang memang nggak ada), error lain tetap 500.
		if errors.Is(err, pgx.ErrNoRows) {
			return echo.NewHTTPError(http.StatusNotFound, "merchant not found")
		}
		return err
	}
	return c.JSON(http.StatusOK, httpx.OK(m))
}

// List menangani GET buat daftar merchant lengkap dengan paginasi/pencarian.
func (h *Handler) List(c *echo.Context) error {
	// ParseQueryParams ambil limit/offset/search dari query string sekaligus ngasih
	// nilai default yang aman, jadi service nggak perlu tahu detail parsing HTTP.
	merchants, err := h.svc.List(c.Request().Context(), dto.ParseQueryParams(c))
	if err != nil {
		return err
	}
	return c.JSON(http.StatusOK, httpx.OK(merchants))
}

// Produce meng-enqueue satu pesan ke queue buat diproses worker (ini sisi producer).
// Pola ini misahin penerimaan request (yang cepat) dari pemrosesannya (yang asinkron):
// API cukup naruh task ke antrean lalu langsung balas, sisanya worker yang ngerjain.
func (h *Handler) Produce(c *echo.Context) error {
	var req dto.ProduceRequest
	if err := c.Bind(&req); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid request body")
	}
	if err := c.Validate(&req); err != nil {
		return err
	}

	// Bikin task-nya (serialisasi payload) pakai helper yang sama dengan sisi worker,
	// biar format payload selalu konsisten antara producer dan consumer.
	task, err := NewMessageTask(req.Message)
	if err != nil {
		return err
	}
	// EnqueueContext naruh task ke Redis/asynq. Context diteruskan biar proses enqueue
	// juga kena batas timeout request.
	if _, err := h.q.EnqueueContext(c.Request().Context(), task); err != nil {
		return err
	}
	// 202 Accepted: request-nya diterima buat diproses nanti, hasilnya belum tentu
	// kelar pas response dikirim — ini semantik yang pas buat kerja asinkron.
	return c.JSON(http.StatusAccepted, httpx.OK(map[string]string{"status": "enqueued"}))
}
