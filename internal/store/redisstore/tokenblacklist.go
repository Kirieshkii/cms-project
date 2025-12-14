package redisstore

import (
	"context"
	"fmt"
	"time"

	storage "github.com/Kirieshkii/cms-project/internal/store"
	"github.com/redis/go-redis/v9"
)

// TokenBlacklistRepository реализует storage.TokenBlacklistRepository используя Redis
type TokenBlacklistRepository struct {
	rdb *redis.Client
}

// NewTokenBlacklistRepository создает новый репозиторий для blacklist токенов
func NewTokenBlacklistRepository(rdb *redis.Client) storage.TokenBlacklistRepository {
	return &TokenBlacklistRepository{
		rdb: rdb,
	}
}

// IsBlacklisted проверяет, находится ли токен с указанным jti в blacklist
func (r *TokenBlacklistRepository) IsBlacklisted(ctx context.Context, jti string) (bool, error) {
	key := fmt.Sprintf("blacklist:jti:%s", jti)
	exists, err := r.rdb.Exists(ctx, key).Result()
	if err != nil {
		return false, fmt.Errorf("ошибка проверки blacklist в Redis: %w", err)
	}
	return exists > 0, nil
}

// AddToBlacklist добавляет токен с указанным jti в blacklist на указанное время
func (r *TokenBlacklistRepository) AddToBlacklist(ctx context.Context, jti string, ttl time.Duration) error {
	if ttl <= 0 {
		// Токен уже истёк, не добавляем в blacklist
		return nil
	}

	key := fmt.Sprintf("blacklist:jti:%s", jti)
	if err := r.rdb.Set(ctx, key, true, ttl).Err(); err != nil {
		return fmt.Errorf("ошибка добавления токена в blacklist Redis: %w", err)
	}

	return nil
}
