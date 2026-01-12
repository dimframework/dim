package dim

import (
	"context"
	"fmt"
	"net/http"
	"reflect"
	"strconv"
	"strings"
	"time"

	"github.com/atfromhome/goreus/pkg/cache"
)

// Package filter provides a flexible HTTP query parameter filtering system with
// extensible constraint validation.
//
// FilterParser parses URL query parameters in the format:
//   ?filters[fieldName]=value&filters[fieldName2]=val1,val2
//
// Supported types:
//   - Basic: *string, *int, *int64, *bool, *UUID
//   - Slices: []string, []int, []int64, []float64, []UUID
//   - Ranges: DateRange, AmountRange, IntRange, TimestampRange (both pointer and non-pointer)
//
// Core Features:
//   - Flexible query parameter parsing with type conversion
//   - Range support with From/To validation (IntRange, AmountRange, etc.)
//   - Max values limit per field (configurable)
//   - Custom timezone support for timestamp parsing
//   - Generic range parser helper reducing code duplication
//   - Type caching using goreus for performance optimization
//   - Extensible constraint validation system (ConstraintValidator interface)
//   - Comprehensive error reporting with field-specific error keys
//
// Constraint Validation:
//   - Built-in "in" constraint for enum validation
//   - Custom validators via ConstraintValidator interface
//   - Multiple constraints per field supported
//   - Format: "fieldName,constraint1:value1,constraint2:value2"
//
// Example usage:
//   type Filters struct {
//       IDs       []int64       `filter:"ids"`
//       Status    *string       `filter:"status,in:active|pending|archived"`
//       Statuses  []string      `filter:"statuses,in:active|pending"`
//       Amount    AmountRange   `filter:"amount"`
//       Date      DateRange     `filter:"date"`
//       CreatedAt TimestampRange `filter:"created_at"`
//   }
//
//   var filters Filters
//   fp := NewFilterParser(r).
//       WithMaxValues(50).
//       WithTimezone(time.LoadLocation("Asia/Jakarta")).
//       RegisterConstraintValidator(&CustomValidator{})
//   fp.Parse(&filters)
//   if fp.HasErrors() {
//       // Handle errors: fp.Errors() returns map[string]string
//       // Key format: "filters[fieldName]"
//   }

// Global Type Cache - Performance Optimization
// Uses goreus in-memory cache for efficient type comparison caching.
// Reduces reflection overhead on repeated type matching operations.
// Capacity: 1000 types with automatic LRU eviction when full.
var typeCache = cache.NewInMemoryCache[string, reflect.Type](1000, 0) // 0 = no TTL

// getCachedType returns a cached Type or caches it if not present.
// Uses type string representation as cache key for reliable generic type matching.
// Leverages goreus in-memory cache with automatic LRU eviction.
func getCachedType(t reflect.Type) reflect.Type {
	if t == nil {
		return nil
	}

	key := t.String()
	ctx := context.Background()

	// Try to get from cache (fast path)
	if cached, ok := typeCache.Get(ctx, key); ok {
		return cached
	}

	// Cache miss, add to cache for future use
	typeCache.Set(ctx, key, t)
	return t
}

// Helper Functions - Type Checking
// typeMatches compares two reflect.Type values for equality.
// Uses Type.String() comparison for reliable generic type matching.
// Uses global cache to optimize repeated comparisons.
// Important: Type aliases with the same underlying type (e.g., IntRange = Range[int64],
// TimestampRange = Range[int64]) cannot be distinguished by typeMatches.
func typeMatches(t1, t2 reflect.Type) bool {
	if t1 == nil || t2 == nil {
		return t1 == t2
	}

	// Cache both types
	getCachedType(t1)
	getCachedType(t2)

	return t1.String() == t2.String()
}

