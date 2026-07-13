// Package scheduler adalah wrapper tipis di atas robfig/cron untuk menjalankan
// tugas terjadwal. Tiap job adalah fungsi Go bebas (boleh query DB, proses langsung,
// atau enqueue ke queue).
package scheduler

import (
	"fmt"

	"github.com/robfig/cron/v3"
)

type Scheduler struct {
	c *cron.Cron
}

func New() *Scheduler {
	return &Scheduler{c: cron.New()}
}

// Add mendaftarkan job pada spec cron (mis. "@every 5m", "0 2 * * *"). Error saat
// spec tidak valid dikembalikan supaya jadwal yang salah ketahuan saat startup,
// bukan diam-diam tidak pernah jalan.
func (s *Scheduler) Add(spec string, job func()) error {
	if _, err := s.c.AddFunc(spec, job); err != nil {
		return fmt.Errorf("daftar cron %q: %w", spec, err)
	}
	return nil
}

// Start menjalankan penjadwal di goroutine terpisah (non-blocking).
func (s *Scheduler) Start() { s.c.Start() }

// Stop menghentikan penjadwalan lalu BLOCK menunggu job yang sedang berjalan selesai
// (Stop() mengembalikan context yang Done saat semua job tuntas). Ini mencegah job
// terpotong di tengah saat graceful shutdown.
func (s *Scheduler) Stop() { <-s.c.Stop().Done() }
