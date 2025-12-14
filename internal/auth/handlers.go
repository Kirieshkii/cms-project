package auth

import (
	"log/slog"
	"net/http"
	"strconv"

	"github.com/Kirieshkii/cms-project/internal/middleware"
	storage "github.com/Kirieshkii/cms-project/internal/store"
	"github.com/gin-gonic/gin"
)

type Handler struct {
	authService *AuthService
	userRepo    storage.UserRepository
	logger      *slog.Logger
}

func NewHandler(authService *AuthService, userRepo storage.UserRepository, logger *slog.Logger) *Handler {
	return &Handler{
		authService: authService,
		userRepo:    userRepo,
		logger:      logger,
	}
}

// getRequestID извлекает request ID из контекста Gin для логирования
func (h *Handler) getRequestID(c *gin.Context) string {
	if requestID := c.GetString(middleware.RequestIDKey); requestID != "" {
		return requestID
	}
	return "unknown"
}

// LoginRequest структура запроса для login
type LoginRequest struct {
	Email    string `json:"email" binding:"required,email"`
	Password string `json:"password" binding:"required"`
}

// RefreshRequest структура запроса для refresh
type RefreshRequest struct {
	RefreshToken string `json:"refresh_token" binding:"required"`
}

// LogoutRequest структура запроса для logout (refresh_token обязателен)
type LogoutRequest struct {
	RefreshToken string `json:"refresh_token" binding:"required"`
}

// ProfileResponse структура ответа для profile
type ProfileResponse struct {
	ID    int64  `json:"id"`
	Email string `json:"email"`
	Role  string `json:"role"`
}

// POST /api/v1/auth/login
func (h *Handler) Login(c *gin.Context) {
	var req LoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		RespondBadRequest(c, "Отсутствуют или неверные учетные данные")
		return
	}

	ctx := c.Request.Context()

	// Находим пользователя по email
	user, err := h.userRepo.FindByEmail(ctx, req.Email)
	if err != nil {
		if err == storage.ErrUserNotFound {
			RespondUnauthorized(c, "Неверный email или пароль")
			return
		}
		h.logger.Error("ошибка поиска пользователя по email",
			"request_id", h.getRequestID(c),
			"email", req.Email,
			"error", err,
		)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Внутренняя ошибка сервера"})
		return
	}

	// Проверяем пароль
	if !user.ComparePassword(req.Password) {
		RespondUnauthorized(c, "Неверный email или пароль")
		return
	}

	// Генерируем токены (НЕ инкрементируем token_version)
	userIDStr := strconv.FormatInt(user.ID, 10)
	tokens, err := h.authService.GenerateTokens(userIDStr, user.Role, user.TokenVersion)
	if err != nil {
		h.logger.Error("ошибка генерации токенов",
			"request_id", h.getRequestID(c),
			"user_id", user.ID,
			"error", err,
		)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Внутренняя ошибка сервера"})
		return
	}

	h.logger.Info("успешный вход пользователя",
		"request_id", h.getRequestID(c),
		"user_id", user.ID,
		"email", user.Email,
		"role", user.Role,
	)

	c.JSON(http.StatusOK, tokens)
}