// Range represents a range of values with from and to bounds.
// Generic type supporting any comparable type for flexible range queries.
//
// Fields:
//   - From: Start value of range
//   - To: End value of range
//   - Valid: true if format is valid, range constraint (From <= To) satisfied, and non-empty
//   - Present: true if parameter exists in request (even if Invalid)
//
// Behavior:
//   - Single value (e.g., "100") sets both From and To to same value
//   - Range format (e.g., "100,500") sets From and To separately
//   - Invalid format or From > To results in Valid=false but Present=true
//   - Empty input results in Valid=false and Present=true
type Range[T any] struct {
	From    T
	To      T
	Valid   bool // true if format is valid and has non-empty value and From <= To
	Present bool // true if parameter exists in request
}

// Type aliases for common range types
//
// DateRange: String format "YYYY-MM-DD"
// Example: "2024-01-15" or "2024-01-01,2024-12-31"
type DateRange = Range[string]

// AmountRange: Floating point amount range
// Example: "100.50" or "100.50,500.75"
type AmountRange = Range[float64]

// IntRange: Integer range
// Example: "100" or "100,500"
type IntRange = Range[int64]

// TimestampRange: Unix timestamp in seconds (from date string parsing)
// Input format: "YYYY-MM-DD" converted to Unix timestamp
// Example: "2024-01-15" or "2024-01-01,2024-12-31"
type TimestampRange = Range[int64]

// FilterParser parses filter parameters from an HTTP request and sets the fields of a target struct accordingly.
type FilterParser struct {
	request             *http.Request
	errors              map[string]string
	MaxValuesPerField   int                            // Maximum number of values allowed per filter field (0 = unlimited)
	TimestampTimezone   *time.Location                 // Timezone for parsing timestamps (nil = UTC)
	constraintValidator map[string]ConstraintValidator // Custom constraint validators (e.g., "in", "regex")
}

// NewFilterParser creates a new FilterParser instance with unlimited values.
// Defaults:
//   - MaxValuesPerField: 0 (unlimited)
//   - TimestampTimezone: nil (UTC)
//   - constraintValidator: built-in validators (e.g., "in" for enums)
func NewFilterParser(r *http.Request) *FilterParser {
	return &FilterParser{
		request:             r,
		errors:              make(map[string]string),
		MaxValuesPerField:   0, // Default: unlimited
		constraintValidator: BuiltinConstraintValidators(),
	}
}

// WithMaxValues sets the maximum number of values allowed per filter field.
// Use 0 for unlimited (default).
// Returns the receiver for method chaining.
//
// Example:
//
//	fp.WithMaxValues(10).Parse(&filters)
func (fp *FilterParser) WithMaxValues(max int) *FilterParser {
	fp.MaxValuesPerField = max
	return fp
}

// WithTimezone sets the timezone for parsing timestamp ranges.
// If nil, UTC is used (default). This affects parseTimestampRange only.
// Returns the receiver for method chaining.
//
// Example:
//
//	jakartaTz, _ := time.LoadLocation("Asia/Jakarta")
//	fp.WithTimezone(jakartaTz).Parse(&filters)
func (fp *FilterParser) WithTimezone(tz *time.Location) *FilterParser {
	fp.TimestampTimezone = tz
	return fp
}

// RegisterConstraintValidator registers a custom constraint validator.
// Replaces any existing validator with the same name (including built-in validators).
// Returns the receiver for method chaining.
//
// Example - custom length validator:
//
//	type MinLengthValidator struct{}
//	func (v *MinLengthValidator) Name() string { return "min_length" }
//	func (v *MinLengthValidator) Validate(values []string, constraint string, fieldType reflect.Type) error {
//	    min := 0
//	    fmt.Sscanf(constraint, "%d", &min)
//	    for _, v := range values {
//	        if len(v) < min {
//	            return fmt.Errorf("minimum length %d required", min)
//	        }
//	    }
//	    return nil
//	}
//
//	fp.RegisterConstraintValidator(&MinLengthValidator())
//	fp.Parse(&filters)
func (fp *FilterParser) RegisterConstraintValidator(validator ConstraintValidator) *FilterParser {
	if validator == nil {
		return fp
	}
	fp.constraintValidator[validator.Name()] = validator
	return fp
}

// HasErrors returns true if any errors were encountered during parsing.
// Check this before accessing filter results.
func (fp *FilterParser) HasErrors() bool {
	return len(fp.errors) > 0
}

