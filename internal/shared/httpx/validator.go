package httpx

import (
	"errors"
	"fmt"
	"net/http"
	"strings"

	"github.com/go-playground/validator/v10"
	"github.com/labstack/echo/v5"
)

// Validator mengadaptasi go-playground/validator ke interface echo.Validator,
// sehingga DTO cukup menandai aturan lewat struct tag `validate:"..."` dan
// handler memanggil c.Validate(&req).
type Validator struct {
	v *validator.Validate
}

func NewValidator() *Validator {
	return &Validator{v: validator.New()}
}

// Validate mengembalikan HTTP 400 dengan pesan singkat & aman bila tidak valid.
func (val *Validator) Validate(i any) error {
	if err := val.v.Struct(i); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, firstError(err))
	}
	return nil
}

// firstError mengubah error validator menjadi pesan singkat (nama field + aturan yang gagal).
// Aman ditampilkan ke client: hanya nama field & nama aturan, bukan detail internal.
func firstError(err error) string {
	var verrs validator.ValidationErrors
	if errors.As(err, &verrs) && len(verrs) > 0 {
		e := verrs[0]
		return fmt.Sprintf("field '%s' gagal aturan '%s'", strings.ToLower(e.Field()), e.Tag())
	}
	return "input tidak valid"
}
