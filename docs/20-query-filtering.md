# Query Parameter Filtering di Framework dim

Pelajari cara menggunakan sistem filtering yang fleksibel untuk query parameter HTTP.

## Daftar Isi

- [Overview](#overview)
- [Konsep Dasar](#konsep-dasar)
- [Tipe Data Supported](#tipe-data-supported)
- [Range Queries](#range-queries)
- [Constraint Validation](#constraint-validation)
- [Custom Validators](#custom-validators)
- [Configuration](#configuration)
- [Praktik Terbaik](#praktik-terbaik)
- [API Reference](#api-reference)

---

## Overview

FilterParser adalah sistem untuk parsing dan validasi query parameter HTTP dengan type-safe casting dan constraint validation.

### Fitur Utama

- **Type-Safe Parsing**: Automatic type conversion dengan error handling
- **Range Queries**: Support untuk range dengan validasi From <= To
- **Constraint Validation**: Extensible enum dan custom constraints
- **Performance**: Type caching menggunakan goreus in-memory cache
- **Error Reporting**: Field-specific error messages

### Format Query Parameter

```
?filters[fieldName]=value
?filters[fieldName2]=val1,val2
?filters[range_field]=100,500
```

**Key format**: `filters[fieldName]` (square brackets)

---

## Konsep Dasar

### Basic Structure

```go
type Filters struct {
    // Basic types
    IDs       []int64       `filter:"ids"`
    Name      *string       `filter:"name"`
    Active    *bool         `filter:"active"`
    
    // Dengan constraints
    Status    *string       `filter:"status,in:active|pending|archived"`
    Statuses  []string      `filter:"statuses,in:active|pending"`
    
    // Range types
    Amount    AmountRange   `filter:"amount"`
    Price     *IntRange     `filter:"price"`
    CreatedAt TimestampRange `filter:"created_at"`
}

var filters Filters
fp := dim.NewFilterParser(r).
    WithMaxValues(50).
    WithTimezone(time.LoadLocation("Asia/Jakarta"))
    
fp.Parse(&filters)

if fp.HasErrors() {
    dim.JsonError(w, 400, "Filter error", fp.Errors())
    return
}

// Use filters.IDs, filters.Status, etc.
```

### Error Response Format

```json
{
  "message": "Filter error",
  "errors": {
    "filters[status]": "nilai tidak valid: invalid (diizinkan: active, pending, archived)",
    "filters[price]": "format angka tidak valid (gunakan 100 atau 100,500)"
  }
}
```

### Supported Pointer Types

```go
*string
*int
*int64
*bool
*UUID
*DateRange
*AmountRange
*IntRange
*TimestampRange
```

### Supported Slice Types

```go
[]string
[]int
[]int64
[]float64
[]UUID
```

---

## Tipe Data Supported

### 1. String (Pointer & Slice)

```go
type Filters struct {
    // Pointer - single value
    Name      *string  `filter:"name"`
    
    // Slice - multiple values
    Tags      []string `filter:"tags"`
}

// Query: ?filters[name]=John&filters[tags]=golang,database
```

**Response jika tidak ada**:
```json
{
  "name": null,
  "tags": null
}
```

### 2. Integer Types (Pointer & Slice)

```go
type Filters struct {
    // Pointer int64
    UserID    *int64   `filter:"user_id"`
    
    // Slice int64
    IDs       []int64  `filter:"ids"`
    
    // Pointer int
    Count     *int     `filter:"count"`
    
    // Slice int
    PageNums  []int    `filter:"page_nums"`
}

// Query: ?filters[user_id]=123&filters[ids]=1,2,3&filters[count]=5
```

**Parsing errors**:
- `?filters[user_id]=abc` → "harus berupa angka: abc"

### 3. Float64 (Slice only)

```go
type Filters struct {
    // Slice float64
    Prices    []float64 `filter:"prices"`
}

// Query: ?filters[prices]=10.5,20.75,99.99
```

### 4. Boolean (Pointer only)

```go
type Filters struct {
    Active    *bool    `filter:"active"`
    Verified  *bool    `filter:"verified"`
}

// Query: ?filters[active]=true&filters[verified]=false
```

**Valid values**: `true`, `false`, `1`, `0`

### 5. UUID (Pointer & Slice)

```go
type Filters struct {
    UserID    *UUID    `filter:"user_id"`
    ResourceIDs []UUID `filter:"resource_ids"`
}

// Query: ?filters[user_id]=550e8400-e29b-41d4-a716-446655440000
```

---

## Range Queries

Range types mendukung query rentang nilai dengan validasi From <= To.

### AmountRange (Float64)

```go
type Filters struct {
    Amount    AmountRange `filter:"amount"`
    Price     *AmountRange `filter:"price"`
}

// Single value: From == To
?filters[amount]=100.50
// Response: {From: 100.50, To: 100.50, Valid: true, Present: true}

// Range value
?filters[amount]=100.50,500.00
// Response: {From: 100.50, To: 500.00, Valid: true, Present: true}

// Invalid: From > To
?filters[amount]=500.00,100.50
// Response: {From: 500.00, To: 100.50, Valid: false, Present: true}
// Error: "format amount tidak valid"

// Invalid format
?filters[amount]=invalid
// Response: {From: 0, To: 0, Valid: false, Present: true}
// Error: "format amount tidak valid"
```

### IntRange (Int64)

```go
type Filters struct {
    Count     IntRange `filter:"count"`
    Pages     *IntRange `filter:"pages"`
}

// Query examples
?filters[count]=100
// Response: {From: 100, To: 100, Valid: true, Present: true}

?filters[count]=100,500
// Response: {From: 100, To: 500, Valid: true, Present: true}

?filters[count]=invalid
// Error: "format angka tidak valid (gunakan 100 atau 100,500)"
```

### DateRange (String)

```go
type Filters struct {
    CreatedOn DateRange `filter:"created_on"`
}

// Single date
?filters[created_on]=2024-01-15
// Response: {From: "2024-01-15", To: "2024-01-15", Valid: true, Present: true}

// Date range
?filters[created_on]=2024-01-01,2024-12-31
// Response: {From: "2024-01-01", To: "2024-12-31", Valid: true, Present: true}

// Invalid format
?filters[created_on]=01-15-2024
// Error: "format tanggal tidak valid (gunakan YYYY-MM-DD atau YYYY-MM-DD,YYYY-MM-DD)"

// Invalid order (From > To)
?filters[created_on]=2024-12-31,2024-01-01
// Response: {From: "2024-12-31", To: "2024-01-01", Valid: false, Present: true}
// Error: "format tanggal tidak valid..."
```

### TimestampRange (Unix Timestamp)

```go
type Filters struct {
    CreatedAt TimestampRange `filter:"created_at"`
}

// Input: YYYY-MM-DD
// Output: Unix timestamp (seconds)

?filters[created_at]=2024-01-15
// Response: {From: 1705276800, To: 1705276800, Valid: true, Present: true}

?filters[created_at]=2024-01-01,2024-01-31
// Response: {
//   From: 1704067200,  // 2024-01-01 00:00:00 UTC
//   To:   1706745600,  // 2024-01-31 00:00:00 UTC
//   Valid: true,
//   Present: true
// }

// Dengan timezone
fp := dim.NewFilterParser(r).WithTimezone(time.LoadLocation("Asia/Jakarta"))
// Parsing menggunakan Asia/Jakarta timezone
```

### Range Field Structure

```go
type Range[T any] struct {
    From    T     // Start value
    To      T     // End value
    Valid   bool  // true jika format valid dan From <= To
    Present bool  // true jika parameter ada di query
}

// Checking results
if filters.Amount.Present && filters.Amount.Valid {
    // Amount range tersedia dan valid
    startAmount := filters.Amount.From
    endAmount := filters.Amount.To
} else if filters.Amount.Present && !filters.Amount.Valid {
    // Amount range ada tapi format/validasi error
    // Error sudah tersimpan di fp.Errors()
}
```

---

## Constraint Validation

Constraints adalah rules untuk validasi nilai yang diizinkan.

### "in" Constraint (Enum)

Enum validation dengan pipe-separated allowed values.

```go
type Filters struct {
    // Pointer string
    Status    *string  `filter:"status,in:active|pending|archived"`
    
    // Slice string
    Statuses  []string `filter:"statuses,in:active|pending"`
}

// Valid
?filters[status]=active        // ✅ Valid
?filters[statuses]=active,pending  // ✅ Valid

// Invalid
?filters[status]=inactive      // ❌ Error
?filters[statuses]=active,invalid  // ❌ Error: "invalid" not allowed

// Error response
{
  "errors": {
    "filters[status]": "nilai tidak valid: inactive (diizinkan: active, pending, archived)"
  }
}
```

### Multiple Constraints Per Field

Constraints dipisahkan dengan comma:

```go
type Filters struct {
    Email  *string  `filter:"email,in:admin|user"`
    // Future: more constraints
}

// Format: fieldName,constraint1:value1,constraint2:value2
```

---

## Custom Validators

Implementasi ConstraintValidator interface untuk constraint custom.

### Implement Custom Validator

```go
// Regex pattern validator
type RegexValidator struct {
    patterns map[string]*regexp.Regexp
}

func NewRegexValidator() *RegexValidator {
    return &RegexValidator{
        patterns: map[string]*regexp.Regexp{
            "phone":  regexp.MustCompile(`^\+?[\d\s\-()]+$`),
            "slug":   regexp.MustCompile(`^[a-z0-9]+(?:-[a-z0-9]+)*$`),
            "alphanumeric": regexp.MustCompile(`^[a-zA-Z0-9]+$`),
        },
    }
}

func (v *RegexValidator) Name() string {
    return "regex"
}

func (v *RegexValidator) Validate(values []string, constraint string, fieldType reflect.Type) error {
    pattern, ok := v.patterns[constraint]
    if !ok {
        return fmt.Errorf("pattern tidak ditemukan: %s", constraint)
    }
    
    for _, val := range values {
        if !pattern.MatchString(val) {
            return fmt.Errorf("tidak sesuai pattern: %s", constraint)
        }
    }
    return nil
}

// Register dan gunakan
fp := dim.NewFilterParser(r)
fp.RegisterConstraintValidator(NewRegexValidator())

type Filters struct {
    Phone     *string `filter:"phone,regex:phone"`
    Slug      *string `filter:"slug,regex:slug"`
    AlphaNum  *string `filter:"alpha_num,regex:alphanumeric"`
}

fp.Parse(&filters)
```

### Min/Max Length Validator

```go
type LengthValidator struct{}

func (v *LengthValidator) Name() string {
    return "length"
}

func (v *LengthValidator) Validate(values []string, constraint string, fieldType reflect.Type) error {
    parts := strings.Split(constraint, "-")
    if len(parts) != 2 {
        return fmt.Errorf("format constraint salah: gunakan min-max")
    }
    
    min, _ := strconv.Atoi(parts[0])
    max, _ := strconv.Atoi(parts[1])
    
    for _, val := range values {
        if len(val) < min || len(val) > max {
            return fmt.Errorf("panjang harus %d-%d karakter", min, max)
        }
    }
    return nil
}

// Usage
type Filters struct {
    Username  *string `filter:"username,length:3-50"`
    Bio       *string `filter:"bio,length:0-500"`
}
```

### Database Unique Check Validator

```go
type UniqueValidator struct {
    db Database
}

func (v *UniqueValidator) Name() string {
    return "unique"
}

func (v *UniqueValidator) Validate(values []string, constraint string, fieldType reflect.Type) error {
    // constraint format: "table:column"
    parts := strings.Split(constraint, ":")
    if len(parts) != 2 {
        return fmt.Errorf("format: table:column")
    }
    
    table, column := parts[0], parts[1]
    
    for _, val := range values {
        var exists bool
        query := fmt.Sprintf("SELECT EXISTS(SELECT 1 FROM %s WHERE %s = $1)", table, column)
        v.db.QueryRow(context.Background(), query, val).Scan(&exists)
        
        if exists {
            return fmt.Errorf("%s '%s' sudah digunakan", column, val)
        }
    }
    return nil
}

// Usage
type Filters struct {
    Username  *string `filter:"username,unique:users:username"`
    Email     *string `filter:"email,unique:users:email"`
}
```

---

## Configuration

### WithMaxValues

Batasi jumlah nilai per field:

```go
fp := dim.NewFilterParser(r).WithMaxValues(50)

type Filters struct {
    IDs []int64 `filter:"ids"`  // Max 50 values
    Tags []string `filter:"tags"`  // Max 50 values
}

// Query: ?filters[ids]=1,2,3,...,51
// Error: "maksimal 50 nilai diperbolehkan, diterima 51"
```

### WithTimezone

Set timezone untuk parsing timestamp:

```go
jakartaTz, _ := time.LoadLocation("Asia/Jakarta")

fp := dim.NewFilterParser(r).WithTimezone(jakartaTz)

type Filters struct {
    CreatedAt TimestampRange `filter:"created_at"`
}

// Parsing menggunakan Asia/Jakarta timezone
// ?filters[created_at]=2024-01-15
// Parsed dalam timezone Asia/Jakarta
```

### RegisterConstraintValidator

Register custom constraint:

```go
fp := dim.NewFilterParser(r).
    RegisterConstraintValidator(NewRegexValidator()).
    RegisterConstraintValidator(NewLengthValidator()).
    RegisterConstraintValidator(NewUniqueValidator())

// Semua validator tersedia untuk digunakan
```

### Method Chaining

```go
fp := dim.NewFilterParser(r).
    WithMaxValues(100).
    WithTimezone(time.LoadLocation("Asia/Jakarta")).
    RegisterConstraintValidator(customValidator1).
    RegisterConstraintValidator(customValidator2)

fp.Parse(&filters)
```

---

## Error Handling

### Check Errors

```go
fp := dim.NewFilterParser(r).Parse(&filters)

if fp.HasErrors() {
    errors := fp.Errors()  // map[string]string
    
    // Error format:
    // Key: "filters[fieldName]"
    // Value: Error message
    
    dim.JsonError(w, 400, "Filter validation failed", errors)
    return
}
```

### Error Messages

| Scenario | Error Message |
|----------|---------------|
| Invalid int | "harus berupa angka: abc" |
| Invalid float | "harus berupa angka desimal: abc" |
| Invalid UUID | "UUID tidak valid: invalid-uuid" |
| Invalid date | "format tanggal tidak valid (gunakan YYYY-MM-DD atau YYYY-MM-DD,YYYY-MM-DD)" |
| Invalid enum | "nilai tidak valid: {value} (diizinkan: {list})" |
| Max values exceeded | "maksimal {max} nilai diperbolehkan, diterima {count}" |
| Invalid range | Format errors dari parser |

### Example Error Response

```json
{
  "message": "Filter validation failed",
  "errors": {
    "filters[ids]": "maksimal 50 nilai diperbolehkan, diterima 60",
    "filters[status]": "nilai tidak valid: inactive (diizinkan: active, pending, archived)",
    "filters[price]": "format angka tidak valid (gunakan 100 atau 100,500)",
    "filters[created_at]": "format tanggal tidak valid (gunakan YYYY-MM-DD atau YYYY-MM-DD,YYYY-MM-DD)"
  }
}
```

---

## Praktik Terbaik

### ✅ DO: Parse & Validate Immediately

```go
// ✅ BAIK
func listUsersHandler(w http.ResponseWriter, r *http.Request) {
    var filters Filters
    fp := dim.NewFilterParser(r).Parse(&filters)
    
    if fp.HasErrors() {
        dim.JsonError(w, 400, "Filter error", fp.Errors())
        return
    }
    
    // Gunakan filters dengan aman
}

// ❌ BURUK - Tunda validasi
func listUsersHandler(w http.ResponseWriter, r *http.Request) {
    var filters Filters
    fp := dim.NewFilterParser(r)
    
    // ... logic lain dulu ...
    
    fp.Parse(&filters)  // Terlambat
}
```

### ✅ DO: Use Pointer for Optional Fields

```go
// ✅ BAIK - Optional field dengan pointer
type Filters struct {
    Name    *string  `filter:"name"`      // Optional
    IDs     []int64  `filter:"ids"`       // Optional slice
    Status  *string  `filter:"status"`    // Optional
}

// ❌ BURUK - Non-pointer untuk optional
type Filters struct {
    Name    string  `filter:"name"`       // Will be "" if not provided
}
```

### ✅ DO: Use Range for Numeric Queries

```go
// ✅ BAIK - Range untuk rentang nilai
type Filters struct {
    PriceRange    AmountRange `filter:"price"`
    AgeRange      IntRange `filter:"age"`
    DateRange     TimestampRange `filter:"date"`
}

// Query: ?filters[price]=100,500
// Response: {From: 100, To: 500, Valid: true}

// ❌ BURUK - Separate min/max fields
type Filters struct {
    PriceMin    *float64 `filter:"price_min"`
    PriceMax    *float64 `filter:"price_max"`
}
```

### ✅ DO: Set Reasonable Limits

```go
// ✅ BAIK - Limit query parameter values
fp := dim.NewFilterParser(r).WithMaxValues(100)

// Query: ?filters[ids]=1,2,3,...,101
// Error: Terlalu banyak nilai

// ❌ BURUK - No limits
fp := dim.NewFilterParser(r)
// Bisa menerima 10000+ values → performance issue
```

### ✅ DO: Use Constraints for Enums

```go
// ✅ BAIK - Enum constraint
type Filters struct {
    Status *string `filter:"status,in:active|pending|archived"`
}

// ❌ BURUK - Manual validation
type Filters struct {
    Status *string `filter:"status"`
}

func handler(w http.ResponseWriter, r *http.Request) {
    if filters.Status != nil && 
       *filters.Status != "active" && 
       *filters.Status != "pending" {
        // Manual validation
    }
}
```

### ✅ DO: Build Query URLs Safely

```go
// ✅ BAIK - url.Values untuk build query
values := url.Values{}
values.Set("filters[status]", "active")
values.Add("filters[ids]", "1")
values.Add("filters[ids]", "2")

url := "http://api.example.com/users?" + values.Encode()

// ❌ BURUK - String concatenation
url := "http://api.example.com/users?filters[status]=" + status
```

---

## API Reference

### FilterParser Methods

```go
// Create new parser
fp := dim.NewFilterParser(r)

// Configuration
fp.WithMaxValues(max int) *FilterParser
fp.WithTimezone(tz *time.Location) *FilterParser
fp.RegisterConstraintValidator(v ConstraintValidator) *FilterParser

// Parsing
fp.Parse(target interface{}) *FilterParser

// Error checking
fp.HasErrors() bool
fp.Errors() map[string]string
```

### Range Types

```go
type Range[T any] struct {
    From    T
    To      T
    Valid   bool   // true if valid format and From <= To
    Present bool   // true if parameter exists
}

type DateRange = Range[string]      // YYYY-MM-DD
type AmountRange = Range[float64]   // Float64
type IntRange = Range[int64]        // Integer
type TimestampRange = Range[int64]  // Unix timestamp
```

### ConstraintValidator Interface

```go
type ConstraintValidator interface {
    Name() string
    Validate(values []string, constraint string, fieldType reflect.Type) error
}

// Built-in validators
type InConstraintValidator struct{}  // Enum validation
```

### Built-in Validators

- **InConstraintValidator**: Enum validation dengan pipe-separated values

---

## Common Patterns

### List with Filters

```go
func listUsersHandler(userStore UserStore) http.HandlerFunc {
    return func(w http.ResponseWriter, r *http.Request) {
        type FilterRequest struct {
            IDs       []int64       `filter:"ids"`
            Status    *string       `filter:"status,in:active|inactive"`
            CreatedAt TimestampRange `filter:"created_at"`
            Name      *string       `filter:"name"`
        }
        
        var filters FilterRequest
        fp := dim.NewFilterParser(r).
            WithMaxValues(100).
            Parse(&filters)
        
        if fp.HasErrors() {
            dim.JsonError(w, 400, "Invalid filters", fp.Errors())
            return
        }
        
        // Build query dengan filters
        users, err := userStore.List(r.Context(), filters)
        if err != nil {
            dim.JsonError(w, 500, "Failed to fetch users", nil)
            return
        }
        
        dim.Json(w, 200, users)
    }
}
```

### Range Filtering

```go
func searchProductsHandler(productStore ProductStore) http.HandlerFunc {
    return func(w http.ResponseWriter, r *http.Request) {
        type FilterRequest struct {
            PriceRange  AmountRange `filter:"price"`
            StockRange  IntRange `filter:"stock"`
            CreatedDate DateRange `filter:"created_date"`
        }
        
        var filters FilterRequest
        fp := dim.NewFilterParser(r).Parse(&filters)
        
        if fp.HasErrors() {
            dim.JsonError(w, 400, "Invalid filters", fp.Errors())
            return
        }
        
        query := productStore.NewQuery()
        
        if filters.PriceRange.Valid {
            query = query.WherePriceBetween(
                filters.PriceRange.From,
                filters.PriceRange.To,
            )
        }
        
        if filters.StockRange.Valid {
            query = query.WhereStockBetween(
                filters.StockRange.From,
                filters.StockRange.To,
            )
        }
        
        products, err := query.Fetch(r.Context())
        if err != nil {
            dim.JsonError(w, 500, "Query failed", nil)
            return
        }
        
        dim.Json(w, 200, products)
    }
}
```

---

## Troubleshooting

### Issue: "maksimal N nilai diperbolehkan"

Terlalu banyak values dalam satu parameter:

```bash
# ❌ Terlalu banyak
?filters[ids]=1,2,3,...,101  (101 values, limit 100)

# ✅ Gunakan limit yang masuk akal
?filters[ids]=1,2,3,...,100  (100 values)

# ✅ Atau increase limit jika perlu
fp := dim.NewFilterParser(r).WithMaxValues(500)
```

### Issue: Range not parsing correctly

Pastikan format dari-ke dengan comma:

```bash
# ❌ Salah
?filters[price]=100 500    # Spasi, bukan comma
?filters[price]=100-500    # Dash, bukan comma

# ✅ Benar
?filters[price]=100,500    # Comma separator
```

### Issue: Timezone not working

Load timezone dengan benar:

```go
// ❌ Salah
tz := time.LoadLocation("Jakarta")  // Not found

// ✅ Benar
tz, _ := time.LoadLocation("Asia/Jakarta")
fp := dim.NewFilterParser(r).WithTimezone(tz)
```

---

## Summary

Query Filtering di dim:
- **Type-Safe**: Automatic type conversion dengan validation
- **Flexible**: Support berbagai tipe data dan range queries
- **Extensible**: Custom constraint validators
- **Performant**: Type caching untuk query berulang
- **User-Friendly**: Clear error messages

Gunakan FilterParser untuk semua query parameter parsing dan validasi!

---

**Related Docs**:
- [Validasi](09-validation.md) - Field validation patterns
- [Error Handling](08-error-handling.md) - Error response formatting
- [Request Context](10-request-context.md) - Accessing request data
- [API Reference](19-api-reference.md) - Complete API docs
