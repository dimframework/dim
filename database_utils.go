package dim

import "strings"

// StripComments removes SQL comments (-- and /* */) and whitespace from the start of the string.
// This is an internal helper exposed for testing purposes.
func StripComments(query string) string {
	var i int
	n := len(query)
	inComment := false
	inBlockComment := false

	for i < n {
		if inComment {
			if query[i] == '\n' {
				inComment = false
			}
			i++
			continue
		}
		if inBlockComment {
			if i+1 < n && query[i] == '*' && query[i+1] == '/' {
				inBlockComment = false
				i += 2
			} else {
				i++
			}
			continue
		}

		c := query[i]
		switch {
		case c == ' ' || c == '\t' || c == '\n' || c == '\r':
			i++
		case i+1 < n && c == '-' && query[i+1] == '-':
			inComment = true
			i += 2
		case i+1 < n && c == '/' && query[i+1] == '*':
			inBlockComment = true
			i += 2
		default:
			return query[i:]
		}
	}
	return ""
}

// IsSafeRead determines if a query is safe to route to a read replica.
// It uses a whitelist approach: only explicitly safe SELECT and CTE queries
// are allowed. Everything else (INSERT, UPDATE, Locking Reads, etc.) returns false.
func IsSafeRead(query string) bool {
	cleanQuery := StripComments(query)
	if len(cleanQuery) == 0 {
		return true // Empty query, safe to read (noop)
	}

	upperQuery := strings.ToUpper(cleanQuery)

	// 2. WHITELIST: Handle SELECT
	// Only route to Read Pool if it's a SELECT without side effects
	if strings.HasPrefix(upperQuery, "SELECT") {
		// Check for Locking Clauses (Must go to Write Pool)
		if strings.Contains(upperQuery, " FOR UPDATE") ||
			strings.Contains(upperQuery, " FOR NO KEY UPDATE") ||
			strings.Contains(upperQuery, " FOR SHARE") ||
			strings.Contains(upperQuery, " FOR KEY SHARE") {
			return false
		}

		// Check for SELECT INTO (Table Creation - Write Operation)
		if strings.Contains(upperQuery, " INTO ") {
			return false
		}

		// It's a standard SELECT -> Safe to Read
		return true
	}

	// 3. WHITELIST: Handle CTEs (WITH)
	// Only route to Read if the CTE contains no write modifications
	if strings.HasPrefix(upperQuery, "WITH") {
		if strings.Contains(upperQuery, "INSERT ") ||
			strings.Contains(upperQuery, "UPDATE ") ||
			strings.Contains(upperQuery, "DELETE ") ||
			strings.Contains(upperQuery, "MERGE ") ||
			strings.Contains(upperQuery, "TRUNCATE ") ||
			strings.Contains(upperQuery, " RETURNING ") {
			return false
		}
		// CTE without write keywords -> Safe to Read
		return true
	}

	// 4. Default Fallback: Assume Write (Not Safe Read)
	return false
}
