package usecase_test

import (
	"context"
	"testing"

	"pr-reviewer-service/internal/domain"
	"pr-reviewer-service/internal/usecase"
	"pr-reviewer-service/tests/mocks"

	"github.com/stretchr/testify/assert"
)

func TestTeamUseCase_CreateTeam_Success(t *testing.T) {
	ctx := context.Background()
	teamRepo := &mocks.TeamRepository{}
	userRepo := &mocks.UserRepository{}
	prRepo := &mocks.PRRepository{}
	uc := usecase.NewTeamUseCase(teamRepo, userRepo, prRepo)

	team := &domain.Team{
		Name: "backend",
		Members: []*domain.User{
			{ID: "u1", Username: "Alice", IsActive: true},
			{ID: "u2", Username: "Bob", IsActive: true},
		},
	}

	teamRepo.On("ExistsTeam", ctx, "backend").Return(false, nil)
	teamRepo.On("Create", ctx, team).Return(nil)

	err := uc.CreateTeam(ctx, team)

	assert.NoError(t, err)
	teamRepo.AssertExpectations(t)
}

func TestTeamUseCase_CreateTeam_ValidationErrors(t *testing.T) {
	ctx := context.Background()
	teamRepo := &mocks.TeamRepository{}
	userRepo := &mocks.UserRepository{}
	prRepo := &mocks.PRRepository{}
	uc := usecase.NewTeamUseCase(teamRepo, userRepo, prRepo)

	testCases := []struct {
		name     string
		team     *domain.Team
		expected error
	}{
		{
			name:     "Empty team name",
			team:     &domain.Team{Name: ""},
			expected: domain.ErrInvalidTeamName,
		},
		{
			name:     "No members",
			team:     &domain.Team{Name: "backend", Members: []*domain.User{}},
			expected: domain.ErrTeamMustHaveMembers,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			err := uc.CreateTeam(ctx, tc.team)
			assert.ErrorIs(t, err, tc.expected)
		})
	}
}

func TestTeamUseCase_GetTeam_Success(t *testing.T) {
	ctx := context.Background()
	teamRepo := &mocks.TeamRepository{}
	userRepo := &mocks.UserRepository{}
	prRepo := &mocks.PRRepository{}
	uc := usecase.NewTeamUseCase(teamRepo, userRepo, prRepo)

	expectedTeam := &domain.Team{
		Name: "backend",
		Members: []*domain.User{
			{ID: "u1", Username: "Alice", IsActive: true},
		},
	}

	teamRepo.On("ExistsTeam", ctx, "backend").Return(true, nil)
	teamRepo.On("GetByName", ctx, "backend").Return(expectedTeam, nil)

	team, err := uc.GetTeam(ctx, "backend")

	assert.NoError(t, err)
	assert.Equal(t, expectedTeam, team)
}

func TestTeamUseCase_GetTeam_NotFound(t *testing.T) {
	ctx := context.Background()
	teamRepo := &mocks.TeamRepository{}
	userRepo := &mocks.UserRepository{}
	prRepo := &mocks.PRRepository{}
	uc := usecase.NewTeamUseCase(teamRepo, userRepo, prRepo)

	teamRepo.On("ExistsTeam", ctx, "nonexistent").Return(false, nil)

	team, err := uc.GetTeam(ctx, "nonexistent")

	assert.ErrorIs(t, err, domain.ErrTeamNotFound)
	assert.Nil(t, team)
}

func TestTeamUseCase_DeactivateTeamUsers_Success(t *testing.T) {
	ctx := context.Background()
	teamRepo := &mocks.TeamRepository{}
	userRepo := &mocks.UserRepository{}
	prRepo := &mocks.PRRepository{}
	uc := usecase.NewTeamUseCase(teamRepo, userRepo, prRepo)

	activeUsers := []*domain.User{
		{ID: "u1", Username: "Alice", TeamName: "backend", IsActive: true},
		{ID: "u2", Username: "Bob", TeamName: "backend", IsActive: true},
	}

	teamRepo.On("ExistsTeam", ctx, "backend").Return(true, nil)
	teamRepo.On("GetActiveUsersFromTeam", ctx, "backend").Return(activeUsers, nil)
	teamRepo.On("GetOpenPRsWithTeamReviewers", ctx, "backend").Return([]string{}, nil)

	userRepo.On("UpdateActiveStatus", ctx, "u1", false).Return(&domain.User{ID: "u1", IsActive: false}, nil)
	userRepo.On("UpdateActiveStatus", ctx, "u2", false).Return(&domain.User{ID: "u2", IsActive: false}, nil)

	result, err := uc.DeactivateTeamUsers(ctx, "backend")

	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, "backend", result.TeamName)
	assert.Equal(t, 2, result.DeactivatedUsers)
	assert.ElementsMatch(t, []string{"u1", "u2"}, result.DeactivatedUserIDs)
}

func TestTeamUseCase_DeactivateTeamUsers_NoActiveUsers(t *testing.T) {
	ctx := context.Background()
	teamRepo := &mocks.TeamRepository{}
	userRepo := &mocks.UserRepository{}
	prRepo := &mocks.PRRepository{}
	uc := usecase.NewTeamUseCase(teamRepo, userRepo, prRepo)

	teamRepo.On("ExistsTeam", ctx, "backend").Return(true, nil)
	teamRepo.On("GetActiveUsersFromTeam", ctx, "backend").Return([]*domain.User{}, nil)

	result, err := uc.DeactivateTeamUsers(ctx, "backend")

	assert.ErrorIs(t, err, domain.ErrNoActiveUsersInTeam)
	assert.Nil(t, result)
}
