# Validasi di Framework dim

Pelajari cara melakukan validasi input dengan validator framework dim.

## Daftar Isi

- [Konsep Validasi](#konsep-validasi)
- [Validator API](#validator-api)
- [Validasi Basic](#validasi-basic)
- [Validasi Email](#validasi-email)
- [Validasi Length](#validasi-length)
- [Custom Validasi](#custom-validasi)
- [Validasi Nested](#validasi-nested)
- [Error Messages](#error-messages)
- [Praktik Terbaik](#best-practices)

---

## Konsep Validasi

### Filosofi Validasi di dim

Framework dim menggunakan model **single-error-per-field**:

```json
{
  "message": "Validation failed",
  "errors": {
    "email": "Invalid email format",
    "password": "Password too weak"
  }
}
```

Bukan multiple errors per field:
```json
{
  "errors": {
    "email": [
      "Email is required",
      "Invalid email format"
    ]
  }
}
```

### Validation Flow

```
Request Input
  ↓
Validator
  ├─ Required check
  ├─ Format check (email, etc)
  ├─ Length check
  ├─ Custom rules
  └─ Collect errors (1 per field)
  ↓
IsValid?
  ├─ YES → Continue
  └─ NO → Return errors
  ↓
Response
```

### Kapan Validasi?

```
┌─────────────────────────────────────────┐
│ Handler menerima request                │
├─────────────────────────────────────────┤
│ 1. Parse JSON                           │
│ 2. Validate input ← Di sini!            │
│ 3. Call service                         │
│ 4. Return response                      │
└─────────────────────────────────────────┘
```

---

## Validator API

### Create Validator

```go
v := dim.NewValidator()
```

### Add Validations

```go
v.Required("email", email)
v.Email("email", email)
v.MinLength("username", username, 3)
v.MaxLength("bio", bio, 500)
```

### Check if Valid

```go
if !v.IsValid() {
    dim.JsonError(w, http.StatusBadRequest, "Validation failed", v.Errors())
    return
}
```

### Get Errors

```go
errors := v.Errors()
// errors = {"email": "Invalid email format", ...}
```

---

## Validasi Basic

### Required

Field harus tidak kosong:

```go
v := dim.NewValidator()
v.Required("email", req.Email)
v.Required("username", req.Username)
v.Required("password", req.Password)

if !v.IsValid() {
    dim.JsonError(w, 400, "Validation failed", v.Errors())
    return
}
```

**Output jika error**:
```json
{
  "message": "Validation failed",
  "errors": {
    "email": "email wajib diisi"
  }
}
```

### NotEmpty

Alias untuk Required (sama):

```go
v.NotEmpty("field_name", value)
```

---

## Validasi Email

### Email Format

```go
v := dim.NewValidator()
v.Email("email", req.Email)

if !v.IsValid() {
    dim.JsonError(w, 400, "Validation failed", v.Errors())
    return
}
```

**Valid emails**:
```
user@example.com
john.doe@example.co.uk
test+tag@domain.org
```

**Invalid emails**:
```
invalid.email
@example.com
user@
user@.com
```

**Pesan error**:
```json
{
  "errors": {
    "email": "email harus berupa alamat email yang valid"
  }
}
```

### Combine Required + Email

```go
v := dim.NewValidator()
v.Required("email", req.Email)     // Check not empty
v.Email("email", req.Email)         // Check format

// Akan set error jika salah satu gagal
if !v.IsValid() {
    dim.JsonError(w, 400, "Validation failed", v.Errors())
    return
}
```

---

### Validasi Length

### MinLength

Memvalidasi panjang string minimal.

```go
v := dim.NewValidator()
v.MinLength("username", req.Username, 3)
v.MinLength("password", req.Password, 8)
```

**Pesan error**:
```json
{
  "errors": {
    "username": "username harus minimal 3 karakter",
    "password": "password harus minimal 8 karakter"
  }
}
```

### MaxLength

Memvalidasi panjang string maksimal.

```go
v := dim.NewValidator()
v.MaxLength("bio", req.Bio, 500)
```

**Pesan error**:
```json
{
  "errors": {
    "bio": "bio tidak boleh melebihi 500 karakter"
  }
}
```

---

## Validasi Lanjutan

Validator juga menyediakan aturan-aturan lain yang lebih spesifik.

### Length

Memvalidasi panjang string harus **tepat** sama dengan nilai yang ditentukan.

```go
v.Length("kode_verifikasi", req.VerificationCode, 6)
```
**Pesan error**: `"kode_verifikasi harus tepat 6 karakter"`

### Pattern

Memvalidasi string berdasarkan *regular expression* (regex).

```go
// Memvalidasi bahwa 'slug' hanya berisi huruf kecil, angka, dan tanda hubung
v.Pattern("slug", req.Slug, `^[a-z0-9]+(?:-[a-z0-9]+)*$`)
```
**Pesan error**: `"format slug tidak valid"`

### In

Memvalidasi bahwa sebuah nilai harus ada di dalam daftar nilai yang diizinkan (seperti enum).

```go
v.In("role", req.Role, "admin", "user", "guest")
```
**Pesan error**: `"role memiliki nilai yang tidak valid"`

### NumRange

Memvalidasi bahwa sebuah angka berada dalam rentang (inklusif).

```go
v.NumRange("age", req.Age, 18, 65)
```
**Pesan error**: `"age harus antara 18 dan 65"`

### Matches

Memvalidasi bahwa dua buah nilai string sama persis. Berguna untuk konfirmasi *password*.

```go
v.Matches("password", req.Password, "password_confirm", req.PasswordConfirm)
```
**Pesan error**: `"password tidak cocok dengan password_confirm"`

---

## Custom Validasi

### Simple Custom Validation

```go
v := dim.NewValidator()

// Match two fields
v.Custom("password_confirm", func() bool {
    return req.Password == req.PasswordConfirm
}, "Password tidak cocok")

// Check if age is adult
v.Custom("age", func() bool {
    age, _ := strconv.Atoi(req.Age)
    return age >= 18
}, "Harus berusia 18 tahun atau lebih")

// Check unique username (call database)
v.Custom("username", func() bool {
    exists, _ := userStore.UsernameExists(ctx, req.Username)
    return !exists
}, "Username sudah digunakan")
```

### Complex Custom Validation

```go
v := dim.NewValidator()

// Strong password validation
v.Custom("password", func() bool {
    pwd := req.Password
    
    // Minimal 8 chars
    if len(pwd) < 8 {
        return false
    }
    
    // Has uppercase
    hasUpper := false
    for _, c := range pwd {
        if unicode.IsUpper(c) {
            hasUpper = true
            break
        }
    }
    
    // Has lowercase
    hasLower := false
    for _, c := range pwd {
        if unicode.IsLower(c) {
            hasLower = true
            break
        }
    }
    
    // Has digit
    hasDigit := false
    for _, c := range pwd {
        if unicode.IsDigit(c) {
            hasDigit = true
            break
        }
    }
    
    return hasUpper && hasLower && hasDigit
}, "Password harus mengandung huruf besar, huruf kecil, dan angka")
```

### Conditional Validation

```go
v := dim.NewValidator()

// Validasi conditional
if req.AccountType == "business" {
    v.Required("company_name", req.CompanyName)
    v.Required("tax_id", req.TaxID)
}

if req.NotifyViaEmail {
    v.Email("notification_email", req.NotificationEmail)
}
```

---

## Validasi Nested

### Struct Validation

```go
type CreateUserRequest struct {
    Email    string `json:"email"`
    Username string `json:"username"`
    Password string `json:"password"`
    Profile  struct {
        FirstName string `json:"first_name"`
        LastName  string `json:"last_name"`
        Bio       string `json:"bio"`
    } `json:"profile"`
}

func createUserHandler(w http.ResponseWriter, r *http.Request) {
    var req CreateUserRequest
    json.NewDecoder(r.Body).Decode(&req)
    
    v := dim.NewValidator()
    
    // Top-level fields
    v.Required("email", req.Email)
    v.Email("email", req.Email)
    
    // Nested fields
    v.Required("profile.first_name", req.Profile.FirstName)
    v.Required("profile.last_name", req.Profile.LastName)
    v.MaxLength("profile.bio", req.Profile.Bio, 500)
    
    if !v.IsValid() {
        dim.JsonError(w, 400, "Validation failed", v.Errors())
        return
    }
}
```

**Error response**:
```json
{
  "message": "Validation failed",
  "errors": {
    "email": "email harus berupa alamat email yang valid",
    "profile.first_name": "profile.first_name wajib diisi"
  }
}
```

### Array Validation

```go
type CreatePostRequest struct {
    Title string   `json:"title"`
    Tags  []string `json:"tags"`
}

func createPostHandler(w http.ResponseWriter, r *http.Request) {
    var req CreatePostRequest
    json.NewDecoder(r.Body).Decode(&req)
    
    v := dim.NewValidator()
    
    v.Required("title", req.Title)
    
    // Validate array length
    if len(req.Tags) == 0 {
        v.AddError("tags", "Minimal satu tag diperlukan")
    }
    
    if len(req.Tags) > 10 {
        v.AddError("tags", "Maksimal 10 tag diizinkan")
    }
    
    // Validate each tag
    for i, tag := range req.Tags {
        if len(tag) < 2 {
            v.AddError("tags", "Setiap tag harus minimal 2 karakter")
            break
        }
    }
    
    if !v.IsValid() {
        dim.JsonError(w, 400, "Validation failed", v.Errors())
        return
    }
}
```

---

## Validasi Field Opsional (Nullable)

Saat membuat endpoint `PATCH`, seringkali Anda perlu membedakan antara *field* yang tidak dikirim, *field* yang dikirim dengan nilai `null`, dan *field* yang dikirim dengan nilai. Tipe `dim.JsonNull[T]` dirancang untuk menangani kasus ini, dan *validator* memiliki metode khusus untuknya.

Aturan validasi `Optional*` hanya akan dijalankan jika *field* `JsonNull[T]` ada di dalam JSON *payload* **dan** nilainya tidak `null`.

### Metode Opsional yang Tersedia

-   `OptionalEmail`
-   `OptionalMinLength`
-   `OptionalMaxLength`
-   `OptionalLength`
-   `OptionalIn`
-   `OptionalMatches` (untuk regex)

### Contoh Penggunaan

Misalkan Anda memiliki *endpoint* untuk memperbarui profil pengguna:

```go
// Tipe JsonNull digunakan untuk field yang bisa null
type UpdateProfileRequest struct {
	Name     dim.JsonNull[string] `json:"name"`
	Email    dim.JsonNull[string] `json:"email"`
	Status   dim.JsonNull[string] `json:"status"`
}

func updateUserHandler(w http.ResponseWriter, r *http.Request) {
    var req UpdateProfileRequest
    if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
        dim.JsonError(w, http.StatusBadRequest, "Invalid JSON", nil)
        return
    }

    v := dim.NewValidator()

    // Validasi hanya jika field "name" ada dan tidak null
    v.OptionalMinLength("name", req.Name, 2)
    v.OptionalMaxLength("name", req.Name, 100)

    // Validasi email hanya jika field "email" ada dan tidak null
    v.OptionalEmail("email", req.Email)
    
    // Validasi enum hanya jika field "status" ada dan tidak null
    v.OptionalIn("status", req.Status, "active", "inactive")

    if !v.IsValid() {
        dim.JsonError(w, http.StatusBadRequest, "Validation failed", v.Errors())
        return
    }
    
    // ... lanjutkan proses update ...
}
```

Dengan `UpdateProfileRequest` di atas:
-   `{}` (JSON kosong): Tidak ada validasi yang berjalan.
-   `{"name": null}`: Tidak ada validasi yang berjalan untuk `name`.
-   `{"name": "X"}`: Validasi `OptionalMinLength` akan gagal untuk `name`.
-   `{"email": "user@example.com"}`: Validasi `OptionalEmail` akan berjalan.
-   `{"email": "invalid"}`: Validasi `OptionalEmail` akan gagal.

---

## Error Messages

### Custom Error Messages

```go
v := dim.NewValidator()

v.Custom("email", func() bool {
    return isValidEmail(req.Email)
}, "Please enter a valid email address")  // Custom message

v.Custom("age", func() bool {
    age, _ := strconv.Atoi(req.Age)
    return age >= 18
}, "You must be at least 18 years old to register")
```

### Localized Messages

```go
// messages_en.go
var ValidationMessages = map[string]map[string]string{
    "email": {
        "required": "Email is required",
        "invalid": "Invalid email format",
    },
    "password": {
        "required": "Password is required",
        "too_short": "Password must be at least 8 characters",
    },
}

// messages_id.go
var ValidationMessageID = map[string]map[string]string{
    "email": {
        "required": "Email diperlukan",
        "invalid": "Format email tidak valid",
    },
    "password": {
        "required": "Password diperlukan",
        "too_short": "Password minimal 8 karakter",
    },
}

// Usage
func getErrorMessage(field, errorType, locale string) string {
    messages := ValidationMessages
    if locale == "id" {
        messages = ValidationMessageID
    }
    return messages[field][errorType]
}
```

---

## Complete Validation Example

### Registration Handler

```go
func registerHandler(authService *AuthService) http.HandlerFunc {
    return func(w http.ResponseWriter, r *http.Request) {
        // 1. Parse request
        var req struct {
            Email            string `json:"email"`
            Username         string `json:"username"`
            Password         string `json:"password"`
            PasswordConfirm  string `json:"password_confirm"`
        }
        
        if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
            dim.JsonError(w, http.StatusBadRequest, "Invalid JSON", nil)
            return
        }
        
        // 2. Validate input
        v := dim.NewValidator()
        
        // Email validation
        v.Required("email", req.Email)
        v.Email("email", req.Email)
        
        // Username validation
        v.Required("username", req.Username)
        v.MinLength("username", req.Username, 3)
        v.MaxLength("username", req.Username, 50)
        
        // Username unique check (custom)
        v.Custom("username", func() bool {
            exists, _ := authService.UsernameExists(r.Context(), req.Username)
            return !exists
        }, "Username sudah digunakan")
        
        // Password validation
        v.Required("password", req.Password)
        v.MinLength("password", req.Password, 8)
        
        v.Custom("password", func() bool {
            pwd := req.Password
            hasUpper := false
            hasDigit := false
            
            for _, c := range pwd {
                if unicode.IsUpper(c) {
                    hasUpper = true
                }
                if unicode.IsDigit(c) {
                    hasDigit = true
                }
            }
            
            return hasUpper && hasDigit
        }, "Password harus mengandung huruf besar dan angka")
        
        // Password confirm validation
        v.Custom("password_confirm", func() bool {
            return req.Password == req.PasswordConfirm
        }, "Password tidak cocok")
        
        // Check if valid
        if !v.IsValid() {
            dim.JsonError(w, http.StatusBadRequest, "Validasi gagal", v.Errors())
            return
        }
        
        // 3. All validations passed, proceed with business logic
        user, err := authService.Register(r.Context(), 
            req.Email, req.Username, req.Password)
        
        if err != nil {
            logger.Error("Gagal melakukan registrasi", "error", err)
            dim.JsonError(w, http.StatusInternalServerError, "Gagal melakukan registrasi", nil)
            return
        }
        
        // 4. Return response
        dim.Json(w, http.StatusCreated, map[string]interface{}{
            "id":       user.ID,
            "email":    user.Email,
            "username": user.Username,
        })
    }
}
```

### Test Invalid Input

```bash
# Missing email
curl -X POST http://localhost:8080/auth/register \
  -H "Content-Type: application/json" \
  -d '{"username": "john", "password": "SecurePass123"}'

# Response:
# {
#   "message": "Validasi gagal",
#   "errors": {
#     "email": "email wajib diisi"
#   }
# }

# Invalid email
curl -X POST http://localhost:8080/auth/register \
  -H "Content-Type: application/json" \
  -d '{"email": "invalid", "username": "john", "password": "SecurePass123"}'

# Response:
# {
#   "message": "Validasi gagal",
#   "errors": {
#     "email": "email harus berupa alamat email yang valid"
#   }
# }

# Passwords don't match
curl -X POST http://localhost:8080/auth/register \
  -H "Content-Type: application/json" \
  -d '{
    "email": "john@example.com",
    "username": "john",
    "password": "SecurePass123",
    "password_confirm": "DifferentPass123"
  }'

# Response:
# {
#   "message": "Validasi gagal",
#   "errors": {
#     "password_confirm": "Password tidak cocok"
#   }
# }
```

---

## Praktik Terbaik

### ✅ DO: Validate Early

```go
// ✅ BAIK - Validasi segera setelah parse
func handler(w http.ResponseWriter, r *http.Request) {
    var req CreateUserRequest
    json.NewDecoder(r.Body).Decode(&req)
    
    v := dim.NewValidator()
    v.Required("email", req.Email)
    // ... validasi lainnya ...
    
    if !v.IsValid() {
        dim.JsonError(w, 400, "Validasi gagal", v.Errors())
        return
    }
    
    // Logika bisnis setelah validasi lolos
}

// ❌ BURUK - Validasi di dalam logika bisnis
func handler(w http.ResponseWriter, r *http.Request) {
    var req CreateUserRequest
    json.NewDecoder(r.Body).Decode(&req)
    
    // Logika kompleks terlebih dahulu
    user, _ := userStore.FindByEmail(req.Email)
    
    // Validasi terlambat
    if req.Email == "" {
        dim.JsonError(w, 400, "Email wajib diisi", nil)
        return
    }
}
```

### ✅ DO: Use Consistent Error Keys

```go
// ✅ BAIK - Nama field yang konsisten
v.AddError("email", "Email tidak valid")
v.AddError("username", "Sudah digunakan")

// ✅ BAIK - Field bersarang dengan notasi titik
v.AddError("profile.first_name", "Wajib diisi")
v.AddError("address.street", "Terlalu pendek")

// ❌ BURUK - Nama tidak konsisten
v.AddError("user_email", "Tidak valid")
v.AddError("name_first", "Wajib diisi")
```

### ✅ DO: Make Error Messages User-Friendly

```go
// ✅ BAIK - Pesan yang ramah pengguna
v.Custom("age", func() bool {
    age, _ := strconv.Atoi(req.Age)
    return age >= 18
}, "Anda harus berusia minimal 18 tahun untuk mendaftar")

// ❌ BURUK - Pesan teknis
v.Custom("age", func() bool {
    age, _ := strconv.Atoi(req.Age)
    return age >= 18
}, "age < 18")
```

### ✅ DO: Validate at Boundaries

```go
// ✅ BAIK - Validasi input eksternal
func handler(w http.ResponseWriter, r *http.Request) {
    var req CreateUserRequest
    json.NewDecoder(r.Body).Decode(&req)  // Input eksternal
    
    v := dim.NewValidator()
    v.Required("email", req.Email)  // Validasi input eksternal
    // ...
}

// ❌ BURUK - Validasi data internal
func handler(w http.ResponseWriter, r *http.Request) {
    // Data internal dari database
    user := getUserFromDB()
    
    // Tidak perlu validasi (sudah tepercaya)
    v := dim.NewValidator()
    v.Required("email", user.Email)  // Tidak perlu
}
```

### ✅ DO: Create Reusable Validators

```go
// ✅ BAIK - Reusable validation functions
func ValidateEmail(v *Validator, email string) {
    v.Required("email", email)
    v.Email("email", email)
}

func ValidateUsername(v *Validator, username string) {
    v.Required("username", username)
    v.MinLength("username", username, 3)
    v.MaxLength("username", username, 50)
}

func ValidatePassword(v *Validator, password string) {
    v.Required("password", password)
    v.MinLength("password", password, 8)
}

// Usage
func registerHandler(w http.ResponseWriter, r *http.Request) {
    var req CreateUserRequest
    json.NewDecoder(r.Body).Decode(&req)
    
    v := dim.NewValidator()
    
    ValidateEmail(v, req.Email)
    ValidateUsername(v, req.Username)
    ValidatePassword(v, req.Password)
    
    if !v.IsValid() {
        dim.JsonError(w, 400, "Validation failed", v.Errors())
        return
    }
}
```

---

## Summary

Validasi di dim:
- **Simple** - Clean validator API
- **Single-error-per-field** - Not multiple errors per field
- **Flexible** - Custom validation rules
- **Reusable** - Extract common validations
- **User-friendly** - Clear error messages

Lihat [Error Handling](08-error-handling.md) untuk detail error handling.

---

**Lihat Juga**:
- [Error Handling](08-error-handling.md) - Error response formatting
- [Autentikasi](05-authentication.md) - Validation di auth handlers
- [Request Context](10-request-context.md) - Access validated data
