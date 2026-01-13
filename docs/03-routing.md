# Routing di Framework dim

Pelajari cara mendaftar dan mengelola routes HTTP di dim menggunakan standar modern Go 1.22+.

## Daftar Isi

- [Konsep Dasar](#konsep-dasar)
- [Route Registration](#route-registration)
- [Path Parameters](#path-parameters)
- [Static Files & SPA](#static-files--spa)
- [Route Grouping](#route-grouping)
- [Middleware Per-Route](#middleware-per-route)
- [Advanced Routing](#advanced-routing)

---

## Konsep Dasar

Framework dim dibangun di atas router standar library `http.ServeMux` yang diperkenalkan di Go 1.22. Ini memberikan performa tinggi, standar kompatibilitas yang sangat baik, dan sintaks yang bersih.

**Struktur Route**:
```
Method Path → Handler
GET /users/{id} → func getUser(w, r)
```

**Fitur Utama**:
- **Method Matching**: `GET /users` vs `POST /users`
- **Wildcards**: `/files/{path...}`
- **Path Parameters**: `/users/{id}` (menggunakan `{}` bukan `:`)

---

## Route Registration

### Inisialisasi Router

```go
router := dim.NewRouter()
```

### HTTP Methods

Daftar route menggunakan helper method yang tersedia:

```go
// GET request
router.Get("/users", listUsersHandler)

// POST request
router.Post("/users", createUserHandler)

// PUT request
router.Put("/users/{id}", updateUserHandler)

// DELETE request
router.Delete("/users/{id}", deleteUserHandler)

// PATCH request
router.Patch("/users/{id}", patchUserHandler)

// OPTIONS, HEAD, dll
router.Options("/users", optionsHandler)
router.Head("/users", headHandler)
```

### Handler Function Signature

Handler menggunakan standar `http.HandlerFunc`:

```go
func handlerName(w http.ResponseWriter, r *http.Request) {
    // ...
}
```

---

## Path Parameters

Gunakan kurung kurawal `{}` untuk mendefinisikan parameter dinamis. Nilai parameter dapat diambil menggunakan method standar `r.PathValue()`.

### Single Parameter

```go
// Definisi: Gunakan {id}
router.Get("/users/{id}", func(w http.ResponseWriter, r *http.Request) {
    // Pengambilan: Gunakan r.PathValue("id")
    id := r.PathValue("id")
    
    dim.Response(w).Ok(dim.Map{
        "id": id,
    })
})

// Request: GET /users/123
// Response: {"id": "123"}
```

### Multiple Parameters

```go
router.Get("/users/{userId}/posts/{postId}", func(w http.ResponseWriter, r *http.Request) {
    userId := r.PathValue("userId")
    postId := r.PathValue("postId")
    
    // ...
})
```

### Wildcards (Catch-All)

Gunakan `...` di akhir nama parameter untuk menangkap sisa path. Ini berguna untuk file server atau proxy.

```go
// Matches /files/a.jpg, /files/docs/report.pdf, dll
router.Get("/files/{path...}", func(w http.ResponseWriter, r *http.Request) {
    filePath := r.PathValue("path")
    // filePath = "docs/report.pdf"
})
```

---

## Static Files & SPA

Dim memiliki dukungan untuk menyajikan aset statis dan aplikasi Single Page Application (SPA) seperti React, Vue, atau Svelte.

Fitur ini mendukung interface `fs.FS`, yang memungkinkan Anda menggunakan:
1.  **Folder Lokal** (`os.DirFS`) untuk development.
2.  **Go Embed** (`embed.FS`) untuk membungkus frontend ke dalam binary Go.

### Static Assets

Untuk menyajikan file seperti CSS, JS, dan Gambar.

```go
// Menggunakan Folder Lokal
router.Static("/assets/", os.DirFS("./public"))

// Menggunakan Embed (Production)
//go:embed public/*
var staticFS embed.FS
publicFS, _ := fs.Sub(staticFS, "public")

router.Static("/assets/", publicFS)
```

Secara otomatis menambahkan header keamanan: `X-Content-Type-Options: nosniff`.

### Single Page Application (SPA)

Untuk SPA, router perlu menangani "fallback" ke `index.html` jika user me-refresh halaman di route klien (misal `/dashboard`).

```go
// 1. Definisikan API route TERLEBIH DAHULU
router.Group("/api", apiHandler)

// 2. Definisikan SPA di akhir
// Ini akan menangani semua route yang tidak cocok dengan API
router.SPA(os.DirFS("./dist"), "index.html")
```

**Fitur Otomatis SPA**:
- **Fallback**: Mengembalikan `index.html` jika file statis tidak ditemukan.
- **Security**: Menambahkan header keamanan pada `index.html`.
- **Anti-Cache**: Menambahkan `Cache-Control: no-cache` pada `index.html` agar user selalu mendapat versi terbaru setelah deploy.

---

## Route Grouping

Kelompokkan routes dengan prefix dan shared middleware.

```go
// Buat grup /api dengan middleware Auth
api := router.Group("/api", dim.RequireAuth(jwtManager))

// Route menjadi: GET /api/users
api.Get("/users", listUsersHandler)

// Nested Grouping
v1 := api.Group("/v1")
v1.Get("/products", listProducts) // GET /api/v1/products
```

---

## Middleware Per-Route

Anda dapat menerapkan middleware secara spesifik untuk satu route saja.

```go
// Middleware RateLimit hanya untuk route login
router.Post("/login", 
    loginHandler, 
    dim.RateLimit(limitConfig),
)

// Middleware CORS khusus untuk public endpoint
router.Get("/public-data", 
    dataHandler,
    dim.Cors(publicConfig),
)
```

Urutan eksekusi:
1. Global Middleware
2. Group Middleware
3. Route Middleware
4. Handler

---

## Advanced Routing

### Custom Middleware untuk Static/SPA

Fungsi `Static` dan `SPA` mendukung parameter variadik untuk middleware tambahan.

```go
// Tambahkan header cache custom untuk static assets
router.Static("/img/", os.DirFS("./img"), func(next dim.HandlerFunc) dim.HandlerFunc {
    return func(w http.ResponseWriter, r *http.Request) {
        w.Header().Set("Cache-Control", "public, max-age=31536000")
        next(w, r)
    }
})
```

---

## Ringkasan

- Gunakan `{param}` untuk path parameters.
- Akses parameter via `r.PathValue("param")`.
- Gunakan `router.Static` dan `router.SPA` dengan `fs.FS` untuk kemudahan deployment.
- Manfaatkan `router.Group` untuk mengorganisir API.