package dim

import (
	"net/http/httptest"
	"reflect"
	"testing"
)

func TestSortParser_Parse(t *testing.T) {
	tests := []struct {
		name          string
		allowedFields []string
		queryString   string
		expected      []SortField
		expectError   bool
	}{
		{
			name:          "No sort param",
			allowedFields: []string{"name", "age"},
			queryString:   "",
			expected:      nil,
			expectError:   false,
		},
		{
			name:          "Single sort ASC",
			allowedFields: []string{"name"},
			queryString:   "?sort=name",
			expected: []SortField{
				{Field: "name", Direction: "ASC"},
			},
			expectError: false,
		},
		{
			name:          "Single sort DESC",
			allowedFields: []string{"name"},
			queryString:   "?sort=-name",
			expected: []SortField{
				{Field: "name", Direction: "DESC"},
			},
			expectError: false,
		},
		{
			name:          "Multiple sort",
			allowedFields: []string{"name", "created_at"},
			queryString:   "?sort=name,-created_at",
			expected: []SortField{
				{Field: "name", Direction: "ASC"},
				{Field: "created_at", Direction: "DESC"},
			},
			expectError: false,
		},
		{
			name:          "Disallowed field",
			allowedFields: []string{"name"},
			queryString:   "?sort=age",
			expected:      nil,
			expectError:   true,
		},
		{
			name:          "Empty field in comma",
			allowedFields: []string{"name", "age"},
			queryString:   "?sort=name,,age",
			expected: []SortField{
				{Field: "name", Direction: "ASC"},
				{Field: "age", Direction: "ASC"},
			},
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			parser := NewSortParser(tt.allowedFields)
			req := httptest.NewRequest("GET", "/"+tt.queryString, nil)

			result, err := parser.Parse(req)

			if tt.expectError {
				if err == nil {
					t.Errorf("Expected error but got none")
				}
				return
			}

			if err != nil {
				t.Errorf("Unexpected error: %v", err)
				return
			}

			if !reflect.DeepEqual(result, tt.expected) {
				t.Errorf("Expected %v, got %v", tt.expected, result)
			}
		})
	}
}

func TestSortField_SQL(t *testing.T) {
	s := SortField{Field: "name", Direction: "DESC"}
	expected := "name DESC"
	if got := s.SQL(); got != expected {
		t.Errorf("Expected %s, got %s", expected, got)
	}
}
