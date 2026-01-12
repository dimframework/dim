package dim

import (
	"context"
	"fmt"
	"strings"
	"time"
)

// User represents a user entity
type User struct {
	ID        int64     `json:"id"`
	Email     string    `json:"email"`
	Name      string    `json:"name"`
	Password  string    `json:"-"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// UpdateUserRequest represents a partial update request for a user
// Fields use JsonNull to distinguish between:
// - Not sent (don't update)
// - Sent as null (clear field - if applicable)
// - Sent with value (update to new value)
type UpdateUserRequest struct {
	Email    JsonNull[string] `json:"email"`
	Name     JsonNull[string] `json:"name"`
	Password JsonNull[string] `json:"password"`
}

// UserStore defines the interface for user storage operations
type UserStore interface {
	Create(ctx context.Context, user *User) error
	FindByID(ctx context.Context, id int64) (*User, error)
	FindByEmail(ctx context.Context, email string) (*User, error)
	Update(ctx context.Context, user *User) error
	Delete(ctx context.Context, id int64) error
	Exists(ctx context.Context, email string) (bool, error)
}

// PostgresUserStore is the PostgreSQL implementation of UserStore
type PostgresUserStore struct {
	db Database
}

// NewPostgresUserStore membuat PostgreSQL user store baru.
// Store ini menangani operasi CRUD untuk user entities.
//
// Parameters:
//   - db: Database instance untuk execute queries
//
// Returns:
//   - *PostgresUserStore: user store instance
//
// Example:
//
//	userStore := NewPostgresUserStore(db)
func NewPostgresUserStore(db Database) *PostgresUserStore {
	return &PostgresUserStore{db: db}
}

// Create membuat user baru dan menyimpannya ke database.
// Auto-generate ID dan timestamps, hanya perlu email, name, password.
//
// Parameters:
//   - ctx: context untuk membatalkan operasi
//   - user: User struct dengan data yang akan disimpan
//
// Returns:
//   - error: error jika INSERT query gagal (misalnya duplicate email)
//
// Example:
//
//	err := userStore.Create(ctx, &user)
func (s *PostgresUserStore) Create(ctx context.Context, user *User) error {
	err := s.db.QueryRow(ctx,
		`INSERT INTO users (email, name, password, created_at, updated_at)
		 VALUES ($1, $2, $3, $4, $5)
		 RETURNING id, created_at, updated_at`,
		user.Email,
		user.Name,
		user.Password,
		time.Now(),
		time.Now(),
	).Scan(&user.ID, &user.CreatedAt, &user.UpdatedAt)

	if err != nil {
		return fmt.Errorf("failed to create user: %w", err)
	}

	return nil
}

// FindByID mencari user berdasarkan ID.
//
// Parameters:
//   - ctx: context untuk membatalkan operasi
//   - id: user ID yang akan dicari
//
// Returns:
//   - *User: User struct jika ditemukan
//   - error: error jika user tidak ditemukan atau query gagal
//
// Example:
//
//	user, err := userStore.FindByID(ctx, 123)
func (s *PostgresUserStore) FindByID(ctx context.Context, id int64) (*User, error) {
	user := &User{}
	err := s.db.QueryRow(ctx,
		`SELECT id, email, name, password, created_at, updated_at
		 FROM users WHERE id = $1`,
		id,
	).Scan(&user.ID, &user.Email, &user.Name, &user.Password, &user.CreatedAt, &user.UpdatedAt)

	if err != nil {
		return nil, fmt.Errorf("failed to find user by id: %w", err)
	}

	return user, nil
}

// FindByEmail mencari user berdasarkan email address.
//
// Parameters:
//   - ctx: context untuk membatalkan operasi
//   - email: alamat email user yang akan dicari
//
// Returns:
//   - *User: User struct jika ditemukan
//   - error: error jika user tidak ditemukan atau query gagal
//
// Example:
//
//	user, err := userStore.FindByEmail(ctx, "user@example.com")
func (s *PostgresUserStore) FindByEmail(ctx context.Context, email string) (*User, error) {
	user := &User{}
	err := s.db.QueryRow(ctx,
		`SELECT id, email, name, password, created_at, updated_at
		 FROM users WHERE email = $1`,
		email,
	).Scan(&user.ID, &user.Email, &user.Name, &user.Password, &user.CreatedAt, &user.UpdatedAt)

	if err != nil {
		return nil, fmt.Errorf("failed to find user by email: %w", err)
	}

	return user, nil
}

// Update mengupdate semua field user dan auto-update updated_at timestamp.
// Replace seluruh user dengan data baru.
//
// Parameters:
//   - ctx: context untuk membatalkan operasi
//   - user: User struct dengan data baru (harus sudah ada ID)
//
// Returns:
//   - error: error jika UPDATE query gagal
//
// Example:
//
//	user.Name = "New Name"
//	err := userStore.Update(ctx, user)
func (s *PostgresUserStore) Update(ctx context.Context, user *User) error {
	user.UpdatedAt = time.Now()

	err := s.db.Exec(ctx,
		`UPDATE users SET email = $1, name = $2, password = $3, updated_at = $4
		 WHERE id = $5`,
		user.Email,
		user.Name,
		user.Password,
		user.UpdatedAt,
		user.ID,
	)

	if err != nil {
		return fmt.Errorf("failed to update user: %w", err)
	}

	return nil
}

// UpdatePartial melakukan partial update user fields berdasarkan request.
// Hanya update field yang Present dan Valid dalam JsonNull wrapper.
// Field yang tidak ada di request tidak akan diubah.
//
// Parameters:
//   - ctx: context untuk membatalkan operasi
//   - id: user ID yang akan diupdate
//   - req: UpdateUserRequest dengan field-field yang akan diupdate
//
// Returns:
//   - error: error jika UPDATE query gagal
//
// Example:
//
//	req := &UpdateUserRequest{
//	  Name: JsonNull[string]{Present: true, Valid: true, Value: "New Name"},
//	}
//	err := userStore.UpdatePartial(ctx, userID, req)
func (s *PostgresUserStore) UpdatePartial(ctx context.Context, id int64, req *UpdateUserRequest) error {
	var setClauses []string
	var args []interface{}
	argIndex := 1

	// Check each field and add to SET clause if present
	if req.Email.Present && req.Email.Valid {
		setClauses = append(setClauses, fmt.Sprintf("email = $%d", argIndex))
		args = append(args, req.Email.Value)
		argIndex++
	}

	if req.Name.Present && req.Name.Valid {
		setClauses = append(setClauses, fmt.Sprintf("name = $%d", argIndex))
		args = append(args, req.Name.Value)
		argIndex++
	}

	if req.Password.Present && req.Password.Valid {
		// Hash password before storing
		hashedPassword, err := HashPassword(req.Password.Value)
		if err != nil {
			return fmt.Errorf("failed to hash password: %w", err)
		}
		setClauses = append(setClauses, fmt.Sprintf("password = $%d", argIndex))
		args = append(args, hashedPassword)
		argIndex++
	}

	if len(setClauses) == 0 {
		return nil // Nothing to update
	}

	// Always update updated_at
	setClauses = append(setClauses, fmt.Sprintf("updated_at = $%d", argIndex))
	args = append(args, time.Now())
	argIndex++

	// Add WHERE id clause
	args = append(args, id)

	// Build final query
	query := fmt.Sprintf(
		"UPDATE users SET %s WHERE id = $%d",
		strings.Join(setClauses, ", "),
		argIndex,
	)

	err := s.db.Exec(ctx, query, args...)
	if err != nil {
		return fmt.Errorf("failed to update user: %w", err)
	}

	return nil
}

// Delete menghapus user dari database berdasarkan ID.
// Cascade delete akan menghapus semua related data (tokens, etc) jika ada foreign key.
//
// Parameters:
//   - ctx: context untuk membatalkan operasi
//   - id: user ID yang akan dihapus
//
// Returns:
//   - error: error jika DELETE query gagal
//
// Example:
//
//	err := userStore.Delete(ctx, userID)
func (s *PostgresUserStore) Delete(ctx context.Context, id int64) error {
	err := s.db.Exec(ctx, "DELETE FROM users WHERE id = $1", id)
	if err != nil {
		return fmt.Errorf("failed to delete user: %w", err)
	}

	return nil
}

// Exists mengecek apakah user dengan email tertentu sudah ada di database.
// Menggunakan EXISTS query untuk efisiensi.
//
// Parameters:
//   - ctx: context untuk membatalkan operasi
//   - email: alamat email yang akan dicek
//
// Returns:
//   - bool: true jika user dengan email tersebut ada
//   - error: error jika query gagal
//
// Example:
//
//	exists, err := userStore.Exists(ctx, "user@example.com")
//	if exists {
//	  return "email sudah terdaftar"
//	}
func (s *PostgresUserStore) Exists(ctx context.Context, email string) (bool, error) {
	var exists bool
	err := s.db.QueryRow(ctx,
		"SELECT EXISTS(SELECT 1 FROM users WHERE email = $1)",
		email,
	).Scan(&exists)

	if err != nil {
		return false, fmt.Errorf("failed to check user existence: %w", err)
	}

	return exists, nil
}

// MockUserStore is a mock implementation for testing
type MockUserStore struct {
	users  map[int64]*User
	nextID int64
}

// NewMockUserStore membuat mock user store untuk testing.
// Mock store menyimpan users dalam memory dan cocok untuk unit tests.
//
// Returns:
//   - *MockUserStore: mock store instance dengan empty users map
//
// Example:
//
//	mockStore := NewMockUserStore()
//	// use in tests
func NewMockUserStore() *MockUserStore {
	return &MockUserStore{
		users:  make(map[int64]*User),
		nextID: 1,
	}
}

// Create membuat user baru dalam mock store (memory).
//
// Parameters:
//   - ctx: context (tidak digunakan dalam mock)
//   - user: User struct yang akan disimpan
//
// Returns:
//   - error: selalu nil untuk mock
//
// Example:
//
//	err := mockStore.Create(ctx, &user)
func (s *MockUserStore) Create(ctx context.Context, user *User) error {
	user.ID = s.nextID
	user.CreatedAt = time.Now()
	user.UpdatedAt = time.Now()
	s.users[user.ID] = user
	s.nextID++
	return nil
}

// FindByID mencari user berdasarkan ID dalam mock store.
//
// Parameters:
//   - ctx: context (tidak digunakan dalam mock)
//   - id: user ID yang akan dicari
//
// Returns:
//   - *User: user jika ditemukan, nil jika tidak
//   - error: error message jika user tidak ditemukan
//
// Example:
//
//	user, err := mockStore.FindByID(ctx, userID)
func (s *MockUserStore) FindByID(ctx context.Context, id int64) (*User, error) {
	if user, exists := s.users[id]; exists {
		return user, nil
	}
	return nil, fmt.Errorf("user not found")
}

// FindByEmail mencari user berdasarkan email dalam mock store.
//
// Parameters:
//   - ctx: context (tidak digunakan dalam mock)
//   - email: alamat email user yang akan dicari
//
// Returns:
//   - *User: user jika ditemukan, nil jika tidak
//   - error: error message jika user tidak ditemukan
//
// Example:
//
//	user, err := mockStore.FindByEmail(ctx, "user@example.com")
func (s *MockUserStore) FindByEmail(ctx context.Context, email string) (*User, error) {
	for _, user := range s.users {
		if user.Email == email {
			return user, nil
		}
	}
	return nil, fmt.Errorf("user not found")
}

// Update mengupdate user dalam mock store.
//
// Parameters:
//   - ctx: context (tidak digunakan dalam mock)
//   - user: User struct dengan data baru (harus sudah ada ID)
//
// Returns:
//   - error: error message jika user tidak ditemukan
//
// Example:
//
//	err := mockStore.Update(ctx, &user)
func (s *MockUserStore) Update(ctx context.Context, user *User) error {
	if _, exists := s.users[user.ID]; !exists {
		return fmt.Errorf("user not found")
	}
	user.UpdatedAt = time.Now()
	s.users[user.ID] = user
	return nil
}

// Delete menghapus user dari mock store.
//
// Parameters:
//   - ctx: context (tidak digunakan dalam mock)
//   - id: user ID yang akan dihapus
//
// Returns:
//   - error: selalu nil untuk mock
//
// Example:
//
//	err := mockStore.Delete(ctx, userID)
func (s *MockUserStore) Delete(ctx context.Context, id int64) error {
	delete(s.users, id)
	return nil
}

// Exists mengecek apakah user dengan email tertentu ada dalam mock store.
//
// Parameters:
//   - ctx: context (tidak digunakan dalam mock)
//   - email: alamat email yang akan dicek
//
// Returns:
//   - bool: true jika user dengan email tersebut ada
//   - error: selalu nil untuk mock
//
// Example:
//
//	exists, err := mockStore.Exists(ctx, "user@example.com")
func (s *MockUserStore) Exists(ctx context.Context, email string) (bool, error) {
	for _, user := range s.users {
		if user.Email == email {
			return true, nil
		}
	}
	return false, nil
}

// UpdatePartial melakukan partial update user fields dalam mock store.
// Hanya update field yang Present dan Valid dalam JsonNull wrapper.
//
// Parameters:
//   - ctx: context (tidak digunakan dalam mock)
//   - id: user ID yang akan diupdate
//   - req: UpdateUserRequest dengan field-field yang akan diupdate
//
// Returns:
//   - error: error message jika user tidak ditemukan
//
// Example:
//
//	req := &UpdateUserRequest{Name: JsonNull[string]{Present: true, Valid: true, Value: "New Name"}}
//	err := mockStore.UpdatePartial(ctx, userID, req)
func (s *MockUserStore) UpdatePartial(ctx context.Context, id int64, req *UpdateUserRequest) error {
	user, exists := s.users[id]
	if !exists {
		return fmt.Errorf("user not found")
	}

	// Update email if present and valid
	if req.Email.Present && req.Email.Valid {
		user.Email = req.Email.Value
	}

	// Update name if present and valid
	if req.Name.Present && req.Name.Valid {
		user.Name = req.Name.Value
	}

	// Update password if present and valid
	if req.Password.Present && req.Password.Valid {
		hashedPassword, err := HashPassword(req.Password.Value)
		if err != nil {
			return fmt.Errorf("failed to hash password: %w", err)
		}
		user.Password = hashedPassword
	}

	user.UpdatedAt = time.Now()
	return nil
}
