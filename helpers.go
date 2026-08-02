package dim

import (
	"crypto/rand"
	"encoding/hex"
	"net"
	"net/http"
	"strings"
	"unicode"
)

// =============================================================================
// CRYPTOGRAPHY HELPERS
// =============================================================================

// GenerateSecureToken menghasilkan token random yang cryptographically secure.
// Token di-generate menggunakan crypto/rand dan di-encode sebagai hex string.
// Berguna untuk session tokens, API keys, CSRF tokens, password reset tokens, dll.
//
// Parameters:
//   - length: jumlah bytes random untuk generate (contoh: 32)
//
// Returns:
//   - string: hex-encoded token string
//   - error: error jika random generation gagal
//
// Example:
//
//	token, err := GenerateSecureToken(32)
//	if err != nil {
//	  return err
//	}
//	// token adalah hex string dengan panjang 64 (32 bytes * 2)
func GenerateSecureToken(length int) (string, error) {
	b := make([]byte, length)
	_, err := rand.Read(b)
	if err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}

// GetClientIP mengembalikan client IP address dari HTTP request.
//
// Bila middleware ClientIP terpasang, fungsi ini mengembalikan IP yang sudah
// diresolusi middleware tersebut sesuai ClientIPConfig.TrustedProxyCount.
// Bila tidak, fungsi ini mengembalikan RemoteAddr.
//
// RemoteAddr adalah default yang aman: header proxy (X-Forwarded-For, X-Real-IP,
// dll) dapat dimanipulasi klien, sehingga tidak pernah dipercaya kecuali aplikasi
// secara eksplisit menyatakan berapa hop proxy yang layak dipercaya.
//
// Parameters:
//   - r: *http.Request yang berisi client information
//
// Returns:
//   - string: client IP address string (IPv4 atau IPv6 format tanpa port)
//
// Example:
//
//	clientIP := GetClientIP(req)  // returns "192.168.1.1" atau "::1"
func GetClientIP(r *http.Request) string {
	if clientIP, ok := clientIPFromContext(r); ok {
		return clientIP
	}
	return CleanIPAddress(r.RemoteAddr)
}

// GetClientIPWithTrustedProxies mengekstrak client IP address dari HTTP request
// dengan memperhitungkan sejumlah hop proxy yang tepercaya.
//
// Membaca X-Forwarded-For dari kanan ke kiri: setiap proxy yang tepercaya
// menambahkan IP di ujung kanan, sehingga IP klien asli berada di posisi
// len(entries) - trustedProxyCount dari kiri (dihitung dari kanan sebanyak trustedProxyCount).
//
// Seluruh baris header X-Forwarded-For digabungkan terlebih dahulu, karena sebagian
// proxy menambahkan baris header baru alih-alih menyambung ke baris yang sudah ada.
//
// Jatuh kembali ke RemoteAddr bila: trustedProxyCount <= 0, header tidak ada,
// jumlah entri lebih sedikit daripada trustedProxyCount (rantai proxy lebih pendek
// daripada yang dikonfigurasi — sisanya berasal dari klien), atau nilai pada indeks
// yang dihitung bukan IP yang sah.
//
// KEAMANAN: mekanisme ini hanya aman bila aplikasi TIDAK dapat dihubungi langsung,
// yakni seluruh trafik wajib melewati proxy tepercaya. Bila origin masih terjangkau
// langsung, klien dapat menyusun sendiri seluruh isi header dan memalsukan hasilnya.
//
// Parameters:
//   - r: *http.Request yang berisi client information
//   - trustedProxyCount: jumlah hop proxy tepercaya di depan aplikasi (dihitung dari kanan).
//     0 = abaikan seluruh header proxy, pakai RemoteAddr saja.
//     1 = satu proxy (misalnya Cloud Run atau satu load balancer).
//
// Returns:
//   - string: client IP address string (IPv4 atau IPv6 format tanpa port)
//
// Example:
//
//	// Di belakang satu proxy (Cloud Run, nginx, dll)
//	clientIP := GetClientIPWithTrustedProxies(req, 1)
//
//	// X-Forwarded-For: spoofed, real_client  →  returns "real_client"
func GetClientIPWithTrustedProxies(r *http.Request, trustedProxyCount int) string {
	if trustedProxyCount <= 0 {
		return CleanIPAddress(r.RemoteAddr)
	}

	// Header.Get hanya mengembalikan baris pertama. Sebagian proxy menambahkan
	// baris X-Forwarded-For terpisah, sehingga baris yang dikirim klien akan
	// terbaca lebih dulu jika tidak digabungkan. Secara semantik baris berulang
	// setara dengan satu baris yang dipisah koma (RFC 9110 §5.3).
	xForwardedFor := strings.Join(r.Header.Values("X-Forwarded-For"), ",")
	if xForwardedFor != "" {
		ips := strings.Split(xForwardedFor, ",")

		// Entri paling kanan ditambahkan proxy terakhir (paling dekat dengan kita).
		// IP klien asli berada di posisi len(ips) - trustedProxyCount.
		idx := len(ips) - trustedProxyCount
		if idx < 0 {
			// Rantai proxy lebih pendek daripada yang dikonfigurasi: tidak ada
			// entri yang dijamin ditulis proxy tepercaya. Entri paling kiri di
			// sini sepenuhnya dikendalikan klien, jadi jangan dipercaya.
			return CleanIPAddress(r.RemoteAddr)
		}

		clientIP := strings.TrimSpace(ips[idx])
		if clientIP != "" {
			cleaned := CleanIPAddress(clientIP)
			// Validasi bahwa hasilnya adalah IP address yang sah.
			if net.ParseIP(cleaned) != nil {
				return cleaned
			}
		}
	}

	return CleanIPAddress(r.RemoteAddr)
}

