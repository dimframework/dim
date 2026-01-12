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

// GetClientIP mengekstrak client IP address dari HTTP request.
// Mengecek X-Forwarded-For, X-Real-IP, X-Forwarded headers (untuk proxy scenarios).
// Falls back ke RemoteAddr jika headers tidak ada.
// Menangani IPv4 dan IPv6 formats dengan port numbers.
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
	xForwardedFor := r.Header.Get("X-Forwarded-For")

	if xForwardedFor != "" {
		ips := strings.Split(strings.TrimSpace(xForwardedFor), ",")

		if len(ips) > 0 {
			clientIP := strings.TrimSpace(ips[0])

			if clientIP != "" {
				return CleanIPAddress(clientIP)
			}
		}
	}

	if realIP := r.Header.Get("X-Real-IP"); realIP != "" {
		return CleanIPAddress(strings.TrimSpace(realIP))
	}

	if forwardedFor := r.Header.Get("X-Forwarded"); forwardedFor != "" {
		return CleanIPAddress(forwardedFor)
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
