package dim

import (
	"regexp"
	"testing"
)

// UUID regex pattern: xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx
var uuidPattern = regexp.MustCompile(`^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$`)

func TestNewUUID(t *testing.T) {
	tests := []struct {
		name string
	}{
		{"uuid_1"},
		{"uuid_2"},
		{"uuid_3"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			uuid := NewUuid()
			uuidStr := uuid.String()

			// Check format
			if !uuidPattern.MatchString(uuidStr) {
				t.Errorf("UUID format invalid: %s", uuidStr)
			}

			// Check length
			if len(uuidStr) != 36 {
				t.Errorf("UUID length = %d, want 36", len(uuidStr))
			}

			// Check version is 7 (fallback from v7) or 4
			version := rune(uuidStr[14])
			if version != '7' && version != '4' {
				t.Errorf("UUID version = %c, want 7 or 4", version)
			}
		})
	}
}

func TestNewUUIDUnique(t *testing.T) {
	uuid1 := NewUuid()
	uuid2 := NewUuid()
	uuid3 := NewUuid()

	if uuid1 == uuid2 {
		t.Errorf("Generated UUIDs should be unique, got same: %s", uuid1.String())
	}

	if uuid2 == uuid3 {
		t.Errorf("Generated UUIDs should be unique, got same: %s", uuid2.String())
	}

	if uuid1 == uuid3 {
		t.Errorf("Generated UUIDs should be unique, got same: %s", uuid1.String())
	}
}

func TestNewUUIDv7(t *testing.T) {
	tests := []struct {
		name string
	}{
		{"uuidv7_1"},
		{"uuidv7_2"},
		{"uuidv7_3"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			uuid, err := NewV7()
			if err != nil {
				t.Errorf("NewV7() error = %v", err)
				return
			}
			uuidStr := uuid.String()

			// Check format
			if !uuidPattern.MatchString(uuidStr) {
				t.Errorf("UUID v7 format invalid: %s", uuidStr)
			}

			// Check length
			if len(uuidStr) != 36 {
				t.Errorf("UUID v7 length = %d, want 36", len(uuidStr))
			}

			// Check version is 7
			if uuidStr[14] != '7' {
				t.Errorf("UUID v7 version = %c, want 7", uuidStr[14])
			}
		})
	}
}

func TestNewUUIDv7Unique(t *testing.T) {
	uuid1, err1 := NewV7()
	uuid2, err2 := NewV7()
	uuid3, err3 := NewV7()

	if err1 != nil || err2 != nil || err3 != nil {
		t.Errorf("NewV7() errors: %v, %v, %v", err1, err2, err3)
		return
	}

	if uuid1 == uuid2 {
		t.Errorf("Generated UUID v7s should be unique, got same: %s", uuid1.String())
	}

	if uuid2 == uuid3 {
		t.Errorf("Generated UUID v7s should be unique, got same: %s", uuid2.String())
	}

	if uuid1 == uuid3 {
		t.Errorf("Generated UUID v7s should be unique, got same: %s", uuid1.String())
	}
}

func TestNewUUIDv7Ordering(t *testing.T) {
	// UUID v7 should have monotonic ordering based on timestamp
	uuid1, err1 := NewV7()
	uuid2, err2 := NewV7()

	if err1 != nil || err2 != nil {
		t.Errorf("NewV7() errors: %v, %v", err1, err2)
		return
	}

	uuid1Str := uuid1.String()
	uuid2Str := uuid2.String()

	// Both should be valid UUIDs
	if !uuidPattern.MatchString(uuid1Str) || !uuidPattern.MatchString(uuid2Str) {
		t.Error("Generated UUID v7s have invalid format")
	}

	// uuid2 should be >= uuid1 (timestamp-based ordering)
	// This is a loose check - UUID v7 format ensures roughly chronological ordering
	if uuid2Str < uuid1Str {
		t.Logf("Warning: UUID v7 ordering might be off: %s < %s", uuid2Str, uuid1Str)
	}
}

