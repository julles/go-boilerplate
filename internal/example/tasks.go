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
func (h *TaskHandler) HandleMessage(ctx context.Context, t *asynq.Task) error {
	var p MessagePayload
	if err := json.Unmarshal(t.Payload(), &p); err != nil {
		// error non-retryable: payload rusak, retry tak akan menolong.
		return fmt.Errorf("unmarshal message payload: %w: %w", err, asynq.SkipRetry)
	}
	slog.InfoContext(ctx, "memproses pesan", "message", p.Message)
	return nil
}

// RegisterTasks mendaftarkan handler task modul ke mux worker.
func RegisterTasks(mux *asynq.ServeMux, svc *Service) {
	h := NewTaskHandler(svc)
	mux.HandleFunc(TypeMessage, h.HandleMessage)
}
