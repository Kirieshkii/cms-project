package storage

import (
	"context"

	"github.com/Kirieshkii/cms-project/internal/user/model"
)

//go:generate mockery --name=UserRepository
type UserRepository interface {
	Create(context.Context, *model.User) error
	GetTokenVersion(context.Context, int64) (int, error)
	FindByEmail(context.Context, string) (*model.User, error)
	FindByID(context.Context, int64) (*model.User, error)
	UpdateTokenVersion(context.Context, int64) error
}
