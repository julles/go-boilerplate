package dto

// ProduceRequest adalah body untuk enqueue satu pesan ke queue.
//
// Sama seperti request lain, ini trust boundary sehingga isi pesan divalidasi
// sebelum dimasukkan ke queue.
type ProduceRequest struct {
	// Message wajib diisi dan tidak boleh kosong (required + min=1) supaya kita
	// tidak mengirim pesan hampa ke queue. max=500 membatasi ukuran pesan agar
	// consumer/broker tidak dibebani payload berlebihan.
	Message string `json:"message" validate:"required,min=1,max=500"`
}
