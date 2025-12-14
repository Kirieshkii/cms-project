package storage

import (
	"context"
	"time"
)

//go:generate mockery --name=TokenBlacklistRepository
type TokenBlacklistRepository interface {
	// IsBlacklisted проверяет, находится ли токен с указанным jti в blacklist
	IsBlacklisted(ctx context.Context, jti string) (bool, error)

	// AddToBlacklist добавляет токен с указанным jti в blacklist на указанное время
	AddToBlacklist(ctx context.Context, jti string, ttl time.Duration) error
}
