package dto

import (
	"strconv"

	"github.com/labstack/echo/v5"
)

// QueryParams adalah parameter query untuk daftar merchant (paginasi + pencarian).
// Nilai-nilai di sini sudah "bersih" (sudah lolos ParseQueryParams), jadi lapisan
// service/repository bisa memakainya langsung tanpa validasi ulang.
type QueryParams struct {
	Search string // kata kunci pencarian (opsional, boleh kosong)
	Limit  int    // jumlah maksimum baris per halaman
	Offset int    // jumlah baris yang dilewati (paginasi)
}

// ParseQueryParams membaca query string dengan default yang aman (limit dibatasi).
//
// Query string berasal dari client, sehingga di sini kita lakukan sanitasi:
// nilai yang tidak valid/di luar batas diganti default alih-alih menolak request,
// supaya endpoint list tetap ramah dipakai.
func ParseQueryParams(c *echo.Context) QueryParams {
	// Default limit 20 bila param tidak ada atau bukan angka.
	limit := atoiDefault(c.QueryParam("limit"), 20)
	// Tolak nilai non-positif dan batasi maksimum 100. Batas atas ini penting
	// untuk mencegah client meminta ribuan baris sekaligus (beban DB & memori,
	// potensi DoS), jadi limit selalu di-clamp ke rentang aman.
	if limit <= 0 || limit > 100 {
		limit = 20 // batasi agar tidak menarik terlalu banyak baris
	}
	// Default offset 0 (halaman pertama).
	offset := atoiDefault(c.QueryParam("offset"), 0)
	// Offset negatif tidak bermakna untuk paginasi dan bisa memicu error di DB,
	// jadi dipaksa minimal 0.
	if offset < 0 {
		offset = 0
	}
	return QueryParams{
		Search: c.QueryParam("search"),
		Limit:  limit,
		Offset: offset,
	}
}

// atoiDefault mengubah string ke int, dan mengembalikan fallback bila string
// kosong atau bukan angka. Dipakai agar param opsional tidak membuat request gagal.
func atoiDefault(s string, fallback int) int {
	if n, err := strconv.Atoi(s); err == nil {
		return n
	}
	return fallback
}
