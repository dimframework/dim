package dim

import (
	"net/http/httptest"
	"testing"
)

func TestPaginationParser_Parse(t *testing.T) {
	parser := NewPaginationParser(10, 50)

	tests := []struct {
		name        string
		queryString string
		expected    *Pagination
		expectError bool
	}{
		{
			name:        "Default values",
			queryString: "",
			expected:    &Pagination{Page: 1, Limit: 10},
			expectError: false,
		},
		{
			name:        "JSON:API style",
			queryString: "?page[number]=2&page[size]=20",
			expected:    &Pagination{Page: 2, Limit: 20},
			expectError: false,
		},
		{
			name:        "Simple style",
			queryString: "?page=3&limit=15",
			expected:    &Pagination{Page: 3, Limit: 15},
			expectError: false,
		},
		{
			name:        "Size alias",
			queryString: "?page=1&size=25",
			expected:    &Pagination{Page: 1, Limit: 25},
			expectError: false,
		},
		{
			name:        "Exceed max limit",
			queryString: "?page[size]=1000",
			expected:    &Pagination{Page: 1, Limit: 50},
			expectError: false,
		},
		{
			name:        "Invalid page",
			queryString: "?page[number]=0",
			expected:    nil,
			expectError: true,
		},
		{
			name:        "Invalid page text",
			queryString: "?page[number]=abc",
			expected:    nil,
			expectError: true,
		},
		{
			name:        "Invalid limit",
			queryString: "?page[size]=-5",
			expected:    nil,
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest("GET", "/"+tt.queryString, nil)
			got, err := parser.Parse(req)

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

			if got.Page != tt.expected.Page || got.Limit != tt.expected.Limit {
				t.Errorf("Expected %+v, got %+v", tt.expected, got)
			}
		})
	}
}

func TestPagination_Offset(t *testing.T) {
	p := &Pagination{Page: 3, Limit: 10}
	if offset := p.Offset(); offset != 20 {
		t.Errorf("Expected offset 20, got %d", offset)
	}
}
