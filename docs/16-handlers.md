# Handler Patterns & Arsitektur di Framework dim

Pelajari pola-pola handler dan arsitektur aplikasi yang baik untuk framework dim.

## Daftar Isi

- [Konsep Handler](#konsep-handler)
- [Handler Patterns](#handler-patterns)
- [Service Injection Pattern](#service-injection-pattern)
- [Struct-based Handler Pattern](#struct-based-handler-pattern)
- [Direct Handler Pattern](#direct-handler-pattern)
- [Error Handling dalam Handler](#error-handling-dalam-handler)
- [Request Parsing & Validation](#request-parsing--validation)
- [Response Formatting](#response-formatting)
- [Middleware Integration](#middleware-integration)
- [Best Practices](#best-practices)

---

## Konsep Handler

### Apa itu Handler?

Handler adalah fungsi atau method yang menangani HTTP request dan mengembalikan response. Handler menerima `http.ResponseWriter` dan `*http.Request` sebagai parameter.

**Signature Handler:**
```go
type HandlerFunc func(w http.ResponseWriter, r *http.Request)
```

### Tanggung Jawab Handler

1. **Parse Request** - Ekstrak data dari request (path params, query, body)
2. **Validasi Input** - Validasi data yang diterima
3. **Business Logic** - Panggil service atau langsung eksekusi logic
4. **Error Handling** - Handle error dengan graceful
5. **Format Response** - Kembalikan response yang terformat dengan baik

---

## Handler Patterns

Framework dim mendukung beberapa pola handler untuk fleksibilitas maksimal. Pilih yang paling sesuai dengan kebutuhan aplikasi Anda.

---

## Service Injection Pattern

**Pattern Terbaik untuk Separation of Concerns**

Handler menerima dependency (service, store) sebagai parameter dan mengembalikan `http.HandlerFunc`.

### Struktur Dasar

```go
// Handler dengan service injection
func LoginHandler(authService *dim.AuthService) http.HandlerFunc {
    return func(w http.ResponseWriter, r *http.Request) {
        // ... parsing dan validasi ...
        
        // Panggil service
        token, err := authService.Login(r.Context(), req.Email, req.Password)
        if err != nil {
            dim.Unauthorized(w, "Login gagal")
            return
        }
        
        dim.OK(w, map[string]string{"token": token})
    }
}
```

### Penggunaan di Router
```go
authService := dim.NewAuthService(...)
router.Post("/auth/login", LoginHandler(authService))
```

### Keuntungan
✅ **Clean Separation**: Handler fokus pada HTTP, service fokus pada business logic.
✅ **Testable**: Mudah untuk me-mock service saat pengujian handler.
✅ **Reusable**: Service dapat digunakan oleh beberapa handler.

---

## Struct-based Handler Pattern

**Pattern untuk Aplikasi Medium-Large dengan Banyak Endpoint**

Handler diorganisir sebagai method dari sebuah struct, yang menyimpan semua dependency.

### Struktur Dasar

```go
// Handler struct dengan dependencies
type UserHandler struct {
    userService  *UserService
    logger       *slog.Logger
}

// Constructor
func NewUserHandler(userService *UserService, logger *slog.Logger) *UserHandler {
    return &UserHandler{
        userService:  userService,
        logger:       logger,
    }
}

// Method-method ini secara langsung menjadi http.HandlerFunc
func (h *UserHandler) GetProfile(w http.ResponseWriter, r *http.Request) {
    user, ok := dim.GetUser(r)
    if !ok {
        dim.Unauthorized(w, "Tidak terautentikasi")
        return
    }
    
    // Gunakan dependency dari struct
    profile, err := h.userService.GetProfile(r.Context(), user.ID)
    if err != nil {
        h.logger.Error("Gagal mengambil profil", "error", err)
        dim.NotFound(w, "Profil tidak ditemukan")
        return
    }
    
    dim.OK(w, profile)
}

func (h *UserHandler) UpdateProfile(w http.ResponseWriter, r *http.Request) {
    // ... logika untuk update profil ...
    // h.userService.Update(...)
}
```

### Penggunaan di Router

```go
func main() {
    // Setup dependencies
    userService := dim.NewUserService(userStore)
    jwtManager := dim.NewJWTManager(cfg.JWT)
    
    // Buat instance handler
    userHandler := NewUserHandler(userService, logger)
    
    // Setup router
    router := dim.NewRouter()
    
    // Daftarkan method handler secara langsung
    api := router.Group("/api", dim.RequireAuth(jwtManager))
    api.Get("/profile", userHandler.GetProfile)
    api.Put("/profile", userHandler.UpdateProfile)
}
```

### Keuntungan
✅ **Terorganisir**: Semua handler yang terkait berada dalam satu struct.
✅ **Reusable Dependencies**: Berbagi dependency antar method handler.
✅ **Scalable**: Mudah untuk menambah method handler baru.

---

## Direct Handler Pattern

**Pattern untuk Endpoint Sederhana**

Fungsi handler didefinisikan secara langsung tanpa struct atau factory.

```go
// Handler langsung
func healthHandler(w http.ResponseWriter, r *http.Request) {
    dim.OK(w, map[string]string{"status": "sehat"})
}

// Penggunaan di Router
router.Get("/health", healthHandler)
```

### Keuntungan
✅ **Sederhana**: Sangat simpel untuk endpoint tanpa dependency.
✅ **Ringan**: Kode boilerplate minimal.

---

(Sisa dokumen tidak perlu diubah dan dihilangkan dari sini untuk keringkasan)