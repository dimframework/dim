# Database Migrations di Framework dim

Pelajari cara melakukan database versioning dan schema management dengan migrations di framework dim.

## Daftar Isi

- [Konsep Migrations](#konsep-migrations)
- [Workflow](#workflow)
- [Membuat Migration (CLI)](#membuat-migration-cli)
- [Struktur Migration](#struktur-migration)
- [Menjalankan Migration](#menjalankan-migration)
- [Rollback](#rollback)
- [Contoh Kode](#contoh-kode)

---

## Konsep Migrations

### Timestamp-based Versioning

Framework dim menggunakan sistem versioning berbasis waktu (Timestamp) dengan format `YYYYMMDDHHMMSS` (contoh: `20260116123000`).
Sistem ini menggantikan integer versioning (1, 2, 3) untuk mencegah konflik saat bekerja dalam tim.

### Auto-Discovery

Sistem migrasi dim menggunakan pattern **Auto-Discovery**. Setiap file migrasi yang di-generate akan memiliki fungsi `init()` yang secara otomatis mendaftarkan dirinya ke framework. Anda tidak perlu lagi mendaftarkan migrasi secara manual di slice atau array.

---

## Workflow

Alur kerja standar dalam pengembangan database:

1.  **Generate**: Buat file migrasi baru dengan command `make:migration`.
2.  **Edit**: Tulis query `Up` (create/alter) dan `Down` (drop/revert) di file yang terbentuk.
3.  **Migrate**: Jalankan command `migrate` untuk menerapkan perubahan.
4.  **Rollback** (Opsional): Jalankan `migrate:rollback` jika ada kesalahan.

---

## Membuat Migration (CLI)

Gunakan command `make:migration` untuk membuat file template migrasi.

```bash
# Membuat migrasi baru (default ke folder 'migrations')
go run . make:migration create_products_table

# Output:
# Migration created: migrations/20260116120000_create_products_table.go
```

### Opsi Tambahan

```bash
# Simpan di direktori spesifik
go run . make:migration add_category_id --dir internal/database/migrations
```

---

## Struktur Migration

Setiap file migrasi yang dihasilkan memiliki struktur berikut:

```go
package migrations

import (
    "context"
    "github.com/jackc/pgx/v5/pgxpool"
    "github.com/dimframework/dim"
)

// init() otomatis dijalankan saat aplikasi start
// dan mendaftarkan migrasi ini ke global registry framework.
func init() {
    dim.Register(dim.Migration{
        Version: 20260116120000,
        Name:    "create_products_table",
        Up:      UpCreateProductsTable,
        Down:    DownCreateProductsTable,
    })
}

// Up: Dijalankan saat migrate
func UpCreateProductsTable(pool *pgxpool.Pool) error {
    _, err := pool.Exec(context.Background(), `
        CREATE TABLE IF NOT EXISTS products (
            id BIGSERIAL PRIMARY KEY,
            name VARCHAR(255) NOT NULL,
            price DECIMAL(10, 2) NOT NULL,
            created_at TIMESTAMP DEFAULT NOW(),
            updated_at TIMESTAMP DEFAULT NOW()
        )
    `)
    return err
}

// Down: Dijalankan saat rollback
func DownCreateProductsTable(pool *pgxpool.Pool) error {
    _, err := pool.Exec(context.Background(), `
        DROP TABLE IF EXISTS products
    `)
    return err
}
```

---

## Setup Aplikasi

Agar migrasi "terbaca" oleh command `migrate`, Anda harus melakukan **Blank Import** folder migrasi Anda di `main.go`. Ini memicu fungsi `init()` di setiap file migrasi untuk berjalan.

```go
// cmd/app/main.go
package main

import (
    "github.com/dimframework/dim"
    
    // PENTING: Import folder migrations agar ter-register
    _ "github.com/username/project/migrations"
)

func main() {
    // ... setup kode ...
}
```

---

## Menjalankan Migration

Framework `dim` menyediakan console command bawaan untuk manajemen migrasi.

### Migrate (Up)

Menjalankan semua migrasi yang belum dieksekusi (framework core + aplikasi).

```bash
go run . migrate
```

### Check Status

Melihat daftar migrasi yang sudah dan belum dijalankan.

```bash
go run . migrate:list
```

### Rollback (Down)

Membatalkan batch migrasi terakhir.

```bash
# Rollback 1 batch terakhir
go run . migrate:rollback

# Rollback spesifik 2 langkah
go run . migrate:rollback --step 2
```

---

## Contoh Kode manual (Advanced)

Jika Anda ingin menjalankan migrasi secara programatik (bukan lewat CLI), Anda bisa mengakses registry:

```go
func main() {
    // 1. Setup DB
    db, _ := dim.NewPostgresDatabase(cfg.Database)
    
    // 2. Gabungkan Migrasi Core + Registered Migrations
    allMigrations := dim.GetFrameworkMigrations()
    allMigrations = append(allMigrations, dim.GetRegisteredMigrations()...)
    
    // 3. Jalankan
    if err := dim.RunMigrations(db, allMigrations); err != nil {
        log.Fatal(err)
    }
}
```
