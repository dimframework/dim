package dim

import (
	"bufio"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

// GetEnv mengambil environment variable berdasarkan key.
// Returns empty string jika variable tidak ada.
//
// Parameters:
//   - key: nama environment variable yang akan diambil
//
// Returns:
//   - string: value dari environment variable, empty string jika tidak ada
//
// Example:
//
//	port := GetEnv("PORT")  // jika PORT=8080, returns "8080"
func GetEnv(key string) string {
	return os.Getenv(key)
}

// GetEnvOrDefault mengambil environment variable atau return default value jika tidak ada.
// Useful untuk provide fallback values untuk configuration.
//
// Parameters:
//   - key: nama environment variable
//   - defaultValue: default value jika variable tidak ada atau empty
//
// Returns:
//   - string: value dari environment variable atau defaultValue
//
// Example:
//
//	port := GetEnvOrDefault("PORT", "8080")  // returns "8080" jika PORT tidak set
//	dbHost := GetEnvOrDefault("DB_HOST", "localhost")
func GetEnvOrDefault(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

// ParseEnvDuration mengurai string durasi dari variabel lingkungan.
//
// Parameters:
//   - value: Nilai string untuk diurai (misalnya, "30s", "15m", "1h").
//
// Returns:
//   - time.Duration: Nilai durasi yang diurai. Mengembalikan 0 jika input kosong.
//   - error: Error jika string durasi tidak valid.
//
// Example:
//
//	timeout, err := ParseEnvDuration("30s") // returns 30 * time.Second, nil
//	invalid, err := ParseEnvDuration("abc") // returns 0, error
func ParseEnvDuration(value string) (time.Duration, error) {
	if value == "" {
		return 0, nil
	}
	d, err := time.ParseDuration(value)
	if err != nil {
		return 0, fmt.Errorf("invalid duration value: %q", value)
	}
	return d, nil
}

// ParseEnvBool mengurai string boolean dari variabel lingkungan.
// Nilai yang dikenali sebagai `true` (tidak case-sensitive) adalah "true", "yes", "1", "on".
// Semua nilai lain dianggap `false`.
//
// Parameters:
//   - s: Nilai string untuk diurai.
//
// Returns:
//   - bool: Nilai boolean yang diurai.
//
// Example:
//
//	debugMode := ParseEnvBool("true")   // returns true
//	featureOn := ParseEnvBool("1")      // returns true
//	disabled := ParseEnvBool("false") // returns false
func ParseEnvBool(s string) bool {
	s = strings.ToLower(strings.TrimSpace(s))
	return s == "true" || s == "yes" || s == "1" || s == "on"
}

// ParseEnvInt mengurai string integer dari variabel lingkungan.
// Spasi di awal dan akhir akan diabaikan.
//
// Parameters:
//   - value: Nilai string untuk diurai.
//
// Returns:
//   - int: Nilai integer yang diurai. Mengembalikan 0 jika input kosong.
//   - error: Error jika string integer tidak valid.
//
// Example:
//
//	port, err := ParseEnvInt(" 8080 ") // returns 8080, nil
//	invalid, err := ParseEnvInt("abc")  // returns 0, error
func ParseEnvInt(value string) (int, error) {
	value = strings.TrimSpace(value)
	if value == "" {
		return 0, nil
	}
	i, err := strconv.Atoi(value)
	if err != nil {
		return 0, fmt.Errorf("invalid integer value: %q", value)
	}
	return i, nil
}

// LoadEnvFile memuat variabel lingkungan dari file .env yang ditentukan.
// Fungsi ini membaca file baris per baris, mengabaikan komentar dan baris kosong.
// Variabel lingkungan hanya akan diatur jika belum ada nilainya.
//
// Parameters:
//   - filename: Path lengkap ke file .env.
//
// Returns:
//   - error: Error jika file tidak dapat dibuka atau dibaca. Mengembalikan `nil` jika file tidak ada.
//
// Example:
//
//	if err := LoadEnvFile(".env.development"); err != nil {
//	    log.Fatalf("Gagal memuat file .env: %v", err)
//	}
func LoadEnvFile(filename string) error {
	// Check if file exists
	if _, err := os.Stat(filename); err != nil {
		if os.IsNotExist(err) {
			slog.Info("env file not found", "path", filename)
			return nil
		}
		return err
	}

	file, err := os.Open(filename)
	if err != nil {
		return err
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	lineNum := 0

	for scanner.Scan() {
		lineNum++
		line := strings.TrimSpace(scanner.Text())

		// Skip empty lines and comments
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		// Parse key=value
		parts := strings.SplitN(line, "=", 2)
		if len(parts) != 2 {
			slog.Warn("invalid env line", "file", filename, "line", lineNum)
			continue
		}

		key := strings.TrimSpace(parts[0])
		value := strings.TrimSpace(parts[1])

		// Remove quotes if present
		if len(value) >= 2 && ((value[0] == '"' && value[len(value)-1] == '"') ||
			(value[0] == '\'' && value[len(value)-1] == '\'')) {
			value = value[1 : len(value)-1]
		}

		// Only set if not already set
		if _, exists := os.LookupEnv(key); !exists {
			os.Setenv(key, value)
		}
	}

	if err := scanner.Err(); err != nil {
		return err
	}

	slog.Info("env file loaded successfully", "path", filename)
	return nil
}

// LoadEnvFileFromPath memuat variabel lingkungan dari file .env di dalam direktori yang ditentukan.
// Ini adalah fungsi pembantu yang membangun path ke .env dan memanggil LoadEnvFile.
//
// Parameters:
//   - dir: Direktori tempat file .env berada.
//
// Returns:
//   - error: Error yang sama seperti LoadEnvFile.
//
// Example:
//
//	// Memuat .env dari direktori saat ini
//	LoadEnvFileFromPath(".")
//
//	// Memuat .env dari direktori konfigurasi
//	LoadEnvFileFromPath("./config")
func LoadEnvFileFromPath(dir string) error {
	envPath := filepath.Join(dir, ".env")
	return LoadEnvFile(envPath)
}
