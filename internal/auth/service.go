package auth

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
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
	logger      *slog.Logger           // для логирования
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

func (s *AuthService) SetLogger(logger *slog.Logger) {
	s.logger = logger
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
		if s.logger != nil {
			s.logger.Debug("ошибка парсинга refresh token", "error", err)
		}
		return nil, err
	}

	claims, ok := token.Claims.(*RefreshClaims)
	if !ok || !token.Valid {
		if s.logger != nil {
			s.logger.Debug("неверный формат refresh token", "valid", token.Valid, "ok", ok)
		}
		return nil, errors.New("неверный формат refresh token")
	}

	// Проверка blacklist (granular logout)
	if s.redisClient != nil {
		key := fmt.Sprintf("blacklist:jti:%s", claims.ID)
		isBlacklisted, err := s.redisClient.Exists(ctx, key).Result()
		if err != nil {
			if s.logger != nil {
				s.logger.Error("ошибка проверки blacklist в Redis", "error", err, "jti", claims.ID, "key", key)
			}
			return nil, err
		}
		if isBlacklisted > 0 {
			if s.logger != nil {
				s.logger.Debug("refresh token находится в blacklist", "jti", claims.ID, "user_id", claims.UserID)
			}
			return nil, errors.New("refresh token находится в blacklist")
		}
		if s.logger != nil {
			s.logger.Debug("refresh token не в blacklist", "jti", claims.ID, "user_id", claims.UserID)
		}
	} else {
		if s.logger != nil {
			s.logger.Debug("Redis не сконфигурирован, пропускаем проверку blacklist", "jti", claims.ID)
		}
	}

	userIDInt, err := strconv.ParseInt(claims.UserID, 10, 64)
	if err != nil {
		if s.logger != nil {
			s.logger.Error("неверный userID в refresh token", "error", err, "user_id", claims.UserID)
		}
		return nil, fmt.Errorf("неверный userID в refresh token: %v", err)
	}

	// Проверка версии (если ты хранишь версию токена у пользователя)
	userVersion, err := s.userRepo.GetTokenVersion(ctx, userIDInt)
	if err != nil {
		if s.logger != nil {
			s.logger.Error("ошибка получения версии из БД", "error", err, "user_id", userIDInt)
		}
		return nil, fmt.Errorf("ошибка получения версии из БД: %w", err)
	}
	if claims.Ver != userVersion {
		if s.logger != nil {
			s.logger.Debug("версии токенов не совпадают", "token_version", claims.Ver, "user_version", userVersion, "user_id", userIDInt)
		}
		return nil, errors.New("версии токенов не совпадают")
	}

	if s.logger != nil {
		s.logger.Debug("refresh token успешно валидирован", "jti", claims.ID, "user_id", claims.UserID, "version", claims.Ver)
	}

	return claims, nil
}

func (s *AuthService) RevokeRefreshToken(ctx context.Context, claims *RefreshClaims) error {
	if s.redisClient == nil {
		if s.logger != nil {
			s.logger.Error("попытка отозвать токен, но Redis не сконфигурирован", "jti", claims.ID, "user_id", claims.UserID)
		}
		return errors.New("redis не сконфигурирован")
	}

	// TTL = время до истечения токена
	ttl := time.Until(claims.ExpiresAt.Time)
	if ttl <= 0 {
		// Токен уже истёк
		if s.logger != nil {
			s.logger.Debug("токен уже истёк, не добавляем в blacklist", "jti", claims.ID, "expires_at", claims.ExpiresAt.Time)
		}
		return nil
	}

	key := fmt.Sprintf("blacklist:jti:%s", claims.ID)

	err := s.redisClient.Set(ctx, key, true, ttl).Err()
	if err != nil {
		if s.logger != nil {
			s.logger.Error("ошибка добавления токена в blacklist Redis", "error", err, "jti", claims.ID, "user_id", claims.UserID, "key", key, "ttl", ttl)
		}
		return err
	}

	if s.logger != nil {
		s.logger.Debug("токен успешно добавлен в blacklist", "jti", claims.ID, "user_id", claims.UserID, "key", key, "ttl", ttl)
	}

	return nil
}
