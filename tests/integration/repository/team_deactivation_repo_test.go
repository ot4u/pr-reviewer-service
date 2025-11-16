package repository_test

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"os"
	"testing"

	"pr-reviewer-service/internal/database"
	"pr-reviewer-service/internal/domain"
	"pr-reviewer-service/internal/repository"

	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
)

type TeamDeactivationRepoTestSuite struct {
	suite.Suite
	db      *sql.DB
	queries *database.Queries
	repo    domain.TeamRepository
	ctx     context.Context
}

func (suite *TeamDeactivationRepoTestSuite) SetupSuite() {
	suite.ctx = context.Background()

	dsn := fmt.Sprintf(
		"postgres://%s:%s@%s:%s/%s?sslmode=disable",
		"postgres", "password", "localhost", "5433", "pr_reviewer_test",
	)

	var err error
	suite.db, err = sql.Open("pgx", dsn)
	if err != nil {
		log.Fatalf("Failed to connect to test database: %v", err)
	}

	err = suite.db.Ping()
	if err != nil {
		log.Fatalf("Failed to ping test database: %v", err)
	}

	suite.queries = database.New(suite.db)
	suite.repo = repository.NewTeamRepository(suite.db, suite.queries)

	suite.cleanDatabase()
	suite.setupTestData()
}

func (suite *TeamDeactivationRepoTestSuite) TearDownTest() {
	suite.cleanDatabase()
	suite.setupTestData()
}

func (suite *TeamDeactivationRepoTestSuite) TearDownSuite() {
	if suite.db != nil {
		suite.db.Close()
	}
}

func (suite *TeamDeactivationRepoTestSuite) cleanDatabase() {
	tables := []string{"reviewers", "pull_requests", "users", "teams"}
	for _, table := range tables {
		_, err := suite.db.ExecContext(suite.ctx, fmt.Sprintf("DELETE FROM %s", table))
		if err != nil {
			log.Printf("Failed to clean table %s: %v", table, err)
		}
	}
}

func (suite *TeamDeactivationRepoTestSuite) setupTestData() {
	// Создаем команды
	teams := []string{"backend", "frontend", "mobile"}
	for _, teamName := range teams {
		_, err := suite.queries.CreateTeam(suite.ctx, teamName)
		if err != nil {
			log.Printf("Failed to create team %s: %v", teamName, err)
		}

		// Создаем пользователей
		users := []struct {
			id       string
			username string
			isActive bool
		}{
			{id: fmt.Sprintf("%s_active1", teamName), username: fmt.Sprintf("%s_Active1", teamName), isActive: true},
			{id: fmt.Sprintf("%s_active2", teamName), username: fmt.Sprintf("%s_Active2", teamName), isActive: true},
			{id: fmt.Sprintf("%s_inactive", teamName), username: fmt.Sprintf("%s_Inactive", teamName), isActive: false},
		}

		for _, user := range users {
			_, err := suite.queries.UpsertUser(suite.ctx, database.UpsertUserParams{
				UserID:   user.id,
				Username: user.username,
				TeamName: teamName,
				IsActive: user.isActive,
			})
			if err != nil {
				log.Printf("Failed to create user %s: %v", user.id, err)
			}
		}
	}

	// Создаем PR с разными статусами и ревьюверами
	prs := []struct {
		id          string
		name        string
		authorID    string
		status      string
		reviewerIDs []string
	}{
		{
			id:          "pr-open-backend",
			name:        "Open PR with backend reviewers",
			authorID:    "frontend_active1",
			status:      "OPEN",
			reviewerIDs: []string{"backend_active1", "backend_active2"},
		},
		{
			id:          "pr-merged-backend",
			name:        "Merged PR with backend reviewers",
			authorID:    "mobile_active1",
			status:      "MERGED",
			reviewerIDs: []string{"backend_active1"},
		},
		{
			id:          "pr-open-mixed",
			name:        "Open PR with mixed reviewers",
			authorID:    "backend_active1",
			status:      "OPEN",
			reviewerIDs: []string{"backend_active1", "frontend_active1"},
		},
		{
			id:          "pr-open-frontend",
			name:        "Open PR with frontend reviewers",
			authorID:    "backend_active1",
			status:      "OPEN",
			reviewerIDs: []string{"frontend_active1", "frontend_active2"},
		},
	}

	for _, prData := range prs {
		// Создаем PR
		_, err := suite.queries.CreatePullRequest(suite.ctx, database.CreatePullRequestParams{
			PullRequestID:   prData.id,
			PullRequestName: prData.name,
			AuthorID:        prData.authorID,
		})
		if err != nil {
			log.Printf("Failed to create PR %s: %v", prData.id, err)
			continue
		}

		// Обновляем статус если нужно
		if prData.status == "MERGED" {
			_, err = suite.queries.MergePullRequest(suite.ctx, prData.id)
			if err != nil {
				log.Printf("Failed to merge PR %s: %v", prData.id, err)
			}
		}

		// Назначаем ревьюверов
		for _, reviewerID := range prData.reviewerIDs {
			err := suite.queries.AssignReviewer(suite.ctx, database.AssignReviewerParams{
				PullRequestID: prData.id,
				UserID:        reviewerID,
			})
			if err != nil {
				log.Printf("Failed to assign reviewer %s to PR %s: %v", reviewerID, prData.id, err)
			}
		}
	}
}

