package dim

import (
	"net/http"
	"net/url"
	"reflect"
	"strconv"
	"testing"
	"time"
)

// TestNewFilterParser tests FilterParser creation
func TestNewFilterParser(t *testing.T) {
	req, _ := http.NewRequest("GET", "http://example.com", nil)
	fp := NewFilterParser(req)

	if fp == nil {
		t.Fatal("NewFilterParser returned nil")
	}
	if fp.request != req {
		t.Error("request not set correctly")
	}
	if len(fp.errors) != 0 {
		t.Error("errors should be empty on creation")
	}
	if fp.MaxValuesPerField != 0 {
		t.Error("MaxValuesPerField should default to 0 (unlimited)")
	}
}

// TestWithMaxValues tests setting max values limit
func TestWithMaxValues(t *testing.T) {
	req, _ := http.NewRequest("GET", "http://example.com", nil)
	fp := NewFilterParser(req).WithMaxValues(10)

	if fp.MaxValuesPerField != 10 {
		t.Errorf("MaxValuesPerField = %d, want 10", fp.MaxValuesPerField)
	}
}

// TestWithMaxValuesChaining tests method chaining
func TestWithMaxValuesChaining(t *testing.T) {
	req, _ := http.NewRequest("GET", "http://example.com", nil)
	fp := NewFilterParser(req).WithMaxValues(5).WithMaxValues(15)

	if fp.MaxValuesPerField != 15 {
		t.Errorf("MaxValuesPerField = %d, want 15 after second call", fp.MaxValuesPerField)
	}
}

// TestParseSliceMaxValuesUnlimited tests parsing without max limit (default)
func TestParseSliceMaxValuesUnlimited(t *testing.T) {
	q := url.Values{}
	q.Set("filters[ids]", "1,2,3,4,5,6,7,8,9,10")

	req, _ := http.NewRequest("GET", "http://example.com?"+q.Encode(), nil)

	type Filters struct {
		IDs []int64 `filter:"ids"`
	}

	var filters Filters
	fp := NewFilterParser(req)
	fp.Parse(&filters)

	if fp.HasErrors() {
		t.Errorf("Parse failed: %v", fp.Errors())
	}

	if len(filters.IDs) != 10 {
		t.Errorf("IDs length = %d, want 10", len(filters.IDs))
	}
}

// TestParseSliceMaxValuesExceeded tests parsing when values exceed limit
func TestParseSliceMaxValuesExceeded(t *testing.T) {
	q := url.Values{}
	q.Set("filters[ids]", "1,2,3,4,5")

	req, _ := http.NewRequest("GET", "http://example.com?"+q.Encode(), nil)

	type Filters struct {
		IDs []int64 `filter:"ids"`
	}

	var filters Filters
	fp := NewFilterParser(req).WithMaxValues(3)
	fp.Parse(&filters)

	if !fp.HasErrors() {
		t.Error("Parse should have errors when values exceed max limit")
	}

	errors := fp.Errors()
	if _, hasError := errors["filters[ids]"]; !hasError {
		t.Error("Expected error for filters[ids]")
	}

	// Check error message
	expectedMsg := "maksimal 3 nilai diperbolehkan, diterima 5"
	if errors["filters[ids]"] != expectedMsg {
		t.Errorf("Error message = %q, want %q", errors["filters[ids]"], expectedMsg)
	}
}

// TestParseSliceMaxValuesExactLimit tests parsing when values exactly match limit
func TestParseSliceMaxValuesExactLimit(t *testing.T) {
	q := url.Values{}
	q.Set("filters[ids]", "1,2,3")

	req, _ := http.NewRequest("GET", "http://example.com?"+q.Encode(), nil)

	type Filters struct {
		IDs []int64 `filter:"ids"`
	}

	var filters Filters
	fp := NewFilterParser(req).WithMaxValues(3)
	fp.Parse(&filters)

	if fp.HasErrors() {
		t.Errorf("Parse failed: %v", fp.Errors())
	}

	if len(filters.IDs) != 3 {
		t.Errorf("IDs length = %d, want 3", len(filters.IDs))
	}
}

// TestParseSliceMaxValuesStrings tests max values with string slices
func TestParseSliceMaxValuesStrings(t *testing.T) {
	q := url.Values{}
	q.Set("filters[tags]", "a,b,c,d,e")

	req, _ := http.NewRequest("GET", "http://example.com?"+q.Encode(), nil)

	type Filters struct {
		Tags []string `filter:"tags"`
	}

	var filters Filters
	fp := NewFilterParser(req).WithMaxValues(3)
	fp.Parse(&filters)

	if !fp.HasErrors() {
		t.Error("Parse should have errors when values exceed max limit for strings")
	}

	errors := fp.Errors()
	if _, hasError := errors["filters[tags]"]; !hasError {
		t.Error("Expected error for filters[tags]")
	}
}

