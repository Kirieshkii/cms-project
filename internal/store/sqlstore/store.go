package sqlstore

import (
	"database/sql"

	storage "github.com/Kirieshkii/cms-project/internal/store"
)

type Store struct {
	db             *sql.DB
	userRepository storage.UserRepository
}

func New(db *sql.DB) *Store {
	return &Store{
		db: db,
	}
}

// метод DB инкапсулирует запрос к локальному полю db *sql.DB
func (s *Store) DB() *sql.DB {
	return s.db
}

// User возвращает БД, инициализируя при первом обращении
func (s *Store) User() storage.UserRepository {
	if s.userRepository == nil {
		s.userRepository = NewUserRepository(s)
	}
	return s.userRepository
}
