// Package httpx berisi envelope response dan error handler bersama.
package httpx

// Response adalah envelope JSON seragam untuk semua endpoint.
type Response struct {
	Success bool   `json:"success"`
	Message string `json:"message,omitempty"`
	Data    any    `json:"data,omitempty"`
}

// OK membungkus data hasil untuk response sukses.
func OK(data any) Response {
	return Response{Success: true, Data: data}
}

// Err membungkus pesan (aman untuk client) untuk response error.
func Err(message string) Response {
	return Response{Success: false, Message: message}
}
