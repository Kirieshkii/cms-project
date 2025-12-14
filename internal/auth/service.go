package auth

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"strconv"
	"time"

	storage "github.com/Kirieshkii/cms-project/internal/store"
	"github.com/Kirieshkii/cms-project/internal/user/model"
	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
)

type AuthService struct {
	jwtSecret     []byte
	accessTTL     time.Duration
	refreshTTL    time.Duration
	blacklistRepo storage.TokenBlacklistRepository // для blacklist refresh-токенов
	userRepo      storage.UserRepository           // для проверки tokenVersion у пользователя
	logger        *slog.Logger                     // для логирования
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

func (s *AuthService) SetBlacklistRepo(repo storage.TokenBlacklistRepository) {
	s.blacklistRepo = repo
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
	if s.blacklistRepo != nil {
		isBlacklisted, err := s.blacklistRepo.IsBlacklisted(ctx, claims.ID)
		if err != nil {
			if s.logger != nil {
				s.logger.Error("ошибка проверки blacklist", "error", err, "jti", claims.ID)
			}
			return nil, err
		}
		if isBlacklisted {
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
			s.logger.Debug("blacklist репозиторий не сконфигурирован, пропускаем проверку blacklist", "jti", claims.ID)
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
	if s.blacklistRepo == nil {
		if s.logger != nil {
			s.logger.Error("попытка отозвать токен, но blacklist репозиторий не сконфигурирован", "jti", claims.ID, "user_id", claims.UserID)
		}
		return errors.New("blacklist репозиторий не сконфигурирован")
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

	err := s.blacklistRepo.AddToBlacklist(ctx, claims.ID, ttl)
	if err != nil {
		if s.logger != nil {
			s.logger.Error("ошибка добавления токена в blacklist", "error", err, "jti", claims.ID, "user_id", claims.UserID, "ttl", ttl)
		}
		return err
	}

	if s.logger != nil {
		s.logger.Debug("токен успешно добавлен в blacklist", "jti", claims.ID, "user_id", claims.UserID, "ttl", ttl)
	}

	return nil
}

// Login выполняет аутентификацию пользователя по email и паролю
func (s *AuthService) Login(ctx context.Context, email, password string) (*TokenPair, error) {
	// Находим пользователя по email
	user, err := s.userRepo.FindByEmail(ctx, email)
	if err != nil {
		if err == storage.ErrUserNotFound {
			return nil, ErrInvalidCredentials
		}
		if s.logger != nil {
			s.logger.Error("ошибка поиска пользователя по email", "email", email, "error", err)
		}
		return nil, fmt.Errorf("ошибка поиска пользователя: %w", err)
	}

	// Проверяем пароль
	if !user.ComparePassword(password) {
		return nil, ErrInvalidCredentials
	}

	// Генерируем токены (НЕ инкрементируем token_version)
	userIDStr := strconv.FormatInt(user.ID, 10)
	tokens, err := s.GenerateTokens(userIDStr, user.Role, user.TokenVersion)
	if err != nil {
		if s.logger != nil {
			s.logger.Error("ошибка генерации токенов", "user_id", user.ID, "error", err)
		}
		return nil, fmt.Errorf("ошибка генерации токенов: %w", err)
	}

	if s.logger != nil {
		s.logger.Info("успешный вход пользователя",
			"user_id", user.ID,
			"email", user.Email,
			"role", user.Role,
		)
	}

	return tokens, nil
}

// Refresh обновляет пару токенов по refresh token
func (s *AuthService) Refresh(ctx context.Context, refreshToken string) (*TokenPair, error) {
	// Валидируем refresh token
	claims, err := s.ValidateRefreshToken(ctx, refreshToken)
	if err != nil {
		return nil, ErrInvalidToken
	}

	// Получаем userID из claims
	userID, err := strconv.ParseInt(claims.UserID, 10, 64)
	if err != nil {
		if s.logger != nil {
			s.logger.Error("неверный userID в refresh token", "user_id", claims.UserID, "error", err)
		}
		return nil, ErrInvalidToken
	}

	// Получаем пользователя по ID для получения роли
	user, err := s.userRepo.FindByID(ctx, userID)
	if err != nil {
		if err == storage.ErrUserNotFound {
			return nil, ErrInvalidToken
		}
		if s.logger != nil {
			s.logger.Error("ошибка поиска пользователя по ID", "user_id", userID, "error", err)
		}
		return nil, fmt.Errorf("ошибка поиска пользователя: %w", err)
	}

	// Старый refresh токен → в blacklist (ротация токена)
	if err := s.RevokeRefreshToken(ctx, claims); err != nil {
		// Если Redis недоступен, логируем, но продолжаем
		if s.logger != nil {
			s.logger.Error("ошибка добавления refresh токена в blacklist",
				"user_id", claims.UserID,
				"jti", claims.ID,
				"error", err,
			)
		}
		// В production лучше обработать эту ошибку более явно
	}

	// Генерируем новые токены с ТОЙ ЖЕ версией (НЕ инкрементируем token_version)
	tokens, err := s.GenerateTokens(claims.UserID, user.Role, claims.Ver)
	if err != nil {
		if s.logger != nil {
			s.logger.Error("ошибка генерации новых токенов при refresh",
				"user_id", claims.UserID,
				"error", err,
			)
		}
		return nil, fmt.Errorf("ошибка генерации токенов: %w", err)
	}

	if s.logger != nil {
		s.logger.Info("успешное обновление токенов",
			"user_id", claims.UserID,
			"old_jti", claims.ID,
			"token_version", claims.Ver,
		)
	}

	return tokens, nil
}

// Logout отзывает refresh token
func (s *AuthService) Logout(ctx context.Context, refreshToken string) error {
	// Валидируем refresh token
	refreshClaims, err := s.ValidateRefreshToken(ctx, refreshToken)
	if err != nil {
		return ErrInvalidToken
	}

	// Отзываем refresh token
	if err := s.RevokeRefreshToken(ctx, refreshClaims); err != nil {
		if s.logger != nil {
			s.logger.Error("ошибка отзыва refresh токена при logout",
				"user_id", refreshClaims.UserID,
				"jti", refreshClaims.ID,
				"error", err,
			)
		}
		return fmt.Errorf("ошибка отзыва токена: %w", err)
	}

	if s.logger != nil {
		s.logger.Info("успешный выход пользователя",
			"user_id", refreshClaims.UserID,
			"jti", refreshClaims.ID,
		)
	}

	return nil
}

// GetProfile возвращает профиль пользователя по access claims
func (s *AuthService) GetProfile(ctx context.Context, accessClaims *AccessClaims) (*model.User, error) {
	// Получаем userID из claims
	userID, err := strconv.ParseInt(accessClaims.UserID, 10, 64)
	if err != nil {
		if s.logger != nil {
			s.logger.Error("неверный userID в access token", "user_id", accessClaims.UserID, "error", err)
		}
		return nil, ErrInvalidToken
	}

	// Получаем пользователя по ID (один запрос включает token_version)
	user, err := s.userRepo.FindByID(ctx, userID)
	if err != nil {
		if err == storage.ErrUserNotFound {
			// Пользователь не найден, но токен валиден - это проблема безопасности
			if s.logger != nil {
				s.logger.Warn("попытка доступа с валидным токеном для несуществующего пользователя",
					"user_id", userID,
					"access_token_jti", accessClaims.ID,
					"token_version", accessClaims.Ver,
					"message", "пользователь был удален или токен выдан для несуществующего пользователя",
				)
			}
			return nil, ErrInvalidToken
		}
		if s.logger != nil {
			s.logger.Error("ошибка поиска пользователя по ID в profile",
				"user_id", userID,
				"error", err,
			)
		}
		return nil, fmt.Errorf("ошибка поиска пользователя: %w", err)
	}

	// Проверяем версию токена
	if accessClaims.Ver != user.TokenVersion {
		return nil, ErrTokenVersionMismatch
	}

	if s.logger != nil {
		s.logger.Info("успешное получение профиля пользователя",
			"user_id", user.ID,
			"email", user.Email,
			"role", user.Role,
		)
	}

	return user, nil
}
