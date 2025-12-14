package config

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/redis/go-redis/v9"
)

// NewRedisClient создаёт клиент Redis, валидируя env и выполняя ping.
// Возвращает ошибку, если обязательные переменные не заданы или Redis недоступен.
func NewRedisClient(ctx context.Context) (*redis.Client, error) {
	host := os.Getenv("REDIS_HOST")
	port := os.Getenv("REDIS_PORT")
	password := os.Getenv("REDIS_PASSWORD") // может быть пустым

	if host == "" || port == "" {
		return nil, fmt.Errorf("переменные окружения REDIS_HOST/REDIS_PORT обязательны")
	}

	addr := fmt.Sprintf("%s:%s", host, port)

	rdb := redis.NewClient(&redis.Options{
		Addr:     addr,
		Password: password,
		DB:       0, // default DB
	})

	// Проверяем соединение с ограниченным таймаутом
	ctxPing, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()

	if err := rdb.Ping(ctxPing).Err(); err != nil {
		_ = rdb.Close()
		return nil, fmt.Errorf("redis ping %s: %w", addr, err)
	}

	return rdb, nil
}
