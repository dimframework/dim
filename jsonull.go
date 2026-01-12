package dim

import "github.com/atfromhome/goreus/pkg/jsonull"

// JsonNull represents a nullable value that distinguishes between:
// - Field not present in JSON (Present=false)
// - Field explicitly set to null (Present=true, Valid=false)
// - Field with a value (Present=true, Valid=true)
//
// This is useful for partial updates in PATCH endpoints where you need to
// distinguish between "don't update this field" vs "set this field to null"
// vs "set this field to a new value".
type JsonNull[T any] = jsonull.JsonNull[T]

// NewJsonNull creates a JsonNull with a valid value
func NewJsonNull[T any](value T) JsonNull[T] {
	return jsonull.NewJsonNull(value)
}

// NewJsonNullNull creates a JsonNull representing an explicit null
func NewJsonNullNull[T any]() JsonNull[T] {
	return jsonull.NewJsonNullNull[T]()
}

// JsonNullFromPtr converts a pointer to JsonNull.
// nil pointer becomes null, non-nil becomes valid value
func JsonNullFromPtr[T any](ptr *T) JsonNull[T] {
	return jsonull.JsonNullFromPtr(ptr)
}
