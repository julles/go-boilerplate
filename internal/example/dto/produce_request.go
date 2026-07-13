package dto

// ProduceRequest adalah body untuk enqueue satu pesan ke queue.
type ProduceRequest struct {
	Message string `json:"message" validate:"required,min=1,max=500"`
}
