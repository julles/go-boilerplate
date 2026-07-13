package example

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"

	"github.com/hibiken/asynq"
)

// TypeMessage adalah type task untuk memproses satu pesan.
const TypeMessage = "example:message"

// MessagePayload adalah isi task example:message.
type MessagePayload struct {
	Message string `json:"message"`
}

// NewMessageTask membuat task example:message dari sebuah pesan (dipakai producer/API).
func NewMessageTask(msg string) (*asynq.Task, error) {
	// Payload diserialisasi ke JSON karena queue hanya membawa byte; worker nanti
	// men-deserialisasi kembali ke struct yang sama. Memakai helper ini (bukan
	// merangkai task manual di handler) menjaga TypeMessage & format payload satu sumber.
	b, err := json.Marshal(MessagePayload{Message: msg})
	if err != nil {
		return nil, fmt.Errorf("marshal message payload: %w", err)
	}
	return asynq.NewTask(TypeMessage, b), nil
}

// TaskHandler memproses task queue memakai service modul yang sama dengan API.
type TaskHandler struct {
	svc *Service
}

func NewTaskHandler(svc *Service) *TaskHandler {
	return &TaskHandler{svc: svc}
}

// HandleMessage memproses satu pesan. Idempoten (hanya mencatat log).
// Idempoten penting di queue: asynq bisa mengirim ulang task yang sama (retry saat
// error, atau at-least-once delivery), jadi handler harus aman bila dijalankan
// lebih dari sekali untuk payload yang sama.
func (h *TaskHandler) HandleMessage(ctx context.Context, t *asynq.Task) error {
	// Deserialisasi payload byte kembali ke struct.
	var p MessagePayload
	if err := json.Unmarshal(t.Payload(), &p); err != nil {
		// error non-retryable: payload rusak, retry tak akan menolong.
		// Menggabungkan asynq.SkipRetry ke error memberi tahu worker agar TIDAK
		// mengulang task ini (payload cacat akan gagal selamanya = buang kuota retry).
		return fmt.Errorf("unmarshal message payload: %w: %w", err, asynq.SkipRetry)
	}
	slog.InfoContext(ctx, "memproses pesan", "message", p.Message)
	// return nil menandai task sukses sehingga asynq menghapusnya dari antrean.
	return nil
}

// RegisterTasks mendaftarkan handler task modul ke mux worker.
// mux memetakan tipe task (TypeMessage) ke fungsi handler-nya, mirip router HTTP
// tetapi untuk pesan queue -- worker memakainya untuk mengarahkan tiap task.
func RegisterTasks(mux *asynq.ServeMux, svc *Service) {
	h := NewTaskHandler(svc)
	mux.HandleFunc(TypeMessage, h.HandleMessage)
}
