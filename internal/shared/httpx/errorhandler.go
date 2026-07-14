package httpx

import (
	"errors"
	"log/slog"
	"net/http"

	"github.com/labstack/echo/v5"
)

// ErrorHandler ngubah error apa pun jadi response JSON yang bentuknya seragam.
// Detail internal TIDAK dikirim ke client (cuma pesan aman); detail lengkapnya masuk log.
func ErrorHandler(c *echo.Context, err error) {
	// Response bisa jadi udah kelanjur ditulis, misal RequestLogger manggil error
	// handler duluan. Committed artinya header/status udah kekirim ke client; kalau
	// kita nulis lagi bakal muncul body dobel plus warning "superfluous
	// response.WriteHeader", jadi mendingan berhenti di sini kalau response udah terkirim.
	if r, _ := echo.UnwrapResponse(c.Response()); r != nil && r.Committed {
		return
	}

	// Default-nya: semua error yang nggak terduga kita anggap 500 dengan pesan generik.
	// Pesan generik ini sengaja dibikin nggak informatif buat client, biar detail
	// internal (bug, koneksi DB, dll) nggak bocor dan disalahgunakan penyerang.
	code := http.StatusInternalServerError
	msg := "internal server error"

	// Kalau error-nya ternyata *echo.HTTPError, berarti ini error yang memang kita
	// bentuk sendiri (mis. 400 dari validator, atau 404 not found). Status sama
	// pesannya udah kita kontrol, jadi aman dipakai apa adanya buat response.
	var he *echo.HTTPError
	if errors.As(err, &he) {
		code = he.Code
		if he.Message != "" {
			msg = he.Message // pesan ini kita set sendiri, jadi aman ditampilkan
		}
	}

	// Detail lengkap error selalu kita catat ke log buat kebutuhan debugging. Di
	// sinilah detail yang nggak dikirim ke client itu disimpan, lengkap dengan
	// konteks request (method, path, status) biar gampang ditelusuri pas investigasi.
	ctx := c.Request().Context()
	slog.ErrorContext(ctx, "request error",
		"error", err,
		"method", c.Request().Method,
		"path", c.Request().URL.Path,
		"status", code,
	)

	// ponytail: skip cek "committed" — handler kita hanya menulis di jalur error, tidak double-write.
	// Menurut spec HTTP, request HEAD nggak boleh punya body, jadi di sini cukup
	// kirim status code-nya aja tanpa isi JSON.
	if c.Request().Method == http.MethodHead {
		_ = c.NoContent(code)
		return
	}
	// Buat method lain, kita kirim envelope error dalam bentuk JSON. Kalau
	// pengirimannya gagal (misal koneksi client keburu putus), ya nggak ada yang
	// bisa kita lakuin selain mencatatnya di log.
	if err := c.JSON(code, Err(msg)); err != nil {
		slog.ErrorContext(ctx, "gagal mengirim error response", "error", err)
	}
}
