# Database Migrations

Kelola perubahan skema database secara terstruktur, versioned, dan aman menggunakan fitur migrasi bawaan dim. Framework ini mendukung **Database Agnostic Migrations**, memungkinkan satu set migrasi berjalan di PostgreSQL maupun SQLite.

## Daftar Isi

- [Konsep Migrations](#konsep-migrations)
- [Workflow](#workflow)
- [Membuat Migration (CLI)](#membuat-migration-cli)
- [Struktur Migration](#struktur-migration)
- [Menjalankan Migration](#menjalankan-migration)
- [Override Default Tables](#override-default-tables)
- [Migrasi Multi-Schema](#migrasi-multi-schema)

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

---

## Struktur Migration

Setiap file migrasi yang dihasilkan memiliki struktur berikut:

```go
package migrations

import (
    "context"
    "github.com/dimframework/dim"
)

func init() {
    dim.Register(dim.Migration{
        Version: 20260116120000,
        Name:    "create_products_table",
        Up:      UpCreateProductsTable,
        Down:    DownCreateProductsTable,
    })
}

func UpCreateProductsTable(db dim.Database) error {
    var query string
    
    // Sesuaikan syntax SQL berdasarkan driver
    if db.DriverName() == "sqlite" {
        query = `
            CREATE TABLE IF NOT EXISTS products (
                id INTEGER PRIMARY KEY AUTOINCREMENT,
                name TEXT NOT NULL,
                created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
            )
        `
    } else {
        // Default: PostgreSQL
        query = `
            CREATE TABLE IF NOT EXISTS products (
                id BIGSERIAL PRIMARY KEY,
                name VARCHAR(255) NOT NULL,
                created_at TIMESTAMP DEFAULT NOW()
            )
        `
    }
    
    return db.Exec(context.Background(), query)
}

func DownCreateProductsTable(db dim.Database) error {
    return db.Exec(context.Background(), `DROP TABLE IF EXISTS products`)
}
```

---

## Menjalankan Migration

```bash
# Migrate (Up)
go run . migrate

# Check Status
go run . migrate:list

# Rollback (Down)
go run . migrate:rollback
```

---

## Override Default Tables

Secara default, `dim` menyertakan migrasi untuk tabel inti seperti `users`, `refresh_tokens`, `password_reset_tokens`, dll.
Jika Anda ingin **mengganti** skema tabel-tabel ini (misalnya menggunakan `BIGSERIAL` untuk ID user alih-alih `UUID`, atau menambahkan kolom baru), Anda dapat menonaktifkan migrasi bawaan framework.

### Langkah-langkah Override

1.  **Disable Framework Migrations** di `init()` utama aplikasi Anda (misal di `cmd/app/main.go` atau file setup lainnya).

    ```go
    // cmd/app/main.go
    func init() {
        // Matikan migrasi bawaan framework
        dim.DisableFrameworkMigrations()
    }
    ```

2.  **Buat Migrasi Pengganti** menggunakan CLI.

    ```bash
    go run . make:migration create_core_tables
    ```

3.  **Definisikan Skema Baru** di file migrasi yang baru dibuat.
    Pastikan Anda tetap membuat tabel-tabel yang dibutuhkan oleh service `dim` (seperti `AuthService`) jika Anda menggunakannya, atau sesuaikan service tersebut.

    Contoh definisi ulang tabel users dengan ID serial:

    ```go
    // migrations/2026xxxx_create_core_tables.go
    func UpCreateCoreTables(db dim.Database) error {
        var query string
        if db.DriverName() == "sqlite" {
             query = `
                CREATE TABLE IF NOT EXISTS users (
                    id INTEGER PRIMARY KEY AUTOINCREMENT,
                    email TEXT UNIQUE NOT NULL,
                    name TEXT,
                    password TEXT NOT NULL,
                    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
                    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
                );
             `
        } else {
             query = `
                CREATE TABLE IF NOT EXISTS users (
                    id BIGSERIAL PRIMARY KEY,
                    email VARCHAR(255) UNIQUE NOT NULL,
                    name VARCHAR(100),
                    password VARCHAR(255) NOT NULL,
                    created_at TIMESTAMP DEFAULT NOW(),
                    updated_at TIMESTAMP DEFAULT NOW()
                );
             `
        }
        
        _, err := db.Exec(context.Background(), query)
        return err
    }
    ```

Dengan cara ini, saat Anda menjalankan `go run . migrate`, framework hanya akan menjalankan migrasi kustom Anda dan mengabaikan versi bawaan.

---

## Dedicated Migration Database Connection

Di beberapa environment, migration perlu dijalankan dengan kredensial atau host berbeda dari koneksi aplikasi — misalnya menggunakan superuser PostgreSQL agar bisa membuat extension atau mengubah schema.

Framework mendukung hal ini melalui `DB_MIGRATION_*` env vars dan `NewMigrationDatabase`.

### Environment Variables

```bash
# Opsional — jika tidak di-set, fallback ke nilai Write connection
DB_MIGRATION_HOST=migration.db.internal   # fallback: DB_WRITE_HOST
DB_MIGRATION_PORT=5432                    # fallback: DB_PORT
DB_MIGRATION_USER=migrate_superuser       # fallback: DB_USER
DB_MIGRATION_PASSWORD=superuser_password  # fallback: DB_PASSWORD
```

### Setup di main.go

```go
cfg, _ := dim.LoadConfig()

// Koneksi aplikasi (read/write biasa)
db, _ := dim.NewPostgresDatabase(cfg.Database)

// Koneksi migration — dibuat hanya jika MigrationHost di-set
var migrationDB dim.Database
if cfg.Database.MigrationHost != "" {
    migrationDB, _ = dim.NewMigrationDatabase(cfg.Database)
}

console := dim.NewConsole(db, router, cfg)
if migrationDB != nil {
    console.WithMigrationDB(migrationDB)
}
console.RegisterBuiltInCommands()
console.Run(os.Args[1:])
```

Jika `DB_MIGRATION_HOST` tidak di-set, semua perintah migrate tetap berjalan normal menggunakan koneksi Write — tidak ada perubahan behavior untuk setup yang sudah ada.
---

## Migrasi Multi-Schema

Aplikasi yang dipecah menjadi kernel + beberapa modul biasanya memberi tiap modul **schema PostgreSQL-nya sendiri** (`module_a`, `module_b`, …), dengan seluruh SQL dikualifikasi penuh (`module_a.<tabel>`) dan tanpa `search_path`. Tiga hal berikut yang dibutuhkan bentuk itu — ketiganya opsional, dan aplikasi yang tidak memakai schema tidak perlu menyentuhnya sama sekali.

### 1. Tabel pencatat per schema

Tanpa `search_path`, tabel pencatat riwayat migrasi mendarat di schema bawaan koneksi — bukan di schema modul yang sedang dimigrasi. `RunMigrationsIn` menerima nama tabelnya:

```go
// migrasi sebuah modul, riwayatnya dicatat di myschema.migrations
err := dim.RunMigrationsIn(db, "myschema.migrations", migrations)

// pasangannya saat rollback
err = dim.RollbackMigrationIn(db, "myschema.migrations", migration)
```

`RunMigrations(db, migrations)` adalah pembungkus tipis dari `RunMigrationsIn(db, "migrations", migrations)`, jadi kode lama tidak berubah perilakunya. Nama tabelnya boleh polos (`migrations`) atau berkualifikasi schema (`myschema.migrations`); nama yang bukan identifier SQL sah ditolak sebelum query apa pun dijalankan.

> **Schema-nya harus sudah ada.** dim tidak menjalankan `CREATE SCHEMA` — pembuatan schema adalah urusan aplikasi, karena ia yang tahu kepemilikan dan hak aksesnya.

Ini juga yang membuat isolasi test berbasis prefix schema bekerja: tiap test punya pencatatnya sendiri, aman-paralel, dan dibersihkan sekaligus lewat `DROP SCHEMA … CASCADE`.

```go
schema := "test_9f2c_myschema" // prefix unik per test
db.Exec(ctx, "CREATE SCHEMA "+schema)
dim.RunMigrationsIn(db, schema+".migrations", mymodule.Migrations(schema))
t.Cleanup(func() { db.Exec(ctx, "DROP SCHEMA "+schema+" CASCADE") })
```

### 2. Mengganti sumber migrasi

`dim.Register()` dipanggil dari `init()`, sehingga string SQL-nya beku sebelum program tahu ke schema mana ia akan bermigrasi. Aplikasi yang perlu merakit migrasinya saat runtime dapat menyerahkan perakitnya ke `SetMigrationSource`, dan perintah `migrate`, `migrate:list`, serta `migrate:rollback` akan memakainya alih-alih registry global:

```go
func main() {
    schema := os.Getenv("APP_SCHEMA") // "myschema", "test_9f2c_myschema", …
    dim.SetMigrationSource(func() []dim.Migration {
        return mymodule.Migrations(schema)
    })
    // …
}
```

Tidak dipanggil = perilaku sekarang, persis: sumbernya tetap `GetAllMigrations()`. Panggil dengan `nil` untuk mengembalikannya ke registry global. Slice yang dikembalikan selalu disalin dan diurutkan berdasarkan `Version` sebelum dipakai, sama seperti `GetAllMigrations()`.

### 3. Flag `--table` pada perintah bawaan

```bash
go run . migrate            --table myschema.migrations
go run . migrate:list       --table myschema.migrations
go run . migrate:rollback   --table myschema.migrations -step 2
```

Tanpa flag = `migrations`, sama seperti sebelumnya.

Efeknya pada rollback: cakupannya menjadi satu modul. Pada riwayat global, `migrate:rollback -step 3` bisa membatalkan satu migrasi kernel dan dua migrasi modul yang tidak berhubungan — satuannya "tiga terakhir", yang tidak punya makna domain. Dengan pencatat per schema, `-step 2` berarti dua migrasi terakhir **milik modul itu**.

Ini sekaligus melunakkan penolakan `migrate:rollback` terhadap migrasi yatim (lihat `-allow-missing`): mencabut sebuah modul berarti `DROP SCHEMA <schema modul itu> CASCADE`, yang menghapus tabelnya **dan** riwayatnya sekaligus — tidak ada baris yatim yang tersisa untuk mengunci rollback modul lain.