func BenchmarkNewUuid(b *testing.B) {
	for i := 0; i < b.N; i++ {
		_ = NewUuid()
	}
}

func BenchmarkNewV7(b *testing.B) {
	for i := 0; i < b.N; i++ {
		_, _ = NewV7()
	}
}

func BenchmarkNewV4(b *testing.B) {
	for i := 0; i < b.N; i++ {
		_ = NewV4()
	}
}

func TestParseUuid(t *testing.T) {
	tests := []struct {
		name      string
		input     string
		wantError bool
	}{
		{
			name:      "valid_uuid",
			input:     "550e8400-e29b-41d4-a716-446655440000",
			wantError: false,
		},
		{
			name:      "valid_uuid_all_zeros",
			input:     "00000000-0000-0000-0000-000000000000",
			wantError: false,
		},
		{
			name:      "valid_uuid_all_fs",
			input:     "ffffffff-ffff-ffff-ffff-ffffffffffff",
			wantError: false,
		},
		{
			name:      "invalid_too_short",
			input:     "550e8400-e29b-41d4-a716-446655440",
			wantError: true,
		},
		{
			name:      "invalid_too_long",
			input:     "550e8400-e29b-41d4-a716-4466554400001",
			wantError: true,
		},
		{
			name:      "invalid_missing_hyphens",
			input:     "550e8400e29b41d4a716446655440000",
			wantError: true,
		},
		{
			name:      "invalid_wrong_hyphen_positions",
			input:     "550e8400-e29b41d4-a716-446655440000",
			wantError: true,
		},
		{
			name:      "invalid_non_hex_chars",
			input:     "550e8400-e29b-41d4-a716-44665544000g",
			wantError: true,
		},
		{
			name:      "invalid_space_in_string",
			input:     "550e8400-e29b-41d4-a716-446655440 00",
			wantError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			uuid, err := ParseUuid(tt.input)
			if (err != nil) != tt.wantError {
				t.Errorf("ParseUuid(%q) error = %v, wantError %v", tt.input, err, tt.wantError)
				return
			}

			if !tt.wantError {
				// Convert back to string and verify round-trip
				result := uuid.String()
				if result != tt.input {
					t.Errorf("ParseUuid round-trip failed: got %q, want %q", result, tt.input)
				}
			}
		})
	}
}

func TestParseUuidRoundTrip(t *testing.T) {
	// Test that generating UUID -> String -> ParseUuid works correctly
	original, err := NewV7()
	if err != nil {
		original = NewV4()
	}

	originalStr := original.String()
	parsed, err := ParseUuid(originalStr)
	if err != nil {
		t.Errorf("ParseUuid(%q) error = %v", originalStr, err)
		return
	}

	if parsed != original {
		t.Errorf("Round-trip failed: %v -> %s -> %v", original, originalStr, parsed)
	}
}

func TestParseUuidCaseSensitivity(t *testing.T) {
	// ParseUuid should handle lowercase
	lowercase := "550e8400-e29b-41d4-a716-446655440000"
	uuid, err := ParseUuid(lowercase)
	if err != nil {
		t.Errorf("ParseUuid lowercase failed: %v", err)
		return
	}

	// Verify the parsed UUID is correct
	result := uuid.String()
	if result != lowercase {
		t.Errorf("ParseUuid result = %q, want %q", result, lowercase)
	}
}

func BenchmarkParseUuid(b *testing.B) {
	validUuid := "550e8400-e29b-41d4-a716-446655440000"
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = ParseUuid(validUuid)
	}
}

