package handler

import (
	"github.com/labstack/echo/v4"
	"github.com/sirupsen/logrus"
)

type BaseHandler struct {
	logger *logrus.Logger
}

func NewBaseHandler(logger *logrus.Logger) *BaseHandler {
	return &BaseHandler{
		logger: logger,
	}
}

func (h *BaseHandler) logRequest(c echo.Context, operation string) *logrus.Entry {
	return h.logger.WithFields(logrus.Fields{
		"operation":  operation,
		"method":     c.Request().Method,
		"path":       c.Request().URL.Path,
		"ip":         c.RealIP(),
		"user_agent": c.Request().UserAgent(),
	})
}
