package main

import (
	"context"
	"log"

	"github.com/Kirieshkii/cms-project/internal/auth"
	"github.com/Kirieshkii/cms-project/internal/config"
	"github.com/Kirieshkii/cms-project/internal/db"
	"github.com/Kirieshkii/cms-project/internal/store/pgxstore"
	"github.com/gin-gonic/gin"
)

func main() {
	ctx := context.Background()

	// Инициализация пула pgxpool
	dsn, err := db.BuildDSNFromEnv()
	if err != nil {
		log.Fatalf("ошибка сборки DSN: %v", err)
	}

	pool, err := db.NewPool(ctx, dsn) // явная проверка ошибки
	if err != nil {
		log.Fatalf("ошибка инициализации пула БД: %v", err)
	}
	defer pool.Close()

	s := pgxstore.New(pool)

	// JWT конфиг
	cfgJWT, err := config.LoadJWT()
	if err != nil {
		log.Fatalf("ошибка загрузки переменных окружения для JWT: %v", err)
	}

	authService := auth.NewAuthService(cfgJWT.Secret, cfgJWT.AccessTTL, cfgJWT.RefreshTTL, s.User())

	// Redis
	rdb, err := config.NewRedisClient(ctx)
	if err != nil {
		log.Fatalf("ошибка инициализации Redis: %v", err)
	}
	defer rdb.Close()
	authService.SetRedis(rdb)

	// Router
	r := gin.Default()

	// Auth handlers + routes
	authHandler := auth.NewHandler(authService, s.User())
	authGroup := r.Group("/api/v1/auth")
	{
		authGroup.POST("/login", authHandler.Login)
		authGroup.POST("/refresh", authHandler.Refresh)

		authGroup.POST("/logout", authHandler.Logout)
		authGroup.GET("/profile", auth.AuthMiddleware(authService), authHandler.Profile)
	}

	if err := r.Run(); err != nil {
		log.Fatalf("ошибка запуска HTTP сервера: %v", err)
	}
}
