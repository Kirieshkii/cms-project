package auth

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

type ErrorResponse struct {
	Error ErrorDetail `json:"error"`
}

type ErrorDetail struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

// Error codes
const (
	ErrCodeInvalidCredentials = "invalid_credentials"
	ErrCodeUnauthorized       = "unauthorized"
	ErrCodeInvalidToken       = "invalid_token"
	ErrCodeForbidden          = "forbidden"
)

// RespondWithError отправляет ошибку в едином формате
func RespondWithError(c *gin.Context, statusCode int, code, message string) {
	c.JSON(statusCode, ErrorResponse{
		Error: ErrorDetail{
			Code:    code,
			Message: message,
		},
	})
}

// RespondUnauthorized отправляет 401 Unauthorized
func RespondUnauthorized(c *gin.Context, message string) {
	if message == "" {
		message = "Не авторизован"
	}
	RespondWithError(c, http.StatusUnauthorized, ErrCodeUnauthorized, message)
}

// RespondBadRequest отправляет 400 Bad Request
func RespondBadRequest(c *gin.Context, message string) {
	if message == "" {
		message = "Неверные учетные данные"
	}
	RespondWithError(c, http.StatusBadRequest, ErrCodeInvalidCredentials, message)
}
