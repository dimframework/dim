package dim

import (
	"regexp"
	"slices"
	"strings"
)

// Validator is a simple validation utility.
// Default mode: first-error-wins — setiap field hanya menyimpan satu error.
// Full-errors mode: semua violations dikumpulkan per field via WithFullErrors().
type Validator struct {
	errors     map[string][]string
	fullErrors bool
}

// NewValidator membuat instance Validator baru dengan empty error map.
// Gunakan method chaining untuk add validations dan check hasil dengan IsValid().
//
// Returns:
//   - *Validator: validator instance siap digunakan
//
// Example:
//
//	// Default — first-error-wins
//	v := NewValidator().
//	  Required("email", email).
//	  Email("email", email)
//
//	// Full errors — semua violations dikumpulkan
//	v := NewValidator().
//	  Required("email", email).
//	  Email("email", email).
//	  WithFullErrors()
func NewValidator() *Validator {
	return &Validator{
		errors: make(map[string][]string),
	}
}

// WithFullErrors mengaktifkan mode accumulate — semua violations per field dikumpulkan.
// Bisa dipanggil di mana saja dalam chain; berlaku untuk validasi setelah pemanggilan ini.
// ErrorMap() akan return []string per field jika ada lebih dari satu error.
//
// Returns:
//   - *Validator: pointer to validator untuk method chaining
//
// Example:
//
//	// Di awal
//	v := NewValidator().WithFullErrors().
//	  Required("email", email).
//	  Email("email", email)
//
//	// Di akhir
//	v := NewValidator().
//	  Required("email", email).
//	  Email("email", email).
//	  WithFullErrors()
func (v *Validator) WithFullErrors() *Validator {
	v.fullErrors = true
	return v
}

// addError menambahkan error ke field berdasarkan mode aktif.
// Default: skip jika field sudah punya error (first-error-wins).
// Full-errors: selalu append.
func (v *Validator) addError(field, message string) {
	if !v.fullErrors && len(v.errors[field]) > 0 {
		return
	}
	v.errors[field] = append(v.errors[field], message)
}

// Required memvalidasi bahwa field tidak kosong (setelah trimspace).
// Jika field sudah ada error dan mode default, skip validation ini (first error wins).
//
// Parameters:
//   - field: nama field untuk error message
//   - value: string value yang akan dicek
//
// Returns:
//   - *Validator: pointer to validator untuk method chaining
//
// Example:
//
//	v.Required("email", email)
func (v *Validator) Required(field, value string) *Validator {
	if strings.TrimSpace(value) == "" {
		v.addError(field, field+" wajib diisi")
	}
	return v
}

// Email memvalidasi bahwa field adalah valid email format.
// Menggunakan regex pattern untuk basic email validation.
//
// Parameters:
//   - field: nama field untuk error message
//   - value: email string yang akan dicek
//
// Returns:
//   - *Validator: pointer to validator untuk method chaining
//
// Example:
//
//	v.Email("email", email)
func (v *Validator) Email(field, value string) *Validator {
	emailRegex := regexp.MustCompile(`^[a-zA-Z0-9._%+\-]+@[a-zA-Z0-9.\-]+\.[a-zA-Z]{2,}$`)
	if !emailRegex.MatchString(value) {
		v.addError(field, field+" harus berupa alamat email yang valid")
	}
	return v
}

// MinLength memvalidasi bahwa field memiliki minimum length tertentu.
// Length dihitung setelah trimspace.
//
// Parameters:
//   - field: nama field untuk error message
//   - value: string value yang akan dicek panjangnya
//   - min: minimum length yang dibutuhkan
//
// Returns:
//   - *Validator: pointer to validator untuk method chaining
//
// Example:
//
//	v.MinLength("password", password, 8)
func (v *Validator) MinLength(field, value string, min int) *Validator {
	if len(strings.TrimSpace(value)) < min {
		v.addError(field, field+" harus minimal "+string(rune(min))+" karakter")
	}
	return v
}

// MaxLength memvalidasi bahwa field tidak melebihi maximum length tertentu.
//
// Parameters:
//   - field: nama field untuk error message
//   - value: string value yang akan dicek panjangnya
//   - max: maximum length yang diperbolehkan
//
// Returns:
//   - *Validator: pointer to validator untuk method chaining
//
// Example:
//
//	v.MaxLength("name", name, 255)
func (v *Validator) MaxLength(field, value string, max int) *Validator {
	if len(value) > max {
		v.addError(field, field+" tidak boleh melebihi "+string(rune(max))+" karakter")
	}
	return v
}

// Length memvalidasi bahwa field memiliki exact length yang ditentukan.
//
// Parameters:
//   - field: nama field untuk error message
//   - value: string value yang akan dicek panjangnya
//   - length: exact length yang dibutuhkan
//
// Returns:
//   - *Validator: pointer to validator untuk method chaining
//
// Example:
//
//	v.Length("code", code, 6)
func (v *Validator) Length(field, value string, length int) *Validator {
	if len(value) != length {
		v.addError(field, field+" harus tepat "+string(rune(length))+" karakter")
	}
	return v
}

