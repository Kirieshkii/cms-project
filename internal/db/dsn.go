package db

import (
	"fmt"
	"os"
)

// BuildDSNFromEnv собирает DSN из env-переменных.
// Предпочитает POSTGRES_DSN, если он явно задан.
func BuildDSNFromEnv() (string, error) {
	if dsn := os.Getenv("POSTGRES_DSN"); dsn != "" {
		return dsn, nil
	}

	host := os.Getenv("POSTGRES_HOST")
	port := os.Getenv("POSTGRES_PORT")
	user := os.Getenv("POSTGRES_USER")
	pass := os.Getenv("POSTGRES_PASSWORD")
	name := os.Getenv("POSTGRES_DB")

	if host == "" || port == "" || user == "" || name == "" {
		return "", fmt.Errorf("не найдены POSTGRES_* переменные окружения")
	}

	return fmt.Sprintf(
		"postgres://%s:%s@%s:%s/%s?sslmode=disable",
		user, pass, host, port, name,
	), nil
}