// Errors returns the errors encountered during parsing.
// Key format: "filters[fieldName]" (e.g., "filters[ids]").
// Returns empty map if no errors occurred.
func (fp *FilterParser) Errors() map[string]string {
	return fp.errors
}

// Main Parsing Logic

// Parse parses the filter parameters from the request and sets the fields of the target struct accordingly.
// Target must be a pointer to a struct with "filter" tags.
// Returns the receiver for method chaining.
//
// Tag Format: "fieldName" or "fieldName,constraint1:value1,constraint2:value2"
// Example struct tags:
//
//	type Filters struct {
//	    // Basic types
//	    IDs         []int64       `filter:"ids"`
//	    Name        *string       `filter:"name"`
//	    Active      *bool         `filter:"active"`
//	    Tags        []string      `filter:"tags"`
//
//	    // Enum constraint (pipe-separated allowed values)
//	    Status      *string       `filter:"status,in:active|pending|archived"`
//	    Statuses    []string      `filter:"statuses,in:active|pending"`
//
//	    // Range types with From/To validation
//	    Amount      AmountRange   `filter:"amount"`            // single: "100" or range: "100,500"
//	    Price       *IntRange     `filter:"price"`             // integer range with optional pointer
//	    CreatedAt   TimestampRange `filter:"created_at"`       // date range "2024-01-01,2024-12-31"
//	    Date        DateRange     `filter:"date"`              // string date range
//	}
//
// Built-in Constraints:
//   - in:val1|val2|val3 : Enum validation for strings (pipe-separated allowed values)
//
// Custom Constraints:
//   - Register via RegisterConstraintValidator() to add custom constraint types
//   - Multiple constraints per field supported
//
// Error Handling:
//   - Check HasErrors() before accessing filter results
//   - Call Errors() to get map[string]string with field-specific error messages
//   - Error keys follow format: "filters[fieldName]"
func (fp *FilterParser) Parse(target interface{}) *FilterParser {
	v := reflect.ValueOf(target)
	if v.Kind() != reflect.Ptr || v.Elem().Kind() != reflect.Struct {
		fp.errors["_parse"] = "Parse requires a pointer to struct"
		return fp
	}

	v = v.Elem()
	t := v.Type()

	for i := 0; i < v.NumField(); i++ {
		field := v.Field(i)
		fieldType := t.Field(i)

		if !field.CanSet() {
			continue
		}

		filterTag := fieldType.Tag.Get("filter")
		if filterTag == "" || filterTag == "-" {
			continue
		}

		// Parse filter tag: "fieldName" or "fieldName,in:val1|val2" or "fieldName,constraint1:val,constraint2:val"
		parts := strings.Split(filterTag, ",")
		fieldName := strings.TrimSpace(parts[0])
		if fieldName == "" {
			continue
		}

		// Extract constraints (e.g., "in:active|pending,min:1" becomes map{in: "active|pending", min: "1"})
		constraints := make(map[string]string)
		for i := 1; i < len(parts); i++ {
			part := strings.TrimSpace(parts[i])
			if idx := strings.Index(part, ":"); idx > 0 {
				key := part[:idx]
				value := strings.TrimSpace(part[idx+1:])
				if value != "" {
					constraints[key] = value
				}
			}
		}

		filterValues := fp.request.URL.Query()["filters["+fieldName+"]"]
		if len(filterValues) == 0 {
			continue
		}

		if err := fp.parseFieldValue(field, fieldType, filterValues, constraints); err != nil {
			fp.errors["filters["+fieldName+"]"] = err.Error()
		}
	}

	return fp
}

// Parse parses the filter parameters from the request and sets the fields of the target struct accordingly.
// Target must be a pointer to a struct with "filter" tags.
// Returns the receiver for method chaining.
//
// Example struct tag:
//
//	type Filters struct {
//	    IDs    []int64      `filter:"ids"`
//	    Status *string      `filter:"status,in:active|pending"`
//	    Amount AmountRange  `filter:"amount"`
//	}
//
// Tag constraints (optional, comma-separated):
//   - in:val1|val2 : enum validation for strings (pipe-separated allowed values)

// Field Value Parsing

