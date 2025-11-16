package handler_test

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"pr-reviewer-service/api"
	"pr-reviewer-service/internal/config"
	"pr-reviewer-service/internal/database"
	"pr-reviewer-service/internal/handler"
	"pr-reviewer-service/internal/repository"
	"pr-reviewer-service/internal/usecase"

	"github.com/labstack/echo/v4"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
)

type PRHandlerTestSuite struct {
	suite.Suite
	db      *sql.DB
	queries *database.Queries
	echo    *echo.Echo
	handler *handler.PRHandler
}

func (suite *PRHandlerTestSuite) SetupSuite() {
	cfg, _ := config.LoadConfig()
	dsn := fmt.Sprintf(
		"postgres://%s:%s@%s:%s/%s?sslmode=disable",
		cfg.DBUser, cfg.DBPassword, "localhost", "5433", "pr_reviewer_test",
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
	suite.cleanDatabase()
	suite.setupTestData()

	suite.echo = echo.New()
	logger := logrus.New()

	prRepo := repository.NewPRRepository(suite.db, suite.queries)
	userRepo := repository.NewUserRepository(suite.db, suite.queries)
	prUC := usecase.NewPRUseCase(prRepo, userRepo)
	suite.handler = handler.NewPRHandler(prUC, logger)
}

func (suite *PRHandlerTestSuite) TearDownTest() {
	suite.cleanDatabase()
	suite.setupTestData()
}

func (suite *PRHandlerTestSuite) TearDownSuite() {
	if suite.db != nil {
		suite.db.Close()
	}
}

func (suite *PRHandlerTestSuite) cleanDatabase() {
	tables := []string{"reviewers", "pull_requests", "users", "teams"}
	for _, table := range tables {
		_, err := suite.db.ExecContext(context.Background(), fmt.Sprintf("DELETE FROM %s", table))
		if err != nil {
			log.Printf("Failed to clean table %s: %v", table, err)
		}
	}
}

func (suite *PRHandlerTestSuite) setupTestData() {
	// Создаем базовые данные для PR тестов
	suite.queries.CreateTeam(context.Background(), "pr-team")
	suite.queries.UpsertUser(context.Background(), database.UpsertUserParams{
		UserID: "pr-author", Username: "PR Author", TeamName: "pr-team", IsActive: true,
	})
	suite.queries.UpsertUser(context.Background(), database.UpsertUserParams{
		UserID: "pr-reviewer", Username: "PR Reviewer", TeamName: "pr-team", IsActive: true,
	})
}

func (suite *PRHandlerTestSuite) TestPostPullRequestCreate_Success() {
	request := api.PostPullRequestCreateJSONBody{
		PullRequestId:   "test-pr",
		PullRequestName: "Test PR",
		AuthorId:        "pr-author",
	}

	requestBody, _ := json.Marshal(request)
	req := httptest.NewRequest(http.MethodPost, "/pullRequest/create", bytes.NewReader(requestBody))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	c := suite.echo.NewContext(req, rec)

	err := suite.handler.PostPullRequestCreate(c)

	assert.NoError(suite.T(), err)
	assert.Equal(suite.T(), http.StatusCreated, rec.Code)
}

func (suite *PRHandlerTestSuite) TestPostPullRequestMerge_Success() {
	// Сначала создаем PR
	suite.queries.CreatePullRequest(context.Background(), database.CreatePullRequestParams{
		PullRequestID: "merge-pr", PullRequestName: "Merge PR", AuthorID: "pr-author",
	})

	request := api.PostPullRequestMergeJSONBody{
		PullRequestId: "merge-pr",
	}

	requestBody, _ := json.Marshal(request)
	req := httptest.NewRequest(http.MethodPost, "/pullRequest/merge", bytes.NewReader(requestBody))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	c := suite.echo.NewContext(req, rec)

	err := suite.handler.PostPullRequestMerge(c)

	assert.NoError(suite.T(), err)
	assert.Equal(suite.T(), http.StatusOK, rec.Code)
}

func TestPRHandlerTestSuite(t *testing.T) {
	if os.Getenv("RUN_INTEGRATION_TESTS") != "1" {
		t.Skip("Skipping integration test. Set RUN_INTEGRATION_TESTS=1 to run.")
	}
	suite.Run(t, new(PRHandlerTestSuite))
}
