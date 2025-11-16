package usecase

import (
	"context"

	"pr-reviewer-service/internal/domain"
)

// StatsUseCase реализует бизнес-логику для работы со статистикой.
type StatsUseCase struct {
	statsRepo domain.StatsRepository
}

// NewStatsUseCase создает новый экземпляр StatsUseCase.
func NewStatsUseCase(statsRepo domain.StatsRepository) domain.StatsUseCase {
	return &StatsUseCase{
		statsRepo: statsRepo,
	}
}

// GetReviewStats возвращает статистику по количеству ревью для каждого пользователя.
func (uc *StatsUseCase) GetStatsReviews(ctx context.Context) ([]*domain.ReviewStat, error) {
	return uc.statsRepo.GetStatsReviews(ctx)
}

// GetPRAssignmentStats возвращает статистику по количеству назначенных ревьюверов на каждый PR.
func (uc *StatsUseCase) GetStatsPrAssignments(ctx context.Context) ([]*domain.PRAssignmentStat, error) {
	return uc.statsRepo.GetStatsPrAssignments(ctx)
}
