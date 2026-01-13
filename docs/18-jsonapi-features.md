# Fitur JSON:API di Framework dim

Pelajari cara menggunakan sistem Filtering, Pagination, dan Sorting yang mengikuti standar JSON:API.

## Daftar Isi

- [Filtering](#filtering)
- [Pagination](#pagination)
- [Sorting](#sorting)
- [Integrasi Lengkap](#integrasi-lengkap)

---

## Filtering

Dim menyediakan `FilterParser` untuk memproses query parameter dengan format `filters[field]`.

### Konsep Dasar

```go
type UserFilters struct {
    IDs    []int64 `filter:"ids"`
    Status *string `filter:"status,in:active|inactive"`
    Name   *string `filter:"name"`
}

var filters UserFilters
fp := dim.NewFilterParser(r)
fp.Parse(&filters)

if fp.HasErrors() {
    dim.JsonError(w, 400, "Filter tidak valid", fp.Errors())
    return
}
```

### Format Query
`?filters[status]=active&filters[ids]=1,2,3`

*Untuk dokumentasi filtering yang lebih mendalam, lihat bagian Filtering di kode sumber.*

---

## Pagination

Dim mendukung standar JSON:API `page[number]` dan `page[size]`, serta fallback ke parameter sederhana `page` dan `limit`.

### Penggunaan `PaginationParser`

```go
// 1. Inisialisasi parser (Default Limit, Max Limit)
parser := dim.NewPaginationParser(10, 100)

// 2. Parse request
pagination, err := parser.Parse(r)
if err != nil {
    dim.JsonAppError(w, err.(*dim.AppError))
    return
}

// 3. Gunakan di Database
// pagination.Page  -> nomor halaman
// pagination.Limit -> ukuran halaman
// pagination.Offset() -> helper untuk (Page-1) * Limit
```

### Format Query yang Didukung
1.  **JSON:API Style**: `?page[number]=2&page[size]=20`
2.  **Simple Style**: `?page=2&limit=20` atau `?page=2&size=20`

---

## Sorting

Menangani pengurutan data dengan format `?sort=field` (ascending) atau `?sort=-field` (descending).

### Penggunaan `SortParser`

```go
// 1. Tentukan field yang diizinkan untuk di-sort (Security)
allowedFields := []string{"created_at", "username", "id"}
parser := dim.NewSortParser(allowedFields)

// 2. Parse request
sortFields, err := parser.Parse(r)
if err != nil {
    dim.JsonAppError(w, err.(*dim.AppError))
    return
}

// 3. Gunakan di Database
for _, s := range sortFields {
    // s.Field     -> "created_at"
    // s.Direction -> "DESC" atau "ASC"
    // s.SQL()     -> "created_at DESC" (helper)
}
```

---

## Integrasi Lengkap

Contoh penggunaan Filtering, Pagination, dan Sorting dalam satu handler:

```go
func listUsersHandler(w http.ResponseWriter, r *http.Request) {
    // 1. Parsing Pagination
    pg, _ := dim.NewPaginationParser(10, 50).Parse(r)

    // 2. Parsing Sort
    sort, _ := dim.NewSortParser([]string{"id", "created_at"}).Parse(r)

    // 3. Parsing Filters
    var f struct {
        Status *string `filter:"status,in:active|inactive"`
    }
    dim.NewFilterParser(r).Parse(&f)

    // 4. Query Database (Contoh logika)
    users, total, _ := userStore.FindAll(r.Context(), f, pg, sort)

    // 5. Response dengan Meta
    dim.JsonPagination(w, http.StatusOK, users, dim.PaginationMeta{
        Page:       pg.Page,
        PerPage:    pg.Limit,
        Total:      total,
        TotalPages: (total + pg.Limit - 1) / pg.Limit,
    })
}
```

---

## Summary

Dengan fitur JSON:API di dim, Anda mendapatkan:
- **Type-safe Filtering** dengan validasi otomatis.
- **Flexible Pagination** yang mendukung berbagai gaya query.
- **Secure Sorting** dengan whitelist field untuk mencegah SQL injection.
