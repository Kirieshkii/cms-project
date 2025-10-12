package sqlstore

import (
	"database/sql"

	storage "github.com/Kirieshkii/cms-project/internal/store"
	"github.com/Kirieshkii/cms-project/internal/user/model"
	"github.com/lib/pq"
)

type UserRepository struct {
	store *Store
}

func NewUserRepository(s *Store) *UserRepository {
	return &UserRepository{store: s}
}

// метод db является сокращением, позволяя вместо r.store.db писать r.db()
func (r *UserRepository) db() *sql.DB {
	return r.store.DB()
}

func (r *UserRepository) Create(u *model.User) error {
	_, err := r.db().Exec(
		`INSERT INTO users (email, encrypted_password) VALUES ($1, $2) RETURNING id`,
		u.Email, u.EncryptedPassword,
	)
	if err != nil {
		if pqErr, ok := err.(*pq.Error); ok {
			if pqErr.Code == "23505" {
				return storage.ErrUserAlreadyExists
			}
		}
		return err
	}

	return nil
}
