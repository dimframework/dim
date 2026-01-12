# Database di Framework dim

Pelajari cara menggunakan database abstraction layer dengan PostgreSQL dan read/write splitting.

## Daftar Isi

- [Konsep Database Layer](#konsep-database-layer)
- [Setup Koneksi](#setup-koneksi)
- [Read/Write Splitting](#readwrite-splitting)
- [Operasi Query](#operasi-query)
- [Transaksi](#transaksi)
- [Connection Pooling](#connection-pooling)
- [Pola Store](#pola-store)
- [User Store](#user-store)
- [Token Store](#token-store)
- [Migrasi](#migrasi)
- [Penanganan Kesalahan](#penanganan-kesalahan)
- [Praktik Terbaik](#praktik-terbaik)

---

## Konsep Database Layer

### Abstraksi Database

Framework dim menggunakan **database interface** yang memungkinkan swap implementasi:

```go
type Database interface {
    Query(ctx context.Context, query string, args ...interface{}) (Rows, error)
    QueryRow(ctx context.Context, query string, args ...interface{}) Row
    Exec(ctx context.Context, query string, args ...interface{}) error
    Begin(ctx context.Context) (Tx, error)
    Close() error
}
```

### Implementasi PostgreSQL

Framework fokus pada implementasi PostgreSQL dengan `pgx/pgxpool`:

```go
type PostgresDatabase struct {
    writePool *pgxpool.Pool      // Single write host
    readPools []*pgxpool.Pool    // Multiple read hosts
    readIndex atomic.Uint32      // Round-robin counter
}
```

### Read/Write Splitting

Framework secara otomatis membagi traffic:

```
┌──────────────────────────────────────────┐
│         Kode Aplikasi                    │
└────────────────────┬─────────────────────┘
                     │
        ┌────────────┴────────────┐
        │                         │
        ▼                         ▼
   Query/QueryRow            Exec
        │                         │
        ▼                         ▼
   Read Pool(s)             Write Pool
   (Multiple Hosts)         (Single Host)
        │                         │
        ▼                         ▼
   Replicas (RO)            Primary (RW)
```

---

## Setup Koneksi

### Load Konfigurasi

```go
cfg, err := dim.LoadConfig()
if err != nil {
    log.Fatal(err)
}

// cfg.Database berisi:
// - WriteHost: Host untuk write operations
// - ReadHosts: Array host untuk read operations
// - Port: Database port
// - Database: Database name
// - Username: User PostgreSQL
// - Password: Password user
// - SSLMode: SSL mode (disable, require, prefer, allow, verify-ca, verify-full)
// - MaxConns: Max connection pool size
```

### Environment Variables

File `.env`:
```bash
# Write host (Primary/Master)
DB_WRITE_HOST=db-primary.example.com

# Read hosts (Replicas) - comma-separated
DB_READ_HOSTS=db-replica1.example.com,db-replica2.example.com

DB_PORT=5432
DB_NAME=myapp_db
DB_USER=postgres
DB_PASSWORD=secretpassword
DB_SSL_MODE=require
DB_MAX_CONNS=25
```

**Perilaku Fallback**:
- Jika `DB_READ_HOSTS` kosong → Gunakan write host untuk read
- Jika read host connect gagal → Fallback ke error handling

### Buat Koneksi

```go
package main

import (
    "log"
    "github.com/nuradiyana/dim"
)

func main() {
    // Load config
    cfg, err := dim.LoadConfig()
    if err != nil {
        log.Fatal("Load config failed:", err)
    }
    
    // Buat database connection
    db, err := dim.NewPostgresDatabase(cfg.Database)
    if err != nil {
        log.Fatal("Database connection failed:", err)
    }
    defer db.Close()
    
    // Sekarang siap untuk query
    // ...
}
```

### Konfigurasi Connection Pool

```go
type DatabaseConfig struct {
    WriteHost string   // Primary host
    ReadHosts []string // Replica hosts (optional)
    Port      int      // Default 5432
    Database  string   // Database name
    Username  string   // Database user
    Password  string   // Database password
    SSLMode   string   // SSL mode
    MaxConns  int      // Max connections per pool
}
```

**Praktik Terbaik untuk MaxConns**:
```
MaxConns = (CPU Cores × 4) + 1

Contoh:
- 4 cores  → MaxConns = 17
- 8 cores  → MaxConns = 33
- 16 cores → MaxConns = 65
```

---

## Read/Write Splitting

### Automatic Routing

Framework secara otomatis route requests:

```go
// ✅ OTOMATIS ke Read Pool
rows, err := db.Query(ctx, "SELECT * FROM users WHERE id = $1", userID)

// ✅ OTOMATIS ke Read Pool
row := db.QueryRow(ctx, "SELECT email FROM users WHERE id = $1", userID)

// ✅ OTOMATIS ke Write Pool
err := db.Exec(ctx, "INSERT INTO users (email, name) VALUES ($1, $2)", email, name)

// ✅ OTOMATIS ke Write Pool (transaction)
tx, _ := db.Begin(ctx)
tx.Exec(ctx, "UPDATE users SET name = $1 WHERE id = $2", name, userID)
```

### Load Balancing untuk Read

Multiple read replicas di-balance dengan **round-robin**:

```
Request 1 → Read Pool 0 → Replica 1
Request 2 → Read Pool 1 → Replica 2
Request 3 → Read Pool 0 → Replica 1
Request 4 → Read Pool 1 → Replica 2
```

Implementation:
```go
func (db *PostgresDatabase) getReadPool() *pgxpool.Pool {
    if len(db.readPools) == 0 {
        return db.writePool  // Fallback ke write pool
    }
    
    // Round-robin load balancing
    idx := db.readIndex.Add(1) - 1
    return db.readPools[idx % uint32(len(db.readPools))]
}
```

### Konsistensi Read Replica

⚠️ **Penting**: Read replicas mungkin lag dari primary!

```go
// Setup dengan single write host dan multiple replicas
DB_WRITE_HOST=db-primary.com
DB_READ_HOSTS=db-replica1.com,db-replica2.com

// Write terjadi di primary
err := db.Exec(ctx, "INSERT INTO users (email) VALUES ($1)", "john@example.com")

// Read segera setelah write mungkin belum konsisten
user, _ := db.QueryRow(ctx, "SELECT * FROM users WHERE email = $1", "john@example.com")

// Jika perlu strong consistency:
// 1. Baca dari write host (primary)
// 2. Atau gunakan read after write pattern di app level
```

---

## Operasi Query

### Query - Multiple Rows

Mengambil multiple rows:

```go
func (store *UserStore) GetAll(ctx context.Context) ([]User, error) {
    rows, err := store.db.Query(ctx, 
        "SELECT id, email, username, created_at FROM users ORDER BY id")
    if err != nil {
        return nil, err
    }
    defer rows.Close()
    
    var users []User
    for rows.Next() {
        var user User
        err := rows.Scan(&user.ID, &user.Email, &user.Username, &user.CreatedAt)
        if err != nil {
            return nil, err
        }
        users = append(users, &user)
    }
    
    if err = rows.Err(); err != nil {
        return nil, err
    }
    
    return users, nil
}
```

### QueryRow - Single Row

Mengambil single row:

```go
func (store *UserStore) FindByID(ctx context.Context, id int64) (User, error) {
    var user User
    err := store.db.QueryRow(ctx,
        "SELECT id, email, username, created_at FROM users WHERE id = $1",
        id,
    ).Scan(&user.ID, &user.Email, &user.Username, &user.CreatedAt)
    
    if err != nil {
        return nil, err
    }
    
    return user, nil
}
```

### Exec - Write Operations

Melakukan insert/update/delete:

```go
func (store *UserStore) Create(ctx context.Context, user *User) error {
    err := store.db.Exec(ctx,
        "INSERT INTO users (email, username, password_hash) VALUES ($1, $2, $3)",
        user.Email,
        user.Username,
        user.PasswordHash,
    )
    
    if err != nil {
        // Handle duplicate key error
        if isDuplicateKeyError(err) {
            return ErrEmailAlreadyExists
        }
        return err
    }
    
    return nil
}

func (store *UserStore) Update(ctx context.Context, user *User) error {
    err := store.db.Exec(ctx,
        "UPDATE users SET email = $1, username = $2 WHERE id = $3",
        user.Email,
        user.Username,
        user.ID,
    )
    
    return err
}

func (store *UserStore) Delete(ctx context.Context, id int64) error {
    err := store.db.Exec(ctx,
        "DELETE FROM users WHERE id = $1",
        id,
    )
    
    return err
}
```

### Penanganan Kesalahan

```go
rows, err := db.Query(ctx, "SELECT * FROM users")
if err != nil {
    // Parse error untuk detail info
    if err == sql.ErrNoRows {
        // No rows found
        return nil
    }
    
    // Other errors
    logger.Error("Query failed", "error", err)
    return nil
}
```

---

## Transaksi

### Basic Transaksi

```go
// Mulai transaction
tx, err := db.Begin(ctx)
if err != nil {
    return err
}

// Jika ada error, rollback otomatis
defer func() {
    if err != nil {
        tx.Rollback(ctx)
    }
}()

// Execute queries
err = tx.Exec(ctx, "INSERT INTO users (email) VALUES ($1)", "john@example.com")
if err != nil {
    return err
}

err = tx.Exec(ctx, "INSERT INTO user_profiles (user_id, bio) VALUES ($1, $2)", userID, bio)
if err != nil {
    return err
}

// Commit jika semua berhasil
return tx.Commit(ctx)
```

### Transaksi Helper

```go
func (store *UserStore) CreateUserWithProfile(ctx context.Context, user *User, profile *UserProfile) error {
    tx, err := store.db.Begin(ctx)
    if err != nil {
        return err
    }
    
    // Rollback jika ada error
    defer func() {
        if err != nil {
            tx.Rollback(ctx)
        }
    }()
    
    // Create user
    err = tx.Exec(ctx,
        "INSERT INTO users (email, username) VALUES ($1, $2)",
        user.Email,
        user.Username,
    )
    if err != nil {
        return err
    }
    
    // Create profile
    err = tx.Exec(ctx,
        "INSERT INTO user_profiles (user_id, bio) VALUES ($1, $2)",
        user.ID,
        profile.Bio,
    )
    if err != nil {
        return err
    }
    
    // Commit
    return tx.Commit(ctx)
}
```

### Transaksi Helper (WithTx)

Untuk menyederhanakan manajemen transaksi, framework menyediakan fungsi pembantu `WithTx`. Fungsi ini secara otomatis akan melakukan `COMMIT` jika fungsi yang Anda berikan berhasil, atau `ROLLBACK` jika terjadi *error* atau *panic*.

```go
func (store *UserStore) CreateUserWithProfile(ctx context.Context, user *User, profile *Profile) error {
    // Gunakan WithTx untuk manajemen transaksi otomatis
    return store.db.WithTx(ctx, func(ctx context.Context, tx pgx.Tx) error {
        // Semua query di dalam fungsi ini berjalan dalam satu transaksi
        
        // Buat user
        err := tx.Exec(ctx,
            "INSERT INTO users (email, username) VALUES ($1, $2)",
            user.Email,
            user.Username,
        )
        if err != nil {
            return err // Rollback akan dipanggil secara otomatis
        }
        
        // Buat profile
        err = tx.Exec(ctx,
            "INSERT INTO user_profiles (user_id, bio) VALUES ($1, $2)",
            user.ID,
            profile.Bio,
        )
        if err != nil {
            return err // Rollback akan dipanggil secara otomatis
        }
        
        // Jika tidak ada error, Commit akan dipanggil secara otomatis
        return nil
    })
}
```

### Savepoints (Advanced)

```go
tx, _ := db.Begin(ctx)
defer tx.Rollback(ctx)

// Query 1
tx.Exec(ctx, "INSERT INTO table1 ...")

// Savepoint
tx.Exec(ctx, "SAVEPOINT sp1")

// Query 2
err := tx.Exec(ctx, "INSERT INTO table2 ...")

if err != nil {
    // Rollback hanya ke savepoint, tidak seluruh transaction
    tx.Exec(ctx, "ROLLBACK TO SAVEPOINT sp1")
}

// Lanjut
tx.Exec(ctx, "INSERT INTO table3 ...")

tx.Commit(ctx)
```

---

## Connection Pooling

### Konfigurasi Pool

```go
// Framework automatic pooling dengan pgx/pgxpool

// Read pool
cfg := pgxpool.Config{
    MaxConns: 25,
    MinConns: 5,
}
readPool, _ := pgxpool.NewWithConfig(ctx, connStr)

// Write pool (same config)
cfg := pgxpool.Config{
    MaxConns: 25,
    MinConns: 5,
}
writePool, _ := pgxpool.NewWithConfig(ctx, connStr)
```

### Monitor Pool Health

```go
func monitorPoolHealth(db *PostgresDatabase, logger *slog.Logger) {
    ticker := time.NewTicker(30 * time.Second)
    defer ticker.Stop()
    
    for range ticker.C {
        stat := db.Stat()
        logger.Info("Pool stats",
            "total_conns", stat.TotalConns(),
            "idle_conns", stat.IdleConns(),
            "open_conns", stat.OpenConns(),
        )
    }
}
```

### Validasi Koneksi

```go
// Framework validates connection sebelum digunakan
err := db.QueryRow(ctx, "SELECT 1").Scan()
if err != nil {
    // Connection invalid, reconnect
    logger.Error("Connection validation failed", "error", err)
}
```

---

## Pola Store

### Store Struct

Store adalah repository pattern untuk data access tanpa menggunakan interface:

```go
type UserStore struct {
    db Database
}

func NewUserStore(db Database) *UserStore {
    return &UserStore{db: db}
}

func (store *UserStore) FindByID(ctx context.Context, id int64) (*User, error) {
    var user User
    err := store.db.QueryRow(ctx,
        "SELECT id, email, username, created_at FROM users WHERE id = $1",
        id,
    ).Scan(&user.ID, &user.Email, &user.Username, &user.CreatedAt)
    
    if err != nil {
        return nil, err
    }
    
    return &user, nil
}
```

### Store Methods

```go
type UserStore struct {
    db Database
}

// Create user baru
func (store *UserStore) Create(ctx context.Context, user *User) error {
    return store.db.Exec(ctx,
        "INSERT INTO users (email, username, password_hash) VALUES ($1, $2, $3)",
        user.Email,
        user.Username,
        user.PasswordHash,
    )
}

// Find by ID
func (store *UserStore) FindByID(ctx context.Context, id int64) (*User, error) {
    var user User
    err := store.db.QueryRow(ctx,
        "SELECT id, email, username, created_at FROM users WHERE id = $1",
        id,
    ).Scan(&user.ID, &user.Email, &user.Username, &user.CreatedAt)
    return &user, err
}

// Find by email
func (store *UserStore) FindByEmail(ctx context.Context, email string) (*User, error) {
    var user User
    err := store.db.QueryRow(ctx,
        "SELECT id, email, username, password_hash FROM users WHERE email = $1",
        email,
    ).Scan(&user.ID, &user.Email, &user.Username, &user.PasswordHash)
    return &user, err
}

// Update user
func (store *UserStore) Update(ctx context.Context, user *User) error {
    return store.db.Exec(ctx,
        "UPDATE users SET email = $1, username = $2 WHERE id = $3",
        user.Email,
        user.Username,
        user.ID,
    )
}

// Delete user
func (store *UserStore) Delete(ctx context.Context, id int64) error {
    return store.db.Exec(ctx,
        "DELETE FROM users WHERE id = $1",
        id,
    )
}
```

### Gunakan Store dalam Service

```go
type AuthService struct {
    userStore  *UserStore
    tokenStore *TokenStore
}

func NewAuthService(userStore *UserStore, tokenStore *TokenStore) *AuthService {
    return &AuthService{
        userStore:  userStore,
        tokenStore: tokenStore,
    }
}

func (s *AuthService) Login(ctx context.Context, email, password string) (string, error) {
    // Query user dari store
    user, err := s.userStore.FindByEmail(ctx, email)
    if err != nil {
        return "", ErrInvalidCredentials
    }
    
    // Verify password
    err = bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(password))
    if err != nil {
        return "", ErrInvalidCredentials
    }
    
    // Generate token
    token, _ := s.generateToken(user.ID)
    return token, nil
}
```

---

## User Store

### User Struct

```go
type User struct {
    ID           int64
    Email        string
    Username     string
    PasswordHash string
    CreatedAt    time.Time
    UpdatedAt    time.Time
}
```

### User Table

```sql
CREATE TABLE IF NOT EXISTS users (
    id BIGSERIAL PRIMARY KEY,
    email VARCHAR(255) UNIQUE NOT NULL,
    username VARCHAR(100) UNIQUE NOT NULL,
    password_hash VARCHAR(255) NOT NULL,
    created_at TIMESTAMP DEFAULT NOW(),
    updated_at TIMESTAMP DEFAULT NOW()
);

CREATE INDEX idx_users_email ON users(email);
CREATE INDEX idx_users_username ON users(username);
```

### User Store Struct

```go
type UserStore struct {
    db Database
}

func NewUserStore(db Database) *UserStore {
    return &UserStore{db: db}
}
```

### User Store Methods

```go
// Create user baru
func (store *UserStore) Create(ctx context.Context, user *User) error {
    return store.db.Exec(ctx,
        "INSERT INTO users (email, username, password_hash) VALUES ($1, $2, $3)",
        user.Email,
        user.Username,
        user.PasswordHash,
    )
}

// Find by email
func (store *UserStore) FindByEmail(ctx context.Context, email string) (*User, error) {
    var user User
    err := store.db.QueryRow(ctx,
        "SELECT id, email, username, password_hash FROM users WHERE email = $1",
        email,
    ).Scan(&user.ID, &user.Email, &user.Username, &user.PasswordHash)
    return &user, err
}

// Find by ID
func (store *UserStore) FindByID(ctx context.Context, id int64) (*User, error) {
    var user User
    err := store.db.QueryRow(ctx,
        "SELECT id, email, username, password_hash FROM users WHERE id = $1",
        id,
    ).Scan(&user.ID, &user.Email, &user.Username, &user.PasswordHash)
    return &user, err
}

// Find by username
func (store *UserStore) FindByUsername(ctx context.Context, username string) (*User, error) {
    var user User
    err := store.db.QueryRow(ctx,
        "SELECT id, email, username FROM users WHERE username = $1",
        username,
    ).Scan(&user.ID, &user.Email, &user.Username)
    return &user, err
}

// Update user
func (store *UserStore) Update(ctx context.Context, user *User) error {
    user.UpdatedAt = time.Now()
    return store.db.Exec(ctx,
        "UPDATE users SET email = $1, username = $2, updated_at = $3 WHERE id = $4",
        user.Email,
        user.Username,
        user.UpdatedAt,
        user.ID,
    )
}

// Delete user
func (store *UserStore) Delete(ctx context.Context, id int64) error {
    return store.db.Exec(ctx,
        "DELETE FROM users WHERE id = $1",
        id,
    )
}
```

---

## Token Store

### Token Structs

```go
type RefreshToken struct {
    ID        int64
    UserID    int64
    TokenHash string
    ExpiresAt time.Time
    CreatedAt time.Time
    RevokedAt *time.Time
}

type PasswordResetToken struct {
    ID        int64
    UserID    int64
    TokenHash string
    ExpiresAt time.Time
    UsedAt    *time.Time
    CreatedAt time.Time
}
```

### Token Tables

```sql
CREATE TABLE IF NOT EXISTS refresh_tokens (
    id BIGSERIAL PRIMARY KEY,
    user_id BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    token_hash VARCHAR(255) UNIQUE NOT NULL,
    expires_at TIMESTAMP NOT NULL,
    created_at TIMESTAMP DEFAULT NOW(),
    revoked_at TIMESTAMP
);

CREATE INDEX idx_refresh_tokens_user_id ON refresh_tokens(user_id);
CREATE INDEX idx_refresh_tokens_token_hash ON refresh_tokens(token_hash);

CREATE TABLE IF NOT EXISTS password_reset_tokens (
    id BIGSERIAL PRIMARY KEY,
    user_id BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    token_hash VARCHAR(255) UNIQUE NOT NULL,
    expires_at TIMESTAMP NOT NULL,
    used_at TIMESTAMP,
    created_at TIMESTAMP DEFAULT NOW()
);

CREATE INDEX idx_password_reset_tokens_user_id ON password_reset_tokens(user_id);
CREATE INDEX idx_password_reset_tokens_token_hash ON password_reset_tokens(token_hash);
```

### Token Store Struct

```go
type TokenStore struct {
    db Database
}

func NewTokenStore(db Database) *TokenStore {
    return &TokenStore{db: db}
}
```

### Token Store Methods

```go
// Save refresh token
func (store *TokenStore) SaveRefreshToken(ctx context.Context, token *RefreshToken) error {
    return store.db.Exec(ctx,
        "INSERT INTO refresh_tokens (user_id, token_hash, expires_at) VALUES ($1, $2, $3)",
        token.UserID,
        token.TokenHash,
        token.ExpiresAt,
    )
}

// Find refresh token
func (store *TokenStore) FindRefreshToken(ctx context.Context, tokenHash string) (*RefreshToken, error) {
    var token RefreshToken
    err := store.db.QueryRow(ctx,
        "SELECT id, user_id, token_hash, expires_at, created_at, revoked_at FROM refresh_tokens WHERE token_hash = $1",
        tokenHash,
    ).Scan(&token.ID, &token.UserID, &token.TokenHash, &token.ExpiresAt, &token.CreatedAt, &token.RevokedAt)
    
    return &token, err
}

// Revoke refresh token
func (store *TokenStore) RevokeRefreshToken(ctx context.Context, tokenHash string) error {
    return store.db.Exec(ctx,
        "UPDATE refresh_tokens SET revoked_at = NOW() WHERE token_hash = $1",
        tokenHash,
    )
}

// Revoke all user tokens
func (store *TokenStore) RevokeAllUserTokens(ctx context.Context, userID int64) error {
    return store.db.Exec(ctx,
        "UPDATE refresh_tokens SET revoked_at = NOW() WHERE user_id = $1",
        userID,
    )
}

// Save password reset token
func (store *TokenStore) SavePasswordResetToken(ctx context.Context, token *PasswordResetToken) error {
    return store.db.Exec(ctx,
        "INSERT INTO password_reset_tokens (user_id, token_hash, expires_at) VALUES ($1, $2, $3)",
        token.UserID,
        token.TokenHash,
        token.ExpiresAt,
    )
}

// Find password reset token
func (store *TokenStore) FindPasswordResetToken(ctx context.Context, tokenHash string) (*PasswordResetToken, error) {
    var token PasswordResetToken
    err := store.db.QueryRow(ctx,
        "SELECT id, user_id, token_hash, expires_at, used_at, created_at FROM password_reset_tokens WHERE token_hash = $1",
        tokenHash,
    ).Scan(&token.ID, &token.UserID, &token.TokenHash, &token.ExpiresAt, &token.UsedAt, &token.CreatedAt)
    
    return &token, err
}

// Mark password reset token as used
func (store *TokenStore) MarkPasswordResetUsed(ctx context.Context, tokenHash string) error {
    return store.db.Exec(ctx,
        "UPDATE password_reset_tokens SET used_at = NOW() WHERE token_hash = $1",
        tokenHash,
    )
}
```

---

## Migrasi

### Migration System

Framework menggunakan Go functions untuk migrations:

```go
type Migration struct {
    Version int
    Name    string
    Up      func(Database) error
    Down    func(Database) error
}
```

### Define Migrations

```go
func GetMigrations() []Migration {
    return []Migration{
        {
            Version: 1,
            Name:    "create_users_table",
            Up: func(db Database) error {
                return db.Exec(context.Background(), `
                    CREATE TABLE IF NOT EXISTS users (
                        id BIGSERIAL PRIMARY KEY,
                        email VARCHAR(255) UNIQUE NOT NULL,
                        username VARCHAR(100) UNIQUE NOT NULL,
                        password_hash VARCHAR(255) NOT NULL,
                        created_at TIMESTAMP DEFAULT NOW(),
                        updated_at TIMESTAMP DEFAULT NOW()
                    )
                `)
            },
            Down: func(db Database) error {
                return db.Exec(context.Background(), "DROP TABLE IF EXISTS users")
            },
        },
        {
            Version: 2,
            Name:    "create_refresh_tokens_table",
            Up: func(db Database) error {
                return db.Exec(context.Background(), `
                    CREATE TABLE IF NOT EXISTS refresh_tokens (
                        id BIGSERIAL PRIMARY KEY,
                        user_id BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
                        token_hash VARCHAR(255) UNIQUE NOT NULL,
                        expires_at TIMESTAMP NOT NULL,
                        created_at TIMESTAMP DEFAULT NOW(),
                        revoked_at TIMESTAMP
                    )
                `)
            },
            Down: func(db Database) error {
                return db.Exec(context.Background(), "DROP TABLE IF EXISTS refresh_tokens")
            },
        },
    }
}
```

### Run Migrations

```go
func main() {
    db, _ := dim.NewPostgresDatabase(cfg.Database)
    defer db.Close()
    
    // Run migrations
    migrations := GetMigrations()
    err := dim.RunMigrations(db, migrations)
    if err != nil {
        log.Fatal("Migration failed:", err)
    }
    
    // Now safe to use database
    logger.Info("Migrations completed")
}
```

---

## Penanganan Kesalahan

### Common Database Errors

```go
// No rows found
if err == sql.ErrNoRows {
    return nil, ErrNotFound
}

// Duplicate key (email/username already exists)
if isDuplicateKeyError(err) {
    return nil, ErrDuplicate
}

// Foreign key violation
if isForeignKeyError(err) {
    return nil, ErrInvalidReference
}

// Connection error
if isConnectionError(err) {
    logger.Error("Database connection error", "error", err)
    return nil, ErrDatabaseUnavailable
}
```

### Helper Functions

```go
func isDuplicateKeyError(err error) bool {
    // PostgreSQL duplicate key error code: 23505
    var pgErr *pgconn.PgError
    if errors.As(err, &pgErr) {
        return pgErr.Code == "23505"
    }
    return false
}

func isForeignKeyError(err error) bool {
    // PostgreSQL foreign key error code: 23503
    var pgErr *pgconn.PgError
    if errors.As(err, &pgErr) {
        return pgErr.Code == "23503"
    }
    return false
}
```

---

## Praktik Terbaik

### ✅ DO: Use Store Pattern

```go
// ✅ BAIK - Abstraction layer dengan struct
type UserStore struct {
    db Database
}

func NewUserStore(db Database) *UserStore {
    return &UserStore{db: db}
}

func (store *UserStore) FindByID(ctx context.Context, id int64) (*User, error) {
    // Implementation
}

// Service menggunakan store
type UserService struct {
    store *UserStore
}

func NewUserService(store *UserStore) *UserService {
    return &UserService{store: store}
}
```

### ❌ DON'T: Direct Database Calls

```go
// ❌ BURUK - Tight coupling ke database
type UserService struct {
    db Database
}

func (s *UserService) GetUser(id int64) *User {
    return s.db.Query("SELECT * FROM users ...")
}
```

### ✅ DO: Use Context

```go
// ✅ BAIK - Context timeout, cancellation
func (store *UserStore) Find(ctx context.Context, id int64) (*User, error) {
    return store.db.QueryRow(ctx, "SELECT * FROM users WHERE id = $1", id)
}
```

### ✅ DO: Use Transactions untuk Multi-Table Operations

```go
// ✅ BAIK - Atomic operations
func (store *UserStore) CreateUserWithProfile(ctx context.Context, user *User, profile *Profile) error {
    tx, _ := store.db.Begin(ctx)
    defer tx.Rollback(ctx)
    
    // Multiple operations
    tx.Exec(ctx, "INSERT INTO users ...")
    tx.Exec(ctx, "INSERT INTO profiles ...")
    
    return tx.Commit(ctx)
}
```

### ✅ DO: Use Prepared Statements (via parameterized queries)

```go
// ✅ BAIK - Prevent SQL injection
func (store *UserStore) FindByEmail(ctx context.Context, email string) (*User, error) {
    return store.db.QueryRow(ctx,
        "SELECT * FROM users WHERE email = $1",  // Parameterized
        email,  // Separate parameter
    )
}

// ❌ BURUK - SQL Injection risk
query := fmt.Sprintf("SELECT * FROM users WHERE email = '%s'", email)
```

### ❌ DON'T: Ignore Context Deadline

```go
// ❌ BURUK - Ignores context timeout
db.Query(context.Background(), "SELECT * FROM large_table")

// ✅ BAIK - Respects context timeout
db.Query(ctx, "SELECT * FROM large_table")  // Will timeout if ctx canceled
```

---

## Ringkasan

Database layer di dim:
- **Abstrak** - Interface-based untuk fleksibilitas
- **Scalable** - Read/write splitting dengan connection pooling
- **Safe** - Parameterized queries, transactions, dan penanganan error
- **Simple** - Pola *store* dan *routing* query otomatis tanpa kompleksitas ORM

---

**Lihat Juga**:
- [Konfigurasi](07-configuration.md) - Database configuration
- [Pola Store](06-database.md#pola-store) - Data access layer
- [Transaksi](06-database.md#transaksi) - Multi-operation atomicity
- [Autentikasi](05-authentication.md) - User authentication storage
- [Migrasi](13-migrations.md) - Database versioning