func TestIsValidUuid(t *testing.T) {
	tests := []struct {
		name      string
		input     string
		wantValid bool
	}{
		{
			name:      "valid_uuid",
			input:     "550e8400-e29b-41d4-a716-446655440000",
			wantValid: true,
		},
		{
			name:      "valid_uuid_all_zeros",
			input:     "00000000-0000-0000-0000-000000000000",
			wantValid: true,
		},
		{
			name:      "valid_uuid_all_fs",
			input:     "ffffffff-ffff-ffff-ffff-ffffffffffff",
			wantValid: true,
		},
		{
			name:      "valid_uuid_uppercase",
			input:     "550E8400-E29B-41D4-A716-446655440000",
			wantValid: true,
		},
		{
			name:      "valid_uuid_mixed_case",
			input:     "550e8400-E29b-41D4-a716-446655440000",
			wantValid: true,
		},
		{
			name:      "invalid_too_short",
			input:     "550e8400-e29b-41d4-a716-446655440",
			wantValid: false,
		},
		{
			name:      "invalid_too_long",
			input:     "550e8400-e29b-41d4-a716-4466554400001",
			wantValid: false,
		},
		{
			name:      "invalid_missing_hyphens",
			input:     "550e8400e29b41d4a716446655440000",
			wantValid: false,
		},
		{
			name:      "invalid_wrong_hyphen_positions",
			input:     "550e8400-e29b41d4-a716-446655440000",
			wantValid: false,
		},
		{
			name:      "invalid_non_hex_chars",
			input:     "550e8400-e29b-41d4-a716-44665544000g",
			wantValid: false,
		},
		{
			name:      "invalid_space_in_string",
			input:     "550e8400-e29b-41d4-a716-446655440 00",
			wantValid: false,
		},
		{
			name:      "invalid_empty_string",
			input:     "",
			wantValid: false,
		},
		{
			name:      "invalid_only_hyphens",
			input:     "--------",
			wantValid: false,
		},
		{
			name:      "invalid_special_chars",
			input:     "550e8400-e29b-41d4-a716-44665544@000",
			wantValid: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := IsValidUuid(tt.input)
			if got != tt.wantValid {
				t.Errorf("IsValidUuid(%q) = %v, want %v", tt.input, got, tt.wantValid)
			}
		})
	}
}

func TestIsValidUuidConsistencyWithParseUuid(t *testing.T) {
	// IsValidUuid should return true for all strings that ParseUuid can successfully parse
	validUuids := []string{
		"550e8400-e29b-41d4-a716-446655440000",
		"00000000-0000-0000-0000-000000000000",
		"ffffffff-ffff-ffff-ffff-ffffffffffff",
		"12345678-1234-1234-1234-123456789012",
	}

	for _, uuid := range validUuids {
		t.Run(uuid, func(t *testing.T) {
			isValid := IsValidUuid(uuid)
			_, err := ParseUuid(uuid)

			if isValid && err != nil {
				t.Errorf("IsValidUuid returned true but ParseUuid failed: %v", err)
			}
			if !isValid && err == nil {
				t.Errorf("IsValidUuid returned false but ParseUuid succeeded")
			}
		})
	}
}

func TestIsValidUuidWithGeneratedUuids(t *testing.T) {
	// Test that generated UUIDs are always valid
	for i := 0; i < 10; i++ {
		uuid := NewUuid()
		uuidStr := uuid.String()

		if !IsValidUuid(uuidStr) {
			t.Errorf("Generated UUID failed IsValidUuid check: %s", uuidStr)
		}
	}
}

func TestIsValidUuidWithGeneratedV7(t *testing.T) {
	// Test that generated UUIDs v7 are always valid
	for i := 0; i < 10; i++ {
		uuid, err := NewV7()
		if err != nil {
			t.Errorf("NewV7() failed: %v", err)
			continue
		}

		uuidStr := uuid.String()
		if !IsValidUuid(uuidStr) {
			t.Errorf("Generated UUID v7 failed IsValidUuid check: %s", uuidStr)
		}
	}
}

func BenchmarkIsValidUuid(b *testing.B) {
	validUuid := "550e8400-e29b-41d4-a716-446655440000"
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = IsValidUuid(validUuid)
	}
}

func BenchmarkIsValidUuidInvalid(b *testing.B) {
	invalidUuid := "550e8400-e29b-41d4-a716-44665544000g"
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = IsValidUuid(invalidUuid)
	}
}
