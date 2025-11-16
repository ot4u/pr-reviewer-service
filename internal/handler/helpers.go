package handler

import (
	"net/http"
	"time"

	"pr-reviewer-service/api"
	"pr-reviewer-service/internal/domain"
)

// Вспомогательные функции преобразования доменных моделей в API модели

func toAPITeam(team *domain.Team) api.Team {
	members := make([]api.TeamMember, len(team.Members))
	for i, member := range team.Members {
		members[i] = api.TeamMember{
			UserId:   member.ID,
			Username: member.Username,
			IsActive: member.IsActive,
		}
	}
	return api.Team{
		TeamName: team.Name,
		Members:  members,
	}
}

func toAPIUser(user *domain.User) api.User {
	return api.User{
		UserId:   user.ID,
		Username: user.Username,
		TeamName: user.TeamName,
		IsActive: user.IsActive,
	}
}

func toAPIPullRequest(pr *domain.PullRequest) api.PullRequest {
	var mergedAt *time.Time = nil
	if pr.MergedAt != nil {
		mergedAt = pr.MergedAt
	}

	return api.PullRequest{
		PullRequestId:     pr.ID,
		PullRequestName:   pr.Name,
		AuthorId:          pr.AuthorID,
		Status:            api.PullRequestStatus(pr.Status),
		AssignedReviewers: pr.AssignedReviewers,
		MergedAt:          mergedAt,
	}
}

func toAPIPRShorts(prs []*domain.PullRequest) []api.PullRequestShort {
	result := make([]api.PullRequestShort, len(prs))
	for i, pr := range prs {
		result[i] = api.PullRequestShort{
			PullRequestId:   pr.ID,
			PullRequestName: pr.Name,
			AuthorId:        pr.AuthorID,
			Status:          api.PullRequestShortStatus(pr.Status),
		}
	}
	return result
}

func toErrorResponse(code, message string) api.ErrorResponse {
	return api.ErrorResponse{
		Error: struct {
			Code    api.ErrorResponseErrorCode `json:"code"`
			Message string                     `json:"message"`
		}{
			Code:    api.ErrorResponseErrorCode(code),
			Message: message,
		},
	}
}

func toAPIErrorResponse(httpErr domain.HTTPError) api.ErrorResponse {
	return api.ErrorResponse{
		Error: struct {
			Code    api.ErrorResponseErrorCode `json:"code"`
			Message string                     `json:"message"`
		}{
			Code:    api.ErrorResponseErrorCode(httpErr.Code),
			Message: httpErr.Message,
		},
	}
}

func getHTTPStatusCode(err error) int {
	switch err {
	// Conflict errors (409)
	case domain.ErrTeamAlreadyExists, domain.ErrPRAlreadyExists,
		domain.ErrPRAlreadyMerged, domain.ErrReviewerNotAssigned,
		domain.ErrNoReviewerCandidate, domain.ErrPartialReassignment,
		domain.ErrNoActiveUsersInTeam:
		return http.StatusConflict

	// Not Found errors (404)
	case domain.ErrUserNotFound, domain.ErrTeamNotFound,
		domain.ErrPRNotFound, domain.ErrPRAuthorNotFound:
		return http.StatusNotFound

	// Bad Request errors (400) - валидация
	case domain.ErrInvalidPRID, domain.ErrInvalidPRName,
		domain.ErrInvalidUserID, domain.ErrInvalidTeamName,
		domain.ErrTeamMustHaveMembers:
		return http.StatusBadRequest

	// Internal Server Error with specific codes (500)
	case domain.ErrTeamDeactivationFailed, domain.ErrPRReassignmentFailed:
		return http.StatusInternalServerError

	default:
		return http.StatusInternalServerError
	}
}
