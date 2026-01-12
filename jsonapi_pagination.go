package dim

import (
	"net/http"
	"strconv"
)

// Pagination struct holds the pagination details
type Pagination struct {
	Page  int `json:"page"`
	Limit int `json:"limit"`
}

// Offset returns the SQL offset
func (p *Pagination) Offset() int {
	return (p.Page - 1) * p.Limit
}

// PaginationParser parses pagination parameters
type PaginationParser struct {
	DefaultLimit int
	MaxLimit     int
}

// NewPaginationParser creates a new PaginationParser
func NewPaginationParser(defaultLimit, maxLimit int) *PaginationParser {
	if defaultLimit <= 0 {
		defaultLimit = 10
	}
	if maxLimit <= 0 {
		maxLimit = 100
	}
	return &PaginationParser{
		DefaultLimit: defaultLimit,
		MaxLimit:     maxLimit,
	}
}

// Parse parses page[number] and page[size]
// Also supports page and limit/size query params as fallback or standard simple pagination
func (p *PaginationParser) Parse(r *http.Request) (*Pagination, error) {
	q := r.URL.Query()

	// Try page[number] and page[size] (JSON:API style)
	pageStr := q.Get("page[number]")
	limitStr := q.Get("page[size]")

	// Fallback to page and limit/size (Simple style)
	if pageStr == "" {
		pageStr = q.Get("page")
	}
	if limitStr == "" {
		limitStr = q.Get("limit")
		if limitStr == "" {
			limitStr = q.Get("size")
		}
	}

	page := 1
	limit := p.DefaultLimit

	var err error

	if pageStr != "" {
		page, err = strconv.Atoi(pageStr)
		if err != nil || page < 1 {
			return nil, NewAppError("Page number must be a positive integer", http.StatusBadRequest)
		}
	}

	if limitStr != "" {
		limit, err = strconv.Atoi(limitStr)
		if err != nil || limit < 1 {
			return nil, NewAppError("Page size must be a positive integer", http.StatusBadRequest)
		}
	}

	if limit > p.MaxLimit {
		limit = p.MaxLimit
	}

	return &Pagination{
		Page:  page,
		Limit: limit,
	}, nil
}
