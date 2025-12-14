package auth

import (
	"log/slog"
	"net/http"
	"strings"

	"github.com/Kirieshkii/cms-project/internal/middleware"
	"github.com/gin-gonic/gin"
)

// AuthMiddleware проверяет Authorization Bearer access токен,
// валидирует его и сохраняет claims в контекст Gin под ключом "claims".
func AuthMiddleware(authService *AuthService, logger *slog.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		requestID := c.GetString(middleware.RequestIDKey)
		if requestID == "" {
			requestID = "unknown"
		}

		header := c.GetHeader("Authorization")
		if header == "" {
			if logger != nil {
				logger.Debug("отсутствует заголовок Authorization",
					"request_id", requestID,
					"path", c.Request.URL.Path,
				)
			}
			RespondWithError(c, http.StatusUnauthorized, ErrCodeUnauthorized, "Требуется заголовок Authorization")
			c.Abort()
			return
		}

		const prefix = "Bearer "
		if !strings.HasPrefix(header, prefix) {
			if logger != nil {
				logger.Debug("неверный формат заголовка Authorization",
					"request_id", requestID,
					"path", c.Request.URL.Path,
					"header_prefix", header[:min(len(header), 20)],
				)
			}
			RespondWithError(c, http.StatusUnauthorized, ErrCodeUnauthorized, "Неверный формат Authorization")
			c.Abort()
			return
		}

		token := strings.TrimSpace(strings.TrimPrefix(header, prefix))
		if token == "" {
			if logger != nil {
				logger.Debug("пустой access token",
					"request_id", requestID,
					"path", c.Request.URL.Path,
				)
			}
			RespondWithError(c, http.StatusUnauthorized, ErrCodeUnauthorized, "Пустой access token")
			c.Abort()
			return
		}

		claims, err := authService.ValidateAccessToken(token)
		if err != nil {
			if logger != nil {
				logger.Debug("ошибка валидации access token",
					"request_id", requestID,
					"path", c.Request.URL.Path,
					"error", err,
				)
			}
			RespondWithError(c, http.StatusUnauthorized, ErrCodeInvalidToken, "Неверный или истекший access token")
			c.Abort()
			return
		}

		// Сохраняем claims для последующих хендлеров (logout/profile).
		c.Set("claims", claims)
		c.Next()
	}
}

// min возвращает минимальное из двух чисел (для совместимости с Go < 1.21)
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
