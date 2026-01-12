package dim

import (
	"testing"
)

func TestValidatorRequired(t *testing.T) {
	v := NewValidator()
	v.Required("email", "")
	v.Required("username", "user123")

	if !v.HasError("email") {
		t.Errorf("Required should add error for empty field")
	}

	if v.HasError("username") {
		t.Errorf("Required should not add error for non-empty field")
	}
}

func TestValidatorEmail(t *testing.T) {
	tests := []struct {
		email   string
		wantErr bool
	}{
		{"test@example.com", false},
		{"user+tag@domain.co.uk", false},
		{"invalid.email", true},
		{"@example.com", true},
		{"test@", true},
		{"", true},
	}

	for _, tt := range tests {
		v := NewValidator()
		v.Email("email", tt.email)
		if (v.HasError("email")) != tt.wantErr {
			t.Errorf("Email(%s) wantErr %v, got %v", tt.email, tt.wantErr, v.HasError("email"))
		}
	}
}

func TestValidatorMinLength(t *testing.T) {
	v := NewValidator()
	v.MinLength("password", "abc", 8)

	if !v.HasError("password") {
		t.Errorf("MinLength should add error for short string")
	}

	v2 := NewValidator()
	v2.MinLength("password", "longenough", 8)
	if v2.HasError("password") {
		t.Errorf("MinLength should not add error for long enough string")
	}
}

func TestValidatorMaxLength(t *testing.T) {
	v := NewValidator()
	v.MaxLength("username", "verylongusername", 10)

	if !v.HasError("username") {
		t.Errorf("MaxLength should add error for long string")
	}

	v2 := NewValidator()
	v2.MaxLength("username", "short", 10)
	if v2.HasError("username") {
		t.Errorf("MaxLength should not add error for short string")
	}
}

func TestValidatorLength(t *testing.T) {
	v := NewValidator()
	v.Length("code", "abc", 5)

	if !v.HasError("code") {
		t.Errorf("Length should add error for wrong length")
	}

	v2 := NewValidator()
	v2.Length("code", "abcde", 5)
	if v2.HasError("code") {
		t.Errorf("Length should not add error for correct length")
	}
}

func TestValidatorChaining(t *testing.T) {
	v := NewValidator().
		Required("email", "").
		Email("email", "invalid")

	// With single-error-per-field design:
	// - Required() adds error to "email"
	// - Email() is skipped because "email" already has error
	// - Result: 1 error
	if v.ErrorCount() != 1 {
		t.Errorf("chaining should create 1 error (only first check per field), got %d", v.ErrorCount())
	}

	// Verify the field has the Required() error
	if v.GetError("email") != "email wajib diisi" {
		t.Errorf("Expected Required error, got: %s", v.GetError("email"))
	}
}

func TestValidatorIsValid(t *testing.T) {
	v := NewValidator()
	if !v.IsValid() {
		t.Errorf("empty validator should be valid")
	}

	v.Required("field", "")
	if v.IsValid() {
		t.Errorf("validator with error should not be valid")
	}
}

func TestValidatorIn(t *testing.T) {
	v := NewValidator()
	v.In("status", "invalid", "active", "inactive", "pending")

	if !v.HasError("status") {
		t.Errorf("In should add error for value not in list")
	}

	v2 := NewValidator()
	v2.In("status", "active", "active", "inactive", "pending")
	if v2.HasError("status") {
		t.Errorf("In should not add error for value in list")
	}
}

func TestValidatorMatches(t *testing.T) {
	v := NewValidator()
	v.Matches("password", "pass123", "password_confirm", "pass456")

	if !v.HasError("password") {
		t.Errorf("Matches should add error for non-matching values")
	}

	v2 := NewValidator()
	v2.Matches("password", "pass123", "password_confirm", "pass123")
	if v2.HasError("password") {
		t.Errorf("Matches should not add error for matching values")
	}
}

func TestValidatorAddError(t *testing.T) {
	v := NewValidator()
	v.AddError("custom", "custom error message")

	if !v.HasError("custom") {
		t.Errorf("AddError should add error")
	}

	if v.GetError("custom") != "custom error message" {
		t.Errorf("GetError should return the error message")
	}
}

func TestValidatorErrors(t *testing.T) {
	v := NewValidator()
	v.Required("email", "").
		Required("username", "").
		Email("email", "invalid")

	errors := v.Errors()

	// With single-error-per-field design:
	// - "email" has error from Required() (Email() is skipped because field already has error)
	// - "username" has error from Required()
	// - Total: 2 errors
	if len(errors) != 2 {
		t.Errorf("Errors should return 2 errors (one per field), got %d", len(errors))
	}

	// Verify that ErrorMap has exactly 2 entries
	errorMap := v.ErrorMap()
	if len(errorMap) != 2 {
		t.Errorf("ErrorMap should have 2 fields, got %d", len(errorMap))
	}

	// Verify specific fields have errors
	if !v.HasError("email") || !v.HasError("username") {
		t.Errorf("Both email and username should have errors")
	}
}

