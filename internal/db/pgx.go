package db

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

// NewPool создаёт и валидирует pgxpool.Pool.
func NewPool(ctx context.Context, dsn string) (*pgxpool.Pool, error) {
	cfg, err := pgxpool.ParseConfig(dsn)
	if err != nil {
		return nil, fmt.Errorf("pgx parse config: %w", err)
	}

	// Production-friendly defaults
	if cfg.MaxConns == 0 {
		cfg.MaxConns = 20
	}
	cfg.MinConns = 2
	cfg.HealthCheckPeriod = 30 * time.Second
	cfg.MaxConnIdleTime = 5 * time.Minute
	cfg.MaxConnLifetime = 30 * time.Minute

	// Ограничиваем время первой инициализации
	ctxInit, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	pool, err := pgxpool.NewWithConfig(ctxInit, cfg)
	if err != nil {
		return nil, fmt.Errorf("pgx new pool: %w", err)
	}

	// Проверяем соединение
	if err := pool.Ping(ctxInit); err != nil {
		pool.Close()
		return nil, fmt.Errorf("pgx ping: %w", err)
	}

	return pool, nil
}

// MustNewPool — удобный helper для main()
func MustNewPool(ctx context.Context, dsn string) *pgxpool.Pool {
	pool, err := NewPool(ctx, dsn)
	if err != nil {
		panic(err)
	}
	return pool
}
