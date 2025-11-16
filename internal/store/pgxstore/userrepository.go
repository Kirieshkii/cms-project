package pgxstore

import (
	"context"
	"errors"

	storage "github.com/Kirieshkii/cms-project/internal/store"
	"github.com/Kirieshkii/cms-project/internal/user/model"

	"github.com/jackc/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

type UserRepository struct {
	pool *pgxpool.Pool
}

func NewUserRepository(pool *pgxpool.Pool) storage.UserRepository {
	return &UserRepository{pool: pool}
}

func (r *UserRepository) Create(ctx context.Context, u *model.User) error {
	const q = `
		INSERT INTO users (email, encrypted_password, role)
		VALUES ($1, $2, $3)
		RETURNING id
	`

	var id int64
	err := r.pool.QueryRow(ctx, q, u.Email, u.EncryptedPassword, u.Role).
		Scan(&id)

	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" {
			return storage.ErrUserAlreadyExists
		}
		return err
	}

	u.ID = id
	return nil
}

func (r *UserRepository) GetTokenVersion(ctx context.Context, id int64) (int, error) {
	const q = `SELECT token_version FROM users WHERE id = $1`

	var version int
	err := r.pool.QueryRow(ctx, q, id).Scan(&version)
	if err != nil {
		return 0, err
	}

	return version, nil
}
