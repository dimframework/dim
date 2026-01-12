package dim

import "fmt"

// AppError represents an application error with optional field-specific validation errors
type AppError struct {
	Message    string            `json:"message"`
	StatusCode int               `json:"-"`
	Errors     map[string]string `json:"errors,omitempty"`
}

// Error mengimplementasikan error interface.
// Mengembalikan string representation dari error dengan message dan field errors jika ada.
// Format: "message" atau "message: {field: error_message, ...}" jika ada field errors.
//
// Returns:
//   - string: error message string
//
// Example:
//
//	appErr := NewAppError("Validasi gagal", 400)
//	appErr.WithFieldError("email", "Email tidak valid")
//	fmt.Println(appErr.Error())  // Output: Validasi gagal: map[email:Email tidak valid]
func (e *AppError) Error() string {
	if len(e.Errors) > 0 {
		return fmt.Sprintf("%s: %v", e.Message, e.Errors)
	}
	return e.Message
}

// NewAppError membuat AppError baru dengan message dan HTTP status code.
// Status code digunakan untuk menentukan HTTP response status saat error di-return ke client.
// Useful untuk error handling yang consistent dengan HTTP semantics.
//
// Parameters:
//   - message: error message string dalam bahasa Indonesia
//   - statusCode: HTTP status code (contoh: 400, 401, 404, 500)
//
// Returns:
//   - *AppError: AppError instance dengan empty field errors
//
// Example:
//
//	appErr := NewAppError("Validasi gagal", 400)
//	appErr.WithFieldError("email", "Email harus valid")
//	return appErr
func NewAppError(message string, statusCode int) *AppError {
	return &AppError{
		Message:    message,
		StatusCode: statusCode,
		Errors:     make(map[string]string),
	}
}

// WithFieldError menambahkan field-specific error ke AppError.
// Berguna untuk validation errors yang related ke specific fields.
// Mendukung method chaining untuk menambahkan multiple field errors.
// Jika field sudah ada, akan overwrite dengan message baru.
//
// Parameters:
//   - field: nama field yang memiliki error
//   - message: error message untuk field ini
//
// Returns:
//   - *AppError: pointer to AppError untuk method chaining
//
// Example:
//
//	appErr := NewAppError("Validasi gagal", 400).
//	  WithFieldError("email", "Email harus valid").
//	  WithFieldError("password", "Password minimal 8 karakter")
func (e *AppError) WithFieldError(field, message string) *AppError {
	if e.Errors == nil {
		e.Errors = make(map[string]string)
	}
	e.Errors[field] = message
	return e
}

// WithFieldErrors menambahkan multiple field-specific errors ke AppError sekaligus.
// Convenience function untuk menambahkan banyak field errors dalam satu call.
// Mendukung method chaining untuk kombinasi dengan WithFieldError.
// Jika fields sudah ada, akan overwrite dengan messages baru.
//
// Parameters:
//   - errors: map[string]string dari field names ke error messages
//
// Returns:
//   - *AppError: pointer to AppError untuk method chaining
//
// Example:
//
//	appErr := NewAppError("Validasi gagal", 400).
//	  WithFieldErrors(map[string]string{
//	    "email": "Email harus valid",
//	    "password": "Password minimal 8 karakter",
//	  })
func (e *AppError) WithFieldErrors(errors map[string]string) *AppError {
	if e.Errors == nil {
		e.Errors = make(map[string]string)
	}
	for field, message := range errors {
		e.Errors[field] = message
	}
	return e
}

// Common error instances
var (
	ErrBadRequest          = NewAppError("Permintaan tidak valid", 400)
	ErrValidation          = NewAppError("Validasi gagal", 400)
	ErrUnauthorized        = NewAppError("Tidak terotorisasi", 401)
	ErrForbidden           = NewAppError("Dilarang", 403)
	ErrNotFound            = NewAppError("Tidak ditemukan", 404)
	ErrConflict            = NewAppError("Konflik", 409)
	ErrInternalServerError = NewAppError("Kesalahan server internal", 500)
)

// IsAppError mengecek apakah error adalah AppError instance.
// Berguna untuk type checking sebelum mengakses AppError-specific fields.
// Gunakan sebelum AsAppError untuk type assertion yang aman.
//
// Parameters:
//   - err: error yang akan di-check tipenya
//
// Returns:
//   - bool: true jika error adalah AppError, false jika tipe lain
//
// Example:
//
//	if IsAppError(err) {
//	  appErr, _ := AsAppError(err)
//	  JsonAppError(w, appErr)
//	}
func IsAppError(err error) bool {
	_, ok := err.(*AppError)
	return ok
}

// AsAppError mengkonversi error menjadi AppError jika possible.
// Type-safe conversion dengan ok flag untuk checking apakah conversion berhasil.
// Returns nil dan false jika error bukan AppError type.
// Gunakan dengan IsAppError untuk safe type conversion.
//
// Parameters:
//   - err: error yang akan dikonversi menjadi AppError
//
// Returns:
//   - *AppError: AppError pointer jika conversion berhasil, nil jika tidak
//   - bool: true jika conversion berhasil, false jika error bukan AppError type
//
// Example:
//
//	appErr, ok := AsAppError(err)
//	if ok {
//	  // appErr is safe to use
//	  appErr.WithFieldError("field", "error message")
//	}
func AsAppError(err error) (*AppError, bool) {
	appErr, ok := err.(*AppError)
	return appErr, ok
}
