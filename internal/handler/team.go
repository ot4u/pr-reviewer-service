package handler

import (
	"net/http"

	"pr-reviewer-service/api"
	"pr-reviewer-service/internal/domain"

	"github.com/labstack/echo/v4"
	"github.com/sirupsen/logrus"
)

// TeamHandler обрабатывает HTTP-запросы для управления командами
type TeamHandler struct {
	*BaseHandler
	teamUseCase domain.TeamUseCase
}

// NewTeamHandler создает новый экземпляр TeamHandler
func NewTeamHandler(teamUseCase domain.TeamUseCase, logger *logrus.Logger) *TeamHandler {
	return &TeamHandler{
		BaseHandler: NewBaseHandler(logger),
		teamUseCase: teamUseCase,
	}
}

// PostTeamAdd обрабатывает создание новой команды с участниками
func (h *TeamHandler) PostTeamAdd(c echo.Context) error {
	logEntry := h.logRequest(c, "create_team")
	logEntry.Info("Creating team")

	var req api.Team
	if err := c.Bind(&req); err != nil {
		logEntry.WithError(err).Warn("Failed to bind request")
		return c.JSON(http.StatusBadRequest, toErrorResponse("INVALID_REQUEST", err.Error()))
	}

	logEntry = logEntry.WithField("team_name", req.TeamName)

	team := &domain.Team{
		Name: req.TeamName,
	}

	for _, member := range req.Members {
		team.Members = append(team.Members, &domain.User{
			ID:       member.UserId,
			Username: member.Username,
			TeamName: req.TeamName,
			IsActive: member.IsActive,
		})
	}

	if err := h.teamUseCase.CreateTeam(c.Request().Context(), team); err != nil {
		logEntry.WithError(err).Error("Failed to create team")
		if httpErr, exists := domain.ToHTTPError(err); exists {
			return c.JSON(getHTTPStatusCode(err), toAPIErrorResponse(httpErr))
		}
		return c.JSON(http.StatusInternalServerError, toErrorResponse("INTERNAL_ERROR", err.Error()))
	}

	logEntry.WithField("members_count", len(team.Members)).Info("Team created successfully")
	return c.JSON(http.StatusCreated, map[string]interface{}{
		"team": toAPITeam(team),
	})
}

// GetTeamGet обрабатывает получение информации о команде по названию
func (h *TeamHandler) GetTeamGet(c echo.Context, params api.GetTeamGetParams) error {
	logEntry := h.logRequest(c, "get_team").WithField("team_name", params.TeamName)
	logEntry.Info("Getting team")

	team, err := h.teamUseCase.GetTeam(c.Request().Context(), params.TeamName)
	if err != nil {
		logEntry.WithError(err).Warn("Team not found")
		if httpErr, exists := domain.ToHTTPError(err); exists {
			return c.JSON(getHTTPStatusCode(err), toAPIErrorResponse(httpErr))
		}
		return c.JSON(http.StatusInternalServerError, toErrorResponse("INTERNAL_ERROR", err.Error()))
	}

	logEntry.WithField("members_count", len(team.Members)).Info("Team retrieved successfully")
	return c.JSON(http.StatusOK, toAPITeam(team))
}

// PostTeamDeactivate обрабатывает массовую деактивацию пользователей команды
func (h *TeamHandler) PostTeamDeactivate(c echo.Context) error {
	var req struct {
		TeamName string `json:"team_name"`
	}

	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, toErrorResponse("INVALID_REQUEST", err.Error()))
	}

	logEntry := h.logRequest(c, "deactivate_team").WithField("team_name", req.TeamName)
	logEntry.Info("Deactivating team users")

	result, err := h.teamUseCase.DeactivateTeamUsers(c.Request().Context(), req.TeamName)
	if err != nil {
		logEntry.WithError(err).Error("Failed to deactivate team users")
		if httpErr, exists := domain.ToHTTPError(err); exists {
			return c.JSON(getHTTPStatusCode(err), toAPIErrorResponse(httpErr))
		}
		return c.JSON(http.StatusInternalServerError, toErrorResponse("INTERNAL_ERROR", err.Error()))
	}

	logEntry.WithFields(logrus.Fields{
		"deactivated_users":    result.DeactivatedUsers,
		"reassigned_prs":       result.ReassignedPRs,
		"failed_reassignments": result.FailedReassignments,
	}).Info("Team users deactivated successfully")

	// Даже при partial reassignment возвращаем 200 с результатом
	return c.JSON(http.StatusOK, map[string]interface{}{
		"team_name":            result.TeamName,
		"deactivated_users":    result.DeactivatedUsers,
		"reassigned_prs":       result.ReassignedPRs,
		"failed_reassignments": result.FailedReassignments,
		"deactivated_user_ids": result.DeactivatedUserIDs,
	})
}
