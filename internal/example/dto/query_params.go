package dto

import (
	"strconv"

	"github.com/labstack/echo/v5"
)

// QueryParams adalah parameter query untuk daftar merchant (paginasi + pencarian).
type QueryParams struct {
	Search string
	Limit  int
	Offset int
}

// ParseQueryParams membaca query string dengan default yang aman (limit dibatasi).
func ParseQueryParams(c *echo.Context) QueryParams {
	limit := atoiDefault(c.QueryParam("limit"), 20)
	if limit <= 0 || limit > 100 {
		limit = 20 // batasi agar tidak menarik terlalu banyak baris
	}
	offset := atoiDefault(c.QueryParam("offset"), 0)
	if offset < 0 {
		offset = 0
	}
	return QueryParams{
		Search: c.QueryParam("search"),
		Limit:  limit,
		Offset: offset,
	}
}

func atoiDefault(s string, fallback int) int {
	if n, err := strconv.Atoi(s); err == nil {
		return n
	}
	return fallback
}
