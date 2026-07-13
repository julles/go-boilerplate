package dto

import "time"

// MerchantResponse adalah bentuk merchant yang dikirim ke client.
type MerchantResponse struct {
	ID        string    `json:"id"`
	Code      string    `json:"code"`
	Status    string    `json:"status"`
	CreatedAt time.Time `json:"created_at"`
}
