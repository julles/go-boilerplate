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
	// Response bisa sudah ditulis (mis. RequestLogger memanggil error handler lebih dulu).
	// Committed berarti header/status sudah dikirim ke client; menulis lagi akan
	// menghasilkan body ganda dan warning "superfluous response.WriteHeader",
	// jadi kita berhenti lebih awal bila response sudah terkirim.
	if r, _ := echo.UnwrapResponse(c.Response()); r != nil && r.Committed {
		return
	}

	// Default: anggap error tak terduga sebagai 500 dengan pesan generik.
	// Pesan generik ini sengaja tidak informatif ke client agar detail internal
	// (bug, koneksi DB, dsb.) tidak bocor dan bisa dimanfaatkan penyerang.
	code := http.StatusInternalServerError
	msg := "internal server error"

	// Bila error berupa *echo.HTTPError, berarti ini error yang memang kita
	// bentuk sendiri (mis. 400 dari validator, 404 not found). Status & pesannya
	// sudah kita kontrol, jadi boleh dipakai apa adanya untuk response.
	var he *echo.HTTPError
	if errors.As(err, &he) {
		code = he.Code
		if he.Message != "" {
			msg = he.Message // pesan ini kita set sendiri, aman ditampilkan
		}
	}

	// Selalu catat detail lengkap ke log untuk debugging. Inilah tempat detail
	// error yang tidak dikirim ke client disimpan, lengkap dengan konteks request
	// (method, path, status) supaya mudah ditelusuri saat investigasi.
	ctx := c.Request().Context()
	slog.ErrorContext(ctx, "request error",
		"error", err,
		"method", c.Request().Method,
		"path", c.Request().URL.Path,
		"status", code,
	)

	// ponytail: skip cek "committed" — handler kita hanya menulis di jalur error, tidak double-write.
	// Request HEAD tidak boleh punya body menurut spec HTTP, jadi cukup kirim
	// status code tanpa isi JSON.
	if c.Request().Method == http.MethodHead {
		_ = c.NoContent(code)
		return
	}
	// Untuk method lain, kirim envelope error JSON. Bila pengiriman gagal (mis.
	// koneksi client putus), tidak ada yang bisa dilakukan selain mencatatnya.
	if err := c.JSON(code, Err(msg)); err != nil {
		slog.ErrorContext(ctx, "gagal mengirim error response", "error", err)
	}
}
