# Database Migrations di Framework dim

Pelajari cara melakukan database versioning dan schema management dengan migrations.

## Daftar Isi

- [Konsep Migrations](#konsep-migrations)
- [Migration Structure](#migration-structure)
- [Define Migrations](#define-migrations)
- [Run Migrations](#run-migrations)
- [Migration Examples](#migration-examples)
- [Rollback](#rollback)
- [Praktik Terbaik](#best-practices)

---

## Konsep Migrations

### Apa itu Migration?

Migration adalah versi dari database schema:

```
Version 1: Create users table
Version 2: Create posts table
Version 3: Add bio column to users
Version 4: Create comments table
Version 5: Add index on posts
```

### Migration Flow

```
Initial Database
  ↓
Migration 1 (Up)  → Add users table
  ↓
Migration 2 (Up)  → Add posts table
  ↓
Migration 3 (Up)  → Add profile table
  ↓
Current Database
```

### Framework Migration System

Framework dim menggunakan **Go functions** untuk migrations:

```go
import "github.com/jackc/pgx/v5/pgxpool"

type Migration struct {
    Version int                      // Sequential version number
    Name    string                   // Description
    Up      func(*pgxpool.Pool) error     // Apply migration
    Down    func(*pgxpool.Pool) error     // Rollback migration
}
```

### Why Migrations?

- **Version control** - Track schema changes
- **Consistency** - Same schema across environments
- **Reversibility** - Rollback if needed
- **Documentation** - Each change documented
- **Team coordination** - Everyone uses same schema

---

## Migration Structure

### Migration Struct

```go
import "github.com/jackc/pgx/v5/pgxpool"

type Migration struct {
    Version int
    Name    string
    Up      func(*pgxpool.Pool) error
    Down    func(*pgxpool.Pool) error
}
```

### Metadata Tracking

Framework otomatis tracking migration history:

```sql
CREATE TABLE migrations (
    version INT PRIMARY KEY,
    name VARCHAR(255) NOT NULL,
    executed_at TIMESTAMP DEFAULT NOW()
)
```

---

## Define Migrations

### Basic Migration

```go
import (
    "context"
    "github.com/jackc/pgx/v5/pgxpool"
)

func getMigrations() []Migration {
    return []Migration{
        {
            Version: 1,
            Name:    "create_users_table",
            Up: func(db *pgxpool.Pool) error {
                _, err := db.Exec(context.Background(), `
                    CREATE TABLE IF NOT EXISTS users (
                        id BIGSERIAL PRIMARY KEY,
                        email VARCHAR(255) UNIQUE NOT NULL,
                        username VARCHAR(100) UNIQUE NOT NULL,
                        password_hash VARCHAR(255) NOT NULL,
                        created_at TIMESTAMP DEFAULT NOW(),
                        updated_at TIMESTAMP DEFAULT NOW()
                    )
                `)
                return err
            },
            Down: func(db *pgxpool.Pool) error {
                _, err := db.Exec(context.Background(), `
                    DROP TABLE IF EXISTS users
                `)
                return err
            },
        },
    }
}
```

### Migration dengan Index

```go
{
    Version: 2,
    Name:    "create_posts_table",
    Up: func(db *pgxpool.Pool) error {
        // Create table
        _, err := db.Exec(context.Background(), `
            CREATE TABLE IF NOT EXISTS posts (
                id BIGSERIAL PRIMARY KEY,
                user_id BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
                title VARCHAR(255) NOT NULL,
                content TEXT NOT NULL,
                created_at TIMESTAMP DEFAULT NOW(),
                updated_at TIMESTAMP DEFAULT NOW()
            )
        `)
        if err != nil {
            return err
        }
        
        // Create index
        _, err = db.Exec(context.Background(), `
            CREATE INDEX IF NOT EXISTS idx_posts_user_id ON posts(user_id)
        `)
        return err
    },
    Down: func(db *pgxpool.Pool) error {
        _, err := db.Exec(context.Background(), `
            DROP TABLE IF EXISTS posts
        `)
        return err
    },
},
```

### Migration dengan Data

```go
{
    Version: 3,
    Name:    "seed_initial_data",
    Up: func(db *pgxpool.Pool) error {
        _, err := db.Exec(context.Background(), `
            INSERT INTO users (email, username, password_hash) VALUES
            ('admin@example.com', 'admin', 'hashed_password'),
            ('user@example.com', 'user', 'hashed_password')
            ON CONFLICT DO NOTHING
        `)
        return err
    },
    Down: func(db *pgxpool.Pool) error {
        _, err := db.Exec(context.Background(), `
            DELETE FROM users WHERE email IN ('admin@example.com', 'user@example.com')
        `)
        return err
    },
},
```

### Migration dengan Multiple Steps

```go
{
    Version: 4,
    Name:    "add_user_roles",
    Up: func(db *pgxpool.Pool) error {
        ctx := context.Background()
        
        // Create role enum
        if _, err := db.Exec(ctx, `
            CREATE TYPE user_role AS ENUM ('user', 'admin', 'moderator')
        `); err != nil {
            return err
        }
        
        // Add role column
        if _, err := db.Exec(ctx, `
            ALTER TABLE users ADD COLUMN role user_role DEFAULT 'user'
        `); err != nil {
            return err
        }
        
        // Set admin role for existing admin
        _, err := db.Exec(ctx, `
            UPDATE users SET role = 'admin' WHERE email = 'admin@example.com'
        `)
        return err
    },
    Down: func(db *pgxpool.Pool) error {
        ctx := context.Background()
        
        // Remove role column
        if _, err := db.Exec(ctx, `
            ALTER TABLE users DROP COLUMN role
        `); err != nil {
            return err
        }
        
        // Drop enum
        _, err := db.Exec(ctx, `
            DROP TYPE user_role
        `)
        return err
    },
},
```

---

## Run Migrations

### Main Application

```go
func main() {
    // Load config
    cfg, err := dim.LoadConfig()
    if err != nil {
        log.Fatal("Config error:", err)
    }
    
    // Connect database
    db, err := dim.NewPostgresDatabase(cfg.Database)
    if err != nil {
        log.Fatal("Database error:", err)
    }
    defer db.Close()
    
    // Define migrations
    migrations := getMigrations()
    
    // Run migrations
    if err := dim.RunMigrations(db, migrations); err != nil {
        log.Fatal("Migration error:", err)
    }
    
    log.Println("Migrations completed successfully")
    
    // Continue with application setup
    router := setupRouter()
    http.ListenAndServe(":8080", router)
}
```

### Separate Migration Command

```go
package main

import (
    "flag"
    "log"
)

func main() {
    action := flag.String("action", "up", "Migration action: up, down, status")
    flag.Parse()
    
    // Load config
    cfg, _ := dim.LoadConfig()
    
    // Connect database
    db, _ := dim.NewPostgresDatabase(cfg.Database)
    defer db.Close()
    
    // Define migrations
    migrations := getMigrations()
    
    switch *action {
    case "up":
        dim.RunMigrations(db, migrations)
        log.Println("Migrations applied")
        
    case "down":
        // Dapatkan migrasi terakhir untuk di-rollback
        if len(migrations) > 0 {
            lastMigration := migrations[len(migrations)-1]
            dim.RollbackMigration(db, lastMigration)
            log.Println("Last migration rolled back")
        }
    case "status":
        // Fungsi status tidak diimplementasikan di framework
        log.Println("Migration status check is not implemented.")
    }
}
```

---

## Migration Examples

### Users Table

```go
{
    Version: 1,
    Name:    "create_users_table",
    Up: func(db *pgxpool.Pool) error {
        _, err := db.Exec(context.Background(), `
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
        `)
        return err
    },
    Down: func(db *pgxpool.Pool) error {
        _, err := db.Exec(context.Background(), `
            DROP TABLE IF EXISTS users
        `)
        return err
    },
},
```

### Posts Table dengan Foreign Key

```go
{
    Version: 2,
    Name:    "create_posts_table",
    Up: func(db *pgxpool.Pool) error {
        _, err := db.Exec(context.Background(), `
            CREATE TABLE IF NOT EXISTS posts (
                id BIGSERIAL PRIMARY KEY,
                user_id BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
                title VARCHAR(255) NOT NULL,
                content TEXT NOT NULL,
                published BOOLEAN DEFAULT false,
                created_at TIMESTAMP DEFAULT NOW(),
                updated_at TIMESTAMP DEFAULT NOW()
            );
            
            CREATE INDEX idx_posts_user_id ON posts(user_id);
            CREATE INDEX idx_posts_published ON posts(published);
        `)
        return err
    },
    Down: func(db *pgxpool.Pool) error {
        _, err := db.Exec(context.Background(), `
            DROP TABLE IF EXISTS posts
        `)
        return err
    },
},
```

### Comments Table

```go
{
    Version: 3,
    Name:    "create_comments_table",
    Up: func(db *pgxpool.Pool) error {
        _, err := db.Exec(context.Background(), `
            CREATE TABLE IF NOT EXISTS comments (
                id BIGSERIAL PRIMARY KEY,
                post_id BIGINT NOT NULL REFERENCES posts(id) ON DELETE CASCADE,
                user_id BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
                content TEXT NOT NULL,
                created_at TIMESTAMP DEFAULT NOW(),
                updated_at TIMESTAMP DEFAULT NOW()
            );
            
            CREATE INDEX idx_comments_post_id ON comments(post_id);
            CREATE INDEX idx_comments_user_id ON comments(user_id);
        `)
        return err
    },
    Down: func(db *pgxpool.Pool) error {
        _, err := db.Exec(context.Background(), `
            DROP TABLE IF EXISTS comments
        `)
        return err
    },
},
```

### Schema Alteration

```go
{
    Version: 4,
    Name:    "add_bio_to_users",
    Up: func(db *pgxpool.Pool) error {
        _, err := db.Exec(context.Background(), `
            ALTER TABLE users ADD COLUMN bio TEXT DEFAULT ''
        `)
        return err
    },
    Down: func(db *pgxpool.Pool) error {
        _, err := db.Exec(context.Background(), `
            ALTER TABLE users DROP COLUMN bio
        `)
        return err
    },
},
```

### Constraints

```go
{
    Version: 5,
    Name:    "add_constraints",
    Up: func(db *pgxpool.Pool) error {
        ctx := context.Background()
        
        // Add check constraint
        if _, err := db.Exec(ctx, `
            ALTER TABLE posts ADD CONSTRAINT check_title_length 
            CHECK (LENGTH(title) > 0 AND LENGTH(title) <= 255)
        `); err != nil {
            return err
        }
        
        // Add unique constraint
        _, err := db.Exec(ctx, `
            ALTER TABLE users ADD CONSTRAINT unique_email_username 
            UNIQUE (email, username)
        `)
        return err
    },
    Down: func(db *pgxpool.Pool) error {
        ctx := context.Background()
        
        if _, err := db.Exec(ctx, `
            ALTER TABLE posts DROP CONSTRAINT check_title_length
        `); err != nil {
            return err
        }
        
        _, err := db.Exec(ctx, `
            ALTER TABLE users DROP CONSTRAINT unique_email_username
        `)
        return err
    },
},
```

---

## Rollback

Framework menyediakan fungsi `RollbackMigration` untuk membatalkan migrasi terakhir yang diterapkan. Fungsi ini akan menjalankan `Down` dari migrasi yang dituju.

### Rollback Satu Migrasi

Fungsi `dim.RollbackMigration` dapat digunakan untuk membatalkan migrasi spesifik. Anda biasanya ingin membatalkan migrasi terakhir.

```go
// main.go
migrations := getMigrations()
db, _ := dim.NewPostgresDatabase(cfg.Database)

if len(migrations) > 0 {
    // Ambil migrasi terakhir dari daftar Anda
    lastMigration := migrations[len(migrations)-1]

    // Jalankan rollback
    if err := dim.RollbackMigration(db, lastMigration); err != nil {
        log.Fatalf("Gagal melakukan rollback: %v", err)
    }
    log.Printf("Rollback untuk migrasi %d (%s) berhasil.", lastMigration.Version, lastMigration.Name)
}
```

---

## Praktik Terbaik

### ✅ DO: Write Reversible Migrations

```go
// ✅ BAIK - Reversible
{
    Version: 1,
    Name:    "create_users_table",
    Up: func(db *pgxpool.Pool) error {
        _, err := db.Exec(context.Background(), `
            CREATE TABLE users (
                id BIGSERIAL PRIMARY KEY,
                email VARCHAR(255) NOT NULL
            )
        `)
        return err
    },
    Down: func(db *pgxpool.Pool) error {
        _, err := db.Exec(context.Background(), `
            DROP TABLE IF EXISTS users
        `)
        return err
    },
},

// ❌ BURUK - Tidak bisa dibalikkan
{
    Version: 2,
    Name:    "delete_old_data",
    Up: func(db *pgxpool.Pool) error {
        _, err := db.Exec(context.Background(), `
            DELETE FROM users WHERE created_at < NOW() - INTERVAL '1 year'
        `)
        return err
    },
    Down: func(db *pgxpool.Pool) error {
        return nil  // Data yang dihapus tidak bisa dikembalikan!
    },
},
```

### ✅ DO: Test Migrations

```go
// Uji di lingkungan development terlebih dahulu
// 1. Jalankan migrasi pada database baru
// 2. Verifikasi skema sudah benar
// 3. Uji dengan data sampel
// 4. Uji proses rollback
// 5. Baru deploy ke produksi
```

### ✅ DO: One Change Per Migration

```go
// ✅ BAIK - Satu perubahan logis
{
    Version: 1,
    Name:    "create_users_table",
    Up: func(db *pgxpool.Pool) error { /* ... */ },
},
{
    Version: 2,
    Name:    "create_posts_table",
    Up: func(db *pgxpool.Pool) error { /* ... */ },
},

// ❌ BURUK - Beberapa perubahan logis digabung
{
    Version: 1,
    Name:    "setup_database",
    Up: func(db *pgxpool.Pool) error {
        db.Exec(context.Background(), `CREATE TABLE users (...)`)
        db.Exec(context.Background(), `CREATE TABLE posts (...)`)
        db.Exec(context.Background(), `CREATE TABLE comments (...)`)
        return nil
    },
},
```

### ✅ DO: Never Modify Existing Migrations

```
❌ JANGAN ubah migrasi yang sudah dijalankan.
✅ BUAT migrasi baru untuk memperbaiki masalah.
```

Migration 1:
```go
{Version: 1, Name: "create_users_table", Up: func(...) {...}}
```

Jika nanti perlu diubah:
```go
{Version: 2, Name: "fix_users_email_length", Up: func(db *pgxpool.Pool) error {
    _, err := db.Exec(context.Background(), `
        ALTER TABLE users ALTER COLUMN email TYPE VARCHAR(255)
    `)
    return err
}}
```

### ✅ DO: Use IF EXISTS / IF NOT EXISTS

```go
// ✅ BAIK - Idempotent
db.Exec(context.Background(), `
    CREATE TABLE IF NOT EXISTS users (...)
`)
db.Exec(context.Background(), `
    CREATE INDEX IF NOT EXISTS idx_users_email ON users(email)
`)
db.Exec(context.Background(), `
    DROP TABLE IF EXISTS old_table
`)

// ❌ BURUK - Tidak idempotent
db.Exec(context.Background(), `
    CREATE TABLE users (...)  // Error jika tabel sudah ada
`)
```

### ✅ DO: Document Migration Purpose

```go
{
    Version: 5,
    Name:    "add_user_roles_system",
    // Nama yang baik menjelaskan TUJUAN, bukan hanya AKSI
    // Bukan: "alter_users_table"
    
    Up: func(db *pgxpool.Pool) error {
        // Komentar yang menjelaskan mengapa
        // Sebelum ini, semua user punya izin yang sama.
        // Migrasi ini menambahkan kontrol akses berbasis peran.
        _, err := db.Exec(context.Background(), `...`)
        return err
    },
},
```

---

## Example Integration

Complete migrations file:

```go
package main

import (
    "context"
    "github.com/nuradiyana/dim"
    "github.com/jackc/pgx/v5/pgxpool"
)

func GetMigrations() []dim.Migration {
    return []dim.Migration{
        // v1: Create users table
        {
            Version: 1,
            Name:    "create_users_table",
            Up: func(db *pgxpool.Pool) error {
                _, err := db.Exec(context.Background(), `
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
                `)
                return err
            },
            Down: func(db *pgxpool.Pool) error {
                _, err := db.Exec(context.Background(), `
                    DROP TABLE IF EXISTS users CASCADE
                `)
                return err
            },
        },
        
        // v2: Create posts table
        {
            Version: 2,
            Name:    "create_posts_table",
            Up: func(db *pgxpool.Pool) error {
                _, err := db.Exec(context.Background(), `
                    CREATE TABLE IF NOT EXISTS posts (
                        id BIGSERIAL PRIMARY KEY,
                        user_id BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
                        title VARCHAR(255) NOT NULL,
                        content TEXT NOT NULL,
                        created_at TIMESTAMP DEFAULT NOW(),
                        updated_at TIMESTAMP DEFAULT NOW()
                    );
                    CREATE INDEX idx_posts_user_id ON posts(user_id);
                `)
                return err
            },
            Down: func(db *pgxpool.Pool) error {
                _, err := db.Exec(context.Background(), `
                    DROP TABLE IF EXISTS posts CASCADE
                `)
                return err
            },
        },
    }
}
```

---

## Summary

Migrations di dim:
- **Version control** - Track schema changes
- **Reversible** - Up dan Down untuk setiap change
- **Go functions** - No external tools needed
- **Idempotent** - Safe to run multiple times
- **Documented** - Self-documenting schema changes

Lihat [Database](06-database.md) untuk database operations.

---

**Lihat Juga**:
- [Database](06-database.md) - Database operations
- [Setup](01-getting-started.md) - Running migrations in main