package dto

import (
	"strconv"

	"github.com/labstack/echo/v5"
)

// QueryParams itu parameter query buat daftar merchant (paginasi + pencarian).
// Nilai-nilai di sini udah "bersih" (udah lolos ParseQueryParams), jadi lapisan
// service/repository bisa langsung pakai tanpa validasi ulang.
type QueryParams struct {
	Search string // kata kunci pencarian (opsional, boleh kosong)
	Limit  int    // jumlah maksimum baris per halaman
	Offset int    // jumlah baris yang dilewati (buat paginasi)
}

// ParseQueryParams baca query string dengan default yang aman (limit dibatasi).
//
// Query string datang dari client, jadi di sini kita sanitasi dulu: nilai yang
// nggak valid/di luar batas kita ganti default alih-alih nolak request, biar
// endpoint list-nya tetap enak dipakai.
func ParseQueryParams(c *echo.Context) QueryParams {
	// Default-nya limit 20 kalau param-nya nggak ada atau bukan angka.
	limit := atoiDefault(c.QueryParam("limit"), 20)
	// Tolak nilai non-positif dan batasi maksimum 100. Batas atas ini penting buat
	// nyegah client minta ribuan baris sekaligus (bikin berat DB & memori, potensi
	// DoS), jadi limit selalu di-clamp ke rentang yang aman.
	if limit <= 0 || limit > 100 {
		limit = 20 // batasi biar nggak narik kebanyakan baris
	}
	// Default-nya offset 0 (halaman pertama).
	offset := atoiDefault(c.QueryParam("offset"), 0)
	// Offset negatif nggak bermakna buat paginasi dan bisa memicu error di DB, jadi
	// dipaksa minimal 0.
	if offset < 0 {
		offset = 0
	}
	return QueryParams{
		Search: c.QueryParam("search"),
		Limit:  limit,
		Offset: offset,
	}
}

// atoiDefault ngubah string ke int, dan balikin fallback kalau string-nya kosong
// atau bukan angka. Dipakai biar param opsional nggak sampai bikin request gagal.
func atoiDefault(s string, fallback int) int {
	if n, err := strconv.Atoi(s); err == nil {
		return n
	}
	return fallback
}
