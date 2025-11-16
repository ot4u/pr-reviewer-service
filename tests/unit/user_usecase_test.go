package usecase_test

import (
	"context"
	"testing"

	"pr-reviewer-service/internal/domain"
	"pr-reviewer-service/internal/usecase"
	"pr-reviewer-service/tests/mocks"

	"github.com/stretchr/testify/assert"
)

func TestUserUseCase_SetUserActive_Success(t *testing.T) {
	ctx := context.Background()
	userRepo := &mocks.UserRepository{}
	prRepo := &mocks.PRRepository{}
	uc := usecase.NewUserUseCase(userRepo, prRepo)

	user := &domain.User{ID: "u1", Username: "Alice", IsActive: true}
	updatedUser := &domain.User{ID: "u1", Username: "Alice", IsActive: false}

	userRepo.On("GetByID", ctx, "u1").Return(user, nil)
	userRepo.On("UpdateActiveStatus", ctx, "u1", false).Return(updatedUser, nil)

	result, err := uc.SetUserActive(ctx, "u1", false)

	assert.NoError(t, err)
	assert.Equal(t, updatedUser, result)
	assert.False(t, result.IsActive)
}

func TestUserUseCase_SetUserActive_UserNotFound(t *testing.T) {
	ctx := context.Background()
	userRepo := &mocks.UserRepository{}
	prRepo := &mocks.PRRepository{}
	uc := usecase.NewUserUseCase(userRepo, prRepo)

	userRepo.On("GetByID", ctx, "nonexistent").Return(nil, assert.AnError)

	result, err := uc.SetUserActive(ctx, "nonexistent", false)

	assert.ErrorIs(t, err, domain.ErrUserNotFound)
	assert.Nil(t, result)
}

func TestUserUseCase_GetUserReviewPRs_Success(t *testing.T) {
	ctx := context.Background()
	userRepo := &mocks.UserRepository{}
	prRepo := &mocks.PRRepository{}
	uc := usecase.NewUserUseCase(userRepo, prRepo)

	user := &domain.User{ID: "u1", Username: "Alice", IsActive: true}
	prs := []*domain.PullRequest{
		{ID: "pr-1", Name: "Feature 1", AuthorID: "u2", Status: "OPEN"},
		{ID: "pr-2", Name: "Feature 2", AuthorID: "u3", Status: "OPEN"},
	}

	userRepo.On("GetByID", ctx, "u1").Return(user, nil)
	prRepo.On("GetUserAssignedPRs", ctx, "u1").Return(prs, nil)

	result, err := uc.GetUserReviewPRs(ctx, "u1")

	assert.NoError(t, err)
	assert.Equal(t, prs, result)
	assert.Len(t, result, 2)
}

func TestUserUseCase_GetUserReviewPRs_UserNotFound(t *testing.T) {
	ctx := context.Background()
	userRepo := &mocks.UserRepository{}
	prRepo := &mocks.PRRepository{}
	uc := usecase.NewUserUseCase(userRepo, prRepo)

	userRepo.On("GetByID", ctx, "nonexistent").Return(nil, assert.AnError)

	result, err := uc.GetUserReviewPRs(ctx, "nonexistent")

	assert.ErrorIs(t, err, domain.ErrUserNotFound)
	assert.Nil(t, result)
}
