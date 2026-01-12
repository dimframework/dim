package dim

import (
	"fmt"
	"strings"

	"golang.org/x/crypto/bcrypt"
)

const (
	// MinPasswordLength is the minimum required password length
	MinPasswordLength = 8
	// BcryptCost is the bcrypt cost factor
	BcryptCost = 12
)

// HashPassword melakukan hash password menggunakan bcrypt algorithm.
// Menggunakan BcryptCost constant untuk set hashing difficulty level.
//
// Parameters:
//   - password: plaintext password yang akan di-hash
//
// Returns:
//   - string: hashed password yang bisa disimpan di database
//   - error: error jika hashing gagal
//
// Example:
//
//	hashedPassword, err := HashPassword("myPassword123!")
//	if err != nil {
//	  return err
//	}
func HashPassword(password string) (string, error) {
	hash, err := bcrypt.GenerateFromPassword([]byte(password), BcryptCost)
	if err != nil {
		return "", fmt.Errorf("failed to hash password: %w", err)
	}
	return string(hash), nil
}

// VerifyPassword memverifikasi plaintext password terhadap hash yang tersimpan.
// Menggunakan bcrypt untuk aman time-constant comparison.
//
// Parameters:
//   - hashedPassword: hashed password dari database
//   - password: plaintext password untuk diverifikasi
//
// Returns:
//   - error: error jika password tidak cocok dengan hash
//
// Example:
//
//	err := VerifyPassword(storedHash, providedPassword)
//	if err != nil {
//	  return "password tidak valid"
//	}
func VerifyPassword(hashedPassword, password string) error {
	return bcrypt.CompareHashAndPassword([]byte(hashedPassword), []byte(password))
}

// PasswordValidator provides password validation utilities
type PasswordValidator struct {
	minLength    int
	requireUpper bool
	requireLower bool
	requireDigit bool
	requireSpec  bool
}

// NewPasswordValidator membuat PasswordValidator baru dengan default settings.
// Default settings: minLength=8, require uppercase, lowercase, digit, dan special char.
//
// Returns:
//   - *PasswordValidator: validator instance dengan default rules
//
// Example:
//
//	validator := NewPasswordValidator()
//	err := validator.Validate(password)
func NewPasswordValidator() *PasswordValidator {
	return &PasswordValidator{
		minLength:    MinPasswordLength,
		requireUpper: true,
		requireLower: true,
		requireDigit: true,
		requireSpec:  true,
	}
}

// SetMinLength sets the minimum password length
func (pv *PasswordValidator) SetMinLength(length int) *PasswordValidator {
	pv.minLength = length
	return pv
}

// RequireUppercase sets whether uppercase letters are required
func (pv *PasswordValidator) RequireUppercase(required bool) *PasswordValidator {
	pv.requireUpper = required
	return pv
}

// RequireLowercase sets whether lowercase letters are required
func (pv *PasswordValidator) RequireLowercase(required bool) *PasswordValidator {
	pv.requireLower = required
	return pv
}

// RequireDigit sets whether digits are required
func (pv *PasswordValidator) RequireDigit(required bool) *PasswordValidator {
	pv.requireDigit = required
	return pv
}

// RequireSpecial sets whether special characters are required
func (pv *PasswordValidator) RequireSpecial(required bool) *PasswordValidator {
	pv.requireSpec = required
	return pv
}

// Validate memvalidasi password terhadap semua configured rules.
// Return error dengan detail field error jika validasi gagal.
//
// Parameters:
//   - password: password string yang akan divalidasi
//
// Returns:
//   - error: AppError dengan field errors jika ada rule yang tidak terpenuhi
//
// Example:
//
//	err := validator.Validate("MyPassword123!")
//	if err != nil {
//	  // handle validation error
//	}
func (pv *PasswordValidator) Validate(password string) error {
	password = strings.TrimSpace(password)

	if len(password) < pv.minLength {
		return NewAppError(
			"Validasi kata sandi gagal",
			400,
		).WithFieldError("password", fmt.Sprintf("Kata sandi harus minimal %d karakter", pv.minLength))
	}

	if pv.requireUpper && !ContainsUppercase(password) {
		return NewAppError(
			"Validasi kata sandi gagal",
			400,
		).WithFieldError("password", "Kata sandi harus mengandung minimal satu huruf besar")
	}

	if pv.requireLower && !ContainsLowercase(password) {
		return NewAppError(
			"Validasi kata sandi gagal",
			400,
		).WithFieldError("password", "Kata sandi harus mengandung minimal satu huruf kecil")
	}

	if pv.requireDigit && !ContainsDigit(password) {
		return NewAppError(
			"Validasi kata sandi gagal",
			400,
		).WithFieldError("password", "Kata sandi harus mengandung minimal satu angka")
	}

	if pv.requireSpec && !ContainsSpecial(password) {
		return NewAppError(
			"Validasi kata sandi gagal",
			400,
		).WithFieldError("password", "Kata sandi harus mengandung minimal satu karakter spesial")
	}

	return nil
}

// ValidatePasswordStrength memvalidasi password strength menggunakan default rules.
// Merupakan convenience function yang membuat validator dan langsung validate.
//
// Parameters:
//   - password: password string yang akan divalidasi
//
// Returns:
//   - error: AppError jika password tidak memenuhi strength requirements
//
// Example:
//
//	err := ValidatePasswordStrength(password)
func ValidatePasswordStrength(password string) error {
	return NewPasswordValidator().Validate(password)
}
