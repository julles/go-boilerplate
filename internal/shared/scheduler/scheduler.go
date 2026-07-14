// Package scheduler cuma wrapper tipis di atas robfig/cron buat ngejalanin tugas
// terjadwal. Tiap job itu fungsi Go biasa yang bebas mau ngapain — boleh query DB,
// proses langsung, atau enqueue ke queue.
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

// Add ndaftarin job ke sebuah spec cron (mis. "@every 5m", "0 2 * * *"). Kalau
// spec-nya nggak valid, error-nya kita balikin biar jadwal yang salah langsung
// ketahuan pas startup, bukan malah diam-diam nggak pernah jalan.
func (s *Scheduler) Add(spec string, job func()) error {
	if _, err := s.c.AddFunc(spec, job); err != nil {
		return fmt.Errorf("daftar cron %q: %w", spec, err)
	}
	return nil
}

// Start ngejalanin scheduler di goroutine terpisah, jadi sifatnya non-blocking.
func (s *Scheduler) Start() { s.c.Start() }

// Stop nyetop penjadwalan lalu BLOCK nungguin job yang lagi jalan sampai kelar
// (Stop() balikin context yang Done pas semua job tuntas). Ini yang nyegah job
// kepotong di tengah jalan pas graceful shutdown.
func (s *Scheduler) Stop() { <-s.c.Stop().Done() }