// parseFieldValue parses a field value for a given field type and value.
// It handles different types of field values and sets the field accordingly.
// Routes to parseSliceValue or parsePointerValue based on field kind.
func (fp *FilterParser) parseFieldValue(field reflect.Value, fieldType reflect.StructField, values []string, constraints map[string]string) error {
	fieldKind := field.Kind()

	if fieldKind == reflect.Slice {
		return fp.parseSliceValue(field, fieldType, values, constraints)
	}

	if fieldKind == reflect.Ptr {
		if len(values) == 0 {
			return nil
		}
		return fp.parsePointerValue(field, fieldType, values[0], constraints)
	}

	// Handle non-pointer Range types
	if typeMatches(fieldType.Type, reflect.TypeOf(DateRange{})) {
		if len(values) == 0 {
			return nil
		}
		dr := parseDateRange(values[0])
		if dr.Present && !dr.Valid {
			return fmt.Errorf("format tanggal tidak valid (gunakan YYYY-MM-DD atau YYYY-MM-DD,YYYY-MM-DD)")
		}
		field.Set(reflect.ValueOf(dr))
		return nil
	}

	if typeMatches(fieldType.Type, reflect.TypeOf(AmountRange{})) {
		if len(values) == 0 {
			return nil
		}
		ar := parseAmountRange(values[0])
		if ar.Present && !ar.Valid {
			return fmt.Errorf("format amount tidak valid")
		}
		field.Set(reflect.ValueOf(ar))
		return nil
	}

	if typeMatches(fieldType.Type, reflect.TypeOf(TimestampRange{})) {
		if len(values) == 0 {
			return nil
		}
		tr := parseTimestampRange(values[0], fp.TimestampTimezone)
		if tr.Present && !tr.Valid {
			return fmt.Errorf("format tanggal tidak valid (gunakan YYYY-MM-DD atau YYYY-MM-DD,YYYY-MM-DD)")
		}
		field.Set(reflect.ValueOf(tr))
		return nil
	}

	if typeMatches(fieldType.Type, reflect.TypeOf(IntRange{})) {
		if len(values) == 0 {
			return nil
		}
		ir := parseIntRange(values[0])
		if ir.Present && !ir.Valid {
			return fmt.Errorf("format angka tidak valid (gunakan 100 atau 100,500)")
		}
		field.Set(reflect.ValueOf(ir))
		return nil
	}

	return fmt.Errorf("field %s must be a pointer or slice type", fieldType.Name)
}

// parseSliceValue parses a slice value for a given field type and value.
// It handles different types of slice values and sets the field accordingly.
func (fp *FilterParser) parseSliceValue(field reflect.Value, fieldType reflect.StructField, values []string, constraints map[string]string) error {
	elemType := field.Type().Elem()

	// Support comma-separated values: if single value contains comma, split it
	if len(values) == 1 && strings.Contains(values[0], ",") {
		parts := strings.Split(values[0], ",")
		values = make([]string, 0, len(parts))
		for _, part := range parts {
			trimmed := strings.TrimSpace(part)
			if trimmed != "" {
				values = append(values, trimmed)
			}
		}
	}

	// Check max values limit
	if fp.MaxValuesPerField > 0 && len(values) > fp.MaxValuesPerField {
		return fmt.Errorf("maksimal %d nilai diperbolehkan, diterima %d", fp.MaxValuesPerField, len(values))
	}

	if typeMatches(elemType, reflect.TypeOf(UUID{})) {
		uuids := make([]UUID, 0, len(values))
		for _, v := range values {
			parsed, err := ParseUuid(v)
			if err != nil {
				return fmt.Errorf("UUID tidak valid: %s", v)
			}
			uuids = append(uuids, parsed)
		}
		field.Set(reflect.ValueOf(uuids))
		return nil
	}

	switch elemType.Kind() {
	case reflect.String:
		stringType := reflect.TypeOf("")
		if !typeMatches(elemType, stringType) {
			slice := reflect.MakeSlice(field.Type(), len(values), len(values))
			for i, v := range values {
				elem := reflect.New(elemType).Elem()
				elem.SetString(v)
				slice.Index(i).Set(elem)
			}
			field.Set(slice)
		} else {
			field.Set(reflect.ValueOf(values))
		}

		// Apply registered constraint validators
		if err := fp.applyConstraints(values, constraints, fieldType.Type); err != nil {
			return err
		}

		return nil

	case reflect.Int64:
		ints := make([]int64, 0, len(values))
		for _, v := range values {
			parsed, err := strconv.ParseInt(v, 10, 64)
			if err != nil {
				return fmt.Errorf("harus berupa angka: %s", v)
			}
			ints = append(ints, parsed)
		}
		field.Set(reflect.ValueOf(ints))
		return nil

	case reflect.Int:
		ints := make([]int, 0, len(values))
		for _, v := range values {
			parsed, err := strconv.Atoi(v)
			if err != nil {
				return fmt.Errorf("harus berupa angka: %s", v)
			}
			ints = append(ints, parsed)
		}
		field.Set(reflect.ValueOf(ints))
		return nil

	case reflect.Float64:
		floats := make([]float64, 0, len(values))
		for _, v := range values {
			parsed, err := strconv.ParseFloat(v, 64)
			if err != nil {
				return fmt.Errorf("harus berupa angka desimal: %s", v)
			}
			floats = append(floats, parsed)
		}
		field.Set(reflect.ValueOf(floats))
		return nil

	default:
		return fmt.Errorf("unsupported slice element type: %s", elemType.Kind())
	}
}

