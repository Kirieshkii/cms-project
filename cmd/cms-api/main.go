package main

import (
	"context"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/Kirieshkii/cms-project/internal/auth"
	"github.com/Kirieshkii/cms-project/internal/config"
	"github.com/Kirieshkii/cms-project/internal/db"
	"github.com/Kirieshkii/cms-project/internal/health"
	"github.com/Kirieshkii/cms-project/internal/logger"
	"github.com/Kirieshkii/cms-project/internal/middleware"
	"github.com/Kirieshkii/cms-project/internal/store/pgxstore"
	"github.com/gin-gonic/gin"
)

func main() {
	// Определяем окружение из переменной ENV (по умолчанию "local" для локальной разработки)
	env := os.Getenv("ENV")
	if env == "" {
		env = "local"
	}

	// Инициализация slog логгера в зависимости от окружения
	logger := logger.SetupLogger(env)

	// Контекст для инициализации ресурсов (БД, Redis)
	rootCtx := context.Background()

	// Инициализация пула pgxpool
	dsn, err := db.BuildDSNFromEnv()
	if err != nil {
		logger.Error("ошибка сборки DSN", "error", err)
		os.Exit(1)
	}

	pool, err := db.NewPool(rootCtx, dsn) // явная проверка ошибки
	if err != nil {
		logger.Error("ошибка инициализации пула БД", "error", err)
		os.Exit(1)
	}
	// Не используем defer - закрываем вручную при graceful shutdown
	logger.Info("пул подключений к БД успешно инициализирован",
		"max_conns", pool.Config().MaxConns,
		"min_conns", pool.Config().MinConns,
	)

	s := pgxstore.New(pool)

	// JWT конфиг
	cfgJWT, err := config.LoadJWT()
	if err != nil {
		logger.Error("ошибка загрузки переменных окружения для JWT", "error", err)
		os.Exit(1)
	}
	logger.Info("JWT конфигурация успешно загружена",
		"access_ttl", cfgJWT.AccessTTL,
		"refresh_ttl", cfgJWT.RefreshTTL,
	)

	authService := auth.NewAuthService(cfgJWT.Secret, cfgJWT.AccessTTL, cfgJWT.RefreshTTL, s.User())

	// Устанавливаем логгер для authService
	authService.SetLogger(logger)

	// Redis
	rdb, err := config.NewRedisClient(rootCtx)
	if err != nil {
		logger.Error("ошибка инициализации Redis", "error", err)
		pool.Close() // Закрываем пул перед выходом
		os.Exit(1)
	}
	// Не используем defer - закрываем вручную при graceful shutdown
	logger.Info("Redis клиент успешно инициализирован")
	authService.SetRedis(rdb)

	// Router
	// Используем gin.New() вместо gin.Default(), чтобы избежать дублирования логов
	// gin.Default() включает встроенный gin.Logger(), который дублирует наш middleware.Logger
	r := gin.New()
	r.Use(gin.Recovery())            // Recovery middleware для обработки паник
	r.Use(middleware.Logger(logger)) // Самописный структурированный slog logger

	// Healthcheck handlers + routes (без middleware логирования для уменьшения шума)
	healthHandler := health.NewHandler(pool, rdb, logger)
	healthGroup := r.Group("/health")

	healthGroup.GET("", healthHandler.Health)      // GET /health
	healthGroup.GET("/ready", healthHandler.Ready) // GET /health/ready
	healthGroup.GET("/live", healthHandler.Live)   // GET /health/live

	// Auth handlers + routes
	authHandler := auth.NewHandler(authService, logger)
	authGroup := r.Group("/api/v1/auth")

	authGroup.POST("/login", authHandler.Login)
	authGroup.POST("/refresh", authHandler.Refresh)
	authGroup.POST("/logout", authHandler.Logout)
	authGroup.GET("/profile", auth.AuthMiddleware(authService, logger), authHandler.Profile)

	// Определяем порт из переменной окружения (по умолчанию 8080)
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}
	addr := ":" + port

	// Определяем таймаут для graceful shutdown из переменной окружения (по умолчанию 10 секунд)
	shutdownTimeoutStr := os.Getenv("SHUTDOWN_TIMEOUT")
	if shutdownTimeoutStr == "" {
		shutdownTimeoutStr = "10s"
	}
	shutdownTimeout, err := time.ParseDuration(shutdownTimeoutStr)
	if err != nil {
		logger.Error("ошибка парсинга SHUTDOWN_TIMEOUT, используем значение по умолчанию 10s", "error", err)
		shutdownTimeout = 10 * time.Second
	}

	// Создаем HTTP сервер
	srv := &http.Server{
		Addr:    addr,
		Handler: r,
		// Настройки для production
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	// Обработка сигналов для graceful shutdown
	// Используем rootCtx как родительский контекст для signal.NotifyContext
	shutdownSignalCtx, stop := signal.NotifyContext(rootCtx, os.Interrupt, syscall.SIGTERM)
	defer stop()

	// Запускаем сервер в отдельной горутине
	go func() {
		logger.Info("HTTP сервер запускается", "addr", addr, "env", env)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Error("ошибка запуска HTTP сервера", "error", err)
			stop() // Останавливаем приложение при ошибке запуска
		}
	}()

	// Ожидаем сигнала завершения
	<-shutdownSignalCtx.Done()
	logger.Info("получен сигнал завершения, начинаем graceful shutdown",
		"timeout", shutdownTimeout,
	)

	// Создаем контекст с таймаутом для shutdown
	shutdownCtx, cancel := context.WithTimeout(context.Background(), shutdownTimeout)
	defer cancel()

	// Graceful shutdown HTTP сервера
	if err := srv.Shutdown(shutdownCtx); err != nil {
		logger.Error("ошибка graceful shutdown HTTP сервера", "error", err)
		// Принудительное закрытие - крайняя мера, если shutdown timeout истек
		// или сервер завис. Close() немедленно закрывает все соединения без ожидания.
		if closeErr := srv.Close(); closeErr != nil {
			logger.Error("ошибка принудительного закрытия HTTP сервера", "error", closeErr)
		}
	} else {
		logger.Info("HTTP сервер корректно остановлен")
	}

	// Закрываем соединения с БД и Redis
	logger.Info("закрываем соединения с БД и Redis")
	pool.Close()
	logger.Info("пул БД закрыт")

	if err := rdb.Close(); err != nil {
		logger.Error("ошибка закрытия Redis клиента", "error", err)
	} else {
		logger.Info("Redis клиент успешно закрыт")
	}

	logger.Info("приложение завершено")
}
