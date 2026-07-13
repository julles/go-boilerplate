package dto

// CreateRequest adalah body untuk membuat merchant baru.
//
// Validasi memakai struct tag go-playground/validator (dijalankan via c.Validate).
// Contoh aturan: required, min, max. Lihat https://pkg.go.dev/github.com/go-playground/validator/v10
type CreateRequest struct {
	Code string `json:"code" validate:"required,min=3,max=50"`
}
