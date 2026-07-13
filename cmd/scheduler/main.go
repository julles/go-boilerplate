package main

import (
	"context"
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	"github.com/julles/go-boilerplate/internal/example"
	"github.com/julles/go-boilerplate/internal/shared/cache"
	"github.com/julles/go-boilerplate/internal/shared/config"
	"github.com/julles/go-boilerplate/internal/shared/database"
	"github.com/julles/go-boilerplate/internal/shared/observability"
	"github.com/julles/go-boilerplate/internal/shared/scheduler"
)

func main() {
	// Siapkan logger global lebih dulu agar error startup pun terformat konsisten.
	observability.SetupLogger()

	// Alur nyata ada di run(); os.Exit hanya dipanggil di sini setelah run() selesai
	// supaya semua `defer` (penutupan DB/Redis) sempat dieksekusi—os.Exit melewati defer.
	if err := run(); err != nil {
		slog.Error("scheduler berhenti karena error", "error", err)
		os.Exit(1)
	}
}

func run() error {
	// Context untuk fase startup (buka pool DB, konek Redis).
	ctx := context.Background()

	// Muat config paling awal; berhenti bila invalid sebelum membuka koneksi apa pun.
	cfg, err := config.Load()
	if err != nil {
		return err
	}

	// Job terjadwal biasanya mengolah/agregasi data, sehingga scheduler tetap butuh
	// akses DB. Pool + defer Close mengikuti pola yang sama dengan API server & worker.
	pool, err := database.NewPool(ctx, cfg.DatabaseURL, cfg.DBPool)
	if err != nil {
		return err
	}
	defer pool.Close()

	// Redis untuk cache yang dipakai service saat job berjalan.
	rdb, err := cache.NewClient(ctx, cfg.RedisURL, cfg.RedisPool)
	if err != nil {
		return err
	}
	defer rdb.Close()

	// Rakit service, lalu daftarkan jadwal tiap modul ke scheduler. RegisterSchedule
	// mengembalikan error (mis. ekspresi cron salah), dan kita berhenti lebih awal bila
	// gagal supaya tidak menjalankan scheduler dengan konfigurasi jadwal yang cacat.
	// Registrasi jadwal tiap modul (satu baris per modul).
	svc := example.NewService(example.NewRepository(pool), cache.New(rdb))
	sch := scheduler.New()
	if err := example.RegisterSchedule(sch, svc); err != nil {
		return err
	}

	// Start menjalankan cron di goroutine latar sendiri dan langsung kembali (non-blocking).
	// Karena itu kita perlu menahan main goroutine secara eksplisit di bawah, kalau tidak
	// proses akan langsung keluar dan tak ada job yang pernah jalan.
	sch.Start()
	slog.Info("scheduler dimulai")

	// Berbeda dengan API/worker yang blocking sendiri, di sini kita pasang penangkap sinyal
	// manual. NotifyContext menghasilkan context yang dibatalkan saat SIGINT/SIGTERM tiba;
	// `<-sigCtx.Done()` memblokir main goroutine sampai sinyal itu datang. stop() (via defer)
	// melepas pendaftaran sinyal saat fungsi selesai.
	// Tunggu SIGINT/SIGTERM, lalu hentikan cron dengan rapi.
	sigCtx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	<-sigCtx.Done()

	// Setelah sinyal diterima, hentikan cron dengan rapi: Stop mencegah job baru dipicu.
	// Baru setelah ini fungsi kembali, sehingga defer di atas menutup Redis lalu DB (LIFO).
	slog.Info("scheduler shutdown")
	sch.Stop()
	return nil
}
