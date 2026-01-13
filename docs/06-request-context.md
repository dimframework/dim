# Request Context di Framework dim

Pelajari cara mengelola request context dan extract data dari request.

## Daftar Isi

- [Konsep Context](#konsep-context)
- [User Context](#user-context)
- [Path Parameters](#path-parameters)
- [Query Parameters](#query-parameters)
- [Header dan Token Auth](#header-dan-token-auth)
- [Request Body](#request-body)
- [Custom Context Values](#custom-context-values)
- [Context Lifecycle](#context-lifecycle)
- [Praktik Terbaik](#best-practices)

---

## Konsep Context

### Apa itu Request Context?

Request context adalah cara untuk membawa data melalui siklus hidup sebuah request. Data dapat ditambahkan di middleware dan kemudian diakses di dalam handler.

```go
// Ambil context dari request
ctx := r.Context()

// Set context dengan value
ctx = context.WithValue(ctx, key, value)
r = r.WithContext(ctx) // Buat request baru dengan context yang sudah diupdate
```

---

## User Context

Cara paling umum untuk menggunakan context adalah untuk menyimpan informasi pengguna yang terautentikasi.

### Mengatur User di Middleware

Middleware `dim.RequireAuth(jwtManager)` secara otomatis menangani proses ini:
1.  Mengekstrak token JWT dari header `Authorization`.
2.  Memverifikasi token menggunakan `jwtManager`.
3.  Jika valid, membuat objek `dim.User` dari *claims* token.
4.  Menyimpan objek `User` ke dalam *request context* menggunakan `dim.SetUser`.

### Mengambil User di Handler

Setelah middleware `dim.RequireAuth` dijalankan, Anda dapat dengan aman mengambil data pengguna di *handler* manapun.

```go
func profileHandler(w http.ResponseWriter, r *http.Request) {
    // Get user dari context
    user, ok := dim.GetUser(r)
    if !ok {
        // Ini seharusnya tidak akan terjadi jika rute dilindungi oleh RequireAuth
        dim.Unauthorized(w, "Unauthorized")
        return
    }
    
    // Gunakan informasi pengguna
    dim.OK(w, user)
}
```

### Struct User

Objek `dim.User` yang disimpan di context berisi `ID` dan `Email` dari *claims* token.

```go
// Dari package dim
type User struct {
    ID    int64
    Email string
}
```

---

## Path Parameters

### Mendeklarasikan Path Parameter

```go
router.Get("/users/:id", getUserHandler)
router.Get("/users/:userId/posts/:postId", getUserPostHandler)
```

### Mengekstrak Satu Parameter

Gunakan `dim.GetParam(r, "key")` untuk mengambil nilai parameter.

```go
func getUserHandler(w http.ResponseWriter, r *http.Request) {
    idStr := dim.GetParam(r, "id")
    
    userID, err := strconv.ParseInt(idStr, 10, 64)
    if err != nil {
        dim.BadRequest(w, "ID tidak valid", nil)
        return
    }
    
    // Gunakan userID
}
```

### Mengekstrak Beberapa Parameter

Gunakan `dim.GetParams(r)` untuk mendapatkan map dari semua parameter.

```go
func getUserPostHandler(w http.ResponseWriter, r *http.Request) {
    params := dim.GetParams(r)
    // params akan menjadi: map[string]string{"userId": "123", "postId": "456"}
    
    userIdStr := params["userId"]
    postIdStr := params["postId"]
}
```

---

## Query Parameters

### Get Single Query Parameter

Gunakan `r.URL.Query().Get()` dari *standard library* atau pembantu `dim.GetQueryParam()`.

```go
func listUsersHandler(w http.ResponseWriter, r *http.Request) {
    // Menggunakan helper
    pageStr := dim.GetQueryParam(r, "page")
    if pageStr == "" {
        pageStr = "1"  // Nilai default
    }
    
    page, _ := strconv.Atoi(pageStr)
    
    users, _ := userStore.GetPage(r.Context(), page, 10)
    dim.OK(w, users)
}
```

### Get Multiple Query Parameters

Gunakan `dim.GetQueryParams()` untuk mengambil beberapa nilai sekaligus.

```go
func searchUsersHandler(w http.ResponseWriter, r *http.Request) {
    // Mengambil beberapa parameter sekaligus
    params := dim.GetQueryParams(r, "page", "limit", "sort")
    
    page, _ := strconv.Atoi(params["page"])
    // ...
}
```

### Get Multiple Values for Same Key

Jika sebuah *query parameter* muncul beberapa kali (misalnya, `?tag=A&tag=B`), gunakan `r.URL.Query()` secara langsung.

```go
func filterUsersHandler(w http.ResponseWriter, r *http.Request) {
    tags := r.URL.Query()["tag"]  // tipe: []string
    // Hasil untuk ?tag=A&tag=B -> ["A", "B"]
}
```

---

## Header dan Token Auth

### Get Header Value

Gunakan `dim.GetHeaderValue()` sebagai pintasan untuk `r.Header.Get()`.

```go
func handler(w http.ResponseWriter, r *http.Request) {
    contentType := dim.GetHeaderValue(r, "Content-Type")
}
```

### Get Auth Token

Fungsi `dim.GetAuthToken()` secara spesifik mengekstrak token dari header `Authorization`. Fungsi ini mengharapkan format `Bearer <token>`.

```go
func myCustomAuthMiddleware(next dim.HandlerFunc) dim.HandlerFunc {
    return func(w http.ResponseWriter, r *http.Request) {
        token, ok := dim.GetAuthToken(r)
        if !ok {
            dim.Unauthorized(w, "Token otorisasi tidak ditemukan atau formatnya salah")
            return
        }
        // ... lanjutkan dengan verifikasi token
        next(w, r)
    }
}
```

---

## Request Body

### Parse JSON Body

```go
type CreateUserRequest struct {
    Email    string `json:"email"`
    Username string `json:"username"`
}

func createUserHandler(w http.ResponseWriter, r *http.Request) {
    var req CreateUserRequest
    if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
        dim.BadRequest(w, "JSON tidak valid", nil)
        return
    }
    defer r.Body.Close()
    
    // ... gunakan req.Email, req.Username
}
```

---

(Sisa dokumen tidak perlu diubah dan dihilangkan dari sini untuk keringkasan)