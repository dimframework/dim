# Security Best Practices di Framework dim

Pelajari cara membangun aplikasi yang aman dengan framework dim.

## Daftar Isi

- [Security Principles](#security-principles)
- [Authentication Security](#authentication-security)
- [Password Security](#password-security)
- [Token Security](#token-security)
- [Input Validation](#input-validation)
- [CORS Security](#cors-security)
- [CSRF Protection](#csrf-protection)
- [Rate Limiting](#rate-limiting)
- [SQL Injection Prevention](#sql-injection-prevention)
- [Sensitive Data](#sensitive-data)
- [HTTPS/TLS](#httpstls)
- [Security Headers](#security-headers)
- [Dependency Security](#dependency-security)

---

## Security Principles

### Defense in Depth

Implementasi multiple security layers:

```
┌─────────────────────────────────┐
│ HTTPS/TLS                       │
├─────────────────────────────────┤
│ Rate Limiting                   │
├─────────────────────────────────┤
│ Input Validation & Sanitization │
├─────────────────────────────────┤
│ Authentication & Authorization  │
├─────────────────────────────────┤
│ Business Logic Validation       │
├─────────────────────────────────┤
│ Logging & Monitoring            │
└─────────────────────────────────┘
```

### Security Mindset

1. **Never trust user input** - Validate everything
2. **Principle of least privilege** - Grant minimum needed access
3. **Fail securely** - Fail closed, not open
4. **Defense in depth** - Multiple security layers
5. **Keep it simple** - Simpler code = fewer bugs
6. **Log and monitor** - Detect threats early

---

## Authentication Security

### ✅ DO: Use Strong Authentication

```go
// ✅ BAIK - JWT dengan secure secret
cfg, _ := dim.LoadConfig()

// Secret dari environment (strong, random)
authService := dim.NewAuthService(userStore, tokenStore, emailSender, &cfg.JWT)

// Verify token
claims, err := verifyJWTToken(token)
if err != nil {
    dim.JsonError(w, 401, "Unauthorized", nil)
    return
}
```

### ✅ DO: Implement Proper Session Handling

```go
// ✅ BAIK - Access token + Refresh token
func loginHandler(authService *AuthService) http.HandlerFunc {
    return func(w http.ResponseWriter, r *http.Request) {
        access, refresh, err := authService.Login(r.Context(), email, password)
        if err != nil {
            dim.JsonError(w, 401, "Invalid credentials", nil)
            return
        }
        
        dim.Json(w, 200, map[string]interface{}{
            "access_token": access,      // Short-lived (15m)
            "refresh_token": refresh,    // Long-lived (7d)
        })
    }
}
```

### ❌ DON'T: Hardcode Credentials

```go
// ❌ BURUK - Hardcoded
const AdminPassword = "admin123"

// ✅ BAIK - From environment
password := os.Getenv("ADMIN_PASSWORD")
```

### ✅ DO: Rate Limit Login Attempts

```go
// ✅ BAIK - Rate limit login endpoint
sensitive := router.Group("/auth", dim.RateLimit(RateLimitConfig{
    PerIP: 5,         // 5 attempts per IP
    PerUser: 10,      // 10 per user
    ResetPeriod: time.Hour,
}))

sensitive.Post("/login", loginHandler)
```

---

## Password Security

### ✅ DO: Hash Passwords Securely

```go
import "golang.org/x/crypto/bcrypt"

// ✅ BAIK - Bcrypt dengan default cost
func hashPassword(password string) (string, error) {
    // Cost 10+ recommended (bcrypt default)
    return bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
}

func verifyPassword(hash, password string) error {
    return bcrypt.CompareHashAndPassword([]byte(hash), []byte(password))
}
```

### ❌ DON'T: Weak Password Hashing

```go
// ❌ BURUK - MD5 (broken)
hash := md5.Sum([]byte(password))

// ❌ BURUK - SHA (no salt)
hash := sha256.Sum256([]byte(password))

// ❌ BURUK - Plain text (NEVER!)
passwordHash := password
```

### ✅ DO: Enforce Strong Passwords

Gunakan `PasswordValidator` bawaan untuk validasi kekuatan password yang mudah dan dapat dikonfigurasi.

```go
import "github.com/dimframework/dim"

// Cara 1: Gunakan validasi default (cara termudah)
// (Minimal 8 karakter, 1 huruf besar, 1 huruf kecil, 1 angka, 1 simbol)
err := dim.ValidatePasswordStrength(req.Password)
if err != nil {
    dim.JsonError(w, 400, "Password terlalu lemah", err.(*dim.AppError).Errors)
    return
}

// Cara 2: Konfigurasi validator secara kustom
validator := dim.NewPasswordValidator().
    SetMinLength(10).          // Minimal 10 karakter
    RequireUppercase(true).    // Wajib ada huruf besar
    RequireSpecial(false)      // Tidak wajib ada karakter spesial

err = validator.Validate(req.Password)
if err != nil {
    dim.JsonError(w, 400, "Password tidak memenuhi syarat", err.(*dim.AppError).Errors)
    return
}
```

Ini jauh lebih baik daripada membuat fungsi validasi manual yang kompleks.
`ValidatePasswordStrength` akan mengembalikan `AppError` dengan pesan yang jelas jika validasi gagal, misalnya:
`{"password": "Kata sandi harus mengandung minimal satu huruf besar"}`.


### ✅ DO: Implement Secure Password Reset

```go
// ✅ BAIK - Secure password reset flow
func forgotPasswordHandler(authService *AuthService) http.HandlerFunc {
    return func(w http.ResponseWriter, r *http.Request) {
        var req struct{ Email string }
        json.NewDecoder(r.Body).Decode(&req)
        
        // Don't reveal if email exists (timing attack resistant)
        _ = authService.RequestPasswordReset(r.Context(), req.Email)
        
        // Same response regardless
        dim.Json(w, 200, map[string]string{
            "message": "If email exists, check inbox",
        })
    }
}

func resetPasswordHandler(authService *AuthService) http.HandlerFunc {
    return func(w http.ResponseWriter, r *http.Request) {
        var req struct {
            Token    string `json:"token"`
            Password string `json:"password"`
        }
        json.NewDecoder(r.Body).Decode(&req)
        
        // Validate password strength
        if err := validatePasswordStrength(req.Password); err != nil {
            dim.JsonError(w, 400, "Password invalid", map[string]string{
                "password": err.Error(),
            })
            return
        }
        
        // Reset (token expires, one-use only)
        err := authService.ResetPassword(r.Context(), req.Token, req.Password)
        if err != nil {
            dim.JsonError(w, 400, "Reset failed", nil)
            return
        }
        
        dim.Json(w, 200, map[string]string{
            "message": "Password reset successfully",
        })
    }
}
```

---

## Token Security

### ✅ DO: Use Strong JWT Secret

```go
// ✅ BAIK - Random 256-bit secret
// Generate with: openssl rand -hex 32
JWT_SECRET=a1b2c3d4e5f6g7h8i9j0k1l2m3n4o5p6

// In code
cfg, _ := dim.LoadConfig()
secret := cfg.JWT.Secret  // From environment

if len(secret) < 32 {
    log.Fatal("JWT_SECRET too short")
}
```

### ✅ DO: Rotate Tokens

```go
// ✅ BAIK - Refresh token rotation
func (s *AuthService) RefreshToken(ctx context.Context, oldRefresh string) (string, string, error) {
    // 1. Verify old token
    claims, err := verifyRefreshToken(oldRefresh)
    if err != nil {
        return "", "", err
    }
    
    // 2. Generate new tokens
    newAccess, _ := s.generateAccessToken(claims.UserID)
    newRefresh, _ := s.generateRefreshToken(claims.UserID)
    
    // 3. Revoke old token (blacklist it)
    s.tokenStore.RevokeRefreshToken(ctx, oldRefresh)
    
    // 4. Store new refresh token
    s.tokenStore.SaveRefreshToken(ctx, newRefresh)
    
    return newAccess, newRefresh, nil
}
```

### ✅ DO: Implement Token Expiry

```go
// ✅ BAIK - Short-lived access token
JWT_ACCESS_TOKEN_EXPIRY=15m    // 15 minutes

// Long-lived refresh token
JWT_REFRESH_TOKEN_EXPIRY=7d    // 7 days

// Verify expiry in middleware
func (s *AuthService) VerifyToken(tokenString string) (*Claims, error) {
    claims := &Claims{}
    
    token, err := jwt.ParseWithClaims(tokenString, claims, func(token *jwt.Token) (interface{}, error) {
        return []byte(s.secret), nil
    })
    
    if err != nil {
        return nil, err
    }
    
    if !token.Valid {
        return nil, errors.New("invalid token")
    }
    
    // Check expiry
    if claims.ExpiresAt.Before(time.Now()) {
        return nil, errors.New("token expired")
    }
    
    return claims, nil
}
```

### ✅ DO: Implement Token Blacklist

```go
// ✅ BAIK - Blacklist for logout
type TokenStore interface {
    RevokeRefreshToken(ctx context.Context, tokenHash string) error
    IsTokenBlacklisted(ctx context.Context, tokenHash string) bool
}

func (s *AuthService) Logout(ctx context.Context, userID int64, refreshToken string) error {
    // Hash token (don't store plain token)
    hash := hashToken(refreshToken)
    
    // Blacklist
    return s.tokenStore.RevokeRefreshToken(ctx, hash)
}

func (s *AuthService) VerifyToken(tokenString string) (*Claims, error) {
    // ... parse token ...
    
    // Check if blacklisted
    if s.tokenStore.IsTokenBlacklisted(context.Background(), hashToken(tokenString)) {
        return nil, errors.New("token revoked")
    }
    
    return claims, nil
}
```

---

## Input Validation

### ✅ DO: Validate All Input

```go
// ✅ BAIK - Comprehensive validation
func registerHandler(authService *AuthService) http.HandlerFunc {
    return func(w http.ResponseWriter, r *http.Request) {
        var req struct {
            Email    string `json:"email"`
            Username string `json:"username"`
            Password string `json:"password"`
        }
        
        if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
            dim.JsonError(w, 400, "Invalid JSON", nil)
            return
        }
        
        // Validate all fields
        v := dim.NewValidator()
        v.Required("email", req.Email)
        v.Email("email", req.Email)
        v.Required("username", req.Username)
        v.MinLength("username", req.Username, 3)
        v.MaxLength("username", req.Username, 50)
        v.Required("password", req.Password)
        v.MinLength("password", req.Password, 8)
        
        if !v.IsValid() {
            dim.JsonError(w, 400, "Validation failed", v.Errors())
            return
        }
        
        // Proceed
    }
}
```

### ✅ DO: Use Parameterized Queries

```go
// ✅ BAIK - Parameterized query (prevents SQL injection)
db.QueryRow(ctx,
    "SELECT * FROM users WHERE email = $1",
    email,  // Separate parameter
)

// ❌ BURUK - String concatenation (SQL injection risk!)
query := fmt.Sprintf("SELECT * FROM users WHERE email = '%s'", email)
db.QueryRow(ctx, query)
```

### ✅ DO: Sanitize User Input

```go
// ✅ BAIK - HTML escape untuk display
import "html"

func displayUserComment(comment string) string {
    return html.EscapeString(comment)
}

// ✅ BAIK - Trim whitespace
username := strings.TrimSpace(req.Username)

// ✅ BAIK - Lowercase email untuk consistency
email := strings.ToLower(req.Email)
```

---

## CORS Security

### ✅ DO: Configure CORS Properly

```go
// ✅ BAIK - Specific origins
CORS_ALLOWED_ORIGINS=https://example.com,https://app.example.com
CORS_ALLOWED_METHODS=GET,POST,PUT,DELETE,PATCH
CORS_ALLOWED_HEADERS=Content-Type,Authorization
CORS_ALLOW_CREDENTIALS=true

// ❌ BURUK - Allow all origins
CORS_ALLOWED_ORIGINS=*

// ❌ BURUK - Allow all methods
CORS_ALLOWED_METHODS=*
```

### ✅ DO: Disable Credentials if Not Needed

```
CORS_ALLOW_CREDENTIALS=false  // If not using cookies/auth
```

---

## CSRF Protection

### ✅ DO: Enable CSRF Protection

```go
// ✅ BAIK - Enable CSRF
CSRF_ENABLED=true

router.Use(dim.CSRF(cfg.CSRF))
```

### ✅ DO: Include Token in Forms

```html
<!-- ✅ BAIK - Include CSRF token -->
<form method="POST" action="/users">
    <input type="hidden" name="_csrf" value="<token>">
    <input type="text" name="email">
    <button>Submit</button>
</form>
```

### ✅ DO: Send Token in API Requests

```javascript
// ✅ BAIK - Include CSRF token in header
fetch('/api/users', {
    method: 'POST',
    headers: {
        'X-CSRF-Token': getCookie('X-CSRF-Token'),
        'Content-Type': 'application/json'
    },
    body: JSON.stringify({email: 'user@example.com'})
})
```

---

## Rate Limiting

### ✅ DO: Implement Rate Limiting

```go
// ✅ BAIK - Rate limit sensitive endpoints
RATE_LIMIT_ENABLED=true
RATE_LIMIT_PER_IP=100         // 100 requests per IP
RATE_LIMIT_PER_USER=200       // 200 per authenticated user
RATE_LIMIT_RESET_PERIOD=1h

// Apply to sensitive routes
auth := router.Group("/auth", dim.RateLimit(cfg.RateLimit))
auth.Post("/login", loginHandler)
auth.Post("/register", registerHandler)
```

---

## SQL Injection Prevention

### ✅ DO: Use Parameterized Queries

```go
// ✅ BAIK
db.QueryRow(ctx,
    "SELECT * FROM users WHERE email = $1 AND status = $2",
    email,
    "active",
)

// ❌ BURUK
query := fmt.Sprintf("SELECT * FROM users WHERE email = '%s'", email)
```

---

## Sensitive Data

### ✅ DO: Never Log Passwords/Tokens

```go
// ✅ BAIK - No sensitive data
logger.Info("User login",
    "user_id", userID,
    "method", "email",
)

// ❌ BURUK
logger.Info("User login",
    "password", password,  // NEVER!
    "token", token,        // NEVER!
)
```

### ✅ DO: Use HTTPS in Production

```
❌ BURUK:  http://example.com
✅ BAIK:   https://example.com
```

### ✅ DO: Store Secrets in Environment Variables

```bash
# ✅ BAIK - Environment variables
JWT_SECRET=from-environment
DB_PASSWORD=from-environment

# ❌ BURUK - Hardcoded
const JWTSecret = "hardcoded"
const DBPassword = "hardcoded"
```

---

## HTTPS/TLS

### ✅ DO: Enforce HTTPS

```go
// ✅ BAIK - Middleware to enforce HTTPS
func enforceHTTPSMiddleware(next HandlerFunc) HandlerFunc {
    return func(w http.ResponseWriter, r *http.Request) {
        if r.Header.Get("X-Forwarded-Proto") == "http" {
            http.Redirect(w, r, "https://"+r.Host+r.RequestURI, http.StatusMovedPermanently)
            return
        }
        next(w, r)
    }
}

router.Use(enforceHTTPSMiddleware)
```

### ✅ DO: Use Strong TLS Configuration

```go
server := &http.Server{
    Addr:    ":443",
    Handler: router,
    TLSConfig: &tls.Config{
        MinVersion:               tls.VersionTLS12,
        CurvePreferences:         []tls.CurveID{tls.CurveP521, tls.CurveP384, tls.CurveP256},
        PreferServerCipherSuites: true,
        CipherSuites: []uint16{
            tls.TLS_AES_256_GCM_SHA384,
            tls.TLS_AES_128_GCM_SHA256,
            tls.TLS_CHACHA20_POLY1305_SHA256,
        },
    },
}

log.Fatal(server.ListenAndServeTLS("cert.pem", "key.pem"))
```

---

## Security Headers

### ✅ DO: Set Security Headers

```go
func securityHeadersMiddleware(next HandlerFunc) HandlerFunc {
    return func(w http.ResponseWriter, r *http.Request) {
        // Prevent MIME sniffing
        w.Header().Set("X-Content-Type-Options", "nosniff")
        
        // Prevent clickjacking
        w.Header().Set("X-Frame-Options", "DENY")
        
        // Enable XSS protection
        w.Header().Set("X-XSS-Protection", "1; mode=block")
        
        // Referrer policy
        w.Header().Set("Referrer-Policy", "strict-origin-when-cross-origin")
        
        // Permissions policy
        w.Header().Set("Permissions-Policy", "camera=(), microphone=(), geolocation=()")
        
        // Strict transport security
        w.Header().Set("Strict-Transport-Security", "max-age=31536000; includeSubDomains")
        
        next(w, r)
    }
}

router.Use(securityHeadersMiddleware)
```

---

## Dependency Security

### ✅ DO: Keep Dependencies Updated

```bash
# Check for vulnerabilities
go list -u -m all

# Update all dependencies
go get -u ./...

# Use vulnerability scanner
go install golang.org/x/vuln/cmd/govulncheck@latest
govulncheck ./...
```

### ✅ DO: Pin Dependency Versions

```
require (
    github.com/golang-jwt/jwt/v5 v5.0.0
    golang.org/x/crypto v0.17.0
)
```

---

## Summary

Security di dim:
- **Authentication** - Strong JWT dengan secure secrets
- **Passwords** - Bcrypt hashing, strong validation
- **Tokens** - Rotation, expiry, blacklist
- **Input** - Validation dan parameterized queries
- **CORS** - Specific origins only
- **CSRF** - Token-based protection
- **Rate Limiting** - Per IP dan per user
- **HTTPS** - Always in production
- **Headers** - Security headers
- **Secrets** - Environment variables only

---

**Lihat Juga**:
- [Autentikasi](05-authentication.md) - Auth details
- [Validasi](09-validation.md) - Input validation
- [Middleware](04-middleware.md) - Security middleware