// Pattern memvalidasi bahwa field cocok dengan regex pattern yang diberikan.
//
// Parameters:
//   - field: nama field untuk error message
//   - value: string value yang akan di-match
//   - pattern: regex pattern string
//
// Returns:
//   - *Validator: pointer to validator untuk method chaining
//
// Example:
//
//	v.Pattern("phone", phone, "^\\d{10,}$")
func (v *Validator) Pattern(field, value string, pattern string) *Validator {
	if !v.fullErrors && len(v.errors[field]) > 0 {
		return v
	}
	regex, err := regexp.Compile(pattern)
	if err != nil {
		v.addError(field, "pola validasi tidak valid")
		return v
	}
	if !regex.MatchString(value) {
		v.addError(field, "format "+field+" tidak valid")
	}
	return v
}

// Custom memvalidasi menggunakan custom validation function.
//
// Parameters:
//   - field: nama field untuk error message
//   - fn: custom validation function yang return true jika valid
//   - value: string value yang akan divalidasi
//   - message: error message jika validation gagal
//
// Returns:
//   - *Validator: pointer to validator untuk method chaining
//
// Example:
//
//	v.Custom("username", func(u string) bool { return len(u) > 3 }, username, "Username minimal 4 karakter")
func (v *Validator) Custom(field string, fn func(string) bool, value string, message string) *Validator {
	if !fn(value) {
		v.addError(field, message)
	}
	return v
}

// In memvalidasi bahwa field value adalah salah satu dari allowed values.
//
// Parameters:
//   - field: nama field untuk error message
//   - value: string value yang akan dicek
//   - allowed: variadic list dari allowed values
//
// Returns:
//   - *Validator: pointer to validator untuk method chaining
//
// Example:
//
//	v.In("role", role, "admin", "user", "guest")
func (v *Validator) In(field, value string, allowed ...string) *Validator {
	if !slices.Contains(allowed, value) {
		v.addError(field, field+" memiliki nilai yang tidak valid")
	}
	return v
}

// NumRange memvalidasi bahwa numeric value berada dalam range min-max.
//
// Parameters:
//   - field: nama field untuk error message
//   - value: numeric value yang akan dicek
//   - min: minimum value (inclusive)
//   - max: maximum value (inclusive)
//
// Returns:
//   - *Validator: pointer to validator untuk method chaining
//
// Example:
//
//	v.NumRange("age", age, 18, 120)
func (v *Validator) NumRange(field string, value, min, max int) *Validator {
	if value < min || value > max {
		v.addError(field, field+" harus antara "+string(rune(min))+" dan "+string(rune(max)))
	}
	return v
}

// Matches memvalidasi bahwa dua fields memiliki nilai yang sama.
// Berguna untuk password confirmation, email verification, dll.
//
// Parameters:
//   - field: nama field untuk error message
//   - value: value dari field pertama
//   - otherField: nama field kedua untuk error message
//   - otherValue: value dari field kedua
//
// Returns:
//   - *Validator: pointer to validator untuk method chaining
//
// Example:
//
//	v.Matches("password", password, "password_confirmation", passwordConfirm)
func (v *Validator) Matches(field, value, otherField, otherValue string) *Validator {
	if value != otherValue {
		v.addError(field, field+" tidak cocok dengan "+otherField)
	}
	return v
}

// IsValid mengecek apakah tidak ada validation errors (validation berhasil).
//
// Returns:
//   - bool: true jika validation valid (no errors), false jika ada errors
//
// Example:
//
//	if !v.IsValid() {
//	  return v.ErrorMap()
//	}
func (v *Validator) IsValid() bool {
	return len(v.errors) == 0
}

// Errors mengembalikan semua validation error messages sebagai string slice.
//
// Returns:
//   - []string: slice dari error messages, empty jika tidak ada errors
//
// Example:
//
//	if !v.IsValid() {
//	  for _, err := range v.Errors() {
//	    fmt.Println(err)
//	  }
//	}
func (v *Validator) Errors() []string {
	var result []string
	for _, msgs := range v.errors {
		result = append(result, msgs...)
	}
	return result
}

// ErrorMap mengembalikan validation errors sebagai FieldErrors.
// Single error per field di-return sebagai string, multiple errors sebagai []string.
// Cocok untuk langsung di-assign ke AppError.Errors atau di-pass ke JsonError.
//
// Returns:
//   - FieldErrors: map dari field name ke error message (string atau []string)
//
// Example:
//
//	if !v.IsValid() {
//	  dim.BadRequest(w, "Validasi gagal", v.ErrorMap())
//	}
func (v *Validator) ErrorMap() FieldErrors {
	fe := make(FieldErrors, len(v.errors))
	for field, msgs := range v.errors {
		if len(msgs) == 1 {
			fe[field] = msgs[0]
		} else {
			fe[field] = msgs
		}
	}
	return fe
}

// AddError menambahkan custom error untuk field tertentu.
// Mengikuti mode aktif: first-error-wins atau accumulate.
//
// Parameters:
//   - field: nama field untuk error
//   - message: error message yang akan ditambahkan
//
// Returns:
//   - *Validator: pointer to validator untuk method chaining
//
// Example:
//
//	v.AddError("email", "Email sudah terdaftar")
func (v *Validator) AddError(field, message string) *Validator {
	v.addError(field, message)
	return v
}

