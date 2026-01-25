# Testing di Framework dim

Pelajari cara melakukan unit testing, integration testing, dan handler testing.

## Daftar Isi

- [Testing Philosophy](#testing-philosophy)
- [Unit Testing](#unit-testing)
- [Mock Interfaces](#mock-interfaces)
- [Handler Testing](#handler-testing)
- [Integration Testing](#integration-testing)
- [Test Utilities](#test-utilities)
- [Running Tests](#running-tests)
- [Test Coverage](#test-coverage)
- [Praktik Terbaik](#best-practices)

---

## Testing Philosophy

### Why Testing?

- **Confidence** - Code berhasil sebagai harapan
- **Documentation** - Tests menunjukkan cara menggunakan kode
- **Regression** - Detect bugs sebelum production
- **Refactoring** - Refactor dengan confidence
- **Design** - Baik testable code = baik design

### Testing Pyramid

```
         / \
        /   \
       / E2E \        (Integration/API tests)
      /_______\
     /         \
    /  Unit     \      (Handler, Service, Store)
   /_____________\
  /               \
 / Util & Helper   \ (Simple functions)
/___________________\
```

---

## Unit Testing

### Basic Unit Test

```go
package main

import "testing"

// Function to test
func Add(a, b int) int {
    return a + b
}

// Unit test
func TestAdd(t *testing.T) {
    tests := []struct {
        name     string
        a, b     int
        expected int
    }{
        {"positive numbers", 2, 3, 5},
        {"negative numbers", -2, -3, -5},
        {"mixed", 5, -3, 2},
        {"zero", 0, 0, 0},
    }
    
    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            result := Add(tt.a, tt.b)
            if result != tt.expected {
                t.Errorf("Add(%d, %d) = %d, want %d", tt.a, tt.b, result, tt.expected)
            }
        })
    }
}
```

### Table-Driven Tests

```go
func TestValidateEmail(t *testing.T) {
    tests := []struct {
        name    string
        email   string
        wantErr bool
    }{
        {"valid email", "user@example.com", false},
        {"missing @", "userexample.com", true},
        {"missing domain", "user@", true},
        {"empty string", "", true},
        {"spaces", "user @example.com", true},
    }
    
    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            err := ValidateEmail(tt.email)
            if (err != nil) != tt.wantErr {
                t.Errorf("ValidateEmail(%q) error = %v, wantErr %v", 
                    tt.email, err, tt.wantErr)
            }
        })
    }
}
```

---

## Mock Interfaces

Pola yang lebih baik dan lebih umum di Go adalah mendefinisikan *interface* untuk *dependency* Anda dan membuat *mock* yang mengimplementasikan *interface* tersebut untuk pengujian. Framework `dim` menyediakan `Database` interface untuk tujuan ini.

### Contoh: Mocking `dim.Database`

Daripada membuat `TestUserStore` yang meniru `UserStore`, kita akan membuat `MockDatabase` yang meniru `dim.Database`. Kemudian kita bisa menguji `UserStore` yang sebenarnya dengan `MockDatabase` ini.

**1. `UserStore` yang Sebenarnya (bergantung pada `Database` interface):**
```go
// Real UserStore
type UserStore struct {
    db dim.Database // Bergantung pada interface, bukan implementasi konkret
}

func NewUserStore(db dim.Database) *UserStore {
    return &UserStore{db: db}
}

func (store *UserStore) Create(ctx context.Context, user *User) error {
    return store.db.Exec(ctx,
        "INSERT INTO users (email, username, password_hash) VALUES ($1, $2, $3)",
        user.Email, user.Username, user.PasswordHash)
}
```

**2. Buat `MockDatabase` untuk Pengujian:**
```go
// MockDatabase mengimplementasikan dim.Database interface
type MockDatabase struct {
    ExecFunc func(ctx context.Context, query string, args ...interface{}) error
    QueryFunc func(ctx context.Context, query string, args ...interface{}) (dim.Rows, error)
    QueryRowFunc func(ctx context.Context, query string, args ...interface{}) dim.Row
    BeginFunc func(ctx context.Context) (dim.Tx, error)
    WithTxFunc func(ctx context.Context, fn dim.TransactionFunc) error
    CloseFunc func() error
    DriverNameFunc func() string
}

// Implementasikan setiap metode interface
func (m *MockDatabase) Exec(ctx context.Context, query string, args ...interface{}) error {
    if m.ExecFunc != nil {
        return m.ExecFunc(ctx, query, args...)
    }
    return nil // Default behavior
}

// ... implementasikan metode lain (Query, QueryRow, Begin, Close) ...

func (m *MockDatabase) DriverName() string {
    if m.DriverNameFunc != nil {
        return m.DriverNameFunc()
    }
    return "mock"
}
```

**3. Gunakan Mock dalam Pengujian:**
```go
func TestUserStore_Create(t *testing.T) {
    // 1. Setup mock
    mockDB := &MockDatabase{}
    
    // 2. Definisikan perilaku mock untuk tes ini
    var execCalled bool
    mockDB.ExecFunc = func(ctx context.Context, query string, args ...interface{}) error {
        execCalled = true
        // Anda bisa memeriksa query dan args di sini jika perlu
        return nil // Simulasikan eksekusi berhasil
    }

    // 3. Injeksi mock ke UserStore yang sebenarnya
    userStore := NewUserStore(mockDB)
    
    // 4. Jalankan metode yang akan diuji
    err := userStore.Create(context.Background(), &User{Email: "test@test.com"})

    // 5. Verifikasi hasil
    if err != nil {
        t.Errorf("Unexpected error: %v", err)
    }
    if !execCalled {
        t.Error("Expected Exec to be called, but it wasn't")
    }
}
```

### Keuntungan Mocking Interface
✅ **Pemisahan yang Kuat**: `UserStore` tidak tahu menahu tentang mock, ia hanya berinteraksi dengan `Database` interface.
✅ **Pengujian Terisolasi**: Anda bisa menguji logika `UserStore` secara terpisah dari database yang sebenarnya.
✅ **Simulasi Error**: Sangat mudah untuk menyimulasikan berbagai kondisi, seperti kegagalan database.

```go
// Contoh simulasi error
func TestUserStore_Create_DBError(t *testing.T) {
    mockDB := &MockDatabase{}
    expectedErr := errors.New("database connection lost")

    mockDB.ExecFunc = func(ctx context.Context, query string, args ...interface{}) error {
        return expectedErr // Simulasikan error
    }

    userStore := NewUserStore(mockDB)
    err := userStore.Create(context.Background(), &User{Email: "test@test.com"})

    if err == nil {
        t.Error("Expected an error, but got nil")
    }
    if !errors.Is(err, expectedErr) {
        t.Errorf("Expected error %v, but got %v", expectedErr, err)
    }
}
```

---

## Handler Testing

### Testing HTTP Handlers

```go
import (
    "bytes"
    "encoding/json"
    "net/http"
    "net/http/httptest"
    "testing"
)

func TestRegisterHandler(t *testing.T) {
    // Setup
    mockService := &MockAuthService{
        RegisterFn: func(ctx context.Context, email, username, password string) (*User, error) {
            return &User{ID: 1, Email: email, Username: username}, nil
        },
    }
    
    handler := registerHandler(mockService)
    
    // Create request
    body := map[string]string{
        "email":    "john@example.com",
        "username": "john",
        "password": "SecurePass123",
    }
    
    bodyBytes, _ := json.Marshal(body)
    req := httptest.NewRequest("POST", "/auth/register", bytes.NewReader(bodyBytes))
    req.Header.Set("Content-Type", "application/json")
    
    // Create response recorder
    rr := httptest.NewRecorder()
    
    // Call handler
    handler.ServeHTTP(rr, req)
    
    // Check status
    if rr.Code != http.StatusCreated {
        t.Errorf("handler returned %d, want %d", rr.Code, http.StatusCreated)
    }
    
    // Check response body
    var user User
    json.NewDecoder(rr.Body).Decode(&user)
    
    if user.Email != "john@example.com" {
        t.Errorf("handler returned email %q, want john@example.com", user.Email)
    }
}
```

### Testing Validation Errors

```go
func TestRegisterHandlerValidation(t *testing.T) {
    mockService := &MockAuthService{}
    handler := registerHandler(mockService)
    
    tests := []struct {
        name       string
        body       map[string]string
        wantStatus int
        wantError  string
    }{
        {
            "missing email",
            map[string]string{"username": "john", "password": "Pass123"},
            http.StatusBadRequest,
            "email",
        },
        {
            "invalid email",
            map[string]string{
                "email":    "invalid",
                "username": "john",
                "password": "Pass123",
            },
            http.StatusBadRequest,
            "email",
        },
        {
            "short password",
            map[string]string{
                "email":    "john@example.com",
                "username": "john",
                "password": "short",
            },
            http.StatusBadRequest,
            "password",
        },
    }
    
    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            bodyBytes, _ := json.Marshal(tt.body)
            req := httptest.NewRequest("POST", "/auth/register", bytes.NewReader(bodyBytes))
            req.Header.Set("Content-Type", "application/json")
            
            rr := httptest.NewRecorder()
            handler.ServeHTTP(rr, req)
            
            if rr.Code != tt.wantStatus {
                t.Errorf("status = %d, want %d", rr.Code, tt.wantStatus)
            }
            
            // Check error contains expected field
            var errResp map[string]interface{}
            json.NewDecoder(rr.Body).Decode(&errResp)
            
            if errors, ok := errResp["errors"].(map[string]interface{}); ok {
                if _, hasField := errors[tt.wantError]; !hasField {
                    t.Errorf("error missing %q field", tt.wantError)
                }
            }
        })
    }
}
```

---

## Integration Testing

### Full Request Flow

```go
func TestAuthFlow(t *testing.T) {
    // Setup database
    db := setupTestDatabase(t)
    defer db.Close()
    
    // Setup stores and services
    userStore := dim.NewPostgresUserStore(db)
    tokenStore := dim.NewDatabaseTokenStore(db)
    cfg := &JWTConfig{
        Secret: "test-secret",
        AccessTokenExpiry: 15 * time.Minute,
        RefreshTokenExpiry: 7 * 24 * time.Hour,
    }
    authService := dim.NewAuthService(userStore, tokenStore, nil, cfg)
    
    // Setup router
    router := dim.NewRouter()
    router.Use(dim.Recovery(logger))
    router.Use(dim.LoggerMiddleware(logger))
    router.Use(dim.CORS(corsConfig))
    
    router.Post("/auth/register", registerHandler(authService))
    router.Post("/auth/login", loginHandler(authService))
    
    // Test registration
    registerBody := map[string]string{
        "email":    "john@example.com",
        "username": "john",
        "password": "SecurePass123",
    }
    
    registerReq := httptest.NewRequest("POST", "/auth/register",
        bytes.NewReader(marshalJSON(registerBody)))
    registerReq.Header.Set("Content-Type", "application/json")
    registerRR := httptest.NewRecorder()
    
    router.ServeHTTP(registerRR, registerReq)
    
    if registerRR.Code != http.StatusCreated {
        t.Fatalf("registration failed: %d", registerRR.Code)
    }
    
    var user User
    json.NewDecoder(registerRR.Body).Decode(&user)
    
    // Test login
    loginBody := map[string]string{
        "email":    "john@example.com",
        "password": "SecurePass123",
    }
    
    loginReq := httptest.NewRequest("POST", "/auth/login",
        bytes.NewReader(marshalJSON(loginBody)))
    loginReq.Header.Set("Content-Type", "application/json")
    loginRR := httptest.NewRecorder()
    
    router.ServeHTTP(loginRR, loginReq)
    
    if loginRR.Code != http.StatusOK {
        t.Fatalf("login failed: %d", loginRR.Code)
    }
    
    var loginResp map[string]interface{}
    json.NewDecoder(loginRR.Body).Decode(&loginResp)
    
    if _, ok := loginResp["access_token"]; !ok {
        t.Error("login response missing access_token")
    }
}
```

---

## Test Utilities

### Test Database

```go
func setupTestDatabase(t *testing.T) Database {
    cfg := DatabaseConfig{
        WriteHost: "localhost",
        ReadHosts: []string{"localhost"},
        Port:      5432,
        Database:  "dim_test",
        Username:  "postgres",
        Password:  "postgres",
    }
    
    db, err := dim.NewPostgresDatabase(cfg)
    if err != nil {
        t.Fatalf("failed to connect to test database: %v", err)
    }
    
    // Run migrations
    migrations := getMigrations()
    if err := dim.RunMigrations(db, migrations); err != nil {
        t.Fatalf("failed to run migrations: %v", err)
    }
    
    return db
}
```

### Helper Functions

```go
func marshalJSON(v interface{}) *bytes.Buffer {
    buf := new(bytes.Buffer)
    json.NewEncoder(buf).Encode(v)
    return buf
}

func newTestRequest(method, path string, body interface{}) *http.Request {
    var reader io.Reader
    if body != nil {
        reader = bytes.NewReader(marshalJSON(body).Bytes())
    }
    
    req := httptest.NewRequest(method, path, reader)
    req.Header.Set("Content-Type", "application/json")
    return req
}
```

---

## Running Tests

### Run All Tests

```bash
go test ./...
```

### Run Specific Package

```bash
go test ./handler
```

### Run Specific Test

```bash
go test -run TestAuthServiceRegister
```

### Run with Output

```bash
go test -v ./...
```

### Run with Timeout

```bash
go test -timeout 30s ./...
```

### Run with Parallel Execution

```bash
go test -parallel 4 ./...
```

---

## Test Coverage

### Generate Coverage

```bash
go test -cover ./...

# Output: coverage: 75.5% of statements
```

### Generate Coverage Report

```bash
go test -coverprofile=coverage.out ./...
go tool cover -html=coverage.out
```

### Coverage by Function

```bash
go tool cover -func=coverage.out
```

### Minimum Coverage

```go
// In Makefile or CI
go test -cover ./... -coverprofile=coverage.out
go tool cover -func=coverage.out | grep total | awk '{
    if ($3 < 80) {
        print "Coverage too low: " $3
        exit 1
    }
}'
```

---

## Praktik Terbaik

### ✅ DO: Write Testable Code

```go
// ✅ BAIK - Dependency injection dengan struct
func NewAuthService(userStore *UserStore, tokenStore *TokenStore) *AuthService {
    return &AuthService{
        userStore:  userStore,
        tokenStore: tokenStore,
    }
}

// ❌ BURUK - Global dependencies
var globalDB *sql.DB

func NewAuthService() *AuthService {
    return &AuthService{
        db: globalDB,  // Can't test without real DB
    }
}
```

### ✅ DO: Create Test Stores

```go
// ✅ BAIK - Test store implementation
type TestUserStore struct {
    Users       map[int64]*User
    CreateCalls int
}

func NewTestUserStore() *TestUserStore {
    return &TestUserStore{
        Users: make(map[int64]*User),
    }
}

func (store *TestUserStore) Create(ctx context.Context, user *User) error {
    store.CreateCalls++
    user.ID = int64(len(store.Users) + 1)
    store.Users[user.ID] = user
    return nil
}

// ❌ BURUK - No way to test without real database
type PostgresUserStore struct {
    db Database
}
```

### ✅ DO: Test Edge Cases

```go
func TestValidateEmail(t *testing.T) {
    tests := []struct {
        name    string
        email   string
        wantErr bool
    }{
        {"valid", "user@example.com", false},
        {"empty", "", true},
        {"no @", "userexample.com", true},
        {"double @", "user@@example.com", true},
        {"space", "user @example.com", true},
        {"unicode", "用户@example.com", false},  // Edge case
    }
    
    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            err := ValidateEmail(tt.email)
            if (err != nil) != tt.wantErr {
                t.Errorf("ValidateEmail(%q) error = %v", tt.email, err)
            }
        })
    }
}
```

### ✅ DO: Test Error Cases

```go
func TestAuthServiceLoginErrors(t *testing.T) {
    mockStore := &MockUserStore{
        FindByEmailFn: func(ctx context.Context, email string) (*User, error) {
            return nil, sql.ErrNoRows  // User not found
        },
    }
    
    service := NewAuthService(mockStore, nil, nil)
    
    _, _, err := service.Login(context.Background(), "nonexistent@example.com", "password")
    
    if err == nil {
        t.Error("Login() should error for non-existent user")
    }
}
```

### ✅ DO: Cleanup Resources

```go
func TestWithDatabase(t *testing.T) {
    db := setupTestDatabase(t)
    defer db.Close()  // Always cleanup
    
    // Test code
}

func TestWithRouter(t *testing.T) {
    // Setup
    router := setupTestRouter()
    
    // Test
    rr := httptest.NewRecorder()
    req := httptest.NewRequest("GET", "/health", nil)
    router.ServeHTTP(rr, req)
    
    // Verify
    if rr.Code != http.StatusOK {
        t.Error("unexpected status")
    }
}
```

### ✅ DO: Clear Test Names

```go
// ✅ BAIK - Descriptive names
func TestRegisterHandlerWithValidInput(t *testing.T) { }
func TestRegisterHandlerWithMissingEmail(t *testing.T) { }
func TestRegisterHandlerWithDuplicateEmail(t *testing.T) { }

// ❌ BURUK - Generic names
func TestRegister(t *testing.T) { }
func TestRegister2(t *testing.T) { }
func TestRegister3(t *testing.T) { }
```

### ❌ DON'T: Test Implementation Details

```go
// ❌ BURUK - Testing private method
func TestPrivateHelper(t *testing.T) {
    result := privateHelper("value")  // Can't test private
}

// ✅ BAIK - Test public interface
func TestPublicMethod(t *testing.T) {
    result := PublicMethod("value")
    // privateHelper called internally
}
```

---

## Test File Structure

```
auth_service.go
auth_service_test.go    ← Test for auth_service

user_store.go
user_store_test.go      ← Test for user_store

handler.go
handler_test.go         ← Test for handlers

testutil.go             ← Helper functions
testutil.go             ← Shared test utilities
```

---

## Summary

Testing di dim:
- **Unit tests** - Test individual components
- **Mocks** - Replace dependencies
- **Handlers** - Test HTTP endpoints
- **Integration** - Test full flows
- **Coverage** - Measure completeness
- **Praktik terbaik** - Testable code design

Lihat [Running Tests](#running-tests) untuk menjalankan test suite.

---

**Lihat Juga**:
- [Architecture](02-architecture.md) - Component design
- [Handlers](16-handlers.md) - Handler patterns
- [Database](06-database.md) - Database operations
