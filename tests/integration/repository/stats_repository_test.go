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

type StatsRepositoryTestSuite struct {
	suite.Suite
	db      *sql.DB
	queries *database.Queries
	repo    domain.StatsRepository
	ctx     context.Context
}

func (suite *StatsRepositoryTestSuite) SetupSuite() {
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
	statsRepo := repository.NewStatsRepository(suite.queries)
	suite.repo = statsRepo

	suite.cleanDatabase()
	suite.setupTestData()
}

func (suite *StatsRepositoryTestSuite) TearDownTest() {
	suite.cleanDatabase()
	suite.setupTestData()
}

func (suite *StatsRepositoryTestSuite) TearDownSuite() {
	if suite.db != nil {
		suite.db.Close()
	}
}

func (suite *StatsRepositoryTestSuite) cleanDatabase() {
	tables := []string{"reviewers", "pull_requests", "users", "teams"}
	for _, table := range tables {
		_, err := suite.db.ExecContext(suite.ctx, fmt.Sprintf("DELETE FROM %s", table))
		if err != nil {
			log.Printf("Failed to clean table %s: %v", table, err)
		}
	}
}

func (suite *StatsRepositoryTestSuite) setupTestData() {
	// Создаем команды
	teams := []string{"backend", "frontend"}
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
			{id: fmt.Sprintf("%s_author", teamName), username: fmt.Sprintf("%s_Author", teamName), isActive: true},
			{id: fmt.Sprintf("%s_reviewer1", teamName), username: fmt.Sprintf("%s_Reviewer1", teamName), isActive: true},
			{id: fmt.Sprintf("%s_reviewer2", teamName), username: fmt.Sprintf("%s_Reviewer2", teamName), isActive: true},
			{id: fmt.Sprintf("%s_reviewer3", teamName), username: fmt.Sprintf("%s_Reviewer3", teamName), isActive: true},
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

	// Создаем тестовые PR с разным количеством ревьюверов
	prs := []struct {
		id          string
		name        string
		authorID    string
		reviewerIDs []string
	}{
		{
			id:          "pr-many-reviews",
			name:        "PR with 3 reviewers",
			authorID:    "backend_author",
			reviewerIDs: []string{"backend_reviewer1", "backend_reviewer2", "backend_reviewer3"},
		},
		{
			id:          "pr-two-reviews",
			name:        "PR with 2 reviewers",
			authorID:    "frontend_author",
			reviewerIDs: []string{"frontend_reviewer1", "frontend_reviewer2"},
		},
		{
			id:          "pr-one-review",
			name:        "PR with 1 reviewer",
			authorID:    "backend_author",
			reviewerIDs: []string{"backend_reviewer1"},
		},
		{
			id:          "pr-no-reviews",
			name:        "PR without reviewers",
			authorID:    "frontend_author",
			reviewerIDs: []string{},
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

func (suite *StatsRepositoryTestSuite) TestGetReviewStats() {
	stats, err := suite.repo.GetStatsReviews(suite.ctx)
	assert.NoError(suite.T(), err)
	assert.NotNil(suite.T(), stats)

	// Проверяем что все пользователи присутствуют в статистике
	userStats := make(map[string]int64)
	for _, stat := range stats {
		userStats[stat.UserID] = stat.ReviewCount
	}

	// Проверяем конкретные значения на основе тестовых данных
	assert.Equal(suite.T(), int64(2), userStats["backend_reviewer1"])  // Назначен на 2 PR
	assert.Equal(suite.T(), int64(1), userStats["backend_reviewer2"])  // Назначен на 1 PR
	assert.Equal(suite.T(), int64(1), userStats["backend_reviewer3"])  // Назначен на 1 PR
	assert.Equal(suite.T(), int64(1), userStats["frontend_reviewer1"]) // Назначен на 1 PR
	assert.Equal(suite.T(), int64(1), userStats["frontend_reviewer2"]) // Назначен на 1 PR

	// Проверяем что авторы имеют 0 назначений (они не ревьюверы)
	assert.Equal(suite.T(), int64(0), userStats["backend_author"])
	assert.Equal(suite.T(), int64(0), userStats["frontend_author"])

	// Проверяем сортировку по убыванию количества ревью
	for i := 0; i < len(stats)-1; i++ {
		assert.GreaterOrEqual(suite.T(), stats[i].ReviewCount, stats[i+1].ReviewCount)
	}
}

func (suite *StatsRepositoryTestSuite) TestGetReviewStats_Empty() {
	// Очищаем БД и проверяем пустую статистику
	suite.cleanDatabase()

	stats, err := suite.repo.GetStatsReviews(suite.ctx)
	assert.NoError(suite.T(), err)
	assert.Empty(suite.T(), stats)
}

func (suite *StatsRepositoryTestSuite) TestGetPRAssignmentStats() {
	stats, err := suite.repo.GetStatsPrAssignments(suite.ctx)
	assert.NoError(suite.T(), err)
	assert.NotNil(suite.T(), stats)

	// Создаем мапу для удобства проверки
	prStats := make(map[string]int64)
	for _, stat := range stats {
		prStats[stat.PRID] = stat.ReviewersCount
	}

	// Проверяем количество ревьюверов для каждого PR
	assert.Equal(suite.T(), int64(3), prStats["pr-many-reviews"])
	assert.Equal(suite.T(), int64(2), prStats["pr-two-reviews"])
	assert.Equal(suite.T(), int64(1), prStats["pr-one-review"])
	assert.Equal(suite.T(), int64(0), prStats["pr-no-reviews"])

	// Проверяем сортировку по убыванию количества ревьюверов
	for i := 0; i < len(stats)-1; i++ {
		assert.GreaterOrEqual(suite.T(), stats[i].ReviewersCount, stats[i+1].ReviewersCount)
	}
}

func (suite *StatsRepositoryTestSuite) TestGetPRAssignmentStats_Empty() {
	// Очищаем БД и проверяем пустую статистику
	suite.cleanDatabase()

	stats, err := suite.repo.GetStatsPrAssignments(suite.ctx)
	assert.NoError(suite.T(), err)
	assert.Empty(suite.T(), stats)
}

func (suite *StatsRepositoryTestSuite) TestGetReviewStats_IncludesAllUsers() {
	stats, err := suite.repo.GetStatsReviews(suite.ctx)
	assert.NoError(suite.T(), err)

	// Проверяем что в статистике есть все пользователи (даже с 0 назначений)
	userIDs := make(map[string]bool)
	for _, stat := range stats {
		userIDs[stat.UserID] = true
	}

	// Должны быть все пользователи из тестовых данных
	expectedUsers := []string{
		"backend_author", "backend_reviewer1", "backend_reviewer2", "backend_reviewer3",
		"frontend_author", "frontend_reviewer1", "frontend_reviewer2", "frontend_reviewer3",
	}

	for _, userID := range expectedUsers {
		assert.True(suite.T(), userIDs[userID], "User %s should be in stats", userID)
	}
}

func TestStatsRepositoryTestSuite(t *testing.T) {
	if os.Getenv("RUN_INTEGRATION_TESTS") != "1" {
		t.Skip("Skipping integration test. Set RUN_INTEGRATION_TESTS=1 to run.")
	}
	suite.Run(t, new(StatsRepositoryTestSuite))
}