// TestParseSliceMaxValuesUUID tests max values with UUID slices
func TestParseSliceMaxValuesUUID(t *testing.T) {
	uuid1 := "550e8400-e29b-41d4-a716-446655440000"
	uuid2 := "550e8400-e29b-41d4-a716-446655440001"
	uuid3 := "550e8400-e29b-41d4-a716-446655440002"

	q := url.Values{}
	q.Set("filters[ids]", uuid1+","+uuid2+","+uuid3)

	req, _ := http.NewRequest("GET", "http://example.com?"+q.Encode(), nil)

	type Filters struct {
		IDs []UUID `filter:"ids"`
	}

	var filters Filters
	fp := NewFilterParser(req).WithMaxValues(2)
	fp.Parse(&filters)

	if !fp.HasErrors() {
		t.Error("Parse should have errors when UUID values exceed max limit")
	}

	errors := fp.Errors()
	if _, hasError := errors["filters[ids]"]; !hasError {
		t.Error("Expected error for filters[ids]")
	}
}

// TestParseSliceMaxValuesZero tests that 0 means unlimited
func TestParseSliceMaxValuesZero(t *testing.T) {
	q := url.Values{}
	q.Set("filters[ids]", "1,2,3,4,5,6,7,8,9,10,11,12,13,14,15")

	req, _ := http.NewRequest("GET", "http://example.com?"+q.Encode(), nil)

	type Filters struct {
		IDs []int64 `filter:"ids"`
	}

	var filters Filters
	fp := NewFilterParser(req).WithMaxValues(0) // Explicitly set to 0
	fp.Parse(&filters)

	if fp.HasErrors() {
		t.Errorf("Parse failed with MaxValuesPerField=0: %v", fp.Errors())
	}

	if len(filters.IDs) != 15 {
		t.Errorf("IDs length = %d, want 15 when max is 0", len(filters.IDs))
	}
}

// TestParseSliceMaxValuesFloat64 tests max values with float64 slices
func TestParseSliceMaxValuesFloat64(t *testing.T) {
	q := url.Values{}
	q.Set("filters[prices]", "10.5,20.3,30.1,40.2,50.5")

	req, _ := http.NewRequest("GET", "http://example.com?"+q.Encode(), nil)

	type Filters struct {
		Prices []float64 `filter:"prices"`
	}

	var filters Filters
	fp := NewFilterParser(req).WithMaxValues(3)
	fp.Parse(&filters)

	if !fp.HasErrors() {
		t.Error("Parse should have errors when float64 values exceed max limit")
	}

	errors := fp.Errors()
	expectedMsg := "maksimal 3 nilai diperbolehkan, diterima 5"
	if errors["filters[prices]"] != expectedMsg {
		t.Errorf("Error message = %q, want %q", errors["filters[prices]"], expectedMsg)
	}
}

// TestParseMultipleFiltersWithMaxValues tests multiple fields with max values
func TestParseMultipleFiltersWithMaxValues(t *testing.T) {
	q := url.Values{}
	q.Set("filters[ids]", "1,2,3,4,5") // Exceeds limit
	q.Set("filters[tags]", "a,b,c")    // Within limit

	req, _ := http.NewRequest("GET", "http://example.com?"+q.Encode(), nil)

	type Filters struct {
		IDs  []int64  `filter:"ids"`
		Tags []string `filter:"tags"`
	}

	var filters Filters
	fp := NewFilterParser(req).WithMaxValues(3)
	fp.Parse(&filters)

	if !fp.HasErrors() {
		t.Error("Parse should have errors for ids field")
	}

	errors := fp.Errors()
	if _, hasError := errors["filters[ids]"]; !hasError {
		t.Error("Expected error for filters[ids]")
	}

	// tags should be set correctly despite ids error
	if len(filters.Tags) != 3 {
		t.Errorf("Tags length = %d, want 3", len(filters.Tags))
	}
}

// TestParseSliceMaxValuesBelow tests parsing when values are below limit
func TestParseSliceMaxValuesBelow(t *testing.T) {
	q := url.Values{}
	q.Set("filters[ids]", "1,2")

	req, _ := http.NewRequest("GET", "http://example.com?"+q.Encode(), nil)

	type Filters struct {
		IDs []int64 `filter:"ids"`
	}

	var filters Filters
	fp := NewFilterParser(req).WithMaxValues(5)
	fp.Parse(&filters)

	if fp.HasErrors() {
		t.Errorf("Parse failed: %v", fp.Errors())
	}

	if len(filters.IDs) != 2 {
		t.Errorf("IDs length = %d, want 2", len(filters.IDs))
	}
}

