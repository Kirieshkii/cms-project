package sqlstore

import (
	"database/sql"

	"github.com/Kirieshkii/cms-project/internal/user/model"
)

type UserRepository struct {
	store *Store
}

func NewUserRepository(s *Store) *UserRepository {
	return &UserRepository{store: s}
}

func (r *UserRepository) db() *sql.DB {
	return r.store.DB()
}

func (r *UserRepository) Create(u *model.User) error {
	_, err := r.db().Exec(
		`INSERT INTO users (email, encrypted_password) VALUES ($1, $2) RETURNING id`,
		u.Email, u.EncryptedPassword,
	)
	if err != nil {
		return err
	}

	return nil
}
