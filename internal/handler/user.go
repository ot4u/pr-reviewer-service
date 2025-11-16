package handler

import (
	"net/http"

	"pr-reviewer-service/api"
	"pr-reviewer-service/internal/domain"

	"github.com/labstack/echo/v4"
	"github.com/sirupsen/logrus"
)

// UserHandler обрабатывает HTTP-запросы, связанные с пользователями.
type UserHandler struct {
	*BaseHandler
	userUseCase domain.UserUseCase
}

// NewUserHandler создает новый экземпляр UserHandler.
func NewUserHandler(userUseCase domain.UserUseCase, logger *logrus.Logger) *UserHandler {
	return &UserHandler{
		BaseHandler: NewBaseHandler(logger),
		userUseCase: userUseCase,
	}
}

// PostUsersSetIsActive обрабатывает запрос для установки статуса активности пользователя.
func (h *UserHandler) PostUsersSetIsActive(c echo.Context) error {
	var req api.PostUsersSetIsActiveJSONBody
	if err := c.Bind(&req); err != nil {
		h.logger.WithError(err).Warn("Failed to bind set active request")
		return c.JSON(http.StatusBadRequest, toErrorResponse("INVALID_REQUEST", err.Error()))
	}

	logEntry := h.logRequest(c, "set_user_active").WithFields(logrus.Fields{
		"user_id":   req.UserId,
		"is_active": req.IsActive,
	})
	logEntry.Info("Setting user active status")

	user, err := h.userUseCase.SetUserActive(c.Request().Context(), req.UserId, req.IsActive)
	if err != nil {
		logEntry.WithError(err).Error("Failed to set user active status")
		if httpErr, exists := domain.ToHTTPError(err); exists {
			return c.JSON(getHTTPStatusCode(err), toAPIErrorResponse(httpErr))
		}
		return c.JSON(http.StatusInternalServerError, toErrorResponse("INTERNAL_ERROR", err.Error()))
	}

	logEntry.Info("User active status updated successfully")
	return c.JSON(http.StatusOK, map[string]interface{}{
		"user": toAPIUser(user),
	})
}

// GetUsersGetReview обрабатывает запрос для получения списка PR, назначенных пользователю на ревью.
func (h *UserHandler) GetUsersGetReview(c echo.Context, params api.GetUsersGetReviewParams) error {
	logEntry := h.logRequest(c, "get_user_reviews").WithField("user_id", params.UserId)
	logEntry.Info("Getting user review PRs")

	prs, err := h.userUseCase.GetUserReviewPRs(c.Request().Context(), params.UserId)
	if err != nil {
		logEntry.WithError(err).Warn("Failed to get user review PRs")
		if httpErr, exists := domain.ToHTTPError(err); exists {
			return c.JSON(getHTTPStatusCode(err), toAPIErrorResponse(httpErr))
		}
		return c.JSON(http.StatusInternalServerError, toErrorResponse("INTERNAL_ERROR", err.Error()))
	}

	logEntry.WithField("prs_count", len(prs)).Info("User review PRs retrieved successfully")
	return c.JSON(http.StatusOK, map[string]interface{}{
		"user_id":       params.UserId,
		"pull_requests": toAPIPRShorts(prs),
	})
}
