package dto

import "time"

// MerchantResponse adalah bentuk merchant yang dikirim ke client.
//
// Sengaja dibuat sebagai DTO terpisah dari entity/model DB: hanya field yang
// memang boleh dilihat client yang diekspos di sini. Dengan begitu kolom internal
// (mis. data audit atau field sensitif) tidak ikut bocor ke response.
type MerchantResponse struct {
	ID        string    `json:"id"`
	Code      string    `json:"code"`
	Status    string    `json:"status"`
	CreatedAt time.Time `json:"created_at"`
}
