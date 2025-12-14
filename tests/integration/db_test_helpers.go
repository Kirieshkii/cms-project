package tests

import (
	"context"
	"errors"
	"fmt"
	"os"
	"time"

	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/postgres" // нужен драйвер
	_ "github.com/golang-migrate/migrate/v4/source/file"       // источник file://
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/joho/godotenv"
	"github.com/stretchr/testify/suite"
)

type DBTestSuite struct {
	suite.Suite
	Pool *pgxpool.Pool
}

// SetupSuite загружает .env.test, создаёт пул и накатывает миграции
func (s *DBTestSuite) SetupSuite() {
	// Загружаем .env.test
	if err := godotenv.Load(".env.test"); err != nil {
		s.T().Fatalf("ошибка загрузки файла .env.test: %v", err)
	}

	// Получаем DSN
	dsn := os.Getenv("PG_TEST_DSN")
	if dsn == "" {
		host := os.Getenv("PG_TEST_HOST")
		port := os.Getenv("PG_TEST_PORT")
		user := os.Getenv("PG_TEST_USER")
		password := os.Getenv("PG_TEST_PASSWORD")
		db := os.Getenv("PG_TEST_DB")
		sslmode := os.Getenv("PG_TEST_SSLMODE")

		if host == "" || port == "" || user == "" || password == "" || db == "" || sslmode == "" {
			s.T().Fatal("PG_TEST_DSN или individual PG_TEST_* переменные должны быть заданы в файле .env.test")
		}

		dsn = fmt.Sprintf(
			"postgres://%s:%s@%s:%s/%s?sslmode=%s",
			user, password, host, port, db, sslmode,
		)
	}

	// Создаём конфиг пула
	cfg, err := pgxpool.ParseConfig(dsn)
	if err != nil {
		s.T().Fatalf("ошибка разбора конфигурации пула: %v", err)
	}

	// Настраиваем пул
	cfg.MaxConns = 10
	cfg.MinConns = 1
	cfg.MaxConnIdleTime = 5 * time.Minute
	cfg.HealthCheckPeriod = 1 * time.Minute

	// Создаём пул с таймаутом
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	pool, err := pgxpool.NewWithConfig(ctx, cfg)
	if err != nil {
		s.T().Fatalf("ошибка создания db pool: %v", err)
	}

	s.Pool = pool

	// Накатываем миграции
	if err := s.runMigrations(dsn); err != nil {
		s.T().Fatalf("ошибка применения миграций: %v", err)
	}
}

// TearDownSuite закрывает пул
func (s *DBTestSuite) TearDownSuite() {
	if s.Pool != nil {
		s.Pool.Close()
	}
}

// CleanupTables очищает указанные таблицы и логирует ошибки
func CleanupTables(ctx context.Context, pool *pgxpool.Pool, tables ...string) error {
	for _, table := range tables {
		if _, err := pool.Exec(ctx, fmt.Sprintf("DELETE FROM %s", table)); err != nil {
			return fmt.Errorf("warning: не удалось очистить таблицу %s: %w\n", table, err)
		}
	}

	return nil
}

// runMigrations накатывает миграции через golang-migrate
func (s *DBTestSuite) runMigrations(dsn string) error {
	m, err := migrate.New(
		"file://./migrations", // путь к миграциям
		dsn,
	)
	if err != nil {
		return err
	}
	defer m.Close()

	if err := m.Up(); err != nil && !errors.Is(err, migrate.ErrNoChange) {
		return err
	}
	return nil
}
