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
	// "@every 1m" adalah spesifikasi interval cron: fungsi callback dijalankan
	// scheduler secara berkala tanpa perlu trigger dari luar.
	return s.Add("@every 1m", func() {
		// context.Background() dipakai karena job ini berjalan mandiri (bukan turunan
		// dari request HTTP), jadi tidak ada context induk yang bisa diwarisi.
		ctx := context.Background()
		n, err := svc.ScanRecent(ctx, 24*time.Hour)
		if err != nil {
			// Job cron tidak punya pemanggil untuk menerima error; satu-satunya cara
			// error terlihat adalah lewat log. return agar tick ini berhenti (tick
			// berikutnya tetap jalan sesuai jadwal).
			slog.ErrorContext(ctx, "scan-recent gagal", "error", err)
			return
		}
		// Catat jumlah yang diproses untuk bukti job berjalan & keperluan monitoring.
		slog.InfoContext(ctx, "scan-recent selesai", "jumlah", n)
	})
}
