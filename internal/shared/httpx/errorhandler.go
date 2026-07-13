package httpx

import (
	"errors"
	"log/slog"
	"net/http"

	"github.com/labstack/echo/v5"
)

// ErrorHandler mengubah error menjadi response JSON seragam.
// Detail internal TIDAK dikirim ke client (hanya pesan aman); detail lengkap masuk log.
func ErrorHandler(c *echo.Context, err error) {
	code := http.StatusInternalServerError
	msg := "terjadi kesalahan pada server"

	var he *echo.HTTPError
	if errors.As(err, &he) {
		code = he.Code
		if he.Message != "" {
			msg = he.Message // pesan ini kita set sendiri, aman ditampilkan
		}
	}

	// Selalu catat detail lengkap ke log untuk debugging.
	ctx := c.Request().Context()
	slog.ErrorContext(ctx, "request error",
		"error", err,
		"method", c.Request().Method,
		"path", c.Request().URL.Path,
		"status", code,
	)

	// ponytail: skip cek "committed" — handler kita hanya menulis di jalur error, tidak double-write.
	if c.Request().Method == http.MethodHead {
		_ = c.NoContent(code)
		return
	}
	if err := c.JSON(code, Err(msg)); err != nil {
		slog.ErrorContext(ctx, "gagal mengirim error response", "error", err)
	}
}