// TestParseSliceMaxValuesInt tests max values with int slices
func TestParseSliceMaxValuesInt(t *testing.T) {
	q := url.Values{}
	q.Set("filters[counts]", "10,20,30,40,50")

	req, _ := http.NewRequest("GET", "http://example.com?"+q.Encode(), nil)

	type Filters struct {
		Counts []int `filter:"counts"`
	}

	var filters Filters
	fp := NewFilterParser(req).WithMaxValues(3)
	fp.Parse(&filters)

	if !fp.HasErrors() {
		t.Error("Parse should have errors when int values exceed max limit")
	}

	errors := fp.Errors()
	expectedMsg := "maksimal 3 nilai diperbolehkan, diterima 5"
	if errors["filters[counts]"] != expectedMsg {
		t.Errorf("Error message = %q, want %q", errors["filters[counts]"], expectedMsg)
	}
}

// TestParseSliceEmptyValues tests that empty value list doesn't trigger max check
func TestParseSliceEmptyValues(t *testing.T) {
	req, _ := http.NewRequest("GET", "http://example.com", nil)

	type Filters struct {
		IDs []int64 `filter:"ids"`
	}

	var filters Filters
	fp := NewFilterParser(req).WithMaxValues(1)
	fp.Parse(&filters)

	if fp.HasErrors() {
		t.Errorf("Parse failed: %v", fp.Errors())
	}

	if filters.IDs != nil {
		t.Errorf("IDs should be nil for empty values")
	}
}

// TestHasErrors tests HasErrors method
func TestHasErrors(t *testing.T) {
	req, _ := http.NewRequest("GET", "http://example.com", nil)
	fp := NewFilterParser(req)

	if fp.HasErrors() {
		t.Error("HasErrors should return false for new FilterParser")
	}

	// Add an error
	fp.errors["test"] = "test error"
	if !fp.HasErrors() {
		t.Error("HasErrors should return true after adding error")
	}
}

// TestErrors tests Errors method
func TestErrors(t *testing.T) {
	req, _ := http.NewRequest("GET", "http://example.com", nil)
	fp := NewFilterParser(req)

	if len(fp.Errors()) != 0 {
		t.Error("Errors should be empty for new FilterParser")
	}

	fp.errors["field1"] = "error1"
	fp.errors["field2"] = "error2"

	errors := fp.Errors()
	if len(errors) != 2 {
		t.Errorf("Errors count = %d, want 2", len(errors))
	}

	if errors["field1"] != "error1" {
		t.Errorf("errors[field1] = %q, want %q", errors["field1"], "error1")
	}

	if errors["field2"] != "error2" {
		t.Errorf("errors[field2] = %q, want %q", errors["field2"], "error2")
	}
}

// TestWithTimezone tests setting timezone for timestamp parsing
func TestWithTimezone(t *testing.T) {
	req, _ := http.NewRequest("GET", "http://example.com", nil)
	jakartaTz, _ := time.LoadLocation("Asia/Jakarta")
	fp := NewFilterParser(req).WithTimezone(jakartaTz)

	if fp.TimestampTimezone != jakartaTz {
		t.Error("Timezone not set correctly")
	}
}

// TestWithTimezoneChaining tests timezone method chaining
func TestWithTimezoneChaining(t *testing.T) {
	req, _ := http.NewRequest("GET", "http://example.com", nil)
	jakartaTz, _ := time.LoadLocation("Asia/Jakarta")
	tokyoTz, _ := time.LoadLocation("Asia/Tokyo")
	fp := NewFilterParser(req).WithTimezone(jakartaTz).WithTimezone(tokyoTz)

	if fp.TimestampTimezone != tokyoTz {
		t.Errorf("Timezone not updated to Tokyo after chaining")
	}
}

// TestParseTimestampRangeUTC tests parsing with UTC timezone (default)
func TestParseTimestampRangeUTC(t *testing.T) {
	q := url.Values{}
	q.Set("filters[date]", "2024-01-15,2024-01-20")

	req, _ := http.NewRequest("GET", "http://example.com?"+q.Encode(), nil)

	type Filters struct {
		Date TimestampRange `filter:"date"`
	}

	var filters Filters
	fp := NewFilterParser(req) // Default UTC
	fp.Parse(&filters)

	if fp.HasErrors() {
		t.Errorf("Parse failed: %v", fp.Errors())
	}

	if !filters.Date.Valid {
		t.Error("TimestampRange should be valid")
	}

	// Verify timestamps are in UTC
	if filters.Date.Present != true || filters.Date.Valid != true {
		t.Error("Date range not properly parsed")
	}
}

