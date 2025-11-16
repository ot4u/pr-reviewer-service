package repository_test

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"os"
	"testing"
	"time"

	"pr-reviewer-service/internal/database"
	"pr-reviewer-service/internal/domain"
	"pr-reviewer-service/internal/repository"

	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
)

type PRRepositoryTestSuite struct {
	suite.Suite
	db      *sql.DB
	queries *database.Queries
	repo    domain.PRRepository
	ctx     context.Context
}

func (suite *PRRepositoryTestSuite) SetupSuite() {
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
	suite.repo = repository.NewPRRepository(suite.db, suite.queries)

	suite.cleanDatabase()
	suite.setupTestData()
}

func (suite *PRRepositoryTestSuite) TearDownTest() {
	suite.cleanDatabase()
	suite.setupTestData()
}

func (suite *PRRepositoryTestSuite) TearDownSuite() {
	if suite.db != nil {
		suite.db.Close()
	}
}

func (suite *PRRepositoryTestSuite) cleanDatabase() {
	tables := []string{"reviewers", "pull_requests", "users", "teams"}
	for _, table := range tables {
		_, err := suite.db.ExecContext(suite.ctx, fmt.Sprintf("DELETE FROM %s", table))
		if err != nil {
			log.Printf("Failed to clean table %s: %v", table, err)
		}
	}
}

func (suite *PRRepositoryTestSuite) setupTestData() {
	// Создаем тестовые команды
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
}

func (suite *PRRepositoryTestSuite) TestCreateWithReviewers_Success() {
	pr := &domain.PullRequest{
		ID:       "pr-001",
		Name:     "Test PR",
		AuthorID: "backend_author",
		Status:   "OPEN",
	}
	reviewerIDs := []string{"backend_reviewer1", "backend_reviewer2"}

	err := suite.repo.CreateWithReviewers(suite.ctx, pr, reviewerIDs)
	assert.NoError(suite.T(), err)

	// Проверяем что PR создался
	createdPR, err := suite.repo.GetByID(suite.ctx, "pr-001")
	assert.NoError(suite.T(), err)
	assert.Equal(suite.T(), "pr-001", createdPR.ID)
	assert.Equal(suite.T(), "Test PR", createdPR.Name)
	assert.Equal(suite.T(), "backend_author", createdPR.AuthorID)
	assert.Equal(suite.T(), "OPEN", createdPR.Status)

	// Проверяем что ревьюверы назначились
	reviewers, err := suite.repo.GetReviewers(suite.ctx, "pr-001")
	assert.NoError(suite.T(), err)
	assert.Equal(suite.T(), 2, len(reviewers))
	assert.Contains(suite.T(), reviewers, "backend_reviewer1")
	assert.Contains(suite.T(), reviewers, "backend_reviewer2")
}

func (suite *PRRepositoryTestSuite) TestCreateWithReviewers_EmptyReviewers() {
	pr := &domain.PullRequest{
		ID:       "pr-002",
		Name:     "PR Without Reviewers",
		AuthorID: "backend_author",
		Status:   "OPEN",
	}

	err := suite.repo.CreateWithReviewers(suite.ctx, pr, []string{})
	assert.NoError(suite.T(), err)

	// Проверяем что PR создался без ревьюверов
	createdPR, err := suite.repo.GetByID(suite.ctx, "pr-002")
	assert.NoError(suite.T(), err)
	assert.Equal(suite.T(), "pr-002", createdPR.ID)

	reviewers, err := suite.repo.GetReviewers(suite.ctx, "pr-002")
	assert.NoError(suite.T(), err)
	assert.Empty(suite.T(), reviewers)
}

func (suite *PRRepositoryTestSuite) TestGetByID_NotFound() {
	pr, err := suite.repo.GetByID(suite.ctx, "nonexistent_pr")
	assert.Error(suite.T(), err)
	assert.Nil(suite.T(), pr)
	assert.Equal(suite.T(), domain.ErrPRNotFound, err)
}

