package dim

import (
	"context"
	"fmt"
	"strings"
)

// timeColumns adalah kolom waktu satu tabel bawaan dim.
type timeColumns struct {
	table   string
	columns []string
}

// frameworkTimeColumns adalah seluruh kolom waktu di tabel bawaan dim, di luar
// applied_at milik tabel pencatat migrasi (lihat ensureMigrationsTable).
var frameworkTimeColumns = []timeColumns{
	{"users", []string{"created_at", "updated_at"}},
	{"refresh_tokens", []string{"created_at", "expires_at", "revoked_at"}},
	{"password_reset_tokens", []string{"created_at", "expires_at", "used_at"}},
	{"token_blocklist", []string{"expires_at"}},
	{"rate_limits", []string{"expires_at"}},
}

const (
	pgTimestamp   = "timestamp without time zone"
	pgTimestamptz = "timestamp with time zone"
)

// convertTimestampsToTimestamptzMigration mengubah kolom waktu tabel bawaan yang
// dibuat sebelum v0.15.0 dari TIMESTAMP ke TIMESTAMPTZ.
//
// Kolom tanpa zona hanya benar selama setiap penulis menormalkan ke UTC dan
// setiap sesi berjalan di zona UTC: pgx membuang zona time.Time saat menulis ke
// kolom tanpa zona, dan perbandingan dengan NOW() menafsirkan jam dinding itu
// di zona sesi. Nilai lama adalah jam dinding UTC — store dim selalu menulis
// .UTC() — sehingga konversinya memakai AT TIME ZONE 'UTC', yang mempertahankan
// instannya. `USING col::timestamptz` akan menggesernya sebesar offset sesi.
//
// Hanya kolom yang masih bertipe TIMESTAMP yang diubah, jadi migrasi ini tidak
// berefek pada database yang dibuat dengan DDL baru dan aman dijalankan ulang.
// Tabel yang tidak ada dilewati. SQLite tidak punya tipe berzona; migrasi ini
// tidak melakukan apa pun di sana.
func convertTimestampsToTimestamptzMigration() Migration {
	return Migration{
		Version: 6,
		Name:    "convert_timestamps_to_timestamptz",
		Up: func(db Database) error {
			return convertFrameworkTimeColumns(db, pgTimestamp, "TIMESTAMPTZ")
		},
		Down: func(db Database) error {
			return convertFrameworkTimeColumns(db, pgTimestamptz, "TIMESTAMP")
		},
	}
}

// convertFrameworkTimeColumns mengonversi semua tabel dalam satu transaksi:
// semuanya berubah, atau tidak ada yang berubah.
func convertFrameworkTimeColumns(db Database, from, to string) error {
	if db.DriverName() == "sqlite" {
		return nil
	}
	return db.WithTx(context.Background(), func(ctx context.Context, tx Tx) error {
		for _, tc := range frameworkTimeColumns {
			if err := convertTimeColumns(ctx, tx, tc.table, from, to, tc.columns...); err != nil {
				return err
			}
		}
		return nil
	})
}

// convertTimeColumns mengubah kolom-kolom table yang bertipe from menjadi to,
// dengan jam dinding dibaca dan ditulis dalam UTC. `col AT TIME ZONE 'UTC'`
// bekerja ke dua arah: TIMESTAMP → instan dari jam dinding UTC, TIMESTAMPTZ →
// jam dinding UTC dari instannya.
//
// Kolom yang tipenya bukan from — sudah dikonversi, atau tabelnya tidak ada —
// dilewati. Semua kolom satu tabel diubah dalam satu ALTER TABLE, sehingga
// tabelnya ditulis ulang sekali. Hanya untuk PostgreSQL.
//
// Pemeriksaan tipe dan ALTER-nya harus berada di transaksi yang sama: di luar
// transaksi, SELECT katalog dapat dirutekan ke read replica yang tertinggal,
// dan kolom yang sudah TIMESTAMPTZ akan dikonversi sekali lagi — menggeser
// nilainya sebesar offset zona sesi.
func convertTimeColumns(ctx context.Context, tx Tx, table, from, to string, columns ...string) error {
	var clauses []string
	for _, col := range columns {
		typ, err := postgresColumnType(ctx, tx, table, col)
		if err != nil {
			return fmt.Errorf("read type of %s.%s: %w", table, col, err)
		}
		if typ != from {
			continue
		}
		clauses = append(clauses, fmt.Sprintf("ALTER COLUMN %s TYPE %s USING %s AT TIME ZONE 'UTC'", col, to, col))
	}
	if len(clauses) == 0 {
		return nil
	}
	query := fmt.Sprintf("ALTER TABLE %s %s", table, strings.Join(clauses, ", "))
	if err := tx.Exec(ctx, query); err != nil {
		return fmt.Errorf("convert %s to %s: %w", table, to, err)
	}
	return nil
}

// postgresColumnType mengembalikan tipe kolom seperti format_type, atau string
// kosong bila tabel atau kolomnya tidak ada. table diselesaikan lewat
// search_path, sama seperti SQL tak-berkualifikasi, dan boleh `schema.tabel`.
func postgresColumnType(ctx context.Context, tx Tx, table, column string) (string, error) {
	var typ string
	err := tx.QueryRow(ctx, `
		SELECT format_type(a.atttypid, a.atttypmod)
		FROM pg_attribute a
		WHERE a.attrelid = to_regclass($1) AND a.attname = $2 AND NOT a.attisdropped`,
		table, column).Scan(&typ)
	if isNoRows(err) {
		return "", nil
	}
	return typ, err
}