func (suite *TeamDeactivationRepoTestSuite) TestGetActiveUsersFromTeam() {
	users, err := suite.repo.GetActiveUsersFromTeam(suite.ctx, "backend")

	assert.NoError(suite.T(), err)
	assert.Equal(suite.T(), 2, len(users)) // Только активные пользователи

	for _, user := range users {
		assert.True(suite.T(), user.IsActive)
		assert.Equal(suite.T(), "backend", user.TeamName)
		assert.Contains(suite.T(), []string{"backend_active1", "backend_active2"}, user.ID)
	}
}

func (suite *TeamDeactivationRepoTestSuite) TestGetActiveUsersFromTeam_Empty() {
	users, err := suite.repo.GetActiveUsersFromTeam(suite.ctx, "nonexistent_team")

	assert.NoError(suite.T(), err)
	assert.Empty(suite.T(), users)
}

func (suite *TeamDeactivationRepoTestSuite) TestGetOpenPRsWithTeamReviewers() {
	prIDs, err := suite.repo.GetOpenPRsWithTeamReviewers(suite.ctx, "backend")

	assert.NoError(suite.T(), err)
	assert.Equal(suite.T(), 2, len(prIDs)) // Должны быть только OPEN PR с backend ревьюверами
	assert.Contains(suite.T(), prIDs, "pr-open-backend")
	assert.Contains(suite.T(), prIDs, "pr-open-mixed")
	assert.NotContains(suite.T(), prIDs, "pr-merged-backend") // MERGED PR не должен быть
	assert.NotContains(suite.T(), prIDs, "pr-open-frontend")  // PR без backend ревьюверов не должен быть
}

func (suite *TeamDeactivationRepoTestSuite) TestGetOpenPRsWithTeamReviewers_Empty() {
	prIDs, err := suite.repo.GetOpenPRsWithTeamReviewers(suite.ctx, "nonexistent_team")

	assert.NoError(suite.T(), err)
	assert.Empty(suite.T(), prIDs)
}

func (suite *TeamDeactivationRepoTestSuite) TestGetPRReviewersFromTeam() {
	reviewerIDs, err := suite.repo.GetPRReviewersFromTeam(suite.ctx, "pr-open-backend", "backend")

	assert.NoError(suite.T(), err)
	assert.Equal(suite.T(), 2, len(reviewerIDs))
	assert.Contains(suite.T(), reviewerIDs, "backend_active1")
	assert.Contains(suite.T(), reviewerIDs, "backend_active2")
}

func (suite *TeamDeactivationRepoTestSuite) TestGetPRReviewersFromTeam_OnlyActive() {
	// Деактивируем одного пользователя
	_, err := suite.queries.UpdateUserActiveStatus(suite.ctx, database.UpdateUserActiveStatusParams{
		UserID:   "backend_active1",
		IsActive: false,
	})
	assert.NoError(suite.T(), err)

	reviewerIDs, err := suite.repo.GetPRReviewersFromTeam(suite.ctx, "pr-open-backend", "backend")

	assert.NoError(suite.T(), err)
	assert.Equal(suite.T(), 1, len(reviewerIDs)) // Только активные ревьюверы
	assert.Contains(suite.T(), reviewerIDs, "backend_active2")
	assert.NotContains(suite.T(), reviewerIDs, "backend_active1") // Деактивированный не должен быть
}

func (suite *TeamDeactivationRepoTestSuite) TestGetPRReviewersFromTeam_Empty() {
	reviewerIDs, err := suite.repo.GetPRReviewersFromTeam(suite.ctx, "nonexistent_pr", "backend")

	assert.NoError(suite.T(), err)
	assert.Empty(suite.T(), reviewerIDs)
}

func (suite *TeamDeactivationRepoTestSuite) TestGetAllTeams() {
	teams, err := suite.repo.GetAllTeams(suite.ctx)

	assert.NoError(suite.T(), err)
	assert.Equal(suite.T(), 3, len(teams)) // backend, frontend, mobile

	teamNames := make([]string, len(teams))
	for i, team := range teams {
		teamNames[i] = team.Name
	}

	assert.Contains(suite.T(), teamNames, "backend")
	assert.Contains(suite.T(), teamNames, "frontend")
	assert.Contains(suite.T(), teamNames, "mobile")
}

func TestTeamDeactivationRepoTestSuite(t *testing.T) {
	if os.Getenv("RUN_INTEGRATION_TESTS") != "1" {
		t.Skip("Skipping integration test. Set RUN_INTEGRATION_TESTS=1 to run.")
	}
	suite.Run(t, new(TeamDeactivationRepoTestSuite))
}
