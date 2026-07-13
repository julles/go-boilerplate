package example

import (
	"context"
	"log/slog"
	"time"

	"github.com/julles/go-boilerplate/internal/shared/scheduler"
)

// RegisterSchedule mendaftarkan cron modul.
// Pola di sini: "select rentang -> proses langsung" tanpa lewat queue.
func RegisterSchedule(s *scheduler.Scheduler, svc *Service) error {
	// Tiap 5 menit: scan merchant yang dibuat dalam 24 jam terakhir, lalu proses.
	return s.Add("@every 1m", func() {
		ctx := context.Background()
		n, err := svc.ScanRecent(ctx, 24*time.Hour)
		if err != nil {
			slog.ErrorContext(ctx, "scan-recent gagal", "error", err)
			return
		}
		slog.InfoContext(ctx, "scan-recent selesai", "jumlah", n)
	})
}