// parsePointerValue parses a pointer value for a given field type and value.
// It handles different types of pointer values and sets the field accordingly.
// Note: Constraints are validated for pointer string types.
func (fp *FilterParser) parsePointerValue(field reflect.Value, fieldType reflect.StructField, value string, constraints map[string]string) error {

	if field.Kind() != reflect.Ptr {
		return fmt.Errorf("field %s must be a pointer type", fieldType.Name)
	}

	elemType := field.Type().Elem()

	// Handle Range types
	if typeMatches(elemType, reflect.TypeOf(DateRange{})) {
		dr := parseDateRange(value)
		if dr.Present && !dr.Valid {
			return fmt.Errorf("format tanggal tidak valid (gunakan YYYY-MM-DD atau YYYY-MM-DD,YYYY-MM-DD)")
		}
		field.Set(reflect.ValueOf(&dr))
		return nil
	}

	if typeMatches(elemType, reflect.TypeOf(AmountRange{})) {
		ar := parseAmountRange(value)
		if ar.Present && !ar.Valid {
			return fmt.Errorf("format amount tidak valid")
		}
		field.Set(reflect.ValueOf(&ar))
		return nil
	}

	if typeMatches(elemType, reflect.TypeOf(IntRange{})) {
		ir := parseIntRange(value)
		if ir.Present && !ir.Valid {
			return fmt.Errorf("format angka tidak valid (gunakan 100 atau 100,500)")
		}
		field.Set(reflect.ValueOf(&ir))
		return nil
	}

	if typeMatches(elemType, reflect.TypeOf(TimestampRange{})) {
		tr := parseTimestampRange(value, fp.TimestampTimezone)
		if tr.Present && !tr.Valid {
			return fmt.Errorf("format tanggal tidak valid (gunakan YYYY-MM-DD atau YYYY-MM-DD,YYYY-MM-DD)")
		}
		field.Set(reflect.ValueOf(&tr))
		return nil
	}

	if typeMatches(elemType, reflect.TypeOf(UUID{})) {
		parsed, err := ParseUuid(value)
		if err != nil {
			return fmt.Errorf("UUID tidak valid")
		}
		field.Set(reflect.ValueOf(&parsed))
		return nil
	}

	switch elemType.Kind() {
	case reflect.String:
		// Apply constraints for pointer string types
		if err := fp.applyConstraints([]string{value}, constraints, fieldType.Type); err != nil {
			return err
		}
		field.Set(reflect.ValueOf(&value))
		return nil

	case reflect.Int:
		parsed, err := strconv.Atoi(value)
		if err != nil {
			return fmt.Errorf("harus berupa angka")
		}
		field.Set(reflect.ValueOf(&parsed))
		return nil

	case reflect.Int64:
		parsed, err := strconv.ParseInt(value, 10, 64)
		if err != nil {
			return fmt.Errorf("harus berupa angka")
		}
		field.Set(reflect.ValueOf(&parsed))
		return nil

	case reflect.Bool:
		parsed, err := strconv.ParseBool(value)
		if err != nil {
			return fmt.Errorf("harus berupa true atau false")
		}
		field.Set(reflect.ValueOf(&parsed))
		return nil

	default:
		return fmt.Errorf("unsupported type: %s", elemType.Kind())
	}
}

