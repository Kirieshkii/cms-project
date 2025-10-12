package storage

import "github.com/Kirieshkii/cms-project/internal/user/model"

//go:generate mockery --name=UserRepository
type UserRepository interface {
	Create(*model.User) error
}
