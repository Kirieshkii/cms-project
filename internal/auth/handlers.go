package auth

import (
	"net/http"
	"strconv"

	storage "github.com/Kirieshkii/cms-project/internal/store"
	"github.com/gin-gonic/gin"
)

type Handler struct {
	authService *AuthService
	userRepo    storage.UserRepository
}

func NewHandler(authService *AuthService, userRepo storage.UserRepository) *Handler {
	return &Handler{
		authService: authService,
		userRepo:    userRepo,
	}
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

// LogoutRequest структура запроса для logout (refresh_token опционален)
type LogoutRequest struct {
	RefreshToken string `json:"refresh_token"`
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
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Внутренняя ошибка сервера"})
		return
	}

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
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Внутренняя ошибка сервера"})
		return
	}

	// Старый refresh токен → в blacklist (ротация токена)
	if err := h.authService.RevokeRefreshToken(ctx, claims); err != nil {
		// Если Redis недоступен, логируем, но продолжаем
		// В production лучше обработать эту ошибку более явно
	}

	// Генерируем новые токены с ТОЙ ЖЕ версией (НЕ инкрементируем token_version)
	tokens, err := h.authService.GenerateTokens(claims.UserID, user.Role, claims.Ver)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Внутренняя ошибка сервера"})
		return
	}

	c.JSON(http.StatusOK, tokens)
}

// POST /api/v1/auth/logout
func (h *Handler) Logout(c *gin.Context) {
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

	// Пытаемся получить refresh_token из body (опционально)
	var req LogoutRequest
	c.ShouldBindJSON(&req) // Игнорируем ошибку, т.к. body опционален

	// Если refresh_token передан - добавляем в blacklist (granular logout)
	if req.RefreshToken != "" {
		refreshClaims, err := h.authService.ValidateRefreshToken(ctx, req.RefreshToken)
		if err == nil {
			// Валидируем, что refresh token принадлежит тому же пользователю
			if refreshClaims.UserID == accessClaims.UserID {
				// Добавляем refresh token в blacklist
				if err := h.authService.RevokeRefreshToken(ctx, refreshClaims); err != nil {
					// Если Redis недоступен, логируем, но продолжаем
				}
			}
		}
		// Если refresh token невалиден, просто игнорируем его
	}

	// НЕ инкрементируем token_version (granular logout)
	// Если нужен глобальный logout (сброс пароля), это делается отдельным методом

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
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Внутренняя ошибка сервера"})
		return
	}

	// Проверяем версию токена
	if accessClaims.Ver != user.TokenVersion {
		RespondUnauthorized(c, "Не авторизован")
		return
	}

	c.JSON(http.StatusOK, ProfileResponse{
		ID:    user.ID,
		Email: user.Email,
		Role:  user.Role,
	})
}
