package dim

import (
	"context"
	"database/sql"
	"fmt"
	"regexp"
	"sync"

	_ "github.com/mattn/go-sqlite3"
)

// SQLiteDatabase is the SQLite implementation of Database interface
type SQLiteDatabase struct {
	db *sql.DB
	mu sync.RWMutex
}

// NewSQLiteDatabase creates a new SQLite database connection
//
// Parameters:
//   - config: DatabaseConfig containing connection configuration (Database field is used as file path)
//
// Returns:
//   - *SQLiteDatabase: database instance ready for use
//   - error: error if connection fails
func NewSQLiteDatabase(config DatabaseConfig) (*SQLiteDatabase, error) {
	db, err := sql.Open("sqlite3", config.Database)
	if err != nil {
		return nil, fmt.Errorf("failed to open sqlite database: %w", err)
	}

	// Set connection pool settings
	// SQLite only supports one writer at a time, so we should limit open connections if needed,
	// but generally for WAL mode multiple readers are fine.
	// For simplicity and safety, we start with what Recap used.
	if config.MaxConns > 0 {
		db.SetMaxOpenConns(config.MaxConns)
	} else {
		db.SetMaxOpenConns(1) // Default to single connection for safety
	}

	// Test connection
	if err := db.Ping(); err != nil {
		db.Close()
		return nil, fmt.Errorf("failed to ping sqlite database: %w", err)
	}

	sqliteDB := &SQLiteDatabase{
		db: db,
	}

	// Apply pragma settings if any (we can use RuntimeParams for this)
	if config.RuntimeParams != nil {
		for key, value := range config.RuntimeParams {
			query := fmt.Sprintf("PRAGMA %s = %s", key, value)
			if _, err := db.Exec(query); err != nil {
				db.Close()
				return nil, fmt.Errorf("failed to set pragma %s: %w", key, err)
			}
		}
	}

	return sqliteDB, nil
}

// Exec executes a write query (INSERT, UPDATE, DELETE)
func (db *SQLiteDatabase) Exec(ctx context.Context, query string, args ...interface{}) error {
	_, err := db.db.ExecContext(ctx, query, args...)
	return err
}

// Query executes a read query (SELECT) and returns multiple rows
func (db *SQLiteDatabase) Query(ctx context.Context, query string, args ...interface{}) (Rows, error) {
	rows, err := db.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	return &sqliteRows{rows: rows}, nil
}

// QueryRow executes a read query that returns a single row
func (db *SQLiteDatabase) QueryRow(ctx context.Context, query string, args ...interface{}) Row {
	row := db.db.QueryRowContext(ctx, query, args...)
	return &sqliteRow{row: row}
}

// Begin starts a new transaction
func (db *SQLiteDatabase) Begin(ctx context.Context) (Tx, error) {
	tx, err := db.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to begin transaction: %w", err)
	}
	return &SQLiteTx{tx: tx}, nil
}

// Close closes the database connection
func (db *SQLiteDatabase) Close() error {
	return db.db.Close()
}

// DriverName returns the driver name
func (db *SQLiteDatabase) DriverName() string {
	return "sqlite"
}

// Rebind replaces $n placeholders with ? for SQLite
func (db *SQLiteDatabase) Rebind(query string) string {
	re := regexp.MustCompile(`\$[0-9]+`)
	return re.ReplaceAllString(query, "?")
}

// WithTx executes a function within a transaction with auto rollback/commit
func (db *SQLiteDatabase) WithTx(ctx context.Context, fn TransactionFunc) error {
	tx, err := db.Begin(ctx)
	if err != nil {
		return err
	}

	defer func() {
		if p := recover(); p != nil {
			tx.Rollback(ctx)
			panic(p)
		}
	}()

	if err := fn(ctx, tx); err != nil {
		tx.Rollback(ctx)
		return err
	}

	return tx.Commit(ctx)
}

// SQLiteTx implements Tx interface for SQLite
type SQLiteTx struct {
	tx *sql.Tx
}

func (t *SQLiteTx) Exec(ctx context.Context, query string, args ...interface{}) error {
	_, err := t.tx.ExecContext(ctx, query, args...)
	return err
}

func (t *SQLiteTx) Query(ctx context.Context, query string, args ...interface{}) (Rows, error) {
	rows, err := t.tx.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	return &sqliteRows{rows: rows}, nil
}

func (t *SQLiteTx) QueryRow(ctx context.Context, query string, args ...interface{}) Row {
	row := t.tx.QueryRowContext(ctx, query, args...)
	return &sqliteRow{row: row}
}

func (t *SQLiteTx) Commit(ctx context.Context) error {
	return t.tx.Commit()
}

func (t *SQLiteTx) Rollback(ctx context.Context) error {
	return t.tx.Rollback()
}

// sqliteRows implements Rows interface
type sqliteRows struct {
	rows *sql.Rows
}

func (r *sqliteRows) Close() {
	r.rows.Close()
}

func (r *sqliteRows) Next() bool {
	return r.rows.Next()
}

func (r *sqliteRows) Scan(dest ...interface{}) error {
	return r.rows.Scan(dest...)
}

func (r *sqliteRows) Err() error {
	return r.rows.Err()
}

// sqliteRow implements Row interface
type sqliteRow struct {
	row *sql.Row
}

func (r *sqliteRow) Scan(dest ...interface{}) error {
	return r.row.Scan(dest...)
}