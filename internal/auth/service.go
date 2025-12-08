package auth

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"time"

	storage "github.com/Kirieshkii/cms-project/internal/store"
	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
)

type AuthService struct {
	jwtSecret   []byte
	accessTTL   time.Duration
	refreshTTL  time.Duration
	redisClient *redis.Client          // для blacklist refresh-токенов
	userRepo    storage.UserRepository // для проверки tokenVersion у пользователя
}

type TokenPair struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
	ExpiresIn    int64  `json:"expires_in"` // TTL access токена в секундах
}

func NewAuthService(secret []byte, accessTTL, refreshTTL time.Duration, userRepo storage.UserRepository) *AuthService {
	return &AuthService{
		jwtSecret:  secret,
		accessTTL:  accessTTL,
		refreshTTL: refreshTTL,
		userRepo:   userRepo,
	}
}

func (s *AuthService) SetRedis(rdb *redis.Client) {
	s.redisClient = rdb
}

func (s *AuthService) GenerateTokens(userID, role string, version int) (*TokenPair, error) {
	now := time.Now()

	// --- Access Token ---
	accessClaims := AccessClaims{
		UserID: userID,
		Role:   role,
		Ver:    version,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(now.Add(s.accessTTL)),
			IssuedAt:  jwt.NewNumericDate(now),
			ID:        uuid.New().String(),
		},
	}

	accessToken, err := jwt.NewWithClaims(jwt.SigningMethodHS256, accessClaims).SignedString(s.jwtSecret)
	if err != nil {
		return nil, err
	}

	// --- Refresh Token ---
	refreshClaims := RefreshClaims{
		UserID: userID,
		Ver:    version,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(now.Add(s.refreshTTL)),
			IssuedAt:  jwt.NewNumericDate(now),
			ID:        uuid.New().String(), // JTI
		},
	}

	refreshToken, err := jwt.NewWithClaims(jwt.SigningMethodHS256, refreshClaims).SignedString(s.jwtSecret)
	if err != nil {
		return nil, err
	}

	return &TokenPair{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
		ExpiresIn:    int64(s.accessTTL.Seconds()),
	}, nil
}

func (s *AuthService) ValidateAccessToken(tokenStr string) (*AccessClaims, error) {
	token, err := jwt.ParseWithClaims(tokenStr, &AccessClaims{}, func(t *jwt.Token) (interface{}, error) {
		return s.jwtSecret, nil
	})
	if err != nil {
		return nil, err
	}

	claims, ok := token.Claims.(*AccessClaims)
	if !ok || !token.Valid {
		return nil, errors.New("неверный формат access token")
	}

	return claims, nil
}

func (s *AuthService) ValidateRefreshToken(ctx context.Context, tokenStr string) (*RefreshClaims, error) {
	token, err := jwt.ParseWithClaims(tokenStr, &RefreshClaims{}, func(t *jwt.Token) (interface{}, error) {
		return s.jwtSecret, nil
	})
	if err != nil {
		return nil, err
	}

	claims, ok := token.Claims.(*RefreshClaims)
	if !ok || !token.Valid {
		return nil, errors.New("неверный формат refresh token")
	}

	// Проверка blacklist (granular logout)
	if s.redisClient != nil {
		isBlacklisted, err := s.redisClient.Exists(ctx,
			fmt.Sprintf("blacklist:jti:%s", claims.ID)).
			Result()
		if err != nil {
			return nil, err
		}
		if isBlacklisted > 0 {
			return nil, errors.New("refresh token находится в blacklist")
		}
	}

	userIDInt, err := strconv.ParseInt(claims.UserID, 10, 64)
	if err != nil {
		return nil, fmt.Errorf("неверный userID в refresh token: %v", err)
	}

	// Проверка версии (если ты хранишь версию токена у пользователя)
	userVersion, err := s.userRepo.GetTokenVersion(ctx, userIDInt)
	if err != nil {
		return nil, fmt.Errorf("ошибка получения версии из БД: %w", err)
	}
	if claims.Ver != userVersion {
		return nil, errors.New("версии токенов не совпадают")
	}

	return claims, nil
}

func (s *AuthService) RevokeRefreshToken(ctx context.Context, claims *RefreshClaims) error {
	if s.redisClient == nil {
		return errors.New("redis не сконфигурирован")
	}

	// TTL = время до истечения токена
	ttl := time.Until(claims.ExpiresAt.Time)
	if ttl <= 0 {
		// Токен уже истёк
		return nil
	}

	key := fmt.Sprintf("blacklist:jti:%s", claims.ID)

	return s.redisClient.Set(ctx, key, true, ttl).Err()
}
