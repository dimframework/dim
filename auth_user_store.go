package dim

import (
	"context"
)

// DatabaseAuthUserStore is a generic implementation of AuthUserStore for SQL databases.
// It assumes a standard 'users' table structure.
type DatabaseAuthUserStore struct {
	db Database
}

// NewDatabaseAuthUserStore creates a new DatabaseAuthUserStore.
func NewDatabaseAuthUserStore(db Database) *DatabaseAuthUserStore {
	return &DatabaseAuthUserStore{db: db}
}

// Deprecated: Use NewDatabaseAuthUserStore instead
func NewAuthUserStore(db Database) *DatabaseAuthUserStore {
	return NewDatabaseAuthUserStore(db)
}

func (s *DatabaseAuthUserStore) FindByEmail(ctx context.Context, email string) (Authenticatable, error) {
	user := &TokenUser{}
	query := s.db.Rebind(`SELECT id, email, password FROM users WHERE email = $1`)
	err := s.db.QueryRow(ctx, query, email).Scan(&user.ID, &user.Email, &user.Password)
	if err != nil {
		return nil, err
	}
	return user, nil
}

func (s *DatabaseAuthUserStore) FindByID(ctx context.Context, id string) (Authenticatable, error) {
	user := &TokenUser{}
	query := s.db.Rebind(`SELECT id, email, password FROM users WHERE id = $1`)
	err := s.db.QueryRow(ctx, query, id).Scan(&user.ID, &user.Email, &user.Password)
	if err != nil {
		return nil, err
	}
	return user, nil
}

func (s *DatabaseAuthUserStore) Update(ctx context.Context, user Authenticatable) error {
	query := s.db.Rebind(`UPDATE users SET email = $1, password = $2 WHERE id = $3`)
	return s.db.Exec(ctx, query, user.GetEmail(), user.GetPassword(), user.GetID())
}
