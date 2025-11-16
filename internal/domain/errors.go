package domain

import "errors"

// Domain errors (для бизнес-логики)
var (
	// Validation errors
	ErrInvalidPRID         = errors.New("invalid pull request id")
	ErrInvalidPRName       = errors.New("invalid pull request name")
	ErrInvalidUserID       = errors.New("invalid user id")
	ErrInvalidTeamName     = errors.New("invalid team name")
	ErrTeamMustHaveMembers = errors.New("team must have members")

	// User errors
	ErrUserNotFound      = errors.New("user not found")
	ErrUserAlreadyExists = errors.New("user already exists")

	// Team errors
	ErrTeamNotFound      = errors.New("team not found")
	ErrTeamAlreadyExists = errors.New("team already exists")

	// PR errors
	ErrPRNotFound       = errors.New("pull request not found")
	ErrPRAlreadyExists  = errors.New("pull request already exists")
	ErrPRAlreadyMerged  = errors.New("pull request already merged")
	ErrPRAuthorNotFound = errors.New("pull request author not found")

	// Reviewer errors
	ErrReviewerNotAssigned = errors.New("reviewer not assigned to this PR")
	ErrNoReviewerCandidate = errors.New("no active reviewer candidate available")

	// Team deactivation errors
	ErrTeamDeactivationFailed = errors.New("team deactivation failed")
	ErrNoActiveUsersInTeam    = errors.New("no active users in team")
	ErrPRReassignmentFailed   = errors.New("PR reassignment failed during team deactivation")

	// Mass operation errors
	ErrPartialReassignment = errors.New("partial reassignment completed with failures")
)

// HTTPError для соответствия OpenAPI
type HTTPError struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

type ErrorResponse struct {
	Error HTTPError `json:"error"`
}

// Маппинг domain ошибок в HTTP ошибки
var ErrorMapping = map[error]HTTPError{
	ErrTeamAlreadyExists:      {Code: "TEAM_EXISTS", Message: "team_name already exists"},
	ErrPRAlreadyExists:        {Code: "PR_EXISTS", Message: "PR id already exists"},
	ErrPRAlreadyMerged:        {Code: "PR_MERGED", Message: "cannot reassign on merged PR"},
	ErrReviewerNotAssigned:    {Code: "NOT_ASSIGNED", Message: "reviewer is not assigned to this PR"},
	ErrNoReviewerCandidate:    {Code: "NO_CANDIDATE", Message: "no active replacement candidate in team"},
	ErrUserNotFound:           {Code: "NOT_FOUND", Message: "user not found"},
	ErrTeamNotFound:           {Code: "NOT_FOUND", Message: "team not found"},
	ErrPRNotFound:             {Code: "NOT_FOUND", Message: "pull request not found"},
	ErrPRAuthorNotFound:       {Code: "NOT_FOUND", Message: "author not found"},
	ErrTeamDeactivationFailed: {Code: "DEACTIVATION_FAILED", Message: "team deactivation failed"},
	ErrNoActiveUsersInTeam:    {Code: "NO_ACTIVE_USERS", Message: "no active users in team to deactivate"},
	ErrPRReassignmentFailed:   {Code: "REASSIGNMENT_FAILED", Message: "PR reassignment failed during deactivation"},
	ErrPartialReassignment:    {Code: "PARTIAL_REASSIGNMENT", Message: "partial reassignment completed with some failures"},
}

// ToHTTPError преобразует domain ошибку в HTTP ошибку
func ToHTTPError(err error) (HTTPError, bool) {
	httpErr, exists := ErrorMapping[err]
	return httpErr, exists
}
