package dim

import (
	"crypto/rand"
	"errors"
	"fmt"
	"time"
)

type UUID [16]byte

// String mengembalikan string representation UUID dalam standard format.
// Format: xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx (32 hex chars + 4 hyphens = 36 chars).
//
// Returns:
//   - string: UUID formatted string
//
// Example:
//
//	uuid := NewUuid()
//	fmt.Println(uuid.String())  // Output: 550e8400-e29b-41d4-a716-446655440000
func (u UUID) String() string {
	return fmt.Sprintf("%08x-%04x-%04x-%04x-%012x",
		u[0:4],
		u[4:6],
		u[6:8],
		u[8:10],
		u[10:16])
}

// NewUuid menghasilkan UUID baru menggunakan v7 (time-based) atau v4 (random) sebagai fallback.
// Prefer v7 karena sortable berdasarkan timestamp, lebih efficient untuk database indexing.
// Falls back ke v4 jika random generation untuk v7 gagal.
//
// Returns:
//   - UUID: newly generated UUID
//
// Example:
//
//	id := NewUuid()
//	user.ID = id.String()
func NewUuid() UUID {
	uuid, err := NewV7()
	if err != nil {
		return NewV4()
	}
	return uuid
}

// NewV7 menghasilkan UUID v7 (time-based dengan random component).
// Format: timestamp (48 bits) | version 7 (4 bits) | random (12 bits) | variant (2 bits) | random (62 bits).
// UUID v7 adalah sortable berdasarkan timestamp untuk efficient database operations.
// Berguna untuk primary keys dalam relational databases.
//
// Returns:
//   - UUID: newly generated v7 UUID
//   - error: error jika random generation gagal
//
// Example:
//
//	id, err := NewV7()
//	if err != nil {
//	  return err
//	}
//	log.Println(id.String())
func NewV7() (UUID, error) {
	// Timestamp in milliseconds (48 bits)
	timestamp := time.Now().UnixMilli()

	// Random bytes for the rest (10 bytes = 80 bits)
	randomBytes := make([]byte, 10)
	_, err := rand.Read(randomBytes)
	if err != nil {
		// Return error instead of fallback
		return UUID{}, err
	}

	// Build UUID v7 as [16]byte array
	var uuid UUID

	// Timestamp (48 bits = 6 bytes)
	uuid[0] = byte(timestamp >> 40)
	uuid[1] = byte(timestamp >> 32)
	uuid[2] = byte(timestamp >> 24)
	uuid[3] = byte(timestamp >> 16)
	uuid[4] = byte(timestamp >> 8)
	uuid[5] = byte(timestamp)

	// Version (4 bits) and random (12 bits)
	uuid[6] = (randomBytes[0] & 0x0f) | 0x70 // version 7
	uuid[7] = randomBytes[1]

	// Variant (2 bits) and random (62 bits)
	uuid[8] = (randomBytes[2] & 0x3f) | 0x80 // variant 1 (RFC 4122)
	copy(uuid[9:], randomBytes[3:])

	return uuid, nil
}

// NewV4 menghasilkan UUID v4 (random-based).
// Format: random (4 bytes) | version 4 (4 bits) | random (12 bits) | variant (2 bits) | random (62 bits).
// UUID v4 adalah fully random dan tidak sortable berdasarkan time.
// Returns zero UUID jika random generation gagal (rare edge case).
//
// Returns:
//   - UUID: newly generated v4 UUID
//
// Example:
//
//	id := NewV4()
//	session.ID = id.String()
func NewV4() UUID {
	randomBytes := make([]byte, 16)
	_, err := rand.Read(randomBytes)
	if err != nil {
		// This should not happen, but return a zero UUID if it does
		return UUID{}
	}

	// Build UUID v4 as [16]byte array
	var uuid UUID
	copy(uuid[:], randomBytes)

	// Set version to 4 (random)
	uuid[6] = (uuid[6] & 0x0f) | 0x40

	// Set variant to RFC 4122
	uuid[8] = (uuid[8] & 0x3f) | 0x80

	return uuid
}

