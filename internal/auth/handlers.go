package auth

import (
	"log/slog"
	"net/http"

	"github.com/Kirieshkii/cms-project/internal/middleware"
	"github.com/gin-gonic/gin"
)

type Handler struct {
	authService *AuthService
	logger      *slog.Logger
}

func NewHandler(authService *AuthService, logger *slog.Logger) *Handler {
	return &Handler{
		authService: authService,
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

	// Вызываем сервис для выполнения бизнес-логики
	tokens, err := h.authService.Login(ctx, req.Email, req.Password)
	if err != nil {
		if err == ErrInvalidCredentials {
			RespondUnauthorized(c, "Неверный email или пароль")
			return
		}
		h.logger.Error("ошибка входа пользователя",
			"request_id", h.getRequestID(c),
			"email", req.Email,
			"error", err,
		)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Внутренняя ошибка сервера"})
		return
	}

	h.logger.Info("успешный вход пользователя",
		"request_id", h.getRequestID(c),
		"email", req.Email,
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

	// Вызываем сервис для выполнения бизнес-логики
	tokens, err := h.authService.Refresh(ctx, req.RefreshToken)
	if err != nil {
		if err == ErrInvalidToken {
			RespondWithError(c, http.StatusUnauthorized, ErrCodeInvalidToken, "Неверный или истекший refresh токен")
			return
		}
		h.logger.Error("ошибка обновления токенов",
			"request_id", h.getRequestID(c),
			"error", err,
		)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Внутренняя ошибка сервера"})
		return
	}

	h.logger.Info("успешное обновление токенов",
		"request_id", h.getRequestID(c),
	)

	c.JSON(http.StatusOK, tokens)
}

// POST /api/v1/auth/logout
func (h *Handler) Logout(c *gin.Context) {
	var req LogoutRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		RespondBadRequest(c, "refresh_token обязателен")
		return
	}

	ctx := c.Request.Context()

	// Вызываем сервис для выполнения бизнес-логики
	if err := h.authService.Logout(ctx, req.RefreshToken); err != nil {
		if err == ErrInvalidToken {
			RespondWithError(c, http.StatusUnauthorized, ErrCodeInvalidToken, "Неверный или истекший refresh токен")
			return
		}
		h.logger.Error("ошибка выхода пользователя",
			"request_id", h.getRequestID(c),
			"error", err,
		)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Внутренняя ошибка сервера"})
		return
	}

	h.logger.Info("успешный выход пользователя",
		"request_id", h.getRequestID(c),
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

	// Вызываем сервис для выполнения бизнес-логики
	user, err := h.authService.GetProfile(ctx, accessClaims)
	if err != nil {
		if err == ErrInvalidToken || err == ErrTokenVersionMismatch {
			RespondUnauthorized(c, "Не авторизован")
			return
		}
		h.logger.Error("ошибка получения профиля пользователя",
			"request_id", h.getRequestID(c),
			"user_id", accessClaims.UserID,
			"error", err,
		)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Внутренняя ошибка сервера"})
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
