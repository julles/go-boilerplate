// Package httpx berisi envelope response dan error handler bersama.
package httpx

// Response adalah envelope JSON seragam untuk semua endpoint.
//
// Semua endpoint memakai bentuk yang sama supaya client cukup menulis satu logika
// parsing: cek `success` dulu, baru ambil `data` atau `message`. omitempty dipakai
// agar field yang tidak relevan (mis. Message saat sukses) tidak ikut dikirim.
type Response struct {
	Success bool   `json:"success"`           // true = sukses, false = error
	Message string `json:"message,omitempty"` // pesan aman untuk client (biasanya saat error)
	Data    any    `json:"data,omitempty"`    // payload hasil (hanya saat sukses)
}

// OK membungkus data hasil untuk response sukses.
func OK(data any) Response {
	return Response{Success: true, Data: data}
}

// Err membungkus pesan untuk response error.
//
// Penting: message di sini harus pesan yang AMAN dilihat client (mis. "invalid input"),
// bukan detail internal seperti stack trace atau query DB. Detail internal dicatat
// di log oleh ErrorHandler, tidak dikirim ke luar.
func Err(message string) Response {
	return Response{Success: false, Message: message}
}
