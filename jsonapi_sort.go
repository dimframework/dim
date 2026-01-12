package dim

import (
	"fmt"
	"net/http"
	"strings"
)

// SortField represents a single sort criterion
type SortField struct {
	Field     string
	Direction string // "ASC" or "DESC"
}

// SQL returns a basic ORDER BY clause
func (s SortField) SQL() string {
	return fmt.Sprintf("%s %s", s.Field, s.Direction)
}

// SortParser parses sort parameters from the request
type SortParser struct {
	AllowedFields map[string]bool
}

// NewSortParser creates a new SortParser with allowed fields
func NewSortParser(allowedFields []string) *SortParser {
	allowed := make(map[string]bool)
	for _, f := range allowedFields {
		allowed[f] = true
	}
	return &SortParser{
		AllowedFields: allowed,
	}
}

// Parse parses the "sort" query parameter
// Format: ?sort=-created_at,title (descending created_at, ascending title)
func (p *SortParser) Parse(r *http.Request) ([]SortField, error) {
	sortParam := r.URL.Query().Get("sort")
	if sortParam == "" {
		return nil, nil
	}

	fields := strings.Split(sortParam, ",")
	result := make([]SortField, 0, len(fields))

	for _, f := range fields {
		f = strings.TrimSpace(f)
		if f == "" {
			continue
		}

		direction := "ASC"
		fieldName := f

		if strings.HasPrefix(f, "-") {
			direction = "DESC"
			fieldName = strings.TrimPrefix(f, "-")
		}

		if len(p.AllowedFields) > 0 && !p.AllowedFields[fieldName] {
			return nil, NewAppError(fmt.Sprintf("Sort field '%s' is not allowed", fieldName), http.StatusBadRequest)
		}

		result = append(result, SortField{
			Field:     fieldName,
			Direction: direction,
		})
	}

	return result, nil
}
