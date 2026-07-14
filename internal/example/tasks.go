package example

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"

	"github.com/hibiken/asynq"
)

// TypeMessage itu type task buat memproses satu pesan.
const TypeMessage = "example:message"

// MessagePayload adalah isi payload dari task example:message.
type MessagePayload struct {
	Message string `json:"message"`
}

// NewMessageTask bikin task example:message dari sebuah pesan (dipakai producer/API).
func NewMessageTask(msg string) (*asynq.Task, error) {
	// Payload-nya diserialisasi ke JSON karena queue cuma bisa bawa byte; nanti worker
	// yang men-deserialisasi balik ke struct yang sama. Pakai helper ini (bukan
	// ngerakit task manual di handler) bikin TypeMessage & format payload satu sumber.
	b, err := json.Marshal(MessagePayload{Message: msg})
	if err != nil {
		return nil, fmt.Errorf("marshal message payload: %w", err)
	}
	return asynq.NewTask(TypeMessage, b), nil
}

// TaskHandler memproses task queue pakai service modul yang sama dengan API.
type TaskHandler struct {
	svc *Service
}

func NewTaskHandler(svc *Service) *TaskHandler {
	return &TaskHandler{svc: svc}
}

// HandleMessage memproses satu pesan. Sifatnya idempoten (di sini cuma nyatet log).
// Idempoten itu penting di queue: asynq bisa ngirim ulang task yang sama (retry pas
// error, atau karena at-least-once delivery), jadi handler-nya harus aman kalau
// dijalankan lebih dari sekali untuk payload yang sama.
func (h *TaskHandler) HandleMessage(ctx context.Context, t *asynq.Task) error {
	// Deserialisasi payload byte-nya balik ke struct.
	var p MessagePayload
	if err := json.Unmarshal(t.Payload(), &p); err != nil {
		// error non-retryable: payload-nya rusak, retry pun nggak bakal nolong.
		// Nge-gabung asynq.SkipRetry ke error itu ngasih tahu worker biar TIDAK
		// ngulang task ini (payload cacat bakal gagal terus = buang-buang kuota retry).
		return fmt.Errorf("unmarshal message payload: %w: %w", err, asynq.SkipRetry)
	}
	slog.InfoContext(ctx, "memproses pesan", "message", p.Message)
	// return nil nandain task-nya sukses, jadi asynq bakal ngehapus dari antrean.
	return nil
}

// RegisterTasks mendaftarkan handler task modul ke mux worker.
// mux ini memetakan tipe task (TypeMessage) ke fungsi handler-nya, mirip router HTTP
// tapi buat pesan queue — worker pakai ini buat ngarahin tiap task ke handler-nya.
func RegisterTasks(mux *asynq.ServeMux, svc *Service) {
	h := NewTaskHandler(svc)
	mux.HandleFunc(TypeMessage, h.HandleMessage)
}