// TestParseTimestampRangeWithTimezone tests parsing with custom timezone
func TestParseTimestampRangeWithTimezone(t *testing.T) {
	q := url.Values{}
	q.Set("filters[date]", "2024-01-15")

	req, _ := http.NewRequest("GET", "http://example.com?"+q.Encode(), nil)

	type Filters struct {
		Date TimestampRange `filter:"date"`
	}

	var filters Filters
	jakartaTz, _ := time.LoadLocation("Asia/Jakarta")
	fp := NewFilterParser(req).WithTimezone(jakartaTz)
	fp.Parse(&filters)

	if fp.HasErrors() {
		t.Errorf("Parse failed: %v", fp.Errors())
	}

	if !filters.Date.Valid {
		t.Error("TimestampRange should be valid with Jakarta timezone")
	}

	// Parse same date with UTC for comparison
	utcTime, _ := time.Parse("2006-01-02", "2024-01-15")
	jakartaTime, _ := time.ParseInLocation("2006-01-02", "2024-01-15", jakartaTz)

	// The Unix timestamps should be different due to timezone difference
	if utcTime.Unix() == jakartaTime.Unix() {
		t.Error("UTC and Jakarta timestamps should be different")
	}
}

// TestParseTimestampRangeMultipleWithTimezone tests range parsing with timezone
func TestParseTimestampRangeMultipleWithTimezone(t *testing.T) {
	q := url.Values{}
	q.Set("filters[period]", "2024-01-01,2024-01-31")

	req, _ := http.NewRequest("GET", "http://example.com?"+q.Encode(), nil)

	type Filters struct {
		Period TimestampRange `filter:"period"`
	}

	var filters Filters
	tokyoTz, _ := time.LoadLocation("Asia/Tokyo")
	fp := NewFilterParser(req).WithTimezone(tokyoTz)
	fp.Parse(&filters)

	if fp.HasErrors() {
		t.Errorf("Parse failed: %v", fp.Errors())
	}

	if !filters.Period.Valid {
		t.Error("TimestampRange should be valid")
	}

	if filters.Period.From >= filters.Period.To {
		t.Error("From should be less than To")
	}
}

// TestParseTimestampRangeNilTimezoneDefaultsToUTC tests nil timezone defaults to UTC
func TestParseTimestampRangeNilTimezoneDefaultsToUTC(t *testing.T) {
	q := url.Values{}
	q.Set("filters[date]", "2024-01-15")

	req, _ := http.NewRequest("GET", "http://example.com?"+q.Encode(), nil)

	type Filters struct {
		Date TimestampRange `filter:"date"`
	}

	var filters Filters
	fp := NewFilterParser(req) // No timezone set, should use UTC
	fp.Parse(&filters)

	if fp.HasErrors() {
		t.Errorf("Parse failed: %v", fp.Errors())
	}

	// Verify it matches UTC parse
	utcTime, _ := time.Parse("2006-01-02", "2024-01-15")
	if filters.Date.From != utcTime.Unix() {
		t.Errorf("From timestamp = %d, want %d (UTC)", filters.Date.From, utcTime.Unix())
	}
}

// TestParseTimestampRangePointerWithTimezone tests pointer TimestampRange with timezone
// NOTE: Due to type alias limitation (IntRange = Range[int64] and TimestampRange = Range[int64])
// we cannot have *TimestampRange fields. Use non-pointer TimestampRange instead.
func TestParseTimestampRangePointerWithTimezone(t *testing.T) {
	q := url.Values{}
	q.Set("filters[date]", "2024-06-01,2024-06-30")

	req, _ := http.NewRequest("GET", "http://example.com?"+q.Encode(), nil)

	type Filters struct {
		Date TimestampRange `filter:"date"` // Non-pointer to avoid type alias collision
	}

	var filters Filters
	bangkokTz, _ := time.LoadLocation("Asia/Bangkok")
	fp := NewFilterParser(req).WithTimezone(bangkokTz)
	fp.Parse(&filters)

	if fp.HasErrors() {
		t.Errorf("Parse failed: %v", fp.Errors())
	}

	if !filters.Date.Valid {
		t.Error("TimestampRange should be valid")
	}
}

