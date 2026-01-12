package dim

import "testing"

func TestStripComments(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "No comments",
			input:    "SELECT * FROM users",
			expected: "SELECT * FROM users",
		},
		{
			name:     "Leading whitespace",
			input:    "   SELECT * FROM users",
			expected: "SELECT * FROM users",
		},
		{
			name:     "Single line comment",
			input:    "-- This is a comment\nSELECT * FROM users",
			expected: "SELECT * FROM users",
		},
		{
			name:     "Multiple single line comments",
			input:    "-- First comment\n-- Second comment\nSELECT * FROM users",
			expected: "SELECT * FROM users",
		},
		{
			name:     "Block comment",
			input:    "/* This is a block comment */SELECT * FROM users",
			expected: "SELECT * FROM users",
		},
		{
			name:     "Multiline block comment",
			input:    "/* \n * Multiline \n */\nSELECT * FROM users",
			expected: "SELECT * FROM users",
		},
		{
			name:     "Nested comments mix",
			input:    "-- Line 1\n/* Block 1 */\n-- Line 2\n   SELECT * FROM users",
			expected: "SELECT * FROM users",
		},
		{
			name:     "Comment inside query (should not be stripped)",
			input:    "SELECT /* hint */ * FROM users",
			expected: "SELECT /* hint */ * FROM users",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := StripComments(tt.input)
			if got != tt.expected {
				t.Errorf("StripComments() = %q, want %q", got, tt.expected)
			}
		})
	}
}

func TestIsSafeRead(t *testing.T) {
	tests := []struct {
		query  string
		isSafe bool // true = READ, false = WRITE
	}{
		// SAFE READ cases associated with SELECT
		{"SELECT * FROM users", true},
		{"select id, name from users", true},
		{"   SELECT * FROM users", true},
		{"-- comment\nSELECT * FROM users", true},
		{"/* block */ SELECT * FROM users", true},
		{"SELECT count(*) FROM users", true},

		// UNSAFE / WRITE cases associated with SELECT
		{"SELECT * FROM users FOR UPDATE", false},
		{"SELECT * FROM users FOR SHARE", false},
		{"SELECT * FROM users FOR NO KEY UPDATE", false},
		{"SELECT * INTO new_table FROM old_table", false},

		// SAFE READ cases associated with CTE
		{"WITH cte AS (SELECT * FROM users) SELECT * FROM cte", true},
		{"with c as (select 1) select * from c", true},

		// UNSAFE / WRITE cases associated with CTE
		{"WITH inserted AS (INSERT INTO users VALUES(1) RETURNING *) SELECT * FROM inserted", false},
		{"WITH updated AS (UPDATE users SET name='x' RETURNING *) SELECT * FROM updated", false},
		{"WITH deleted AS (DELETE FROM users RETURNING *) SELECT * FROM deleted", false},

		// UNSAFE / WRITE cases (Direct)
		{"INSERT INTO users VALUES(1)", false},
		{"UPDATE users SET name='x'", false},
		{"DELETE FROM users", false},
		{"TRUNCATE users", false},
		{"MERGE INTO users ...", false},
		{"CALL my_proc()", false},
		{"DROP TABLE users", false},
		{"ALTER TABLE users ...", false},
		{"CREATE TABLE users ...", false},

		// Ambiguous / Fallback to WRITE (Not Safe)
		{"EXPLAIN ANALYZE SELECT * FROM users", false},
		{"SHOW timezone", false},
		{"COPY users FROM stdin", false},
		{"BEGIN", false},
	}

	for _, tt := range tests {
		t.Run(tt.query, func(t *testing.T) {
			got := IsSafeRead(tt.query)
			if got != tt.isSafe {
				expectedStr := "WRITE"
				if tt.isSafe {
					expectedStr = "READ"
				}
				actualStr := "WRITE"
				if got {
					actualStr = "READ"
				}
				t.Errorf("expected %s but got %s for query: %q", expectedStr, actualStr, tt.query)
			}
		})
	}
}
