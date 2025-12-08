package auth

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
)

// AuthMiddleware проверяет Authorization Bearer access токен,
// валидирует его и сохраняет claims в контекст Gin под ключом "claims".
func AuthMiddleware(authService *AuthService) gin.HandlerFunc {
	return func(c *gin.Context) {
		header := c.GetHeader("Authorization")
		if header == "" {
			RespondWithError(c, http.StatusUnauthorized, ErrCodeUnauthorized, "Требуется заголовок Authorization")
			c.Abort()
			return
		}

		const prefix = "Bearer "
		if !strings.HasPrefix(header, prefix) {
			RespondWithError(c, http.StatusUnauthorized, ErrCodeUnauthorized, "Неверный формат Authorization")
			c.Abort()
			return
		}

		token := strings.TrimSpace(strings.TrimPrefix(header, prefix))
		if token == "" {
			RespondWithError(c, http.StatusUnauthorized, ErrCodeUnauthorized, "Пустой access token")
			c.Abort()
			return
		}

		claims, err := authService.ValidateAccessToken(token)
		if err != nil {
			RespondWithError(c, http.StatusUnauthorized, ErrCodeInvalidToken, "Неверный или истекший access token")
			c.Abort()
			return
		}

		// Сохраняем claims для последующих хендлеров (logout/profile).
		c.Set("claims", claims)
		c.Next()
	}
}
