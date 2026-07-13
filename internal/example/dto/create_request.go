package dto

// CreateRequest adalah body untuk membuat merchant baru.
//
// Ini adalah trust boundary: data datang dari client dan TIDAK boleh dipercaya
// begitu saja. Validasi memakai struct tag go-playground/validator (dijalankan
// via c.Validate) supaya aturan input terpusat di DTO, bukan tersebar di handler.
// Contoh aturan: required, min, max. Lihat https://pkg.go.dev/github.com/go-playground/validator/v10
type CreateRequest struct {
	// Code wajib diisi (required) agar merchant tidak dibuat tanpa identitas.
	// min=3 mencegah kode terlalu pendek/tidak bermakna; max=50 membatasi panjang
	// supaya aman disimpan ke kolom DB dan tidak dipakai untuk abuse (payload besar).
	Code string `json:"code" validate:"required,min=3,max=50"`
}