func TestValidatorErrorCount(t *testing.T) {
	v := NewValidator()
	if v.ErrorCount() != 0 {
		t.Errorf("ErrorCount should be 0 for new validator")
	}

	v.Required("field1", "").
		Required("field2", "")

	if v.ErrorCount() != 2 {
		t.Errorf("ErrorCount should be 2, got %d", v.ErrorCount())
	}
}

func TestOptionalEmailNotPresent(t *testing.T) {
	v := NewValidator()
	jn := JsonNull[string]{
		Value:   "",
		Valid:   false,
		Present: false, // Field not sent
	}

	v.OptionalEmail("email", jn)

	if !v.IsValid() {
		t.Errorf("OptionalEmail should not validate when field not present")
	}
}

func TestOptionalEmailNull(t *testing.T) {
	v := NewValidator()
	jn := JsonNull[string]{
		Value:   "",
		Valid:   false,
		Present: true, // Field sent as null
	}

	v.OptionalEmail("email", jn)

	if !v.IsValid() {
		t.Errorf("OptionalEmail should not validate when field is null")
	}
}

func TestOptionalEmailValid(t *testing.T) {
	v := NewValidator()
	jn := JsonNull[string]{
		Value:   "test@example.com",
		Valid:   true,
		Present: true,
	}

	v.OptionalEmail("email", jn)

	if !v.IsValid() {
		t.Errorf("OptionalEmail should validate valid email")
	}
}

func TestOptionalEmailInvalid(t *testing.T) {
	v := NewValidator()
	jn := JsonNull[string]{
		Value:   "invalid-email",
		Valid:   true,
		Present: true,
	}

	v.OptionalEmail("email", jn)

	if v.IsValid() {
		t.Errorf("OptionalEmail should fail for invalid email")
	}

	if !v.HasError("email") {
		t.Errorf("OptionalEmail should set error for invalid email")
	}
}

func TestOptionalMinLength(t *testing.T) {
	tests := []struct {
		name    string
		value   JsonNull[string]
		min     int
		wantErr bool
	}{
		{
			name: "not_present",
			value: JsonNull[string]{
				Present: false,
			},
			min:     3,
			wantErr: false,
		},
		{
			name: "null",
			value: JsonNull[string]{
				Valid:   false,
				Present: true,
			},
			min:     3,
			wantErr: false,
		},
		{
			name: "valid_meets_min",
			value: JsonNull[string]{
				Value:   "hello",
				Valid:   true,
				Present: true,
			},
			min:     3,
			wantErr: false,
		},
		{
			name: "valid_below_min",
			value: JsonNull[string]{
				Value:   "hi",
				Valid:   true,
				Present: true,
			},
			min:     3,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			v := NewValidator()
			v.OptionalMinLength("field", tt.value, tt.min)

			if (v.HasError("field")) != tt.wantErr {
				t.Errorf("OptionalMinLength error = %v, want %v", v.HasError("field"), tt.wantErr)
			}
		})
	}
}

func TestOptionalMaxLength(t *testing.T) {
	tests := []struct {
		name    string
		value   JsonNull[string]
		max     int
		wantErr bool
	}{
		{
			name: "not_present",
			value: JsonNull[string]{
				Present: false,
			},
			max:     3,
			wantErr: false,
		},
		{
			name: "null",
			value: JsonNull[string]{
				Valid:   false,
				Present: true,
			},
			max:     3,
			wantErr: false,
		},
		{
			name: "valid_within_max",
			value: JsonNull[string]{
				Value:   "hi",
				Valid:   true,
				Present: true,
			},
			max:     10,
			wantErr: false,
		},
		{
			name: "valid_exceeds_max",
			value: JsonNull[string]{
				Value:   "this is too long",
				Valid:   true,
				Present: true,
			},
			max:     5,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			v := NewValidator()
			v.OptionalMaxLength("field", tt.value, tt.max)

			if (v.HasError("field")) != tt.wantErr {
				t.Errorf("OptionalMaxLength error = %v, want %v", v.HasError("field"), tt.wantErr)
			}
		})
	}
}

func TestOptionalIn(t *testing.T) {
	tests := []struct {
		name    string
		value   JsonNull[string]
		allowed []string
		wantErr bool
	}{
		{
			name: "not_present",
			value: JsonNull[string]{
				Present: false,
			},
			allowed: []string{"admin", "user"},
			wantErr: false,
		},
		{
			name: "null",
			value: JsonNull[string]{
				Valid:   false,
				Present: true,
			},
			allowed: []string{"admin", "user"},
			wantErr: false,
		},
		{
			name: "valid_in_list",
			value: JsonNull[string]{
				Value:   "admin",
				Valid:   true,
				Present: true,
			},
			allowed: []string{"admin", "user"},
			wantErr: false,
		},
		{
			name: "valid_not_in_list",
			value: JsonNull[string]{
				Value:   "superuser",
				Valid:   true,
				Present: true,
			},
			allowed: []string{"admin", "user"},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			v := NewValidator()
			v.OptionalIn("role", tt.value, tt.allowed...)

			if (v.HasError("role")) != tt.wantErr {
				t.Errorf("OptionalIn error = %v, want %v", v.HasError("role"), tt.wantErr)
			}
		})
	}
}
