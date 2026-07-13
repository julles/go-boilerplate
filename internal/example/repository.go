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

// Merchant adalah representasi satu baris tabel merchant.merchants.
type Merchant struct {
	ID        string
	Code      string
	Status    string
	CreatedAt time.Time
}

// Repository mengakses tabel merchant.merchants. Semua query parameterized ($1,$2)
// untuk mencegah SQL injection.
type Repository struct {
	db *pgxpool.Pool
}

func NewRepository(db *pgxpool.Pool) *Repository {
	return &Repository{db: db}
}

// Create menyimpan merchant baru (kolom lain memakai default DB) dan mengembalikan baris hasilnya.
func (r *Repository) Create(ctx context.Context, code string) (Merchant, error) {
	// $1 adalah placeholder parameterized: nilai code dikirim terpisah dari teks SQL,
	// sehingga driver yang meng-escape-nya dan SQL injection menjadi mustahil.
	// RETURNING dipakai agar id/status/created_at hasil default DB langsung didapat
	// dalam satu roundtrip (tanpa perlu SELECT ulang setelah INSERT).
	const q = `
		INSERT INTO merchant.merchants (code)
		VALUES ($1)
		RETURNING id::text, code, status::text, created_at`

	// QueryRow+Scan: eksekusi lalu salin kolom hasil ke field struct sesuai urutan
	// di SELECT/RETURNING. id & status di-cast ::text agar dipetakan ke string Go.
	var m Merchant
	if err := r.db.QueryRow(ctx, q, code).Scan(&m.ID, &m.Code, &m.Status, &m.CreatedAt); err != nil {
		// Bungkus error dengan %w agar konteks ("insert merchant") jelas di log
		// namun error asli tetap bisa di-unwrap/dicek pemanggil.
		return Merchant{}, fmt.Errorf("insert merchant: %w", err)
	}
	return m, nil
}

// GetByID mengambil satu merchant. Mengembalikan (Merchant{}, pgx.ErrNoRows) bila tidak ada.
func (r *Repository) GetByID(ctx context.Context, id string) (Merchant, error) {
	const q = `
		SELECT id::text, code, status::text, created_at
		FROM merchant.merchants
		WHERE id = $1`

	var m Merchant
	if err := r.db.QueryRow(ctx, q, id).Scan(&m.ID, &m.Code, &m.Status, &m.CreatedAt); err != nil {
		// id yang bukan UUID valid (22P02) tak mungkin cocok → anggap tidak ada.
		// Tanpa penanganan ini, id sembarang dari user memicu error tipe invalid_text
		// (500). Kita ubah jadi ErrNoRows supaya handler bisa membalas 404 yang benar.
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "22P02" {
			return Merchant{}, pgx.ErrNoRows
		}
		// Error lain (termasuk pgx.ErrNoRows saat baris memang tak ada) diteruskan
		// apa adanya agar bisa dibedakan pemanggil.
		return Merchant{}, err
	}
	return m, nil
}

// ListRecent mengambil merchant yang dibuat dalam rentang [from, to] (satu query, parameterized).
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
	// defer Close memastikan koneksi/rows dikembalikan ke pool meski loop keluar
	// lebih awal karena error (mencegah kebocoran koneksi).
	defer rows.Close()

	merchants := make([]Merchant, 0)
	// rows.Next() memajukan cursor baris demi baris; Scan menyalin kolom ke struct.
	for rows.Next() {
		var m Merchant
		if err := rows.Scan(&m.ID, &m.Code, &m.Status, &m.CreatedAt); err != nil {
			return nil, fmt.Errorf("scan merchant: %w", err)
		}
		merchants = append(merchants, m)
	}
	// Wajib cek rows.Err() setelah loop: Next() mengembalikan false baik saat data
	// habis maupun saat terjadi error di tengah iterasi -- tanpa cek ini, error
	// (mis. koneksi putus) akan tertelan diam-diam.
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterasi merchants recent: %w", err)
	}
	return merchants, nil
}

// List mengambil daftar merchant dengan paginasi + pencarian opsional (satu query, hindari N+1).
func (r *Repository) List(ctx context.Context, p dto.QueryParams) ([]Merchant, error) {
	// Pencarian opsional dibuat dalam satu query: bila $1 (search) kosong, klausa
	// "$1 = ''" bernilai true sehingga filter code diabaikan (ambil semua). Bila
	// terisi, ILIKE '%..%' melakukan pencarian substring case-insensitive. Nilai
	// search tetap parameterized ($1), jadi aman dari SQL injection walau dipakai
	// dalam pola LIKE. LIMIT/OFFSET juga parameterized untuk paginasi.
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

	// Kapasitas awal = Limit karena hasil paling banyak sejumlah itu (hemat realokasi).
	merchants := make([]Merchant, 0, p.Limit)
	for rows.Next() {
		var m Merchant
		if err := rows.Scan(&m.ID, &m.Code, &m.Status, &m.CreatedAt); err != nil {
			return nil, fmt.Errorf("scan merchant: %w", err)
		}
		merchants = append(merchants, m)
	}
	// Cek error iterasi (lihat ListRecent) agar kegagalan di tengah stream tidak lolos.
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterasi merchants: %w", err)
	}
	return merchants, nil
}
