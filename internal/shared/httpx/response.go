// Package httpx berisi envelope response dan error handler yang dipakai bersama.
package httpx

// Response adalah envelope JSON yang bentuknya seragam untuk semua endpoint.
//
// Semua endpoint pakai bentuk yang sama biar client cukup nulis satu logika parsing:
// cek `success` dulu, baru ambil `data` atau `message`. omitempty kita pakai supaya
// field yang lagi nggak relevan (mis. Message pas sukses) nggak ikut kekirim.
type Response struct {
	Success bool   `json:"success"`           // true = sukses, false = error
	Message string `json:"message,omitempty"` // pesan yang aman buat client (biasanya muncul pas error)
	Data    any    `json:"data,omitempty"`    // payload hasilnya (cuma diisi pas sukses)
}

// OK membungkus data hasil untuk response yang sukses.
func OK(data any) Response {
	return Response{Success: true, Data: data}
}

// Err membungkus pesan untuk response yang error.
//
// Penting: message di sini harus pesan yang AMAN dilihat client (mis. "invalid input"),
// bukan detail internal kayak stack trace atau query DB. Detail internal biar dicatat
// di log sama ErrorHandler aja, nggak usah dikirim keluar.
func Err(message string) Response {
	return Response{Success: false, Message: message}
}
