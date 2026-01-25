package dim

import (
	"context"
	"fmt"
	"maps"
	"sync/atomic"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// PostgresTx wraps pgx.Tx to implement Tx interface
type PostgresTx struct {
	tx pgx.Tx
}

func (p *PostgresTx) Exec(ctx context.Context, query string, args ...interface{}) error {
	_, err := p.tx.Exec(ctx, query, args...)
	return err
}

func (p *PostgresTx) Query(ctx context.Context, query string, args ...interface{}) (Rows, error) {
	return p.tx.Query(ctx, query, args...)
}

func (p *PostgresTx) QueryRow(ctx context.Context, query string, args ...interface{}) Row {
	return p.tx.QueryRow(ctx, query, args...)
}

func (p *PostgresTx) Commit(ctx context.Context) error {
	return p.tx.Commit(ctx)
}

func (p *PostgresTx) Rollback(ctx context.Context) error {
	return p.tx.Rollback(ctx)
}

// PgxTx returns the underlying pgx.Tx for advanced usage
func (p *PostgresTx) PgxTx() pgx.Tx {
	return p.tx
}

// PostgresDatabase is the PostgreSQL implementation of Database interface
// It supports read/write connection splitting with load balancing on read connections
type PostgresDatabase struct {
	writePool   *pgxpool.Pool
	readPools   []*pgxpool.Pool
	readIndex   atomic.Uint32
	hookManager *hookManager
}

// NewPostgresDatabase membuat koneksi database PostgreSQL baru dengan mendukung read/write splitting.
// Membuat write pool untuk operasi INSERT/UPDATE/DELETE dan read pools untuk operasi SELECT.
// Jika tidak ada read hosts yang dikonfigurasi, menggunakan write pool untuk reads.
//
// Parameters:
//   - config: DatabaseConfig berisi konfigurasi koneksi database
//
// Returns:
//   - *PostgresDatabase: instance database yang siap digunakan
//   - error: error jika gagal membuat connection pool
//
// Example:
//
//	db, err := NewPostgresDatabase(config)
//	if err != nil {
//	  log.Fatal(err)
//	}
func NewPostgresDatabase(config DatabaseConfig) (*PostgresDatabase, error) {
	// Initialize hook manager
	hm := &hookManager{
		hooks: make([]QueryHook, 0),
	}

	// Create write connection pool
	writeConnString := formatConnectionString(config.WriteHost, config.Port, config.Database, config.Username, config.Password, config.SSLMode)
	writePool, err := createConnectionPool(writeConnString, config.MaxConns, config.RuntimeParams, config.QueryExecMode, hm)
	if err != nil {
		return nil, fmt.Errorf("failed to create write connection pool: %w", err)
	}

	// Create read connection pools
	var readPools []*pgxpool.Pool
	if len(config.ReadHosts) > 0 {
		for _, host := range config.ReadHosts {
			readConnString := formatConnectionString(host, config.Port, config.Database, config.Username, config.Password, config.SSLMode)
			readPool, err := createConnectionPool(readConnString, config.MaxConns, config.RuntimeParams, config.QueryExecMode, hm)
			if err != nil {
				// Close previously created pools on error
				writePool.Close()
				for _, pool := range readPools {
					pool.Close()
				}
				return nil, fmt.Errorf("failed to create read connection pool for host %s: %w", host, err)
			}
			readPools = append(readPools, readPool)
		}
	} else {
		// If no read hosts specified, use write pool for reads
		readPools = append(readPools, writePool)
	}

	return &PostgresDatabase{
		writePool:   writePool,
		readPools:   readPools,
		readIndex:   atomic.Uint32{},
		hookManager: hm,
	}, nil
}

// AddHook adds a new query hook to the database.
// Thread-safe.
func (db *PostgresDatabase) AddHook(hook QueryHook) {
	db.hookManager.Add(hook)
}

// Exec mengeksekusi write query (INSERT, UPDATE, DELETE) ke write connection pool.
// Semua operasi write selalu dikirim ke write pool untuk consistency.
// Gunakan sticky mode jika perlu subsequent reads ke write connection yang sama.
//
// Parameters:
//   - ctx: context untuk membatalkan operasi
//   - query: SQL query untuk dieksekusi
//   - args: parameter untuk query
//
// Returns:
//   - error: error jika query execution gagal
//
// Example:
//
//	err := db.Exec(ctx, "INSERT INTO users (email, name) VALUES ($1, $2)", email, name)
func (db *PostgresDatabase) Exec(ctx context.Context, query string, args ...interface{}) error {
	_, err := db.writePool.Exec(ctx, query, args...)
	return err
}

// Query mengeksekusi read query (SELECT) dengan routing based on sticky mode.
// Menggunakan decision tree untuk menentukan pool mana yang digunakan:
// 1. Jika query adalah write operation, route ke write pool
// 2. Jika sticky mode enabled dan ada write dalam request, route ke write pool
// 3. Otherwise: route ke read pool dengan round-robin load balancing
//
// Parameters:
//   - ctx: context untuk membatalkan operasi
//   - query: SQL SELECT query
//   - args: parameter untuk query
//
// Returns:
//   - Rows: result set dari query
//   - error: error jika query execution gagal
//
// Example:
//
//	rows, err := db.Query(ctx, "SELECT id, email FROM users WHERE id = $1", userID)
func (db *PostgresDatabase) Query(ctx context.Context, query string, args ...interface{}) (Rows, error) {
	// Decision tree for routing
	pool := db.routeReadQuery(query)
	rows, err := pool.Query(ctx, query, args...)
	return rows, err
}

// QueryRow mengeksekusi read query yang mengembalikan single row dengan routing based on sticky mode.
// Menggunakan decision tree yang sama dengan Query untuk menentukan pool mana yang digunakan.
//
// Parameters:
//   - ctx: context untuk membatalkan operasi
//   - query: SQL SELECT query
//   - args: parameter untuk query
//
// Returns:
//   - Row: single row result yang bisa di-scan
//
// Example:
//
//	err := db.QueryRow(ctx, "SELECT email FROM users WHERE id = $1", userID).Scan(&email)
func (db *PostgresDatabase) QueryRow(ctx context.Context, query string, args ...interface{}) Row {
	pool := db.routeReadQuery(query)
	return pool.QueryRow(ctx, query, args...)
}

// routeReadQuery determines which pool to use for a read query.
// It uses a "Whitelist" approach (Default to Write) for maximum safety.
// Only queries explicitly identified as SAFE READS are routed to the read pool.
// Everything else (Writes, ambiguous queries, unknown commands) goes to the write pool.
func (db *PostgresDatabase) routeReadQuery(query string) *pgxpool.Pool {
	if IsSafeRead(query) {
		return db.getReadPool()
	}

	return db.writePool
}

// Begin memulai transaction baru di write connection.
// Semua transaction selalu dibuat di write pool untuk consistency.
//
// Parameters:
//   - ctx: context untuk membatalkan operasi
//
// Returns:
//   - Tx: transaction object yang bisa digunakan untuk execute queries
//   - error: error jika gagal membuat transaction
//
// Example:
//
//	tx, err := db.Begin(ctx)
//	if err != nil {
//	  return err
//	}
//	defer tx.Rollback(ctx)
func (db *PostgresDatabase) Begin(ctx context.Context) (Tx, error) {
	tx, err := db.writePool.Begin(ctx)
	if err != nil {
		return nil, err
	}
	return &PostgresTx{tx: tx}, nil
}

// Close menutup semua connection pools (write dan read).
// Harus dipanggil sebelum aplikasi shutdown untuk cleanup yang proper.
//
// Returns:
//   - error: selalu nil, tapi disediakan untuk compatibility dengan interface
//
// Example:
//
//	defer db.Close()
func (db *PostgresDatabase) Close() error {
	db.writePool.Close()
	for _, pool := range db.readPools {
		// Only close if it's different from writePool
		if pool != db.writePool {
			pool.Close()
		}
	}
	return nil
}

// DriverName returns the driver name
func (db *PostgresDatabase) DriverName() string {
	return "postgres"
}

// Rebind returns the query as is for PostgreSQL (using $1, $2)
func (db *PostgresDatabase) Rebind(query string) string {
	return query
}

// getReadPool returns a read pool using round-robin load balancing
func (db *PostgresDatabase) getReadPool() *pgxpool.Pool {
	if len(db.readPools) == 0 {
		return db.writePool
	}

	if len(db.readPools) == 1 {
		return db.readPools[0]
	}

	// Round-robin load balancing
	index := db.readIndex.Add(1) - 1
	return db.readPools[index%uint32(len(db.readPools))]
}

// createConnectionPool creates a connection pool with the specified size
// Applies custom RuntimeParams and QueryExecMode for pgbouncer compatibility and custom configuration
func createConnectionPool(connString string, maxConns int, runtimeParams map[string]string, queryExecMode string, hm *hookManager) (*pgxpool.Pool, error) {
	config, err := pgxpool.ParseConfig(connString)
	if err != nil {
		return nil, fmt.Errorf("failed to parse connection string: %w", err)
	}

	config.MaxConns = int32(maxConns)

	// Apply custom runtime parameters
	if config.ConnConfig.RuntimeParams == nil {
		config.ConnConfig.RuntimeParams = make(map[string]string)
	}
	maps.Copy(config.ConnConfig.RuntimeParams, runtimeParams)

	// Apply query execution mode (simple protocol for pgbouncer compatibility)
	if queryExecMode == "simple" {
		config.ConnConfig.DefaultQueryExecMode = pgx.QueryExecModeSimpleProtocol
	}

	// Apply Query Tracer for Observability
	config.ConnConfig.Tracer = &dbTracer{hm: hm}

	pool, err := pgxpool.NewWithConfig(context.Background(), config)
	if err != nil {
		return nil, fmt.Errorf("failed to create connection pool: %w", err)
	}

	// Test connection
	if err := pool.Ping(context.Background()); err != nil {
		pool.Close()
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	return pool, nil
}

// formatConnectionString formats a PostgreSQL connection string
func formatConnectionString(host string, port int, database, username, password, sslmode string) string {
	return fmt.Sprintf("postgres://%s:%s@%s:%d/%s?sslmode=%s",
		username,
		password,
		host,
		port,
		database,
		sslmode,
	)
}

// WithTx mengeksekusi function dalam transaction dengan auto rollback/commit.
// Jika fn return error, transaction di-rollback. Jika sukses, transaction di-commit.
//
// Parameters:
//   - ctx: context untuk membatalkan operasi
//   - fn: function yang berisi query operations dalam transaction
//
// Returns:
//   - error: error dari fn execution atau commit/rollback
//
// Example:
//
//	err := db.WithTx(ctx, func(ctx context.Context, tx Tx) error {
//	  return tx.Exec(ctx, "INSERT INTO users VALUES ($1)", email)
//	})
func (db *PostgresDatabase) WithTx(ctx context.Context, fn TransactionFunc) error {
	tx, err := db.Begin(ctx)
	if err != nil {
		return err
	}

	defer func() {
		if err := recover(); err != nil {
			tx.Rollback(ctx) //nolint:errcheck
			panic(err)
		}
	}()

	if err := fn(ctx, tx); err != nil {
		tx.Rollback(ctx) //nolint:errcheck
		return err
	}

	return tx.Commit(ctx)
}
