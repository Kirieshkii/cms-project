package middleware

import (
	"log/slog"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// RequestIDKey ключ для хранения request ID в контексте Gin
const RequestIDKey = "request_id"

// Logger возвращает middleware для логирования HTTP запросов с использованием slog.
// Логирует: метод, путь, IP адрес, request ID, статус-код, количество байт, время обработки.
func Logger(log *slog.Logger) gin.HandlerFunc {
	log = log.With(
		slog.String("component", "middleware/logger"),
	)

	log.Info("logger middleware активирован")

	return func(c *gin.Context) {
		// Генерируем request ID если его еще нет
		requestID := c.GetString(RequestIDKey)
		if requestID == "" {
			requestID = uuid.New().String()
			c.Set(RequestIDKey, requestID)
		}

		// Создаем логгер с контекстом запроса
		entry := log.With(
			slog.String("method", c.Request.Method),
			slog.String("path", c.Request.URL.Path),
			slog.String("remote_addr", c.ClientIP()),
			slog.String("user_agent", c.Request.UserAgent()),
			slog.String("request_id", requestID),
		)

		// Засекаем время начала обработки
		start := time.Now()

		// Обрабатываем запрос
		c.Next()

		// Логируем после обработки
		entry.Info("request completed",
			slog.Int("status", c.Writer.Status()),
			slog.Int("bytes", c.Writer.Size()),
			slog.String("duration", time.Since(start).String()),
		)
	}
}
