package dim

import (
	"context"
	"database/sql"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"
)

// GCRARateLimitStore mengimplementasikan RateLimitStore dengan Generic Cell Rate
// Algorithm — token bucket yang disimpan sebagai satu stempel waktu per kunci.
//
// Berbeda dari DatabaseRateLimitStore (fixed window): kunci yang ditolak pulih
// sedikit demi sedikit mengikuti laju limit/window, bukan menunggu window habis;
// request yang ditolak tidak menulis apa pun, sehingga retry selama penolakan
// tidak memperpanjang penolakannya; dan tidak ada lonjakan 2× di batas window.
//
// Semantiknya: `limit` request boleh datang sekaligus (burst), sesudah itu satu
// request tiap window/limit. Untuk limit=100 dan window=15 menit: 100 sekaligus,
// lalu satu tiap 9 detik.
//
// Memakai tabel rate_limits yang sama dengan DatabaseRateLimitStore — kolom
// expires_at menyimpan "waktu kedatangan teoretis" (TAT) kunci itu, kolom count
// tidak dipakai. Jalankan GetRateLimitMigrations atau InitSchema seperti biasa.
// Kedua store tidak boleh memakai kunci yang sama secara bergantian, karena
// keduanya membaca expires_at dengan makna berbeda.
//
// PostgreSQL: satu UPSERT atomik dengan klausa WHERE pada DO UPDATE — baris
// hanya ditulis bila request diizinkan, dan penolakan terbaca dari tidak adanya
// baris yang kembali. SQLite tidak punya aritmetika interval yang sepadan, jadi
// di sana keputusannya dihitung di Go dalam satu transaksi; satu-satunya penulis
// SQLite membuatnya tetap atomik.
type GCRARateLimitStore struct {
	db Database
}

// NewGCRARateLimitStore membuat store rate limit GCRA di atas database SQL.
//
// Parameters:
//   - db: koneksi database yang mengimplementasikan interface Database
func NewGCRARateLimitStore(db Database) *GCRARateLimitStore {
	return &GCRARateLimitStore{db: db}
}

// InitSchema membuat tabel rate_limits bila belum ada. Tabelnya sama dengan milik
// DatabaseRateLimitStore.
func (s *GCRARateLimitStore) InitSchema(ctx context.Context) error {
	return NewDatabaseRateLimitStore(s.db).InitSchema(ctx)
}

// Allow mengecek apakah request diizinkan dan, bila ya, memajukan TAT kunci.
func (s *GCRARateLimitStore) Allow(ctx context.Context, key string, limit int, window time.Duration) (bool, error) {
	allowed, _, err := s.AllowWithRetryAfter(ctx, key, limit, window)
	return allowed, err
}

// AllowWithRetryAfter sama dengan Allow, ditambah durasi sampai request berikutnya
// pada kunci ini akan diizinkan bila yang sekarang ditolak.
func (s *GCRARateLimitStore) AllowWithRetryAfter(ctx context.Context, key string, limit int, window time.Duration) (bool, time.Duration, error) {
	if limit <= 0 || window <= 0 {
		return false, window, nil
	}

	// Mikrodetik: presisi TIMESTAMP PostgreSQL, supaya nilai yang ditulis dan
	// yang dibaca kembali sama.
	now := time.Now().UTC().Truncate(time.Microsecond)

	if s.db.DriverName() == "sqlite" {
		return s.allowInTransaction(ctx, key, now, limit, window)
	}
	return s.allowWithUpsert(ctx, key, now, limit, window)
}

