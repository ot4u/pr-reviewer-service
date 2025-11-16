package handler

import (
	"net/http"

	"pr-reviewer-service/api"
	"pr-reviewer-service/internal/domain"

	"github.com/labstack/echo/v4"
	"github.com/sirupsen/logrus"
)

// PRHandler обрабатывает HTTP-запросы связанные с пул-реквестами
type PRHandler struct {
	*BaseHandler
	prUseCase domain.PRUseCase
}

// NewPRHandler создает новый экземпляр PRHandler
func NewPRHandler(prUseCase domain.PRUseCase, logger *logrus.Logger) *PRHandler {
	return &PRHandler{
		BaseHandler: NewBaseHandler(logger),
		prUseCase:   prUseCase,
	}
}

// PostPullRequestCreate обрабатывает создание нового пул-реквеста
func (h *PRHandler) PostPullRequestCreate(c echo.Context) error {
	var req api.PostPullRequestCreateJSONBody
	if err := c.Bind(&req); err != nil {
		h.logger.WithError(err).Warn("Failed to bind create PR request")
		return c.JSON(http.StatusBadRequest, toErrorResponse("INVALID_REQUEST", err.Error()))
	}

	logEntry := h.logRequest(c, "create_pr").WithFields(logrus.Fields{
		"pr_id":   req.PullRequestId,
		"pr_name": req.PullRequestName,
		"author":  req.AuthorId,
	})
	logEntry.Info("Creating pull request")

	pr, err := h.prUseCase.CreatePR(c.Request().Context(), req.PullRequestId, req.PullRequestName, req.AuthorId)
	if err != nil {
		logEntry.WithError(err).Error("Failed to create PR")
		if httpErr, exists := domain.ToHTTPError(err); exists {
			return c.JSON(getHTTPStatusCode(err), toAPIErrorResponse(httpErr))
		}
		return c.JSON(http.StatusInternalServerError, toErrorResponse("INTERNAL_ERROR", err.Error()))
	}

	logEntry.WithField("reviewers_count", len(pr.AssignedReviewers)).Info("PR created successfully")
	return c.JSON(http.StatusCreated, map[string]interface{}{
		"pr": toAPIPullRequest(pr),
	})
}

// PostPullRequestMerge обрабатывает мерж существующего пул-реквеста
func (h *PRHandler) PostPullRequestMerge(c echo.Context) error {
	var req api.PostPullRequestMergeJSONBody
	if err := c.Bind(&req); err != nil {
		h.logger.WithError(err).Warn("Failed to bind merge PR request")
		return c.JSON(http.StatusBadRequest, toErrorResponse("INVALID_REQUEST", err.Error()))
	}

	logEntry := h.logRequest(c, "merge_pr").WithField("pr_id", req.PullRequestId)
	logEntry.Info("Merging pull request")

	pr, err := h.prUseCase.MergePR(c.Request().Context(), req.PullRequestId)
	if err != nil {
		logEntry.WithError(err).Error("Failed to merge PR")
		if httpErr, exists := domain.ToHTTPError(err); exists {
			return c.JSON(getHTTPStatusCode(err), toAPIErrorResponse(httpErr))
		}
		return c.JSON(http.StatusInternalServerError, toErrorResponse("INTERNAL_ERROR", err.Error()))
	}

	logEntry.Info("PR merged successfully")
	return c.JSON(http.StatusOK, map[string]interface{}{
		"pr": toAPIPullRequest(pr),
	})
}

// PostPullRequestReassign обрабатывает переназначение ревьювера пул-реквеста
func (h *PRHandler) PostPullRequestReassign(c echo.Context) error {
	var req api.PostPullRequestReassignJSONBody
	if err := c.Bind(&req); err != nil {
		h.logger.WithError(err).Warn("Failed to bind reassign reviewer request")
		return c.JSON(http.StatusBadRequest, toErrorResponse("INVALID_REQUEST", err.Error()))
	}

	logEntry := h.logRequest(c, "reassign_reviewer").WithFields(logrus.Fields{
		"pr_id":        req.PullRequestId,
		"old_reviewer": req.OldUserId,
	})
	logEntry.Info("Reassigning reviewer")

	pr, newReviewerID, err := h.prUseCase.ReassignReviewer(c.Request().Context(), req.PullRequestId, req.OldUserId)
	if err != nil {
		logEntry.WithError(err).Error("Failed to reassign reviewer")
		if httpErr, exists := domain.ToHTTPError(err); exists {
			return c.JSON(getHTTPStatusCode(err), toAPIErrorResponse(httpErr))
		}
		return c.JSON(http.StatusInternalServerError, toErrorResponse("INTERNAL_ERROR", err.Error()))
	}

	logEntry.WithField("new_reviewer", newReviewerID).Info("Reviewer reassigned successfully")
	return c.JSON(http.StatusOK, map[string]interface{}{
		"pr":          toAPIPullRequest(pr),
		"replaced_by": newReviewerID,
	})
}