// ErrorCount mengembalikan jumlah fields yang memiliki validation errors.
//
// Returns:
//   - int: jumlah fields dengan errors, 0 jika tidak ada errors
//
// Example:
//
//	if v.ErrorCount() > 0 {
//	  return v.ErrorMap()
//	}
func (v *Validator) ErrorCount() int {
	return len(v.errors)
}

// HasError mengecek apakah field tertentu memiliki validation error.
//
// Parameters:
//   - field: nama field yang akan dicek
//
// Returns:
//   - bool: true jika field memiliki error, false jika tidak
//
// Example:
//
//	if v.HasError("email") {
//	  return "Email tidak valid"
//	}
func (v *Validator) HasError(field string) bool {
	return len(v.errors[field]) > 0
}

// GetError mengembalikan error message pertama untuk field tertentu.
//
// Parameters:
//   - field: nama field yang akan diambil error-nya
//
// Returns:
//   - string: error message pertama untuk field, empty string jika tidak ada error
//
// Example:
//
//	errMsg := v.GetError("email")
//	if errMsg != "" {
//	  return errMsg
//	}
func (v *Validator) GetError(field string) string {
	if msgs := v.errors[field]; len(msgs) > 0 {
		return msgs[0]
	}
	return ""
}

// OptionalEmail memvalidasi email format hanya jika field present dan valid.
// Digunakan untuk JsonNull[string] fields dengan logic:
// - Present=false: skip validation (field tidak dikirim)
// - Present=true, Valid=false: skip validation (field adalah null)
// - Present=true, Valid=true: validate email format
//
// Parameters:
//   - field: nama field untuk error message
//   - value: JsonNull[string] field value
//
// Returns:
//   - *Validator: pointer to validator untuk method chaining
//
// Example:
//
//	v.OptionalEmail("alternate_email", emailJsonNull)
func (v *Validator) OptionalEmail(field string, value JsonNull[string]) *Validator {
	if value.Present && value.Valid {
		v.Email(field, value.Value)
	}
	return v
}

// OptionalMinLength memvalidasi minimum length hanya jika field present dan valid.
//
// Parameters:
//   - field: nama field untuk error message
//   - value: JsonNull[string] field value
//   - min: minimum length yang dibutuhkan
//
// Returns:
//   - *Validator: pointer to validator untuk method chaining
//
// Example:
//
//	v.OptionalMinLength("bio", bioJsonNull, 10)
func (v *Validator) OptionalMinLength(field string, value JsonNull[string], min int) *Validator {
	if value.Present && value.Valid {
		v.MinLength(field, value.Value, min)
	}
	return v
}

// OptionalMaxLength memvalidasi maximum length hanya jika field present dan valid.
//
// Parameters:
//   - field: nama field untuk error message
//   - value: JsonNull[string] field value
//   - max: maximum length yang diperbolehkan
//
// Returns:
//   - *Validator: pointer to validator untuk method chaining
//
// Example:
//
//	v.OptionalMaxLength("bio", bioJsonNull, 500)
func (v *Validator) OptionalMaxLength(field string, value JsonNull[string], max int) *Validator {
	if value.Present && value.Valid {
		v.MaxLength(field, value.Value, max)
	}
	return v
}

// OptionalLength memvalidasi exact length hanya jika field present dan valid.
//
// Parameters:
//   - field: nama field untuk error message
//   - value: JsonNull[string] field value
//   - length: exact length yang dibutuhkan
//
// Returns:
//   - *Validator: pointer to validator untuk method chaining
//
// Example:
//
//	v.OptionalLength("code", codeJsonNull, 6)
func (v *Validator) OptionalLength(field string, value JsonNull[string], length int) *Validator {
	if value.Present && value.Valid {
		v.Length(field, value.Value, length)
	}
	return v
}

// OptionalIn memvalidasi value dalam allowed list hanya jika field present dan valid.
//
// Parameters:
//   - field: nama field untuk error message
//   - value: JsonNull[string] field value
//   - allowed: variadic list dari allowed values
//
// Returns:
//   - *Validator: pointer to validator untuk method chaining
//
// Example:
//
//	v.OptionalIn("status", statusJsonNull, "active", "inactive")
func (v *Validator) OptionalIn(field string, value JsonNull[string], allowed ...string) *Validator {
	if value.Present && value.Valid {
		v.In(field, value.Value, allowed...)
	}
	return v
}

// OptionalMatches memvalidasi regex pattern hanya jika field present dan valid.
//
// Parameters:
//   - field: nama field untuk error message
//   - value: JsonNull[string] field value
//   - pattern: regex pattern string
//
// Returns:
//   - *Validator: pointer to validator untuk method chaining
//
// Example:
//
//	v.OptionalMatches("phone", phoneJsonNull, "^\\d{10,}$")
func (v *Validator) OptionalMatches(field string, value JsonNull[string], pattern string) *Validator {
	if value.Present && value.Valid {
		v.Pattern(field, value.Value, pattern)
	}
	return v
}