// allowWithUpsert adalah jalur PostgreSQL: satu pernyataan.
//
// Klausa WHERE pada DO UPDATE membuat baris yang tidak memenuhi syarat tidak
// ditulis dan tidak dikembalikan, jadi "tidak ada baris" berarti ditolak. Setiap
// placeholder muncul tepat sekali (lihat DatabaseRateLimitStore.Allow): nilai yang
// sama dikirim berulang sebagai placeholder terpisah.
func (s *GCRARateLimitStore) allowWithUpsert(ctx context.Context, key string, now time.Time, limit int, window time.Duration) (bool, time.Duration, error) {
	interval := window / time.Duration(limit)
	deadline := now.Add(window)

	query := `
		INSERT INTO rate_limits (key, count, expires_at)
		VALUES ($1, 0, $2)
		ON CONFLICT (key) DO UPDATE
		SET expires_at = GREATEST(rate_limits.expires_at, $3::timestamp) + $4::interval
		WHERE GREATEST(rate_limits.expires_at, $5::timestamp) + $6::interval <= $7::timestamp
		RETURNING expires_at
	`
	var newTAT time.Time
	err := s.db.QueryRow(ctx, s.db.Rebind(query), key, now.Add(interval), now, interval, now, interval, deadline).Scan(&newTAT)
	if err == nil {
		return true, 0, nil
	}
	if !isNoRows(err) {
		return false, 0, err
	}

	// Ditolak. Sisa tunggunya dihitung dari TAT yang tersimpan; bacaan ini tanpa kunci
	// dan tanpa tulisan.
	var tat time.Time
	if err := s.db.QueryRow(ctx, s.db.Rebind("SELECT expires_at FROM rate_limits WHERE key = $1"), key).Scan(&tat); err != nil {
		return false, 0, err
	}
	return false, gcraDecide(tat, now, limit, window).retryAfter, nil
}

// allowInTransaction adalah jalur SQLite: baris dipastikan ada, TAT dibaca,
// keputusan dihitung di Go, dan TAT ditulis hanya bila diizinkan — semuanya dalam
// satu transaksi.
func (s *GCRARateLimitStore) allowInTransaction(ctx context.Context, key string, now time.Time, limit int, window time.Duration) (bool, time.Duration, error) {
	var decision gcraDecision
	err := s.db.WithTx(ctx, func(ctx context.Context, tx Tx) error {
		ensure := s.db.Rebind("INSERT INTO rate_limits (key, count, expires_at) VALUES ($1, 0, $2) ON CONFLICT (key) DO NOTHING")
		if err := tx.Exec(ctx, ensure, key, now); err != nil {
			return err
		}

		var tat time.Time
		if err := tx.QueryRow(ctx, s.db.Rebind("SELECT expires_at FROM rate_limits WHERE key = $1"), key).Scan(&tat); err != nil {
			return err
		}

		decision = gcraDecide(tat, now, limit, window)
		if !decision.allowed {
			return nil
		}
		return tx.Exec(ctx, s.db.Rebind("UPDATE rate_limits SET expires_at = $1 WHERE key = $2"), decision.newTAT, key)
	})
	if err != nil {
		return false, 0, err
	}
	return decision.allowed, decision.retryAfter, nil
}

// Close tidak melakukan apa-apa; koneksi database dikelola di luar store.
func (s *GCRARateLimitStore) Close() error {
	return nil
}

// gcraDecision adalah hasil perhitungan GCRA atas satu request.
type gcraDecision struct {
	allowed    bool
	newTAT     time.Time
	retryAfter time.Duration
}

// gcraDecide memutuskan satu request dari TAT yang tersimpan (nol bila kunci belum
// ada), tanpa menyentuh database. Ini padanan Go dari klausa SQL di allowWithUpsert.
//
// Interval emisi T = window/limit. TAT baru = max(TAT lama, now) + T. Diizinkan
// bila TAT baru - now <= window; selain itu retryAfter = (TAT baru - now) - window,
// yaitu lamanya sampai ketimpangan itu hilang dengan sendirinya.
func gcraDecide(tat, now time.Time, limit int, window time.Duration) gcraDecision {
	if limit <= 0 || window <= 0 {
		return gcraDecision{allowed: false, retryAfter: window}
	}

	interval := window / time.Duration(limit)
	base := now
	if tat.After(now) {
		base = tat
	}
	newTAT := base.Add(interval)

	ahead := newTAT.Sub(now)
	if ahead <= window {
		return gcraDecision{allowed: true, newTAT: newTAT}
	}
	return gcraDecision{allowed: false, retryAfter: ahead - window}
}

// isNoRows mengenali "tidak ada baris" dari kedua driver: pgx untuk PostgreSQL,
// database/sql untuk SQLite.
func isNoRows(err error) bool {
	return errors.Is(err, sql.ErrNoRows) || errors.Is(err, pgx.ErrNoRows)
}