func (suite *PRRepositoryTestSuite) TestExistsPr() {
	// Создаем PR
	pr := &domain.PullRequest{
		ID:       "pr-003",
		Name:     "Test PR",
		AuthorID: "backend_author",
		Status:   "OPEN",
	}
	err := suite.repo.CreateWithReviewers(suite.ctx, pr, []string{})
	assert.NoError(suite.T(), err)

	// Проверяем что существует
	exists, err := suite.repo.ExistsPr(suite.ctx, "pr-003")
	assert.NoError(suite.T(), err)
	assert.True(suite.T(), exists)

	// Проверяем что несуществующий PR не существует
	exists, err = suite.repo.ExistsPr(suite.ctx, "nonexistent_pr")
	assert.NoError(suite.T(), err)
	assert.False(suite.T(), exists)
}

func (suite *PRRepositoryTestSuite) TestMerge_Success() {
	// Создаем OPEN PR
	pr := &domain.PullRequest{
		ID:       "pr-004",
		Name:     "PR to Merge",
		AuthorID: "backend_author",
		Status:   "OPEN",
	}
	err := suite.repo.CreateWithReviewers(suite.ctx, pr, []string{"backend_reviewer1"})
	assert.NoError(suite.T(), err)

	// Мержим PR
	mergedPR, err := suite.repo.Merge(suite.ctx, "pr-004")
	assert.NoError(suite.T(), err)
	assert.Equal(suite.T(), "pr-004", mergedPR.ID)
	assert.Equal(suite.T(), "MERGED", mergedPR.Status)
	assert.NotNil(suite.T(), mergedPR.MergedAt)

	// Проверяем что статус изменился в БД
	dbPR, err := suite.repo.GetByID(suite.ctx, "pr-004")
	assert.NoError(suite.T(), err)
	assert.Equal(suite.T(), "MERGED", dbPR.Status)
	assert.NotNil(suite.T(), dbPR.MergedAt)
}

func (suite *PRRepositoryTestSuite) TestMerge_AlreadyMerged() {
	// Создаем и мержим PR
	pr := &domain.PullRequest{
		ID:       "pr-005",
		Name:     "Already Merged PR",
		AuthorID: "backend_author",
		Status:   "OPEN",
	}
	err := suite.repo.CreateWithReviewers(suite.ctx, pr, []string{})
	assert.NoError(suite.T(), err)

	// Первый мерж
	mergedPR1, err := suite.repo.Merge(suite.ctx, "pr-005")
	assert.NoError(suite.T(), err)
	firstMergeTime := mergedPR1.MergedAt

	// Второй мерж (идемпотентный)
	time.Sleep(100 * time.Millisecond) // Чтобы время было разное
	mergedPR2, err := suite.repo.Merge(suite.ctx, "pr-005")
	assert.NoError(suite.T(), err)

	// Проверяем что merged_at не изменился при повторном мерже
	assert.Equal(suite.T(), firstMergeTime, mergedPR2.MergedAt)
}

func (suite *PRRepositoryTestSuite) TestReassignReviewer_Success() {
	// Создаем PR с ревьювером
	pr := &domain.PullRequest{
		ID:       "pr-006",
		Name:     "PR for Reassignment",
		AuthorID: "backend_author",
		Status:   "OPEN",
	}
	err := suite.repo.CreateWithReviewers(suite.ctx, pr, []string{"backend_reviewer1"})
	assert.NoError(suite.T(), err)

	// Переназначаем ревьювера
	err = suite.repo.ReassignReviewer(suite.ctx, "pr-006", "backend_reviewer1", "backend_reviewer2")
	assert.NoError(suite.T(), err)

	// Проверяем что ревьювер изменился
	reviewers, err := suite.repo.GetReviewers(suite.ctx, "pr-006")
	assert.NoError(suite.T(), err)
	assert.Equal(suite.T(), 1, len(reviewers))
	assert.Equal(suite.T(), "backend_reviewer2", reviewers[0])
	assert.NotContains(suite.T(), reviewers, "backend_reviewer1")
}

