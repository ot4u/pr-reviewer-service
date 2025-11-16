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

type UserHandlerTestSuite struct {
	suite.Suite
	db      *sql.DB
	queries *database.Queries
	echo    *echo.Echo
	handler *handler.UserHandler
}

func (suite *UserHandlerTestSuite) SetupSuite() {
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

	userRepo := repository.NewUserRepository(suite.db, suite.queries)
	prRepo := repository.NewPRRepository(suite.db, suite.queries)
	userUC := usecase.NewUserUseCase(userRepo, prRepo)
	suite.handler = handler.NewUserHandler(userUC, logger)
}

func (suite *UserHandlerTestSuite) TearDownTest() {
	suite.cleanDatabase()
	suite.setupTestData()
}

func (suite *UserHandlerTestSuite) TearDownSuite() {
	if suite.db != nil {
		suite.db.Close()
	}
}

func (suite *UserHandlerTestSuite) cleanDatabase() {
	tables := []string{"reviewers", "pull_requests", "users", "teams"}
	for _, table := range tables {
		_, err := suite.db.ExecContext(context.Background(), fmt.Sprintf("DELETE FROM %s", table))
		if err != nil {
			log.Printf("Failed to clean table %s: %v", table, err)
		}
	}
}

func (suite *UserHandlerTestSuite) setupTestData() {
	// Создаем базовые данные
	suite.queries.CreateTeam(context.Background(), "test-team")
	suite.queries.UpsertUser(context.Background(), database.UpsertUserParams{
		UserID: "test-user", Username: "Test User", TeamName: "test-team", IsActive: true,
	})
}

func (suite *UserHandlerTestSuite) TestPostUsersSetIsActive_Success() {
	request := api.PostUsersSetIsActiveJSONBody{
		UserId:   "test-user",
		IsActive: false,
	}

	requestBody, _ := json.Marshal(request)
	req := httptest.NewRequest(http.MethodPost, "/users/setIsActive", bytes.NewReader(requestBody))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	c := suite.echo.NewContext(req, rec)

	err := suite.handler.PostUsersSetIsActive(c)

	assert.NoError(suite.T(), err)
	assert.Equal(suite.T(), http.StatusOK, rec.Code)
}

func (suite *UserHandlerTestSuite) TestGetUsersGetReview_Success() {
	req := httptest.NewRequest(http.MethodGet, "/users/getReview?user_id=test-user", nil)
	rec := httptest.NewRecorder()

	c := suite.echo.NewContext(req, rec)

	err := suite.handler.GetUsersGetReview(c, api.GetUsersGetReviewParams{UserId: "test-user"})

	assert.NoError(suite.T(), err)
	assert.Equal(suite.T(), http.StatusOK, rec.Code)
}

func TestUserHandlerTestSuite(t *testing.T) {
	if os.Getenv("RUN_INTEGRATION_TESTS") != "1" {
		t.Skip("Skipping integration test. Set RUN_INTEGRATION_TESTS=1 to run.")
	}
	suite.Run(t, new(UserHandlerTestSuite))
}
