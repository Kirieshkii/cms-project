package auth

import (
	"github.com/golang-jwt/jwt/v5"
)

// Короткоживущий Access Token
type AccessClaims struct {
	UserID string `json:"sub"` // Subject — ID пользователя
	Role   string `json:"role"`
	Ver    int    `json:"ver"` // Версия токена (для инвалидации по version)
	jwt.RegisteredClaims
}

// Долго живущий Refresh Token
type RefreshClaims struct {
	UserID string `json:"sub"`
	Ver    int    `json:"ver"`
	jwt.RegisteredClaims
}
