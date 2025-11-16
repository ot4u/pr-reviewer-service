package usecase_test

import (
	"context"
	"errors"
	"testing"

	"pr-reviewer-service/internal/domain"
	"pr-reviewer-service/internal/usecase"
	"pr-reviewer-service/tests/mocks"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func TestPRUseCase_CreatePR_Success(t *testing.T) {
	// Setup
	ctx := context.Background()
	prRepo := &mocks.PRRepository{}
	userRepo := &mocks.UserRepository{}
	uc := usecase.NewPRUseCase(prRepo, userRepo)

	// Test data
	author := &domain.User{ID: "u1", Username: "Alice", TeamName: "backend", IsActive: true}
	candidates := []*domain.User{
		{ID: "u2", Username: "Bob", TeamName: "backend", IsActive: true},
		{ID: "u3", Username: "Charlie", TeamName: "backend", IsActive: true},
	}

	// Mock expectations
	userRepo.On("GetByID", ctx, "u1").Return(author, nil)
	prRepo.On("ExistsPr", ctx, "pr-1001").Return(false, nil)
	userRepo.On("GetActiveUsersByTeam", ctx, "backend", "u1").Return(candidates, nil)
	prRepo.On("CreateWithReviewers", ctx, mock.AnythingOfType("*domain.PullRequest"), []string{"u2", "u3"}).Return(nil)

	// Execute
	pr, err := uc.CreatePR(ctx, "pr-1001", "Add feature", "u1")

	// Assert
	assert.NoError(t, err)
	assert.NotNil(t, pr)
	assert.Equal(t, "pr-1001", pr.ID)
	assert.Equal(t, "Add feature", pr.Name)
	assert.Equal(t, "u1", pr.AuthorID)
	assert.Equal(t, "OPEN", pr.Status)
	assert.ElementsMatch(t, []string{"u2", "u3"}, pr.AssignedReviewers)

	// Verify mocks
	userRepo.AssertExpectations(t)
	prRepo.AssertExpectations(t)
}

func TestPRUseCase_CreatePR_ValidationErrors(t *testing.T) {
	ctx := context.Background()
	prRepo := &mocks.PRRepository{}
	userRepo := &mocks.UserRepository{}
	uc := usecase.NewPRUseCase(prRepo, userRepo)

	testCases := []struct {
		name     string
		prID     string
		prName   string
		authorID string
		expected error
	}{
		{"Empty PR ID", "", "Test PR", "u1", domain.ErrInvalidPRID},
		{"Empty PR Name", "pr-1", "", "u1", domain.ErrInvalidPRName},
		{"Empty Author ID", "pr-1", "Test PR", "", domain.ErrInvalidUserID},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			pr, err := uc.CreatePR(ctx, tc.prID, tc.prName, tc.authorID)
			assert.ErrorIs(t, err, tc.expected)
			assert.Nil(t, pr)
		})
	}
}

func TestPRUseCase_CreatePR_AuthorNotFound(t *testing.T) {
	ctx := context.Background()
	prRepo := &mocks.PRRepository{}
	userRepo := &mocks.UserRepository{}
	uc := usecase.NewPRUseCase(prRepo, userRepo)

	userRepo.On("GetByID", ctx, "u1").Return(nil, errors.New("not found"))

	pr, err := uc.CreatePR(ctx, "pr-1001", "Add feature", "u1")

	assert.ErrorIs(t, err, domain.ErrPRAuthorNotFound)
	assert.Nil(t, pr)
}

func TestPRUseCase_CreatePR_AlreadyExists(t *testing.T) {
	ctx := context.Background()
	prRepo := &mocks.PRRepository{}
	userRepo := &mocks.UserRepository{}
	uc := usecase.NewPRUseCase(prRepo, userRepo)

	author := &domain.User{ID: "u1", Username: "Alice", TeamName: "backend", IsActive: true}
	userRepo.On("GetByID", ctx, "u1").Return(author, nil)
	prRepo.On("ExistsPr", ctx, "pr-1001").Return(true, nil)

	pr, err := uc.CreatePR(ctx, "pr-1001", "Add feature", "u1")

	assert.ErrorIs(t, err, domain.ErrPRAlreadyExists)
	assert.Nil(t, pr)
}

