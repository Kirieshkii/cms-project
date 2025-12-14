package health

import (
	"context"
	"log/slog"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/redis/go-redis/v9"
)

// Handler обрабатывает healthcheck endpoints
type Handler struct {
	pool   *pgxpool.Pool
	rdb    *redis.Client
	logger *slog.Logger
}

// NewHandler создает новый healthcheck handler
func NewHandler(pool *pgxpool.Pool, rdb *redis.Client, logger *slog.Logger) *Handler {
	return &Handler{
		pool:   pool,
		rdb:    rdb,
		logger: logger,
	}
}

// HealthResponse структура ответа для healthcheck
type HealthResponse struct {
	Status  string            `json:"status"`
	Service string            `json:"service"`
	Checks  map[string]string `json:"checks,omitempty"`
}

// Health базовая проверка здоровья сервиса (всегда возвращает 200)
// Используется для проверки, что HTTP сервер работает
// GET /health
func (h *Handler) Health(c *gin.Context) {
	c.JSON(http.StatusOK, HealthResponse{
		Status:  "ok",
		Service: "cms-api",
	})
}

// Ready проверка готовности сервиса (readiness probe)
// Проверяет доступность БД и Redis
// GET /health/ready
func (h *Handler) Ready(c *gin.Context) {
	ctx, cancel := context.WithTimeout(c.Request.Context(), 2*time.Second)
	defer cancel()

	checks := make(map[string]string)
	allHealthy := true

	// Проверка PostgreSQL
	if h.pool != nil {
		if err := h.pool.Ping(ctx); err != nil {
			checks["postgres"] = "unhealthy: " + err.Error()
			allHealthy = false
			h.logger.Warn("healthcheck: PostgreSQL недоступен",
				"error", err,
			)
		} else {
			checks["postgres"] = "healthy"
		}
	} else {
		checks["postgres"] = "unhealthy: pool is nil"
		allHealthy = false
	}

	// Проверка Redis
	if h.rdb != nil {
		if err := h.rdb.Ping(ctx).Err(); err != nil {
			checks["redis"] = "unhealthy: " + err.Error()
			allHealthy = false
			h.logger.Warn("healthcheck: Redis недоступен",
				"error", err,
			)
		} else {
			checks["redis"] = "healthy"
		}
	} else {
		checks["redis"] = "unhealthy: client is nil"
		allHealthy = false
	}

	if allHealthy {
		c.JSON(http.StatusOK, HealthResponse{
			Status:  "ready",
			Service: "cms-api",
			Checks:  checks,
		})
	} else {
		c.JSON(http.StatusServiceUnavailable, HealthResponse{
			Status:  "not ready",
			Service: "cms-api",
			Checks:  checks,
		})
	}
}

// Live проверка живучести сервиса (liveness probe)
// Простая проверка, что сервер работает
// GET /health/live
func (h *Handler) Live(c *gin.Context) {
	c.JSON(http.StatusOK, HealthResponse{
		Status:  "alive",
		Service: "cms-api",
	})
}
