package domain

import "context"

// TeamUseCase определяет бизнес-логику для работы с командами.
type TeamUseCase interface {
	CreateTeam(ctx context.Context, team *Team) error
	GetTeam(ctx context.Context, teamName string) (*Team, error)
	DeactivateTeamUsers(ctx context.Context, teamName string) (*TeamDeactivationResult, error)
}

// UserUseCase определяет бизнес-логику для работы с пользователями.
type UserUseCase interface {
	SetUserActive(ctx context.Context, userID string, isActive bool) (*User, error)
	GetUserReviewPRs(ctx context.Context, userID string) ([]*PullRequest, error)
}

// PRUseCase определяет бизнес-логику для работы с Pull Request'ами.
type PRUseCase interface {
	CreatePR(ctx context.Context, prID, prName, authorID string) (*PullRequest, error)
	MergePR(ctx context.Context, prID string) (*PullRequest, error)
	ReassignReviewer(ctx context.Context, prID, oldReviewerID string) (*PullRequest, string, error)
}

// StatsUseCase определяет бизнес-логику для работы со статистикой.
type StatsUseCase interface {
	GetStatsReviews(ctx context.Context) ([]*ReviewStat, error)
	GetStatsPrAssignments(ctx context.Context) ([]*PRAssignmentStat, error)
}
