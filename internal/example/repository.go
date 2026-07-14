package example

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/julles/go-boilerplate/internal/example/dto"
)

// Merchant merepresentasikan satu baris di tabel merchant.merchants.
type Merchant struct {
	ID        string
	Code      string
	Status    string
	CreatedAt time.Time
}

// Repository yang megang akses ke tabel merchant.merchants. Semua query-nya
// parameterized ($1, $2) buat mencegah SQL injection.
type Repository struct {
	db *pgxpool.Pool
}

func NewRepository(db *pgxpool.Pool) *Repository {
	return &Repository{db: db}
}

// Create nyimpen merchant baru (kolom lain pakai default DB) lalu balikin baris hasilnya.
func (r *Repository) Create(ctx context.Context, code string) (Merchant, error) {
	// $1 itu placeholder parameterized: nilai code dikirim terpisah dari teks SQL,
	// jadi driver yang urus escaping-nya dan SQL injection jadi mustahil terjadi.
	// RETURNING kita pakai biar id/status/created_at hasil default DB langsung didapat
	// dalam satu roundtrip — nggak perlu SELECT ulang setelah INSERT.
	const q = `
		INSERT INTO merchant.merchants (code)
		VALUES ($1)
		RETURNING id::text, code, status::text, created_at`

	// QueryRow+Scan: eksekusi query lalu salin kolom hasilnya ke field struct sesuai
	// urutan di SELECT/RETURNING. id & status di-cast ::text biar kepetakan ke string Go.
	var m Merchant
	if err := r.db.QueryRow(ctx, q, code).Scan(&m.ID, &m.Code, &m.Status, &m.CreatedAt); err != nil {
		// Bungkus error pakai %w biar konteksnya ("insert merchant") kelihatan jelas di
		// log, tapi error aslinya tetap bisa di-unwrap/dicek sama pemanggil.
		return Merchant{}, fmt.Errorf("insert merchant: %w", err)
	}
	return m, nil
}

// GetByID ambil satu merchant. Balikin (Merchant{}, pgx.ErrNoRows) kalau datanya nggak ada.
func (r *Repository) GetByID(ctx context.Context, id string) (Merchant, error) {
	const q = `
		SELECT id::text, code, status::text, created_at
		FROM merchant.merchants
		WHERE id = $1`

	var m Merchant
	if err := r.db.QueryRow(ctx, q, id).Scan(&m.ID, &m.Code, &m.Status, &m.CreatedAt); err != nil {
		// id yang bukan UUID valid (kode 22P02) mustahil cocok → anggap aja nggak ada.
		// Tanpa penanganan ini, id ngasal dari user bakal memicu error invalid_text
		// (500). Kita ubah jadi ErrNoRows biar handler bisa balas 404 yang bener.
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "22P02" {
			return Merchant{}, pgx.ErrNoRows
		}
		// Error lain (termasuk pgx.ErrNoRows pas baris-nya memang nggak ada) diteruskan
		// apa adanya biar bisa dibedakan sama pemanggil.
		return Merchant{}, err
	}
	return m, nil
}

// ListRecent ambil merchant yang dibuat dalam rentang [from, to] — satu query, parameterized.
func (r *Repository) ListRecent(ctx context.Context, from, to time.Time) ([]Merchant, error) {
	const q = `
		SELECT id::text, code, status::text, created_at
		FROM merchant.merchants
		WHERE created_at BETWEEN $1 AND $2
		ORDER BY created_at DESC`

	rows, err := r.db.Query(ctx, q, from, to)
	if err != nil {
		return nil, fmt.Errorf("query merchants recent: %w", err)
	}
	// defer Close mastiin koneksi/rows dikembalikan ke connection pool walau loop
	// keluar lebih awal gara-gara error — ini yang mencegah kebocoran koneksi.
	defer rows.Close()

	merchants := make([]Merchant, 0)
	// rows.Next() majuin cursor baris demi baris; Scan-nya nyalin kolom ke struct.
	for rows.Next() {
		var m Merchant
		if err := rows.Scan(&m.ID, &m.Code, &m.Status, &m.CreatedAt); err != nil {
			return nil, fmt.Errorf("scan merchant: %w", err)
		}
		merchants = append(merchants, m)
	}
	// Wajib cek rows.Err() setelah loop: Next() balikin false baik pas data-nya habis
	// maupun pas ada error di tengah iterasi — tanpa cek ini, error (misal koneksi
	// putus) bakal ketelan diam-diam.
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterasi merchants recent: %w", err)
	}
	return merchants, nil
}

// List ambil daftar merchant dengan paginasi + pencarian opsional — satu query, biar hindari N+1.
func (r *Repository) List(ctx context.Context, p dto.QueryParams) ([]Merchant, error) {
	// Pencarian opsional-nya kita bikin dalam satu query: kalau $1 (search) kosong,
	// klausa "$1 = ''" bernilai true sehingga filter code diabaikan (ambil semua).
	// Kalau terisi, ILIKE '%..%' bakal nyari substring secara case-insensitive. Nilai
	// search tetap parameterized ($1), jadi aman dari SQL injection walau dipakai dalam
	// pola LIKE. LIMIT/OFFSET juga parameterized buat paginasi.
	const q = `
		SELECT id::text, code, status::text, created_at
		FROM merchant.merchants
		WHERE ($1 = '' OR code ILIKE '%' || $1 || '%')
		ORDER BY created_at DESC
		LIMIT $2 OFFSET $3`

	rows, err := r.db.Query(ctx, q, p.Search, p.Limit, p.Offset)
	if err != nil {
		return nil, fmt.Errorf("query merchants: %w", err)
	}
	defer rows.Close()

	// Kapasitas awal = Limit karena hasilnya paling banyak ya segitu — hemat realokasi.
	merchants := make([]Merchant, 0, p.Limit)
	for rows.Next() {
		var m Merchant
		if err := rows.Scan(&m.ID, &m.Code, &m.Status, &m.CreatedAt); err != nil {
			return nil, fmt.Errorf("scan merchant: %w", err)
		}
		merchants = append(merchants, m)
	}
	// Cek error iterasi (sama seperti di ListRecent) biar kegagalan di tengah stream nggak lolos.
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterasi merchants: %w", err)
	}
	return merchants, nil
}
