package dto

// CreateRequest itu body request buat bikin merchant baru.
//
// Ini trust boundary: datanya datang dari client dan TIDAK boleh dipercaya begitu
// aja. Validasinya pakai struct tag go-playground/validator (dijalanin via c.Validate)
// biar aturan input-nya terpusat di DTO, nggak berserakan di handler. Contoh aturan:
// required, min, max. Lihat https://pkg.go.dev/github.com/go-playground/validator/v10
type CreateRequest struct {
	// Code wajib diisi (required) biar merchant nggak dibuat tanpa identitas.
	// min=3 nyegah kode-nya kependekan/nggak bermakna; max=50 batasin panjangnya biar
	// aman disimpan ke kolom DB dan nggak dipakai buat abuse (payload gede).
	Code string `json:"code" validate:"required,min=3,max=50"`
}
