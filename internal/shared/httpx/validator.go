package httpx

import (
	"errors"
	"fmt"
	"net/http"
	"strings"

	"github.com/go-playground/validator/v10"
	"github.com/labstack/echo/v5"
)

// Validator ngadaptasi go-playground/validator ke interface echo.Validator, jadi
// tiap DTO cukup nandain aturannya lewat struct tag `validate:"..."` dan handler
// tinggal manggil c.Validate(&req).
type Validator struct {
	v *validator.Validate
}

func NewValidator() *Validator {
	return &Validator{v: validator.New()}
}

// Validate balikin HTTP 400 dengan pesan singkat dan aman kalau datanya nggak valid.
//
// Kita bungkus sebagai echo.HTTPError 400 biar ErrorHandler ngenalinnya sebagai error
// yang aman ditampilkan, bukan 500 generik. Ini yang bikin kita bisa bedain antara
// "input dari client-nya salah" dan "ada error internal di server".
func (val *Validator) Validate(i any) error {
	if err := val.v.Struct(i); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, firstError(err))
	}
	return nil
}

// firstError ngubah error dari validator jadi pesan singkat, isinya nama field plus
// aturan yang gagal. Aman ditampilkan ke client karena cuma nama field dan nama
// aturannya, bukan detail internal.
func firstError(err error) string {
	// Validator bisa balikin banyak error sekaligus; kita cukup ambil yang pertama
	// aja biar pesannya tetap ringkas tapi udah cukup ngasih tau client apa yang salah.
	var verrs validator.ValidationErrors
	if errors.As(err, &verrs) && len(verrs) > 0 {
		e := verrs[0]
		// Yang kita bocorkan cuma nama field sama nama aturannya (mis. "code",
		// "required"), bukan nilai yang dikirim client atau detail internal lain.
		return fmt.Sprintf("field '%s' failed on the '%s' rule", strings.ToLower(e.Field()), e.Tag())
	}
	// Fallback kalau error-nya ternyata bukan ValidationErrors (jarang kejadian):
	// balikin pesan generik yang aman aja.
	return "invalid input"
}
