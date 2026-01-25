package dim

import (
	"context"
)

// Rows represents query result rows
type Rows interface {
	Close()
	Next() bool
	Scan(dest ...interface{}) error
	Err() error
}

// Row represents a single query result row
type Row interface {
	Scan(dest ...interface{}) error
}

// Database is the interface for database operations
type Database interface {
	Exec(ctx context.Context, query string, args ...interface{}) error
	Query(ctx context.Context, query string, args ...interface{}) (Rows, error)
	QueryRow(ctx context.Context, query string, args ...interface{}) Row
	Begin(ctx context.Context) (Tx, error)
	WithTx(ctx context.Context, fn TransactionFunc) error
	Close() error
	DriverName() string
	Rebind(query string) string
}

// Tx represents a database transaction
type Tx interface {
	Exec(ctx context.Context, query string, args ...interface{}) error
	Query(ctx context.Context, query string, args ...interface{}) (Rows, error)
	QueryRow(ctx context.Context, query string, args ...interface{}) Row
	Commit(ctx context.Context) error
	Rollback(ctx context.Context) error
}

// TransactionFunc is a function that performs operations within a transaction
type TransactionFunc func(ctx context.Context, tx Tx) error