package main

import (
	"context"
	"os"
	"os/signal"
	"syscall"
	"time"

	"pr-reviewer-service/api"
	"pr-reviewer-service/internal/config"
	"pr-reviewer-service/internal/database"
	"pr-reviewer-service/internal/handler"
	"pr-reviewer-service/internal/repository"
	"pr-reviewer-service/internal/usecase"

	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	"github.com/sirupsen/logrus"
)

func main() {
	// Логгер
	logger := logrus.New()
	logger.SetLevel(logrus.InfoLevel)
	logger.SetFormatter(&logrus.JSONFormatter{})

	// Конфиг
	cfg, err := config.LoadConfig()
	if err != nil {
		logger.Warnf(".env not found: %v", err)
	}

	// База данных (database/sql)
	db, err := database.NewPostgresDB(cfg)
	if err != nil {
		logger.Fatalf("Database connection failed: %v", err)
	}
	defer db.Close()
	logger.Info("Database connected")

	// SQLC queries
	queries := database.New(db)

	// Репозитории
	teamRepo := repository.NewTeamRepository(db, queries)
	userRepo := repository.NewUserRepository(db, queries)
	prRepo := repository.NewPRRepository(db, queries)
	statsRepo := repository.NewStatsRepository(queries)

	// Use Cases
	teamUC := usecase.NewTeamUseCase(teamRepo, userRepo, prRepo)
	userUC := usecase.NewUserUseCase(userRepo, prRepo)
	prUC := usecase.NewPRUseCase(prRepo, userRepo)
	statsUC := usecase.NewStatsUseCase(statsRepo)

	// Echo + Handlers
	e := echo.New()
	e.Use(middleware.Recover())
	e.Use(middleware.CORS())
	e.Use(handler.LoggingMiddleware(logger))

	// Handlers
	apiHandler := handler.NewAPIHandler(teamUC, userUC, prUC, statsUC, logger)
	api.RegisterHandlers(e, apiHandler)

	e.GET("/health", func(c echo.Context) error {
		return c.JSON(200, map[string]string{"status": "ok"})
	})

	// Запуск сервера
	go func() {
		if err := e.Start(":8080"); err != nil {
			logger.Infof("Server stopped: %v", err)
		}
	}()

	// Graceful shutdown
	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt, syscall.SIGTERM)
	<-stop

	logger.Info("Shutting down...")
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := e.Shutdown(ctx); err != nil {
		logger.Fatalf("Shutdown failed: %v", err)
	}

	logger.Info("Server exited")
}