// CleanIPAddress menghapus port number dari IP address string.
// Menangani IPv6 format dengan bracket notation dan IPv4:port format.
// Returns IP tanpa port atau original string jika tidak ada port.
//
// Parameters:
//   - address: IP address string dengan atau tanpa port (contoh: "192.168.1.1:8080" atau "[::1]:8080")
//
// Returns:
//   - string: clean IP address tanpa port
//
// Example:
//
//	CleanIPAddress("192.168.1.1:8080")   // returns "192.168.1.1"
//	CleanIPAddress("[::1]:8080")         // returns "::1"
//	CleanIPAddress("192.168.1.1")        // returns "192.168.1.1"
func CleanIPAddress(address string) string {
	// Handle IPv6 format [::1]:port
	if strings.HasPrefix(address, "[") {
		if closeBracket := strings.Index(address, "]"); closeBracket != -1 {
			return address[1:closeBracket]
		}
	}

	// Handle IPv4 format ip:port
	if host, _, err := net.SplitHostPort(address); err == nil {
		return host
	}

	// Return as-is jika tidak ada port
	return address
}

// ToCamelCase converts snake_case or anything to CamelCase (PascalCase)
// example: create_users -> CreateUsers
func ToCamelCase(s string) string {
	parts := strings.FieldsFunc(s, func(r rune) bool {
		return !unicode.IsLetter(r) && !unicode.IsNumber(r)
	})

	for i, part := range parts {
		if len(part) > 0 {
			runes := []rune(part)
			runes[0] = unicode.ToUpper(runes[0])
			parts[i] = string(runes)
		}
	}

	return strings.Join(parts, "")
}

// GetCookie mengambil nilai cookie dari HTTP request berdasarkan nama.
// Returns empty string jika cookie tidak ditemukan.
//
// Parameters:
//   - r: *http.Request yang berisi cookies
//   - name: nama cookie yang akan diambil
//
// Returns:
//   - string: cookie value, empty string jika tidak ditemukan
//
// Example:
//
//	sessionID := GetCookie(req, "session_id")  // returns cookie value atau ""
func GetCookie(r *http.Request, name string) string {
	cookie, err := r.Cookie(name)
	if err != nil {
		return ""
	}
	return cookie.Value
}

// IsHexChar mengecek apakah byte adalah valid hex digit (0-9, a-f, A-F).
// Berguna untuk validasi hex string seperti hash, color codes, dll.
//
// Parameters:
//   - char: byte character yang akan dicek
//
// Returns:
//   - bool: true jika char adalah hex digit, false sebaliknya
//
// Example:
//
//	IsHexChar('A')  // returns true
//	IsHexChar('5')  // returns true
//	IsHexChar('G')  // returns false
func IsHexChar(char byte) bool {
	return (char >= '0' && char <= '9') ||
		(char >= 'a' && char <= 'f') ||
		(char >= 'A' && char <= 'F')
}

// IsValidDateFormat memvalidasi apakah string adalah valid YYYY-MM-DD format.
// Strict validation: harus exactly 10 characters, hyphens di positions 4 dan 7.
// Tidak mengvalidasi actual date validity (misalnya February 30).
//
// Parameters:
//   - date: date string yang akan divalidasi
//
// Returns:
//   - bool: true jika format valid YYYY-MM-DD, false sebaliknya
//
// Example:
//
//	IsValidDateFormat("2024-01-15")  // returns true
//	IsValidDateFormat("2024-1-15")   // returns false (missing leading zero)
//	IsValidDateFormat("01/15/2024")  // returns false (wrong separator)
func IsValidDateFormat(date string) bool {
	if len(date) != 10 {
		return false
	}

	// Check hyphen positions
	if date[4] != '-' || date[7] != '-' {
		return false
	}

	// Check if all non-hyphen characters are digits
	for i, c := range date {
		if i == 4 || i == 7 {
			continue
		}
		if c < '0' || c > '9' {
			return false
		}
	}

	return true
}

// ContainsRune mengecek apakah string contains any rune yang match predicate function.
// Generic helper untuk custom character checking dengan flexible predicates.
//
// Parameters:
//   - s: string yang akan dicek
//   - predicate: function yang return true untuk matching runes
//
// Returns:
//   - bool: true jika ada rune yang match, false sebaliknya
//
// Example:
//
//	ContainsRune("Hello123", unicode.IsDigit)  // returns true
//	ContainsRune("Hello", unicode.IsDigit)     // returns false
func ContainsRune(s string, predicate func(rune) bool) bool {
	for _, r := range s {
		if predicate(r) {
			return true
		}
	}
	return false
}