func TestPRUseCase_CreatePR_NoReviewerCandidates(t *testing.T) {
	ctx := context.Background()
	prRepo := &mocks.PRRepository{}
	userRepo := &mocks.UserRepository{}
	uc := usecase.NewPRUseCase(prRepo, userRepo)

	author := &domain.User{ID: "u1", Username: "Alice", TeamName: "backend", IsActive: true}
	userRepo.On("GetByID", ctx, "u1").Return(author, nil)
	prRepo.On("ExistsPr", ctx, "pr-1001").Return(false, nil)
	userRepo.On("GetActiveUsersByTeam", ctx, "backend", "u1").Return([]*domain.User{}, nil)

	pr, err := uc.CreatePR(ctx, "pr-1001", "Add feature", "u1")

	assert.ErrorIs(t, err, domain.ErrNoReviewerCandidate)
	assert.Nil(t, pr)
}

func TestPRUseCase_MergePR_Success(t *testing.T) {
	ctx := context.Background()
	prRepo := &mocks.PRRepository{}
	userRepo := &mocks.UserRepository{}
	uc := usecase.NewPRUseCase(prRepo, userRepo)

	mergedPR := &domain.PullRequest{
		ID:       "pr-1001",
		Name:     "Add feature",
		AuthorID: "u1",
		Status:   "MERGED",
	}

	prRepo.On("ExistsPr", ctx, "pr-1001").Return(true, nil)
	prRepo.On("Merge", ctx, "pr-1001").Return(mergedPR, nil)

	result, err := uc.MergePR(ctx, "pr-1001")

	assert.NoError(t, err)
	assert.Equal(t, mergedPR, result)
}

func TestPRUseCase_MergePR_NotFound(t *testing.T) {
	ctx := context.Background()
	prRepo := &mocks.PRRepository{}
	userRepo := &mocks.UserRepository{}
	uc := usecase.NewPRUseCase(prRepo, userRepo)

	prRepo.On("ExistsPr", ctx, "pr-1001").Return(false, nil)

	result, err := uc.MergePR(ctx, "pr-1001")

	assert.ErrorIs(t, err, domain.ErrPRNotFound)
	assert.Nil(t, result)
}

func TestPRUseCase_ReassignReviewer_Success(t *testing.T) {
	ctx := context.Background()
	prRepo := &mocks.PRRepository{}
	userRepo := &mocks.UserRepository{}
	uc := usecase.NewPRUseCase(prRepo, userRepo)

	pr := &domain.PullRequest{
		ID:       "pr-1001",
		Name:     "Add feature",
		AuthorID: "u1",
		Status:   "OPEN",
	}
	oldReviewer := &domain.User{ID: "u2", Username: "Bob", TeamName: "backend", IsActive: true}
	candidates := []*domain.User{
		{ID: "u3", Username: "Charlie", TeamName: "backend", IsActive: true},
	}
	updatedPR := &domain.PullRequest{
		ID:                "pr-1001",
		Name:              "Add feature",
		AuthorID:          "u1",
		Status:            "OPEN",
		AssignedReviewers: []string{"u3"},
	}

	prRepo.On("GetByID", ctx, "pr-1001").Return(pr, nil).Once()
	prRepo.On("IsUserReviewer", ctx, "pr-1001", "u2").Return(true, nil)
	userRepo.On("GetByID", ctx, "u2").Return(oldReviewer, nil)
	userRepo.On("GetActiveUsersByTeam", ctx, "backend", "u1").Return(candidates, nil)
	prRepo.On("ReassignReviewer", ctx, "pr-1001", "u2", "u3").Return(nil)
	prRepo.On("GetByID", ctx, "pr-1001").Return(updatedPR, nil).Once()

	resultPR, newReviewerID, err := uc.ReassignReviewer(ctx, "pr-1001", "u2")

	assert.NoError(t, err)
	assert.Equal(t, updatedPR, resultPR)
	assert.Equal(t, "u3", newReviewerID)

	// Проверяем что все моки были вызваны
	prRepo.AssertExpectations(t)
	userRepo.AssertExpectations(t)
}

