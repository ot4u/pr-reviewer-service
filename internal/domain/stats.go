package domain

import "context"

// ReviewStat представляет статистику по ревью для конкретного пользователя.
type ReviewStat struct {
	UserID      string
	Username    string
	ReviewCount int64
}

// PRAssignmentStat представляет статистику по назначению ревьюверов на PR.
type PRAssignmentStat struct {
	PRID           string
	PRName         string
	ReviewersCount int64
}

// StatsRepository определяет контракт для работы со статистическими данными.
type StatsRepository interface {
	GetStatsReviews(ctx context.Context) ([]*ReviewStat, error)
	GetStatsPrAssignments(ctx context.Context) ([]*PRAssignmentStat, error)
}
