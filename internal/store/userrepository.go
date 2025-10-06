package storage

import "github.com/Kirieshkii/cms-project/internal/user/model"

type UserRepository interface {
	Create(*model.User) error
}
