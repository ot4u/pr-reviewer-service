package handler

import (
	"time"

	"github.com/labstack/echo/v4"
	"github.com/sirupsen/logrus"
)

// LoggingMiddleware добавляет структурированное логирование
func LoggingMiddleware(logger *logrus.Logger) echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			start := time.Now()

			// Выполняем запрос
			err := next(c)

			// Логируем детали запроса
			latency := time.Since(start)
			status := c.Response().Status

			entry := logger.WithFields(logrus.Fields{
				"method":     c.Request().Method,
				"uri":        c.Request().URL.Path,
				"status":     status,
				"latency":    latency,
				"user_agent": c.Request().UserAgent(),
				"ip":         c.RealIP(),
			})

			if err != nil {
				entry = entry.WithField("error", err.Error())
			}

			if status >= 500 {
				entry.Error("Server error")
			} else if status >= 400 {
				entry.Warn("Client error")
			} else {
				entry.Info("Request processed")
			}

			return err
		}
	}
}
