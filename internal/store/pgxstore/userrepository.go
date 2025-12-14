package pgxstore

import (
	"context"
	"errors"

	storage "github.com/Kirieshkii/cms-project/internal/store"
	"github.com/Kirieshkii/cms-project/internal/user/model"

	"github.com/jackc/pgconn"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type UserRepository struct {
	pool *pgxpool.Pool
}

func NewUserRepository(pool *pgxpool.Pool) storage.UserRepository {
	return &UserRepository{
		pool: pool,
	}
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
			// ErrUserAlreadyExists - бизнес-ошибка, не логируем (логируется на уровне Service/Handler)
			return storage.ErrUserAlreadyExists
		}
		// Возвращаем ошибку без логирования - логирование на уровне Service/Handler
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
		// Возвращаем ошибку без логирования - логирование на уровне Service/Handler
		return 0, err
	}

	return version, nil
}

// FindByEmail находит пользователя по email
func (r *UserRepository) FindByEmail(ctx context.Context, email string) (*model.User, error) {
	const q = `
		SELECT id, email, encrypted_password, role, token_version
		FROM users
		WHERE email = $1
	`

	u := &model.User{}
	err := r.pool.QueryRow(ctx, q, email).Scan(
		&u.ID,
		&u.Email,
		&u.EncryptedPassword,
		&u.Role,
		&u.TokenVersion,
	)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			// ErrUserNotFound - бизнес-ошибка, не логируем (логируется на уровне Handler)
			return nil, storage.ErrUserNotFound
		}
		// Возвращаем ошибку без логирования - логирование на уровне Handler
		return nil, err
	}

	return u, nil
}

// FindByID находит пользователя по ID
func (r *UserRepository) FindByID(ctx context.Context, id int64) (*model.User, error) {
	const q = `
		SELECT id, email, encrypted_password, role, token_version
		FROM users
		WHERE id = $1
	`

	u := &model.User{}
	err := r.pool.QueryRow(ctx, q, id).Scan(
		&u.ID,
		&u.Email,
		&u.EncryptedPassword,
		&u.Role,
		&u.TokenVersion,
	)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			// ErrUserNotFound - бизнес-ошибка, не логируем (логируется на уровне Handler)
			return nil, storage.ErrUserNotFound
		}
		// Возвращаем ошибку без логирования - логирование на уровне Handler
		return nil, err
	}

	return u, nil
}

// UpdateTokenVersion инкрементирует token_version пользователя
func (r *UserRepository) UpdateTokenVersion(ctx context.Context, id int64) error {
	const q = `
		UPDATE users
		SET token_version = token_version + 1
		WHERE id = $1
	`

	result, err := r.pool.Exec(ctx, q, id)
	if err != nil {
		// Возвращаем ошибку без логирования - логирование на уровне Service/Handler
		return err
	}

	if result.RowsAffected() == 0 {
		// ErrUserNotFound - бизнес-ошибка, не логируем (логируется на уровне Service/Handler)
		return storage.ErrUserNotFound
	}

	return nil
}