func (suite *PRRepositoryTestSuite) TestGetUserAssignedPRs() {
	// Создаем PR где пользователь ревьювер
	pr1 := &domain.PullRequest{
		ID:       "pr-007",
		Name:     "PR 1",
		AuthorID: "backend_author",
		Status:   "OPEN",
	}
	err := suite.repo.CreateWithReviewers(suite.ctx, pr1, []string{"backend_reviewer1"})
	assert.NoError(suite.T(), err)

	// Создаем второй PR где тот же пользователь ревьювер
	pr2 := &domain.PullRequest{
		ID:       "pr-008",
		Name:     "PR 2",
		AuthorID: "frontend_author",
		Status:   "MERGED",
	}
	err = suite.repo.CreateWithReviewers(suite.ctx, pr2, []string{"backend_reviewer1"})
	assert.NoError(suite.T(), err)

	// Получаем PR назначенные пользователю
	userPRs, err := suite.repo.GetUserAssignedPRs(suite.ctx, "backend_reviewer1")
	assert.NoError(suite.T(), err)
	assert.Equal(suite.T(), 2, len(userPRs))

	// Проверяем что оба PR вернулись
	prIDs := make([]string, len(userPRs))
	for i, pr := range userPRs {
		prIDs[i] = pr.ID
	}
	assert.Contains(suite.T(), prIDs, "pr-007")
	assert.Contains(suite.T(), prIDs, "pr-008")
}

func (suite *PRRepositoryTestSuite) TestGetUserAssignedPRs_Empty() {
	userPRs, err := suite.repo.GetUserAssignedPRs(suite.ctx, "user_without_prs")
	assert.NoError(suite.T(), err)
	assert.Empty(suite.T(), userPRs)
}

func (suite *PRRepositoryTestSuite) TestIsUserReviewer() {
	// Создаем PR с ревьювером
	pr := &domain.PullRequest{
		ID:       "pr-009",
		Name:     "Test PR",
		AuthorID: "backend_author",
		Status:   "OPEN",
	}
	err := suite.repo.CreateWithReviewers(suite.ctx, pr, []string{"backend_reviewer1"})
	assert.NoError(suite.T(), err)

	// Проверяем что пользователь является ревьювером
	isReviewer, err := suite.repo.IsUserReviewer(suite.ctx, "pr-009", "backend_reviewer1")
	assert.NoError(suite.T(), err)
	assert.True(suite.T(), isReviewer)

	// Проверяем что другой пользователь не является ревьювером
	isReviewer, err = suite.repo.IsUserReviewer(suite.ctx, "pr-009", "backend_reviewer2")
	assert.NoError(suite.T(), err)
	assert.False(suite.T(), isReviewer)
}

func (suite *PRRepositoryTestSuite) TestGetReviewers() {
	// Создаем PR с несколькими ревьюверами
	pr := &domain.PullRequest{
		ID:       "pr-010",
		Name:     "Test PR",
		AuthorID: "backend_author",
		Status:   "OPEN",
	}
	reviewerIDs := []string{"backend_reviewer1", "backend_reviewer2"}
	err := suite.repo.CreateWithReviewers(suite.ctx, pr, reviewerIDs)
	assert.NoError(suite.T(), err)

	// Получаем ревьюверов
	reviewers, err := suite.repo.GetReviewers(suite.ctx, "pr-010")
	assert.NoError(suite.T(), err)
	assert.Equal(suite.T(), 2, len(reviewers))
	assert.Contains(suite.T(), reviewers, "backend_reviewer1")
	assert.Contains(suite.T(), reviewers, "backend_reviewer2")
}

func TestPRRepositoryTestSuite(t *testing.T) {
	if os.Getenv("RUN_INTEGRATION_TESTS") != "1" {
		t.Skip("Skipping integration test. Set RUN_INTEGRATION_TESTS=1 to run.")
	}
	suite.Run(t, new(PRRepositoryTestSuite))
}