// TestParseTimestampRangeInvalidWithTimezone tests invalid date with timezone
func TestParseTimestampRangeInvalidWithTimezone(t *testing.T) {
	q := url.Values{}
	q.Set("filters[date]", "2024-13-45") // Invalid date

	req, _ := http.NewRequest("GET", "http://example.com?"+q.Encode(), nil)

	type Filters struct {
		Date TimestampRange `filter:"date"`
	}

	var filters Filters
	shanghaiTz, _ := time.LoadLocation("Asia/Shanghai")
	fp := NewFilterParser(req).WithTimezone(shanghaiTz)
	fp.Parse(&filters)

	if !fp.HasErrors() {
		t.Error("Parse should have errors for invalid date")
	}
}

// TestParseTimestampRangeReverseOrderWithTimezone tests From > To validation with timezone
func TestParseTimestampRangeReverseOrderWithTimezone(t *testing.T) {
	q := url.Values{}
	q.Set("filters[date]", "2024-12-31,2024-01-01") // From > To

	req, _ := http.NewRequest("GET", "http://example.com?"+q.Encode(), nil)

	type Filters struct {
		Date TimestampRange `filter:"date"`
	}

	var filters Filters
	delhiTz, _ := time.LoadLocation("Asia/Kolkata")
	fp := NewFilterParser(req).WithTimezone(delhiTz)
	fp.Parse(&filters)

	if !fp.HasErrors() {
		t.Error("Parse should have errors when From > To")
	}
}

// Type Caching Tests

// TestGetCachedType tests caching of reflect.Type
func TestGetCachedType(t *testing.T) {
	rt1 := reflect.TypeOf(DateRange{})
	rt2 := reflect.TypeOf(DateRange{})

	// Both should return the same cached instance
	cached1 := getCachedType(rt1)
	cached2 := getCachedType(rt2)

	if cached1.String() != cached2.String() {
		t.Errorf("Cached types should match: %s vs %s", cached1.String(), cached2.String())
	}
}

// TestGetCachedTypeNil tests caching with nil type
func TestGetCachedTypeNil(t *testing.T) {
	result := getCachedType(nil)
	if result != nil {
		t.Error("getCachedType(nil) should return nil")
	}
}

// TestTypeMatchesWithCache tests typeMatches with caching
func TestTypeMatchesWithCache(t *testing.T) {
	rt1 := reflect.TypeOf(AmountRange{})
	rt2 := reflect.TypeOf(AmountRange{})

	// First call should cache types
	result1 := typeMatches(rt1, rt2)
	if !result1 {
		t.Error("AmountRange types should match")
	}

	// Second call should use cache
	result2 := typeMatches(rt1, rt2)
	if !result2 {
		t.Error("AmountRange types should still match (from cache)")
	}
}

// TestTypeMatchesDifferentTypes tests typeMatches with different types
func TestTypeMatchesDifferentTypes(t *testing.T) {
	rt1 := reflect.TypeOf(IntRange{})
	rt2 := reflect.TypeOf(AmountRange{})

	result := typeMatches(rt1, rt2)
	if result {
		t.Error("IntRange and AmountRange should not match")
	}
}

// TestTypeMatchesNilTypes tests typeMatches with nil types
func TestTypeMatchesNilTypes(t *testing.T) {
	if !typeMatches(nil, nil) {
		t.Error("typeMatches(nil, nil) should return true")
	}

	rt := reflect.TypeOf(DateRange{})
	if typeMatches(rt, nil) {
		t.Error("typeMatches(type, nil) should return false")
	}

	if typeMatches(nil, rt) {
		t.Error("typeMatches(nil, type) should return false")
	}
}

// TestTypeCacheMultipleCalls tests that cache reduces reflection overhead
func TestTypeCacheMultipleCalls(t *testing.T) {
	// Create multiple types
	types := []reflect.Type{
		reflect.TypeOf(DateRange{}),
		reflect.TypeOf(AmountRange{}),
		reflect.TypeOf(IntRange{}),
		reflect.TypeOf(TimestampRange{}),
		reflect.TypeOf(UUID{}),
	}

	// Cache them all
	for _, rt := range types {
		getCachedType(rt)
	}

	// Verify they're all cached by checking they can be retrieved
	for _, rt := range types {
		cached := getCachedType(rt)
		if cached.String() != rt.String() {
			t.Errorf("Cached type mismatch: %s vs %s", cached.String(), rt.String())
		}
	}
}

