package mocks

import storage "github.com/Kirieshkii/cms-project/internal/store"

type Store struct {
	UserRepo *UserRepository
}

func (s *Store) User() storage.UserRepository {
	return s.UserRepo
}
