package repository

import (
	"context"
	"fmt"

	"pr-reviewer-service/internal/database"
	"pr-reviewer-service/internal/domain"
)

// StatsRepository реализует domain.StatsRepository для работы со статистикой.
type StatsRepository struct {
	queries *database.Queries
}

// NewStatsRepository создает новый экземпляр StatsRepository.
func NewStatsRepository(queries *database.Queries) domain.StatsRepository {
	return &StatsRepository{
		queries: queries,
	}
}

// GetStatsReviews возвращает статистику по количеству ревью для каждого пользователя.
func (r *StatsRepository) GetStatsReviews(ctx context.Context) ([]*domain.ReviewStat, error) {
	stats, err := r.queries.GetReviewStats(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get users stats: %w", err)
	}

	result := make([]*domain.ReviewStat, len(stats))
	for i, stat := range stats {
		result[i] = &domain.ReviewStat{
			UserID:      stat.UserID,
			Username:    stat.Username,
			ReviewCount: stat.ReviewCount,
		}
	}

	return result, nil
}

// GetStatsPrAssignments возвращает статистику по количеству назначенных ревьюверов на каждый PR.
func (r *StatsRepository) GetStatsPrAssignments(ctx context.Context) ([]*domain.PRAssignmentStat, error) {
	stats, err := r.queries.GetPRAssignmentStats(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to pull request stats: %w", err)
	}

	result := make([]*domain.PRAssignmentStat, len(stats))
	for i, stat := range stats {
		result[i] = &domain.PRAssignmentStat{
			PRID:           stat.PullRequestID,
			PRName:         stat.PullRequestName,
			ReviewersCount: stat.ReviewersCount,
		}
	}

	return result, nil
}
