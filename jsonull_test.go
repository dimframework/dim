package dim

import (
	"encoding/json"
	"testing"
)

func TestNewJsonNull(t *testing.T) {
	tests := []struct {
		name          string
		value         string
		expectValid   bool
		expectPresent bool
		expectValue   string
	}{
		{
			name:          "valid_value",
			value:         "test",
			expectValid:   true,
			expectPresent: true,
			expectValue:   "test",
		},
		{
			name:          "empty_string",
			value:         "",
			expectValid:   true,
			expectPresent: true,
			expectValue:   "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			jn := NewJsonNull(tt.value)
			if jn.Value != tt.expectValue {
				t.Errorf("Value = %v, want %v", jn.Value, tt.expectValue)
			}
			if jn.Valid != tt.expectValid {
				t.Errorf("Valid = %v, want %v", jn.Valid, tt.expectValid)
			}
			if jn.Present != tt.expectPresent {
				t.Errorf("Present = %v, want %v", jn.Present, tt.expectPresent)
			}
		})
	}
}

func TestNewJsonNullNull(t *testing.T) {
	jn := NewJsonNullNull[string]()

	if jn.Valid {
		t.Errorf("Valid = %v, want false", jn.Valid)
	}
	if !jn.Present {
		t.Errorf("Present = %v, want true", jn.Present)
	}
}

func TestJsonNullFromPtr(t *testing.T) {
	tests := []struct {
		name          string
		ptr           *string
		expectValid   bool
		expectPresent bool
		expectValue   string
	}{
		{
			name:          "nil_pointer",
			ptr:           nil,
			expectValid:   false,
			expectPresent: true,
			expectValue:   "",
		},
		{
			name:          "valid_pointer",
			ptr:           strPtr("hello"),
			expectValid:   true,
			expectPresent: true,
			expectValue:   "hello",
		},
		{
			name:          "pointer_to_empty_string",
			ptr:           strPtr(""),
			expectValid:   true,
			expectPresent: true,
			expectValue:   "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			jn := JsonNullFromPtr(tt.ptr)
			if jn.Value != tt.expectValue {
				t.Errorf("Value = %v, want %v", jn.Value, tt.expectValue)
			}
			if jn.Valid != tt.expectValid {
				t.Errorf("Valid = %v, want %v", jn.Valid, tt.expectValid)
			}
			if jn.Present != tt.expectPresent {
				t.Errorf("Present = %v, want %v", jn.Present, tt.expectPresent)
			}
		})
	}
}

func TestJsonNullMarshalUnmarshal(t *testing.T) {
	tests := []struct {
		name     string
		json     string
		expect   JsonNull[string]
		expectOk bool
	}{
		{
			name: "valid_value",
			json: `{"email":"test@example.com"}`,
			expect: JsonNull[string]{
				Value:   "test@example.com",
				Valid:   true,
				Present: true,
			},
			expectOk: true,
		},
		{
			name: "null_value",
			json: `{"email":null}`,
			expect: JsonNull[string]{
				Value:   "",
				Valid:   false,
				Present: true,
			},
			expectOk: true,
		},
		{
			name: "missing_field",
			json: `{}`,
			expect: JsonNull[string]{
				Value:   "",
				Valid:   false,
				Present: false,
			},
			expectOk: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			type testStruct struct {
				Email JsonNull[string] `json:"email"`
			}

			var result testStruct
			err := json.Unmarshal([]byte(tt.json), &result)

			if (err == nil) != tt.expectOk {
				t.Errorf("error = %v, want ok = %v", err, tt.expectOk)
			}

			if result.Email.Value != tt.expect.Value {
				t.Errorf("Value = %v, want %v", result.Email.Value, tt.expect.Value)
			}
			if result.Email.Valid != tt.expect.Valid {
				t.Errorf("Valid = %v, want %v", result.Email.Valid, tt.expect.Valid)
			}
			if result.Email.Present != tt.expect.Present {
				t.Errorf("Present = %v, want %v", result.Email.Present, tt.expect.Present)
			}
		})
	}
}

func TestJsonNullMarshal(t *testing.T) {
	tests := []struct {
		name   string
		value  JsonNull[string]
		expect string
	}{
		{
			name: "valid_value",
			value: JsonNull[string]{
				Value:   "test@example.com",
				Valid:   true,
				Present: true,
			},
			expect: `"test@example.com"`,
		},
		{
			name: "null_value",
			value: JsonNull[string]{
				Value:   "",
				Valid:   false,
				Present: true,
			},
			expect: `null`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			data, err := json.Marshal(tt.value)
			if err != nil {
				t.Errorf("Marshal error = %v", err)
			}

			if string(data) != tt.expect {
				t.Errorf("Marshal = %s, want %s", string(data), tt.expect)
			}
		})
	}
}

// Helper function to create string pointer
func strPtr(s string) *string {
	return &s
}