// parseDateRange parses "YYYY-MM-DD,YYYY-MM-DD" format
// If only one date provided, To is set to From
// Empty string is considered invalid
//
// Note: Does not use generic parseRange helper because validation is based on
// format check (IsValidDateFormat) rather than numeric comparison.
func parseDateRange(value string) DateRange {
	value = strings.TrimSpace(value)
	if value == "" {
		return DateRange{Present: true, Valid: false}
	}

	parts := strings.Split(value, ",")
	from := strings.TrimSpace(parts[0])

	if from == "" {
		return DateRange{Present: true, Valid: false}
	}

	// Default to = from (single date)
	to := from
	if len(parts) > 1 {
		toStr := strings.TrimSpace(parts[1])
		if toStr != "" {
			to = toStr
		}
	}

	// Validate YYYY-MM-DD format
	if !IsValidDateFormat(from) || !IsValidDateFormat(to) {
		return DateRange{From: from, To: to, Present: true, Valid: false}
	}

	return DateRange{From: from, To: to, Present: true, Valid: true}
}

// parseAmountRange parses amount range in "100.50" or "100.50,500.00" format.
// If only one amount provided, To is set to From.
// Empty string is considered invalid (Present: true, Valid: false).
// Validates that From <= To; if From > To, Valid is false.
//
// Uses generic parseRange helper to reduce code duplication.
//
// Examples:
//
//	"100.50"         → AmountRange{From: 100.50, To: 100.50, Valid: true}
//	"100.50,500.00"  → AmountRange{From: 100.50, To: 500.00, Valid: true}
//	"500.00,100.50"  → AmountRange{From: 500.00, To: 100.50, Valid: false}
//	"invalid"        → AmountRange{Present: true, Valid: false}
func parseAmountRange(value string) AmountRange {
	return parseRange(
		value,
		func(s string) (float64, error) { return strconv.ParseFloat(s, 64) },
		func(from, to float64) bool { return from <= to },
	)
}

// parseIntRange parses integer range in "100" or "100,500" format.
// If only one value provided, To is set to From.
// Empty string is considered invalid (Present: true, Valid: false).
// Validates that From <= To; if From > To, Valid is false.
//
// Uses generic parseRange helper to reduce code duplication.
//
// Examples:
//
//	"100"    → IntRange{From: 100, To: 100, Valid: true}
//	"100,500" → IntRange{From: 100, To: 500, Valid: true}
//	"500,100" → IntRange{From: 500, To: 100, Valid: false}
//	"abc"    → IntRange{Present: true, Valid: false}
func parseIntRange(value string) IntRange {
	return parseRange(
		value,
		func(s string) (int64, error) { return strconv.ParseInt(s, 10, 64) },
		func(from, to int64) bool { return from <= to },
	)
}

// parseTimestampRange parses date strings and converts to Unix timestamps.
// Format: "YYYY-MM-DD" or "YYYY-MM-DD,YYYY-MM-DD"
// If only one date provided, To is set to From.
// Returns Unix timestamp in seconds.
// If timezone is nil, UTC is used.
// Validates that From <= To; if From > To, Valid is false.
//
// Uses generic parseRange helper with time parsing function.
//
// Examples (with UTC):
//
//	"2024-01-15"           → TimestampRange{From: 1705276800, To: 1705276800, Valid: true}
//	"2024-01-01,2024-01-31" → TimestampRange{From: 1704067200, To: 1706745600, Valid: true}
//	"2024-01-31,2024-01-01" → TimestampRange{..., Valid: false}
//	"2024-13-45"           → TimestampRange{Present: true, Valid: false}
func parseTimestampRange(value string, tz *time.Location) TimestampRange {
	// Default to UTC if no timezone specified
	if tz == nil {
		tz = time.UTC
	}

	// Use generic parseRange with time.ParseInLocation parser
	return parseRange(
		value,
		func(s string) (int64, error) {
			t, err := time.ParseInLocation("2006-01-02", s, tz)
			if err != nil {
				return 0, err
			}
			return t.Unix(), nil
		},
		func(from, to int64) bool { return from <= to },
	)
}

