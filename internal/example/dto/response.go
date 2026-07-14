package dto

import "time"

// MerchantResponse itu bentuk merchant yang dikirim ke client.
//
// Sengaja dibikin DTO terpisah dari entity/model DB: cuma field yang memang boleh
// dilihat client yang diekspos di sini. Dengan begitu kolom internal (misal data
// audit atau field sensitif) nggak ikut bocor ke response.
type MerchantResponse struct {
	ID        string    `json:"id"`
	Code      string    `json:"code"`
	Status    string    `json:"status"`
	CreatedAt time.Time `json:"created_at"`
}
