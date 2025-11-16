package pgxstore

import (
	storage "github.com/Kirieshkii/cms-project/internal/store"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Store struct {
	pool           *pgxpool.Pool
	userRepository storage.UserRepository
}

func New(pool *pgxpool.Pool) storage.Store {
	s := &Store{pool: pool}
	s.userRepository = NewUserRepository(pool)
	return s
}

func (s *Store) User() storage.UserRepository {
	return s.userRepository
}
