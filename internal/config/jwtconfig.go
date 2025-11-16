package config

import (
	"fmt"
	"os"
	"time"
)

type JWT struct {
	Secret     []byte
	AccessTTL  time.Duration
	RefreshTTL time.Duration
}

func LoadJWT() (*JWT, error) {
	secret, err := getEnv("JWT_SECRET")
	if err != nil {
		return nil, err
	}
	accessTTL, err := getEnvDuration("ACCESS_TOKEN_TTL")
	if err != nil {
		return nil, err
	}
	refreshTTL, err := getEnvDuration("REFRESH_TOKEN_TTL")
	if err != nil {
		return nil, err
	}

	return &JWT{
		Secret:     []byte(secret),
		AccessTTL:  accessTTL,
		RefreshTTL: refreshTTL,
	}, nil
}

func getEnv(key string) (string, error) {
	value := os.Getenv(key)
	if value == "" {
		return "", fmt.Errorf("переменная окружения %q обязательна", key)
	}

	return value, nil
}

func getEnvDuration(key string) (time.Duration, error) {
	val, err := getEnv(key)
	if err != nil {
		return 0, err
	}

	d, err := time.ParseDuration(val)
	if err != nil {
		return 0, fmt.Errorf("неверный %q=%q: %w", key, val, err)
	}

	return d, nil
}