func TestPRUseCase_ReassignReviewer_PRNotFound(t *testing.T) {
	ctx := context.Background()
	prRepo := &mocks.PRRepository{}
	userRepo := &mocks.UserRepository{}
	uc := usecase.NewPRUseCase(prRepo, userRepo)

	prRepo.On("GetByID", ctx, "pr-1001").Return(nil, errors.New("not found"))

	resultPR, newReviewerID, err := uc.ReassignReviewer(ctx, "pr-1001", "u2")

	assert.ErrorIs(t, err, domain.ErrPRNotFound)
	assert.Nil(t, resultPR)
	assert.Equal(t, "", newReviewerID)
}

func TestPRUseCase_ReassignReviewer_PRAlreadyMerged(t *testing.T) {
	ctx := context.Background()
	prRepo := &mocks.PRRepository{}
	userRepo := &mocks.UserRepository{}
	uc := usecase.NewPRUseCase(prRepo, userRepo)

	pr := &domain.PullRequest{
		ID:       "pr-1001",
		Name:     "Add feature",
		AuthorID: "u1",
		Status:   "MERGED",
	}

	prRepo.On("GetByID", ctx, "pr-1001").Return(pr, nil)

	resultPR, newReviewerID, err := uc.ReassignReviewer(ctx, "pr-1001", "u2")

	assert.ErrorIs(t, err, domain.ErrPRAlreadyMerged)
	assert.Nil(t, resultPR)
	assert.Equal(t, "", newReviewerID)
}

func TestPRUseCase_ReassignReviewer_ReviewerNotAssigned(t *testing.T) {
	ctx := context.Background()
	prRepo := &mocks.PRRepository{}
	userRepo := &mocks.UserRepository{}
	uc := usecase.NewPRUseCase(prRepo, userRepo)

	pr := &domain.PullRequest{
		ID:       "pr-1001",
		Name:     "Add feature",
		AuthorID: "u1",
		Status:   "OPEN",
	}

	prRepo.On("GetByID", ctx, "pr-1001").Return(pr, nil)
	prRepo.On("IsUserReviewer", ctx, "pr-1001", "u2").Return(false, nil)

	resultPR, newReviewerID, err := uc.ReassignReviewer(ctx, "pr-1001", "u2")

	assert.ErrorIs(t, err, domain.ErrReviewerNotAssigned)
	assert.Nil(t, resultPR)
	assert.Equal(t, "", newReviewerID)
}

func TestPRUseCase_MergePR_Idempotent(t *testing.T) {
	ctx := context.Background()
	prRepo := &mocks.PRRepository{}
	userRepo := &mocks.UserRepository{}
	uc := usecase.NewPRUseCase(prRepo, userRepo)

	mergedPR := &domain.PullRequest{
		ID:       "pr-1001",
		Name:     "Add feature",
		AuthorID: "u1",
		Status:   "MERGED",
	}

	// Первый вызов - мердж
	prRepo.On("ExistsPr", ctx, "pr-1001").Return(true, nil).Once()
	prRepo.On("Merge", ctx, "pr-1001").Return(mergedPR, nil).Once()

	// Второй вызов - идемпотентный, возвращает тот же результат
	prRepo.On("ExistsPr", ctx, "pr-1001").Return(true, nil).Once()
	prRepo.On("Merge", ctx, "pr-1001").Return(mergedPR, nil).Once()

	// Первый мердж
	result1, err1 := uc.MergePR(ctx, "pr-1001")
	assert.NoError(t, err1)
	assert.Equal(t, "MERGED", result1.Status)

	// Второй мердж (идемпотентный)
	result2, err2 := uc.MergePR(ctx, "pr-1001")
	assert.NoError(t, err2)
	assert.Equal(t, "MERGED", result2.Status)
	assert.Equal(t, result1, result2)

	prRepo.AssertExpectations(t)
}