// Generic Range Parser Helper

// parseRange is a generic helper for parsing range values with custom parsers and validators.
// Reduces code duplication across parseIntRange, parseAmountRange, etc.
//
// Parameters:
//   - value: raw string value from query parameter (e.g., "100,500")
//   - parser: function to parse individual string value to type T (e.g., strconv.ParseInt)
//   - validator: function to validate from <= to relationship (return true if valid)
//
// Behavior:
//   - If value is empty: returns Range with Present=true, Valid=false
//   - If parsing fails: returns Range with Present=true, Valid=false
//   - If from > to: returns Range with Present=true, Valid=false (parsed values included)
//   - If single value: sets To = From
//
// Returns Range with From, To, Present, and Valid fields set appropriately.
//
// Example:
//
//	parseRange("100,500",
//	    func(s string) (int64, error) { return strconv.ParseInt(s, 10, 64) },
//	    func(from, to int64) bool { return from <= to },
//	)
//	// Returns: Range[int64]{From: 100, To: 500, Present: true, Valid: true}
func parseRange[T any](value string, parser func(string) (T, error), validator func(T, T) bool) Range[T] {
	value = strings.TrimSpace(value)
	if value == "" {
		var zero T
		return Range[T]{From: zero, To: zero, Present: true, Valid: false}
	}

	parts := strings.Split(value, ",")
	fromStr := strings.TrimSpace(parts[0])

	if fromStr == "" {
		var zero T
		return Range[T]{From: zero, To: zero, Present: true, Valid: false}
	}

	// Parse from value
	from, err := parser(fromStr)
	if err != nil {
		var zero T
		return Range[T]{From: zero, To: zero, Present: true, Valid: false}
	}

	// Default to = from (single value case)
	to := from
	if len(parts) > 1 && strings.TrimSpace(parts[1]) != "" {
		parsed, err := parser(strings.TrimSpace(parts[1]))
		if err != nil {
			return Range[T]{From: from, To: to, Present: true, Valid: false}
		}
		to = parsed
	}

	// Validate from <= to relationship
	if !validator(from, to) {
		return Range[T]{From: from, To: to, Present: true, Valid: false}
	}

	return Range[T]{From: from, To: to, Present: true, Valid: true}
}

// applyConstraints applies all registered constraint validators to the values.
// Processes all constraints found in the constraints map using registered validators.
// Returns error if any constraint validation fails.
func (fp *FilterParser) applyConstraints(values []string, constraints map[string]string, fieldType reflect.Type) error {
	for constraintName, constraintValue := range constraints {
		validator, ok := fp.constraintValidator[constraintName]
		if !ok {
			// Skip unknown constraints (allows future extensibility)
			continue
		}

		if err := validator.Validate(values, constraintValue, fieldType); err != nil {
			return err
		}
	}
	return nil
}

// Constraint Validator Interface

