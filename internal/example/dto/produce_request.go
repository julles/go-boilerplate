package dto

// ProduceRequest itu body request buat enqueue satu pesan ke queue.
//
// Sama kayak request lainnya, ini trust boundary — jadi isi pesannya divalidasi dulu
// sebelum dimasukin ke queue.
type ProduceRequest struct {
	// Message wajib diisi dan nggak boleh kosong (required + min=1) biar kita nggak
	// ngirim pesan hampa ke queue. max=500 batasin ukuran pesannya biar consumer/broker
	// nggak kebebanan payload yang kegedean.
	Message string `json:"message" validate:"required,min=1,max=500"`
}
