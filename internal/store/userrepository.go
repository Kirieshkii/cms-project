package storage

import (
	"context"

	"github.com/Kirieshkii/cms-project/internal/user/model"
)

//go:generate mockery --name=UserRepository
type UserRepository interface {
	Create(context.Context, *model.User) error
	GetTokenVersion(context.Context, int64) (int, error)
}
