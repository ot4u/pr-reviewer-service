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

type TeamHandlerTestSuite struct {
	suite.Suite
	db      *sql.DB
	queries *database.Queries
	echo    *echo.Echo
	handler *handler.TeamHandler
}

func (suite *TeamHandlerTestSuite) SetupSuite() {
	// Настройка БД
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

	// Очищаем БД
	suite.cleanDatabase()

	// Настройка Echo и handler'ов
	suite.echo = echo.New()
	logger := logrus.New()

	// Инициализация репозиториев и use cases
	teamRepo := repository.NewTeamRepository(suite.db, suite.queries)
	userRepo := repository.NewUserRepository(suite.db, suite.queries)
	prRepo := repository.NewPRRepository(suite.db, suite.queries)

	teamUC := usecase.NewTeamUseCase(teamRepo, userRepo, prRepo)
	suite.handler = handler.NewTeamHandler(teamUC, logger)
}

func (suite *TeamHandlerTestSuite) TearDownTest() {
	suite.cleanDatabase()
}

func (suite *TeamHandlerTestSuite) TearDownSuite() {
	if suite.db != nil {
		suite.db.Close()
	}
}

func (suite *TeamHandlerTestSuite) cleanDatabase() {
	tables := []string{"reviewers", "pull_requests", "users", "teams"}
	for _, table := range tables {
		_, err := suite.db.ExecContext(context.Background(), fmt.Sprintf("DELETE FROM %s", table))
		if err != nil {
			log.Printf("Failed to clean table %s: %v", table, err)
		}
	}
}

func (suite *TeamHandlerTestSuite) TestPostTeamAdd_Success() {
	// Подготовка тестовых данных
	teamRequest := api.Team{
		TeamName: "integration-team",
		Members: []api.TeamMember{
			{UserId: "user1", Username: "User One", IsActive: true},
			{UserId: "user2", Username: "User Two", IsActive: true},
		},
	}

	requestBody, _ := json.Marshal(teamRequest)
	req := httptest.NewRequest(http.MethodPost, "/team/add", bytes.NewReader(requestBody))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	c := suite.echo.NewContext(req, rec)

	// Вызов handler'а
	err := suite.handler.PostTeamAdd(c)

	// Проверки
	assert.NoError(suite.T(), err)
	assert.Equal(suite.T(), http.StatusCreated, rec.Code)

	var response map[string]interface{}
	json.Unmarshal(rec.Body.Bytes(), &response)

	assert.Contains(suite.T(), response, "team")
	team := response["team"].(map[string]interface{})
	assert.Equal(suite.T(), "integration-team", team["team_name"])
}

func (suite *TeamHandlerTestSuite) TestPostTeamAdd_TeamExists() {
	// Сначала создаем команду
	teamRequest := api.Team{
		TeamName: "duplicate-team",
		Members: []api.TeamMember{
			{UserId: "user1", Username: "User One", IsActive: true},
		},
	}

	requestBody, _ := json.Marshal(teamRequest)
	req := httptest.NewRequest(http.MethodPost, "/team/add", bytes.NewReader(requestBody))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	c := suite.echo.NewContext(req, rec)
	suite.handler.PostTeamAdd(c) // Первое создание

	// Пытаемся создать again
	req2 := httptest.NewRequest(http.MethodPost, "/team/add", bytes.NewReader(requestBody))
	req2.Header.Set("Content-Type", "application/json")
	rec2 := httptest.NewRecorder()

	c2 := suite.echo.NewContext(req2, rec2)
	err := suite.handler.PostTeamAdd(c2)

	// Проверки
	assert.NoError(suite.T(), err)
	assert.Equal(suite.T(), http.StatusConflict, rec2.Code)

	var response map[string]interface{}
	json.Unmarshal(rec2.Body.Bytes(), &response)

	assert.Contains(suite.T(), response, "error")
	errorObj := response["error"].(map[string]interface{})
	assert.Equal(suite.T(), "TEAM_EXISTS", errorObj["code"])
}

func (suite *TeamHandlerTestSuite) TestGetTeamGet_Success() {
	// Сначала создаем команду
	teamRequest := api.Team{
		TeamName: "test-get-team",
		Members: []api.TeamMember{
			{UserId: "user1", Username: "User One", IsActive: true},
			{UserId: "user2", Username: "User Two", IsActive: false},
		},
	}

	requestBody, _ := json.Marshal(teamRequest)
	req := httptest.NewRequest(http.MethodPost, "/team/add", bytes.NewReader(requestBody))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	c := suite.echo.NewContext(req, rec)
	suite.handler.PostTeamAdd(c)

	// Теперь получаем команду
	reqGet := httptest.NewRequest(http.MethodGet, "/team/get?team_name=test-get-team", nil)
	recGet := httptest.NewRecorder()

	cGet := suite.echo.NewContext(reqGet, recGet)

	err := suite.handler.GetTeamGet(cGet, api.GetTeamGetParams{TeamName: "test-get-team"})

	// Проверки
	assert.NoError(suite.T(), err)
	assert.Equal(suite.T(), http.StatusOK, recGet.Code)

	var response api.Team
	json.Unmarshal(recGet.Body.Bytes(), &response)

	assert.Equal(suite.T(), "test-get-team", response.TeamName)
	assert.Equal(suite.T(), 2, len(response.Members))
}

func (suite *TeamHandlerTestSuite) TestGetTeamGet_NotFound() {
	req := httptest.NewRequest(http.MethodGet, "/team/get?team_name=nonexistent", nil)
	rec := httptest.NewRecorder()

	c := suite.echo.NewContext(req, rec)

	err := suite.handler.GetTeamGet(c, api.GetTeamGetParams{TeamName: "nonexistent"})

	// Проверки
	assert.NoError(suite.T(), err)
	assert.Equal(suite.T(), http.StatusNotFound, rec.Code)

	var response map[string]interface{}
	json.Unmarshal(rec.Body.Bytes(), &response)

	assert.Contains(suite.T(), response, "error")
	errorObj := response["error"].(map[string]interface{})
	assert.Equal(suite.T(), "NOT_FOUND", errorObj["code"])
}

func TestTeamHandlerTestSuite(t *testing.T) {
	if os.Getenv("RUN_INTEGRATION_TESTS") != "1" {
		t.Skip("Skipping integration test. Set RUN_INTEGRATION_TESTS=1 to run.")
	}
	suite.Run(t, new(TeamHandlerTestSuite))
}
