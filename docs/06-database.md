# Database di Framework dim

Pelajari cara menggunakan database abstraction layer dengan PostgreSQL, read/write splitting, dan fitur observability otomatis.

## Daftar Isi

- [Konsep Database Layer](#konsep-database-layer)
- [Setup Koneksi](#setup-koneksi)
- [Observability & Security](#observability--security)
- [Read/Write Splitting](#readwrite-splitting)
- [Operasi Query](#operasi-query)
- [Transaksi](#transaksi)
- [Praktik Terbaik](#praktik-terbaik)

---

## Konsep Database Layer

Framework dim menggunakan wrapper di atas driver `pgx/v5` yang menyediakan fitur:

1.  **Read/Write Splitting**: Memisahkan query ke *primary* dan *replica*.
2.  **Observability**: Tracer otomatis untuk logging query.
3.  **Security**: Masking otomatis data sensitif di log.
4.  **Connection Pooling**: Manajemen koneksi yang efisien.

---

## Setup Koneksi

### Konfigurasi

Gunakan struct `DatabaseConfig` untuk mengatur koneksi:

```go
config := dim.DatabaseConfig{
    WriteHost:     "db-primary",
    ReadHosts:     []string{"db-replica-1", "db-replica-2"},
    Port:          5432,
    Database:      "myapp",
    Username:      "user",
    Password:      "secret",
    SSLMode:       "require",
    MaxConns:      25,
    
    // Opsional: Parameter runtime PostgreSQL
    RuntimeParams: map[string]string{
        "search_path": "public,app",
    },
}
```

### Inisialisasi

```go
db, err := dim.NewPostgresDatabase(config)
if err != nil {
    log.Fatal(err)
}
defer db.Close()

// Opsional: Jalankan migrasi inti framework
// Ini akan membuat tabel users, tokens, dan rate_limits jika belum ada.
if err := dim.RunMigrations(db, dim.GetFrameworkMigrations()); err != nil {
    log.Fatal("Gagal migrasi:", err)
}
```

---

## Observability & Security

Salah satu fitur unggulan dim adalah sistem *tracing* bawaan.

### Automatic Query Logging

Setiap query yang dijalankan melalui wrapper `dim` akan otomatis dicatat ke log (jika level logger diatur ke Debug/Info). Informasi yang dicatat meliputi:
- SQL Query
- Durasi eksekusi
- Error (jika ada)

### Sensitive Data Masking

Demi keamanan, framework secara otomatis mendeteksi dan menyembunyikan data sensitif di dalam argumen query sebelum ditulis ke log.

**Keyword Sensitif**:
- `password`, `email`, `token`, `secret`, `api_key`

**Contoh Log**:

Query Asli:
```go
db.Exec(ctx, "INSERT INTO users (email, password) VALUES ($1, $2)", "user@example.com", "rahasia123")
```

Log Output (Otomatis):
```text
level=INFO msg="query executed" sql="INSERT INTO users (email, password) VALUES ($1, $2)" args=["*****", "*****"] duration=2ms
```

Anda tidak perlu konfigurasi tambahan, fitur ini aktif secara default untuk mencegah kebocoran data (PII Leak) di log server.

---

## Read/Write Splitting

### Routing Otomatis

Framework memiliki logika routing cerdas:

1.  **Exec (INSERT/UPDATE/DELETE)**: Selalu ke **Write Pool** (Primary).
2.  **Query/QueryRow (SELECT)**:
    *   Secara default ke **Read Pool** (Replica) menggunakan *Round-Robin*.
    *   Jika terdeteksi "unsafe" atau dalam transaksi, diarahkan ke **Write Pool**.

```go
// Masuk ke Read Pool (Replica 1 atau 2)
db.Query(ctx, "SELECT * FROM users")

// Masuk ke Write Pool (Primary)
db.Exec(ctx, "UPDATE users SET name=$1", "John")
```

---

## Operasi Query

API `dim.Database` kompatibel dengan standar `database/sql` namun menggunakan `pgx` di belakang layar.

### Query Row

```go
var name string
err := db.QueryRow(ctx, "SELECT name FROM users WHERE id=$1", 1).Scan(&name)
if err != nil {
    // Handle error (termasuk pgx.ErrNoRows)
}
```

### Query Multiple Rows

```go
rows, err := db.Query(ctx, "SELECT id, name FROM users")
if err != nil {
    return err
}
defer rows.Close()

for rows.Next() {
    var u User
    if err := rows.Scan(&u.ID, &u.Name); err != nil {
        return err
    }
    // ...
}
```

---

## Transaksi

### Transaksi Manual

```go
tx, err := db.Begin(ctx)
if err != nil {
    return err
}
defer tx.Rollback(ctx)

// Operasi dalam transaksi
if _, err := tx.Exec(ctx, "UPDATE ..."); err != nil {
    return err
}

if err := tx.Commit(ctx); err != nil {
    return err
}
```

### Transaksi Otomatis (`WithTx`)

Helper `WithTx` menangani commit/rollback secara otomatis. Sangat disarankan untuk menghindari *dangling transaction*.

```go
err := db.WithTx(ctx, func(ctx context.Context, tx pgx.Tx) error {
    // Query 1
    if _, err := tx.Exec(ctx, "INSERT INTO balance ..."); err != nil {
        return err // Otomatis Rollback
    }

    // Query 2
    if _, err := tx.Exec(ctx, "UPDATE wallet ..."); err != nil {
        return err // Otomatis Rollback
    }

    return nil // Otomatis Commit jika return nil
})
```

---

## Praktik Terbaik

1.  **Gunakan `WithTx`**: Mencegah lupa `Rollback` atau `Commit`.
2.  **Gunakan Context**: Selalu pass `r.Context()` dari handler HTTP agar query bisa dibatalkan jika klien putus koneksi.
3.  **Prepared Statements**: Selalu gunakan placeholder `$1`, `$2` untuk argumen. Jangan string concatenation (rawan SQL Injection).
4.  **Cek Error**: Selalu cek error dari `Scan` dan `Exec`.