// POST /api/v1/auth/refresh
func (h *Handler) Refresh(c *gin.Context) {
	var req RefreshRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		RespondWithError(c, http.StatusUnauthorized, ErrCodeInvalidToken, "Неверный или истекший refresh токен")
		return
	}

	ctx := c.Request.Context()

	// Валидируем refresh token
	claims, err := h.authService.ValidateRefreshToken(ctx, req.RefreshToken)
	if err != nil {
		RespondWithError(c, http.StatusUnauthorized, ErrCodeInvalidToken, "Неверный или истекший refresh токен")
		return
	}

	// Получаем userID из claims
	userID, err := strconv.ParseInt(claims.UserID, 10, 64)
	if err != nil {
		RespondWithError(c, http.StatusUnauthorized, ErrCodeInvalidToken, "Неверный или истекший refresh токен")
		return
	}

	// Получаем пользователя по ID для получения роли
	user, err := h.userRepo.FindByID(ctx, userID)
	if err != nil {
		if err == storage.ErrUserNotFound {
			RespondWithError(c, http.StatusUnauthorized, ErrCodeInvalidToken, "Неверный или истекший refresh токен")
			return
		}
		h.logger.Error("ошибка поиска пользователя по ID",
			"request_id", h.getRequestID(c),
			"user_id", userID,
			"error", err,
		)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Внутренняя ошибка сервера"})
		return
	}

	// Старый refresh токен → в blacklist (ротация токена)
	if err := h.authService.RevokeRefreshToken(ctx, claims); err != nil {
		// Если Redis недоступен, логируем, но продолжаем
		h.logger.Error("ошибка добавления refresh токена в blacklist",
			"request_id", h.getRequestID(c),
			"user_id", claims.UserID,
			"jti", claims.ID,
			"error", err,
		)
		// В production лучше обработать эту ошибку более явно
	}

	// Генерируем новые токены с ТОЙ ЖЕ версией (НЕ инкрементируем token_version)
	tokens, err := h.authService.GenerateTokens(claims.UserID, user.Role, claims.Ver)
	if err != nil {
		h.logger.Error("ошибка генерации новых токенов при refresh",
			"request_id", h.getRequestID(c),
			"user_id", claims.UserID,
			"error", err,
		)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Внутренняя ошибка сервера"})
		return
	}

	h.logger.Info("успешное обновление токенов",
		"request_id", h.getRequestID(c),
		"user_id", claims.UserID,
		"old_jti", claims.ID,
		"token_version", claims.Ver,
	)

	c.JSON(http.StatusOK, tokens)
}

// POST /api/v1/auth/logout
func (h *Handler) Logout(c *gin.Context) {
	ctx := c.Request.Context()

	var req LogoutRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		RespondBadRequest(c, "refresh_token обязателен")
		return
	}

	refreshClaims, err := h.authService.ValidateRefreshToken(ctx, req.RefreshToken)
	if err != nil {
		RespondWithError(c, http.StatusUnauthorized, ErrCodeInvalidToken, "Неверный или истекший refresh токен")
		return
	}

	if err := h.authService.RevokeRefreshToken(ctx, refreshClaims); err != nil {
		h.logger.Error("ошибка отзыва refresh токена при logout",
			"request_id", h.getRequestID(c),
			"user_id", refreshClaims.UserID,
			"jti", refreshClaims.ID,
			"error", err,
		)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Внутренняя ошибка сервера"})
		return
	}

	h.logger.Info("успешный выход пользователя",
		"request_id", h.getRequestID(c),
		"user_id", refreshClaims.UserID,
		"jti", refreshClaims.ID,
	)

	c.Status(http.StatusNoContent)
}

// GET /api/v1/auth/profile
func (h *Handler) Profile(c *gin.Context) {
	// Получаем claims из middleware
	claims, exists := c.Get("claims")
	if !exists {
		RespondUnauthorized(c, "Не авторизован")
		return
	}

	//возможно стоит добавить интерфейс, чтобы не улететь в панику при неверном типе во время приведения
	accessClaims, ok := claims.(*AccessClaims)
	if !ok {
		RespondUnauthorized(c, "Не авторизован")
		return
	}

	ctx := c.Request.Context()

	// Получаем userID из claims
	userID, err := strconv.ParseInt(accessClaims.UserID, 10, 64)
	if err != nil {
		RespondUnauthorized(c, "Не авторизован")
		return
	}

	// Получаем пользователя по ID (один запрос включает token_version)
	user, err := h.userRepo.FindByID(ctx, userID)
	if err != nil {
		if err == storage.ErrUserNotFound {
			RespondUnauthorized(c, "Не авторизован")
			return
		}
		h.logger.Error("ошибка поиска пользователя по ID в profile",
			"request_id", h.getRequestID(c),
			"user_id", userID,
			"error", err,
		)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Внутренняя ошибка сервера"})
		return
	}

	// Проверяем версию токена
	if accessClaims.Ver != user.TokenVersion {
		RespondUnauthorized(c, "Не авторизован")
		return
	}

	h.logger.Info("успешное получение профиля пользователя",
		"request_id", h.getRequestID(c),
		"user_id", user.ID,
		"email", user.Email,
		"role", user.Role,
	)

	c.JSON(http.StatusOK, ProfileResponse{
		ID:    user.ID,
		Email: user.Email,
		Role:  user.Role,
	})
}
