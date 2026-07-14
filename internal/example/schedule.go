package example

import (
	"context"
	"log/slog"
	"time"

	"github.com/julles/go-boilerplate/internal/shared/scheduler"
)

// RegisterSchedule mendaftarkan cron milik modul ini.
// Polanya: "select rentang lalu proses langsung", tanpa mampir ke queue.
func RegisterSchedule(s *scheduler.Scheduler, svc *Service) error {
	// Tiap 5 menit: scan merchant yang dibuat dalam 24 jam terakhir, lalu diproses.
	// "@every 1m" itu spesifikasi interval cron: fungsi callback-nya dijalankan
	// scheduler secara berkala tanpa perlu trigger dari luar.
	return s.Add("@every 1m", func() {
		// Pakai context.Background() karena job ini jalan mandiri (bukan turunan dari
		// request HTTP), jadi nggak ada context induk yang bisa diwarisi.
		ctx := context.Background()
		n, err := svc.ScanRecent(ctx, 24*time.Hour)
		if err != nil {
			// Job cron nggak punya pemanggil buat nampung error; satu-satunya cara
			// error-nya kelihatan ya lewat log. return biar tick ini berhenti (tick
			// berikutnya tetap jalan sesuai jadwal).
			slog.ErrorContext(ctx, "scan-recent gagal", "error", err)
			return
		}
		// Catat jumlah yang diproses sebagai bukti job jalan & buat keperluan monitoring.
		slog.InfoContext(ctx, "scan-recent selesai", "jumlah", n)
	})
}
