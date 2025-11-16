package handler

import (
	"net/http"

	"pr-reviewer-service/internal/domain"

	"github.com/labstack/echo/v4"
	"github.com/sirupsen/logrus"
)

// StatsHandler обрабатывает HTTP-запросы для получения статистических данных.
type StatsHandler struct {
	*BaseHandler
	statsUseCase domain.StatsUseCase
}

// NewStatsHandler создает новый экземпляр StatsHandler.
func NewStatsHandler(statsUseCase domain.StatsUseCase, logger *logrus.Logger) *StatsHandler {
	return &StatsHandler{
		BaseHandler:  NewBaseHandler(logger),
		statsUseCase: statsUseCase,
	}
}

// GetReviewStats обрабатывает GET запрос для получения статистики по ревью пользователей.
func (h *StatsHandler) GetStatsReviews(c echo.Context) error {
	logEntry := h.logRequest(c, "get_review_stats")
	logEntry.Info("Getting review statistics")

	stats, err := h.statsUseCase.GetStatsReviews(c.Request().Context())
	if err != nil {
		logEntry.WithError(err).Error("Failed to get review stats")
		return c.JSON(http.StatusInternalServerError, toErrorResponse("INTERNAL_ERROR", err.Error()))
	}

	logEntry.WithField("stats_count", len(stats)).Info("Review stats retrieved")
	return c.JSON(http.StatusOK, map[string]interface{}{
		"stats": stats,
	})
}

// GetPRAssignmentStats обрабатывает GET запрос для получения статистики по назначению ревьюверов на PR.
func (h *StatsHandler) GetStatsPrAssignments(c echo.Context) error {
	logEntry := h.logRequest(c, "get_pr_assignment_stats")
	logEntry.Info("Getting PR assignment statistics")

	stats, err := h.statsUseCase.GetStatsPrAssignments(c.Request().Context())
	if err != nil {
		logEntry.WithError(err).Error("Failed to get PR assignment stats")
		return c.JSON(http.StatusInternalServerError, toErrorResponse("INTERNAL_ERROR", err.Error()))
	}

	logEntry.WithField("stats_count", len(stats)).Info("PR assignment stats retrieved")
	return c.JSON(http.StatusOK, map[string]interface{}{
		"stats": stats,
	})
}