// BenchmarkTypeMatchesWithoutCache benchmarks type matching without cache utilization
func BenchmarkTypeMatchesWithoutCache(b *testing.B) {
	rt1 := reflect.TypeOf(DateRange{})
	rt2 := reflect.TypeOf(DateRange{})

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = typeMatches(rt1, rt2)
	}
}

// BenchmarkGetCachedType benchmarks type caching
func BenchmarkGetCachedType(b *testing.B) {
	rt := reflect.TypeOf(AmountRange{})

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = getCachedType(rt)
	}
}

// BenchmarkTypeMatchesMultipleCalls benchmarks repeated type matching (cache benefits)
func BenchmarkTypeMatchesMultipleCalls(b *testing.B) {
	rt1 := reflect.TypeOf(IntRange{})
	rt2 := reflect.TypeOf(IntRange{})

	// Warm up cache
	typeMatches(rt1, rt2)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = typeMatches(rt1, rt2)
	}
}

// Constraint Validator Tests

// TestBuiltinConstraintValidators tests BuiltinConstraintValidators function
func TestBuiltinConstraintValidators(t *testing.T) {
	validators := BuiltinConstraintValidators()

	if validators == nil {
		t.Fatal("BuiltinConstraintValidators returned nil")
	}

	if len(validators) == 0 {
		t.Error("BuiltinConstraintValidators should return at least one validator")
	}

	if _, ok := validators["in"]; !ok {
		t.Error("Expected 'in' validator in built-in validators")
	}
}

// DummyValidator is a test validator implementation
type DummyValidator struct{}

func (v *DummyValidator) Name() string {
	return "dummy"
}

func (v *DummyValidator) Validate([]string, string, reflect.Type) error {
	return nil
}

// TestRegisterConstraintValidator tests RegisterConstraintValidator method
func TestRegisterConstraintValidator(t *testing.T) {
	req, _ := http.NewRequest("GET", "http://example.com", nil)
	fp := NewFilterParser(req)

	fp.RegisterConstraintValidator(&DummyValidator{})

	if _, ok := fp.constraintValidator["dummy"]; !ok {
		t.Error("Custom validator not registered")
	}

	// Verify chaining works
	result := fp.RegisterConstraintValidator(&DummyValidator{})
	if result != fp {
		t.Error("RegisterConstraintValidator should return receiver for chaining")
	}
}