// ParseUUID mengparse UUID string dalam standard format.
// Format: xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx (exactly 36 characters).
// Case-insensitive hex character validation.
// Returns error jika format invalid atau string bukan valid UUID.
//
// Parameters:
//   - s: UUID string untuk di-parse
//
// Returns:
//   - UUID: parsed UUID
//   - error: error jika string bukan valid UUID format
//
// Example:
//
//	uuid, err := ParseUUID("550e8400-e29b-41d4-a716-446655440000")
//	if err != nil {
//	  log.Fatal(err)
//	}
//	user.ID = uuid
func ParseUuid(s string) (UUID, error) {
	// Validate length (36 chars: 32 hex digits + 4 hyphens)
	if len(s) != 36 {
		return UUID{}, errors.New("invalid UUID string length")
	}

	// Validate hyphen positions
	if s[8] != '-' || s[13] != '-' || s[18] != '-' || s[23] != '-' {
		return UUID{}, errors.New("invalid UUID format: hyphens at wrong positions")
	}

	var uuid UUID
	var byteIndex int

	// Parse segments (without hyphens)
	segments := []string{
		s[0:8],   // time_low (4 bytes)
		s[9:13],  // time_mid (2 bytes)
		s[14:18], // time_hi_version (2 bytes)
		s[19:23], // clock_seq (2 bytes)
		s[24:36], // node (6 bytes)
	}

	for _, segment := range segments {
		for i := 0; i < len(segment); i += 2 {
			// Parse hex byte manually to validate characters
			var byte uint8
			for j := 0; j < 2; j++ {
				char := segment[i+j]
				byte <<= 4

				if char >= '0' && char <= '9' {
					byte |= uint8(char - '0')
				} else if char >= 'a' && char <= 'f' {
					byte |= uint8(char - 'a' + 10)
				} else if char >= 'A' && char <= 'F' {
					byte |= uint8(char - 'A' + 10)
				} else {
					return UUID{}, errors.New("invalid UUID hex characters")
				}
			}
			uuid[byteIndex] = byte
			byteIndex++
		}
	}

	return uuid, nil
}

// ParseUUIDFromString adalah alias untuk ParseUUID untuk consistency dengan naming conventions.
// Mengparse UUID string dalam standard format (xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx).
//
// Parameters:
//   - s: UUID string untuk di-parse
//
// Returns:
//   - UUID: parsed UUID
//   - error: error jika string bukan valid UUID format
//
// Example:
//
//	uuid, err := ParseUUIDFromString("550e8400-e29b-41d4-a716-446655440000")
func ParseUUIDFromString(s string) (UUID, error) {
	return ParseUuid(s)
}

// IsValidUuid mengecek apakah string adalah valid UUID tanpa melakukan full parsing.
// Lebih efficient daripada ParseUUID jika hanya perlu validation.
// Validates format, length, hyphen positions, dan hex characters.
//
// Parameters:
//   - s: string yang akan di-check
//
// Returns:
//   - bool: true jika valid UUID format, false sebaliknya
//
// Example:
//
//	if IsValidUuid(userID) {
//	  uuid, _ := ParseUUID(userID)
//	}
func IsValidUuid(s string) bool {
	// Validate length (36 chars: 32 hex digits + 4 hyphens)
	if len(s) != 36 {
		return false
	}

	// Validate hyphen positions
	if s[8] != '-' || s[13] != '-' || s[18] != '-' || s[23] != '-' {
		return false
	}

	// Validate hex characters in segments
	segments := []string{
		s[0:8],   // time_low (4 bytes)
		s[9:13],  // time_mid (2 bytes)
		s[14:18], // time_hi_version (2 bytes)
		s[19:23], // clock_seq (2 bytes)
		s[24:36], // node (6 bytes)
	}

	for _, segment := range segments {
		for _, char := range segment {
			if !IsHexChar(byte(char)) {
				return false
			}
		}
	}

	return true
}