// ContainsUppercase mengecek apakah string contains uppercase letters (A-Z).
// Berguna untuk password validation dan character set checking.
//
// Parameters:
//   - s: string yang akan dicek
//
// Returns:
//   - bool: true jika ada uppercase letters, false sebaliknya
//
// Example:
//
//	ContainsUppercase("Hello")    // returns true
//	ContainsUppercase("hello")    // returns false
func ContainsUppercase(s string) bool {
	return ContainsRune(s, unicode.IsUpper)
}

// ContainsLowercase mengecek apakah string contains lowercase letters (a-z).
// Berguna untuk password validation dan character set checking.
//
// Parameters:
//   - s: string yang akan dicek
//
// Returns:
//   - bool: true jika ada lowercase letters, false sebaliknya
//
// Example:
//
//	ContainsLowercase("Hello")    // returns true
//	ContainsLowercase("HELLO")    // returns false
func ContainsLowercase(s string) bool {
	return ContainsRune(s, unicode.IsLower)
}

// ContainsDigit mengecek apakah string contains digits (0-9).
// Berguna untuk password validation dan number checking.
//
// Parameters:
//   - s: string yang akan dicek
//
// Returns:
//   - bool: true jika ada digits, false sebaliknya
//
// Example:
//
//	ContainsDigit("Hello123")    // returns true
//	ContainsDigit("Hello")       // returns false
func ContainsDigit(s string) bool {
	return ContainsRune(s, unicode.IsDigit)
}

// ContainsSpecial mengecek apakah string contains special characters.
// Supported special characters: !@#$%^&*()-_=+[]{}|;:',.<>?/\~`
// Berguna untuk password strength validation.
//
// Parameters:
//   - s: string yang akan dicek
//
// Returns:
//   - bool: true jika ada special characters, false sebaliknya
//
// Example:
//
//	ContainsSpecial("Hello!123")    // returns true
//	ContainsSpecial("Hello123")     // returns false
func ContainsSpecial(s string) bool {
	specialChars := "!@#$%^&*()-_=+[]{}|;:',.<>?/\\~`"
	for _, char := range s {
		if strings.ContainsRune(specialChars, char) {
			return true
		}
	}
	return false
}

// IsSafeHttpMethod mengecek apakah HTTP method adalah safe (tidak mengubah state).
// Safe methods: GET, HEAD, OPTIONS (tidak memiliki side effects).
// Berguna untuk CSRF protection, caching, dan conditional logic.
//
// Parameters:
//   - method: HTTP method string (GET, POST, PUT, DELETE, PATCH, OPTIONS, HEAD)
//
// Returns:
//   - bool: true jika method adalah safe, false sebaliknya
//
// Example:
//
//	IsSafeHttpMethod("GET")     // returns true
//	IsSafeHttpMethod("POST")    // returns false
//	IsSafeHttpMethod("OPTIONS") // returns true
func IsSafeHttpMethod(method string) bool {
	return method == http.MethodGet || method == http.MethodHead || method == http.MethodOptions
}

// PathMatches mengecek apakah path match any pattern dalam list.
// Supports simple glob patterns dengan * wildcard.
// Pattern: "*" cocok semua path, "/webhooks/*" cocok /webhooks/anything.
//
// Parameters:
//   - path: URL path yang akan dicek
//   - patterns: list pattern untuk matching (exact atau glob)
//
// Returns:
//   - bool: true jika path cocok dengan any pattern, false sebaliknya
//
// Example:
//
//	PathMatches("/webhooks/github", []string{"/webhooks/*"})  // returns true
//	PathMatches("/admin", []string{"/admin", "/api/*"})       // returns true
//	PathMatches("/users", []string{"/admin/*"})               // returns false
func PathMatches(path string, patterns []string) bool {
	for _, pattern := range patterns {
		if SimpleGlobMatch(path, pattern) {
			return true
		}
	}
	return false
}

// SimpleGlobMatch mengimplementasikan basic glob matching dengan * wildcard saja.
// Supports exact match dan trailing wildcard pattern.
// Pattern "*" cocok semua path, "/path/*" cocok /path/anything dan /path/anything/nested.
//
// Parameters:
//   - path: URL path yang akan dicek
//   - pattern: glob pattern untuk matching
//
// Returns:
//   - bool: true jika path cocok dengan pattern, false sebaliknya
//
// Example:
//
//	SimpleGlobMatch("/webhooks/github", "/webhooks/*")  // returns true
//	SimpleGlobMatch("/admin/users", "/admin")            // returns false
//	SimpleGlobMatch("/anything", "*")                    // returns true
func SimpleGlobMatch(path, pattern string) bool {
	if pattern == "*" {
		return true
	}

	// Simple glob: /webhooks/* matches /webhooks/any/path
	if strings.HasSuffix(pattern, "/*") {
		prefix := strings.TrimSuffix(pattern, "/*")
		return strings.HasPrefix(path, prefix+"/")
	}

	// Exact match
	return path == pattern
}