// TestInConstraintValidator tests InConstraintValidator
func TestInConstraintValidator(t *testing.T) {
	validator := &InConstraintValidator{}

	tests := []struct {
		name       string
		values     []string
		constraint string
		wantErr    bool
	}{
		{
			name:       "valid single value",
			values:     []string{"active"},
			constraint: "active|pending",
			wantErr:    false,
		},
		{
			name:       "valid multiple values",
			values:     []string{"active", "pending"},
			constraint: "active|pending|archived",
			wantErr:    false,
		},
		{
			name:       "invalid value",
			values:     []string{"invalid"},
			constraint: "active|pending",
			wantErr:    true,
		},
		{
			name:       "empty constraint",
			values:     []string{"value"},
			constraint: "",
			wantErr:    true,
		},
		{
			name:       "whitespace handling",
			values:     []string{"active"},
			constraint: "active | pending | archived",
			wantErr:    false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validator.Validate(tt.values, tt.constraint, reflect.TypeOf(""))
			if (err != nil) != tt.wantErr {
				t.Errorf("Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

// TestInConstraintValidatorName tests InConstraintValidator.Name()
func TestInConstraintValidatorName(t *testing.T) {
	validator := &InConstraintValidator{}
	if validator.Name() != "in" {
		t.Errorf("Name() = %s, want 'in'", validator.Name())
	}
}

// TestParseWithConstraintValidator tests parsing with constraint validators
func TestParseWithConstraintValidator(t *testing.T) {
	q := url.Values{}
	q.Set("filters[status]", "active")

	req, _ := http.NewRequest("GET", "http://example.com?"+q.Encode(), nil)

	type Filters struct {
		Status *string `filter:"status,in:active|pending"`
	}

	var filters Filters
	fp := NewFilterParser(req)
	fp.Parse(&filters)

	if fp.HasErrors() {
		t.Errorf("Parse failed: %v", fp.Errors())
	}

	if filters.Status == nil || *filters.Status != "active" {
		t.Errorf("Status = %v, want 'active'", filters.Status)
	}
}

// TestParseWithConstraintValidatorFailure tests parsing with failed constraint validation
func TestParseWithConstraintValidatorFailure(t *testing.T) {
	q := url.Values{}
	q.Set("filters[status]", "invalid")

	req, _ := http.NewRequest("GET", "http://example.com?"+q.Encode(), nil)

	type Filters struct {
		Status *string `filter:"status,in:active|pending"`
	}

	var filters Filters
	fp := NewFilterParser(req)
	fp.Parse(&filters)

	if !fp.HasErrors() {
		t.Error("Parse should have errors for invalid constraint value")
	}

	errors := fp.Errors()
	if _, hasError := errors["filters[status]"]; !hasError {
		t.Error("Expected error for filters[status]")
	}
}

// TestParseWithSliceConstraintValidator tests slice parsing with constraint validation
func TestParseWithSliceConstraintValidator(t *testing.T) {
	q := url.Values{}
	q.Set("filters[statuses]", "active,pending")

	req, _ := http.NewRequest("GET", "http://example.com?"+q.Encode(), nil)

	type Filters struct {
		Statuses []string `filter:"statuses,in:active|pending|archived"`
	}

	var filters Filters
	fp := NewFilterParser(req)
	fp.Parse(&filters)

	if fp.HasErrors() {
		t.Errorf("Parse failed: %v", fp.Errors())
	}

	if len(filters.Statuses) != 2 {
		t.Errorf("Statuses length = %d, want 2", len(filters.Statuses))
	}
}

// TestParseWithSliceConstraintValidatorFailure tests slice parsing with failed constraint validation
func TestParseWithSliceConstraintValidatorFailure(t *testing.T) {
	q := url.Values{}
	q.Set("filters[statuses]", "active,invalid")

	req, _ := http.NewRequest("GET", "http://example.com?"+q.Encode(), nil)

	type Filters struct {
		Statuses []string `filter:"statuses,in:active|pending"`
	}

	var filters Filters
	fp := NewFilterParser(req)
	fp.Parse(&filters)

	if !fp.HasErrors() {
		t.Error("Parse should have errors for invalid value in slice")
	}
}

// TestApplyConstraints tests applyConstraints method
func TestApplyConstraints(t *testing.T) {
	req, _ := http.NewRequest("GET", "http://example.com", nil)
	fp := NewFilterParser(req)

	constraints := map[string]string{
		"in": "active|pending",
	}

	tests := []struct {
		name    string
		values  []string
		wantErr bool
	}{
		{
			name:    "valid values",
			values:  []string{"active", "pending"},
			wantErr: false,
		},
		{
			name:    "invalid value",
			values:  []string{"invalid"},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := fp.applyConstraints(tt.values, constraints, reflect.TypeOf(""))
			if (err != nil) != tt.wantErr {
				t.Errorf("applyConstraints() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

// Generic Range Parser Tests

// TestParseRangeInt64 tests parseRange with int64 type
func TestParseRangeInt64(t *testing.T) {
	tests := []struct {
		name      string
		value     string
		wantFrom  int64
		wantTo    int64
		wantValid bool
		wantError string
	}{
		{
			name:      "valid single value",
			value:     "100",
			wantFrom:  100,
			wantTo:    100,
			wantValid: true,
		},
		{
			name:      "valid range",
			value:     "100,500",
			wantFrom:  100,
			wantTo:    500,
			wantValid: true,
		},
		{
			name:      "invalid range from > to",
			value:     "500,100",
			wantFrom:  500,
			wantTo:    100,
			wantValid: false,
		},
		{
			name:      "empty value",
			value:     "",
			wantFrom:  0,
			wantTo:    0,
			wantValid: false,
		},
		{
			name:      "whitespace only",
			value:     "   ",
			wantFrom:  0,
			wantTo:    0,
			wantValid: false,
		},
		{
			name:      "invalid parse first value",
			value:     "abc",
			wantFrom:  0,
			wantTo:    0,
			wantValid: false,
		},
		{
			name:      "invalid parse second value",
			value:     "100,abc",
			wantFrom:  100,
			wantTo:    100,
			wantValid: false,
		},
		{
			name:      "negative values valid range",
			value:     "-500,-100",
			wantFrom:  -500,
			wantTo:    -100,
			wantValid: true,
		},
		{
			name:      "zero value",
			value:     "0",
			wantFrom:  0,
			wantTo:    0,
			wantValid: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := parseRange(
				tt.value,
				func(s string) (int64, error) { return strconv.ParseInt(s, 10, 64) },
				func(from, to int64) bool { return from <= to },
			)

			if result.From != tt.wantFrom {
				t.Errorf("From = %d, want %d", result.From, tt.wantFrom)
			}
			if result.To != tt.wantTo {
				t.Errorf("To = %d, want %d", result.To, tt.wantTo)
			}
			if result.Valid != tt.wantValid {
				t.Errorf("Valid = %v, want %v", result.Valid, tt.wantValid)
			}
			if !result.Present {
				t.Error("Present should always be true")
			}
		})
	}
}

// TestParseRangeFloat64 tests parseRange with float64 type
func TestParseRangeFloat64(t *testing.T) {
	tests := []struct {
		name      string
		value     string
		wantFrom  float64
		wantTo    float64
		wantValid bool
	}{
		{
			name:      "valid single decimal",
			value:     "100.50",
			wantFrom:  100.50,
			wantTo:    100.50,
			wantValid: true,
		},
		{
			name:      "valid range decimals",
			value:     "100.50,500.75",
			wantFrom:  100.50,
			wantTo:    500.75,
			wantValid: true,
		},
		{
			name:      "invalid range from > to",
			value:     "500.75,100.50",
			wantFrom:  500.75,
			wantTo:    100.50,
			wantValid: false,
		},
		{
			name:      "integer parsing as float",
			value:     "100,500",
			wantFrom:  100.0,
			wantTo:    500.0,
			wantValid: true,
		},
		{
			name:      "scientific notation",
			value:     "1.5e2,3.5e2",
			wantFrom:  150.0,
			wantTo:    350.0,
			wantValid: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := parseRange(
				tt.value,
				func(s string) (float64, error) { return strconv.ParseFloat(s, 64) },
				func(from, to float64) bool { return from <= to },
			)

			if result.From != tt.wantFrom {
				t.Errorf("From = %f, want %f", result.From, tt.wantFrom)
			}
			if result.To != tt.wantTo {
				t.Errorf("To = %f, want %f", result.To, tt.wantTo)
			}
			if result.Valid != tt.wantValid {
				t.Errorf("Valid = %v, want %v", result.Valid, tt.wantValid)
			}
		})
	}
}

// TestParseRangeCustomValidator tests parseRange with custom validator
func TestParseRangeCustomValidator(t *testing.T) {
	tests := []struct {
		name      string
		value     string
		validator func(int64, int64) bool
		wantValid bool
	}{
		{
			name:      "default validator from <= to",
			value:     "100,500",
			validator: func(from, to int64) bool { return from <= to },
			wantValid: true,
		},
		{
			name:      "strict validator from < to",
			value:     "100,100",
			validator: func(from, to int64) bool { return from < to },
			wantValid: false,
		},
		{
			name:      "strict validator from < to valid",
			value:     "100,200",
			validator: func(from, to int64) bool { return from < to },
			wantValid: true,
		},
		{
			name:      "custom validator modulo check",
			value:     "10,20",
			validator: func(from, to int64) bool { return (from+to)%2 == 0 },
			wantValid: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := parseRange(
				tt.value,
				func(s string) (int64, error) { return strconv.ParseInt(s, 10, 64) },
				tt.validator,
			)

			if result.Valid != tt.wantValid {
				t.Errorf("Valid = %v, want %v", result.Valid, tt.wantValid)
			}
		})
	}
}

// TestParseRangeEquivalence tests that refactored parsers produce same results as before
func TestParseRangeEquivalence(t *testing.T) {
	tests := []struct {
		name        string
		valueAmount string
		valueInt    string
		checkAmount func(AmountRange) bool
		checkInt    func(IntRange) bool
	}{
		{
			name:        "amount range valid",
			valueAmount: "100.50,500.75",
			checkAmount: func(ar AmountRange) bool {
				return ar.From == 100.50 && ar.To == 500.75 && ar.Valid
			},
		},
		{
			name:        "amount range invalid order",
			valueAmount: "500.75,100.50",
			checkAmount: func(ar AmountRange) bool {
				return ar.From == 500.75 && ar.To == 100.50 && !ar.Valid
			},
		},
		{
			name:     "int range valid",
			valueInt: "100,500",
			checkInt: func(ir IntRange) bool {
				return ir.From == 100 && ir.To == 500 && ir.Valid
			},
		},
		{
			name:     "int range invalid order",
			valueInt: "500,100",
			checkInt: func(ir IntRange) bool {
				return ir.From == 500 && ir.To == 100 && !ir.Valid
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.checkAmount != nil {
				ar := parseAmountRange(tt.valueAmount)
				if !tt.checkAmount(ar) {
					t.Errorf("parseAmountRange mismatch: %+v", ar)
				}
			}
			if tt.checkInt != nil {
				ir := parseIntRange(tt.valueInt)
				if !tt.checkInt(ir) {
					t.Errorf("parseIntRange mismatch: %+v", ir)
				}
			}
		})
	}
}
