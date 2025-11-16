package usecase_test

import (
	"context"
	"testing"

	"pr-reviewer-service/internal/domain"
	"pr-reviewer-service/internal/usecase"
	"pr-reviewer-service/tests/mocks"

	"github.com/stretchr/testify/assert"
)

func TestStatsUseCase_GetStatsReviews_Success(t *testing.T) {
	ctx := context.Background()
	statsRepo := &mocks.StatsRepository{}
	uc := usecase.NewStatsUseCase(statsRepo)

	expectedStats := []*domain.ReviewStat{
		{UserID: "u1", Username: "Alice", ReviewCount: 5},
		{UserID: "u2", Username: "Bob", ReviewCount: 3},
	}

	statsRepo.On("GetStatsReviews", ctx).Return(expectedStats, nil)

	result, err := uc.GetStatsReviews(ctx)

	assert.NoError(t, err)
	assert.Equal(t, expectedStats, result)
	assert.Len(t, result, 2)
}

func TestStatsUseCase_GetStatsPrAssignments_Success(t *testing.T) {
	ctx := context.Background()
	statsRepo := &mocks.StatsRepository{}
	uc := usecase.NewStatsUseCase(statsRepo)

	// Используем правильные названия полей согласно OpenAPI спецификации
	expectedStats := []*domain.PRAssignmentStat{
		{
			PRID:           "pr-1",
			PRName:         "Feature 1",
			ReviewersCount: 2,
		},
		{
			PRID:           "pr-2",
			PRName:         "Feature 2",
			ReviewersCount: 1,
		},
	}

	statsRepo.On("GetStatsPrAssignments", ctx).Return(expectedStats, nil)

	result, err := uc.GetStatsPrAssignments(ctx)

	assert.NoError(t, err)
	assert.Equal(t, expectedStats, result)
	assert.Len(t, result, 2)
}