// ConstraintValidator defines an interface for custom constraint validation.
// Enables extensible validation system where users can implement custom constraints
// without modifying FilterParser.
//
// Design:
//   - Single interface for all constraint types (extensible)
//   - Registered via FilterParser.RegisterConstraintValidator()
//   - Multiple validators per field supported (applied sequentially)
//   - Unknown constraints gracefully skipped (forward-compatible)
//
// Use Cases:
//   - Enum validation (built-in "in" constraint)
//   - Min/max length constraints
//   - Regex pattern matching
//   - Custom business rule validation
//   - Type-specific constraints
//
// Implementations:
//   - Must be thread-safe (may be used concurrently)
//   - Should provide clear, actionable error messages
//   - Can access field type for type-aware validation
//
// Example implementation:
//
//	type RegexValidator struct {
//	    patterns map[string]*regexp.Regexp
//	}
//	func (v *RegexValidator) Name() string { return "regex" }
//	func (v *RegexValidator) Validate(values []string, constraint string, _ reflect.Type) error {
//	    pattern := v.patterns[constraint]
//	    for _, v := range values {
//	        if !pattern.MatchString(v) {
//	            return fmt.Errorf("does not match pattern: %s", constraint)
//	        }
//	    }
//	    return nil
//	}
type ConstraintValidator interface {
	// Name returns the constraint name (e.g., "in", "min", "max", "regex").
	// Used to match constraint tags in struct fields.
	// Must be unique within FilterParser's registered validators.
	Name() string

	// Validate validates the value(s) against the constraint.
	// Called during FilterParser.Parse() for fields with matching constraint.
	//
	// Parameters:
	//   - values: slice of string values to validate (may be single element for pointer types)
	//   - constraint: constraint parameter from tag (e.g., "active|pending" for tag "in:active|pending")
	//   - fieldType: the target field type (use for type-specific validation logic)
	//
	// Returns:
	//   - nil if validation passes
	//   - error with descriptive message if validation fails
	//   - Error message appears in fp.Errors() map with key "filters[fieldName]"
	Validate(values []string, constraint string, fieldType reflect.Type) error
}

// BuiltinConstraintValidators returns a map of built-in constraint validators.
// These validators handle common constraints like "in" for enums.
// Can be extended by adding custom validators to FilterParser.
func BuiltinConstraintValidators() map[string]ConstraintValidator {
	return map[string]ConstraintValidator{
		"in": &InConstraintValidator{},
	}
}

// InConstraintValidator implements ConstraintValidator for enum validation.
// Validates that values are in a predefined list of allowed values.
//
// Format:
//   - Tag: "fieldName,in:value1|value2|value3"
//   - Separator: pipe (|) between allowed values
//   - Whitespace: automatically trimmed from values and constraint
//
// Behavior:
//   - Applied to both pointer string and slice string types
//   - Single value (pointer) or multiple values (slice) validated
//   - Returns error if any value not in allowed list
//   - User-friendly error message listing allowed values
//
// Example usage:
//
//	type Filters struct {
//	    Status    *string   `filter:"status,in:active|pending|archived"`
//	    Statuses  []string  `filter:"statuses,in:active|pending"`
//	}
type InConstraintValidator struct{}

// Name returns the constraint name
func (v *InConstraintValidator) Name() string {
	return "in"
}

// Validate checks if values are in the allowed list.
// Constraint format: "value1|value2|value3" (pipe-separated)
//
// Process:
//  1. Parses constraint string by pipe separator
//  2. Trims whitespace from each allowed value
//  3. Validates each input value exists in allowed set
//  4. Returns first error encountered, or nil if all valid
//
// Error messages:
//   - "constraint tidak valid: tidak ada nilai yang diizinkan" - empty constraint
//   - "nilai tidak valid: {value} (diizinkan: {list})" - value not in allowed set
func (v *InConstraintValidator) Validate(values []string, constraint string, fieldType reflect.Type) error {
	// Build set of allowed values (pipe-separated)
	allowedValues := make(map[string]bool)
	for _, val := range strings.Split(constraint, "|") {
		trimmed := strings.TrimSpace(val)
		if trimmed != "" {
			allowedValues[trimmed] = true
		}
	}

	if len(allowedValues) == 0 {
		return fmt.Errorf("constraint tidak valid: tidak ada nilai yang diizinkan")
	}

	// Validate each value
	for _, value := range values {
		if !allowedValues[value] {
			// Build user-friendly list of allowed values
			allowed := make([]string, 0, len(allowedValues))
			for k := range allowedValues {
				allowed = append(allowed, k)
			}
			return fmt.Errorf("nilai tidak valid: %s (diizinkan: %s)", value, strings.Join(allowed, ", "))
		}
	}

	return nil
}
