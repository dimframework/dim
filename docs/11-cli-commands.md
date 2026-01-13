# CLI & Console Commands

Framework dim dilengkapi dengan sistem CLI (Command Line Interface) yang powerful untuk membantu tugas-tugas development dan operasional.

## Daftar Isi
- [Setup Console](#setup-console)
- [Menggunakan CLI](#menggunakan-cli)
- [Built-in Commands](#built-in-commands)
  - [serve](#serve)
  - [migrate](#migrate)
  - [migrate:rollback](#migrate-rollback)
  - [migrate:list](#migrate-list)
  - [route:list](#route-list)
- [Custom Commands](#custom-commands)

---

## Setup Console

Agar aplikasi Anda mendukung CLI, Anda perlu menginisialisasi `Console` di `main.go`. Console bertindak sebagai entry point yang mem-parsing arguments dan menjalankan command yang sesuai.

```go
package main

import (
    "log"
    "os"
    "github.com/nuradiyana/dim"
)

func main() {
    // 1. Setup Dependencies (Config, DB, Router)
    config, _ := dim.LoadConfig()
    db, _ := dim.NewPostgresDatabase(config.Database)
    router := dim.NewRouter()
    
    // Register routes...
    router.Get("/users", listUsers)
    // Build router agar siap untuk introspeksi
    router.Build()

    // 2. Inisialisasi Console
    console := dim.NewConsole(db, router, config)

    // 3. Register Built-in Commands
    console.RegisterBuiltInCommands()

    // 4. Jalankan Console dengan os.Args
    if err := console.Run(os.Args[1:]); err != nil {
        log.Fatal(err)
    }
}
```

---

## Menggunakan CLI

Setelah setup selesai, Anda bisa menjalankan command melalui `go run main.go`.

Format umum:
```bash
go run main.go [command] [flags]
```

Jika dijalankan tanpa argumen, default-nya adalah command `serve`.

```bash
# Menjalankan server (default)
go run main.go

# Menjalankan command spesifik
go run main.go migrate
```

Untuk melihat bantuan:
```bash
go run main.go help
go run main.go [command] -h
```

---

## Built-in Commands

Framework dim menyertakan beberapa command bawaan yang sangat berguna.

### `serve`
Menjalankan HTTP server aplikasi.

**Usage:**
```bash
go run main.go serve [flags]
```

**Flags:**
- `-port`: Override port server dari konfigurasi (Contoh: `-port 3000`)

---


### `migrate`
Menjalankan semua pending database migrations. Command ini akan mengeksekusi file migrasi yang belum pernah dijalankan sebelumnya.

**Usage:**
```bash
go run main.go migrate [flags]
```

**Flags:**
- `-v`: Verbose mode, menampilkan detail setiap step migrasi.

---


### `migrate:rollback`
Membatalkan (rollback) migrasi database terakhir.

**Usage:**
```bash
go run main.go migrate:rollback [flags]
```

**Flags:**
- `-step`: Jumlah batch migrasi yang ingin di-rollback (Default: 1).

---


### `migrate:list`
Menampilkan status semua migrasi (Applied vs Pending). Sangat berguna untuk mengecek sinkronisasi database.

**Usage:**
```bash
go run main.go migrate:list
```

**Output:**
```
Migration Status:

Version    Name                                     Status     Applied At
-------------------------------------------------------------------------
1          create_users_table                       Applied    2025-01-14 10:00:00
2          add_profile_column                       Pending    -
```

---


### `route:list`
Menampilkan daftar semua route yang terdaftar di aplikasi, lengkap dengan HTTP Method, Path, Handler function, dan Middleware yang aktif.

**Usage:**
```bash
go run main.go route:list
```

**Output:**
```
Registered Routes (5 total):

GET     /users                         -> main.getUsersHandler              [dim.LoggerMiddleware]
POST    /users                         -> main.createUserHandler            [dim.LoggerMiddleware, dim.AuthMiddleware]
```

---

## Custom Commands

Anda dapat membuat command sendiri untuk tugas spesifik seperti seeding data, clearing cache, atau cron jobs.

### 1. Buat Struct Command

Implementasikan interface `dim.Command`.

```go
type HelloCommand struct{}

func (c *HelloCommand) Name() string {
    return "hello"
}

func (c *HelloCommand) Description() string {
    return "Says hello to the world"
}

func (c *HelloCommand) Execute(ctx *dim.CommandContext) error {
    fmt.Println("Hello, World!")
    return nil
}
```


### 2. (Opsional) Tambahkan Flags

Jika command butuh input parameter, implementasikan `dim.FlaggedCommand`.

```go
type SeedCommand struct {
    count int
}

// ... Name & Description methods ...

func (c *SeedCommand) DefineFlags(fs *flag.FlagSet) {
    fs.IntVar(&c.count, "count", 10, "Number of items to seed")
}

func (c *SeedCommand) Execute(ctx *dim.CommandContext) error {
    // Akses dependencies via ctx
    // ctx.DB, ctx.Config, dll
    
    fmt.Printf("Seeding %d items into DB...\n", c.count)
    return nil
}
```


### 3. Register Command

Daftarkan command baru Anda di `main.go`.

```go
func main() {
    // ... setup ...
    console := dim.NewConsole(db, router, config)
    
    // Register custom commands
    console.Register(&HelloCommand{})
    console.Register(&SeedCommand{}) // Usage: go run main.go seed -count 50
    
    console.Run(os.Args[1:])
}
```

---

## Tips

- Gunakan `ctx.DB` di dalam `Execute` untuk mengakses database.
- Gunakan `ctx.Config` untuk membaca konfigurasi environment.
- Nama command sebaiknya menggunakan format `verb` atau `noun:verb` (contoh: `cache:clear`, `user:create`).

```
