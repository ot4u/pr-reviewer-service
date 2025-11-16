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

type UserRepositoryTestSuite struct {
	suite.Suite
	db      *sql.DB
	queries *database.Queries
	repo    domain.UserRepository
	ctx     context.Context
}

func (suite *UserRepositoryTestSuite) SetupSuite() {
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
	suite.repo = repository.NewUserRepository(suite.db, suite.queries)

	suite.cleanDatabase()
	suite.setupTestData()
}

func (suite *UserRepositoryTestSuite) TearDownTest() {
	suite.cleanDatabase()
	suite.setupTestData()
}

func (suite *UserRepositoryTestSuite) TearDownSuite() {
	if suite.db != nil {
		suite.db.Close()
	}
}

func (suite *UserRepositoryTestSuite) cleanDatabase() {
	tables := []string{"reviewers", "pull_requests", "users", "teams"}
	for _, table := range tables {
		_, err := suite.db.ExecContext(suite.ctx, fmt.Sprintf("DELETE FROM %s", table))
		if err != nil {
			log.Printf("Failed to clean table %s: %v", table, err)
		}
	}
}

func (suite *UserRepositoryTestSuite) setupTestData() {
	// Создаем тестовые команды и пользователей
	teams := []string{"backend", "frontend", "mobile"}

	for _, teamName := range teams {
		// Создаем команду
		_, err := suite.queries.CreateTeam(suite.ctx, teamName)
		if err != nil {
			log.Printf("Failed to create team %s: %v", teamName, err)
		}

		// Создаем пользователей для команды
		users := []struct {
			id       string
			username string
			isActive bool
		}{
			{id: fmt.Sprintf("%s_user1", teamName), username: fmt.Sprintf("%s_Alice", teamName), isActive: true},
			{id: fmt.Sprintf("%s_user2", teamName), username: fmt.Sprintf("%s_Bob", teamName), isActive: true},
			{id: fmt.Sprintf("%s_user3", teamName), username: fmt.Sprintf("%s_Charlie", teamName), isActive: false},
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

func (suite *UserRepositoryTestSuite) TestGetByID_Success() {
	user, err := suite.repo.GetByID(suite.ctx, "backend_user1")

	assert.NoError(suite.T(), err)
	assert.NotNil(suite.T(), user)
	assert.Equal(suite.T(), "backend_user1", user.ID)
	assert.Equal(suite.T(), "backend_Alice", user.Username)
	assert.Equal(suite.T(), "backend", user.TeamName)
	assert.True(suite.T(), user.IsActive)
}

func (suite *UserRepositoryTestSuite) TestGetByID_NotFound() {
	user, err := suite.repo.GetByID(suite.ctx, "nonexistent_user")

	assert.Error(suite.T(), err)
	assert.Nil(suite.T(), user)
	assert.Equal(suite.T(), domain.ErrUserNotFound, err)
}

func (suite *UserRepositoryTestSuite) TestGetActiveUsersByTeam_Success() {
	// Получаем активных пользователей backend команды (исключая указанного пользователя)
	users, err := suite.repo.GetActiveUsersByTeam(suite.ctx, "backend", "backend_user1")

	assert.NoError(suite.T(), err)
	assert.Equal(suite.T(), 1, len(users)) // Должен вернуть только backend_user2 (активный)
	assert.Equal(suite.T(), "backend_user2", users[0].ID)
	assert.Equal(suite.T(), "backend_Bob", users[0].Username)
	assert.True(suite.T(), users[0].IsActive)
}

func (suite *UserRepositoryTestSuite) TestGetActiveUsersByTeam_ExcludeAuthor() {
	// Проверяем что автор исключается из результатов
	users, err := suite.repo.GetActiveUsersByTeam(suite.ctx, "backend", "backend_user2")

	assert.NoError(suite.T(), err)
	assert.Equal(suite.T(), 1, len(users)) // Должен вернуть только backend_user1
	assert.Equal(suite.T(), "backend_user1", users[0].ID)
	assert.NotEqual(suite.T(), "backend_user2", users[0].ID) // backend_user2 исключен
}

func (suite *UserRepositoryTestSuite) TestGetActiveUsersByTeam_OnlyActive() {
	// Проверяем что возвращаются только активные пользователи
	users, err := suite.repo.GetActiveUsersByTeam(suite.ctx, "backend", "nonexistent")

	assert.NoError(suite.T(), err)
	// Должны вернуться 2 активных пользователя (user1 и user2), user3 неактивный
	assert.Equal(suite.T(), 2, len(users))

	for _, user := range users {
		assert.True(suite.T(), user.IsActive)
		assert.NotEqual(suite.T(), "backend_user3", user.ID) // user3 неактивный
	}
}

func (suite *UserRepositoryTestSuite) TestGetActiveUsersByTeam_EmptyResult() {
	// Получаем активных пользователей несуществующей команды
	users, err := suite.repo.GetActiveUsersByTeam(suite.ctx, "nonexistent_team", "some_user")

	assert.NoError(suite.T(), err)
	assert.Empty(suite.T(), users)
}

func (suite *UserRepositoryTestSuite) TestGetActiveUsersByTeam_LimitTwo() {
	// Создаем дополнительных активных пользователей для теста лимита
	extraUsers := []struct {
		id       string
		username string
	}{
		{id: "backend_user4", username: "backend_David"},
		{id: "backend_user5", username: "backend_Eve"},
	}

	for _, user := range extraUsers {
		_, err := suite.queries.UpsertUser(suite.ctx, database.UpsertUserParams{
			UserID:   user.id,
			Username: user.username,
			TeamName: "backend",
			IsActive: true,
		})
		assert.NoError(suite.T(), err)
	}

	// Получаем активных пользователей - должен вернуть максимум 2 случайных
	users, err := suite.repo.GetActiveUsersByTeam(suite.ctx, "backend", "nonexistent")

	assert.NoError(suite.T(), err)
	assert.LessOrEqual(suite.T(), len(users), 2) // Максимум 2 пользователя
}

func (suite *UserRepositoryTestSuite) TestUpdateActiveStatus_Activate() {
	// Деактивируем пользователя
	user, err := suite.repo.UpdateActiveStatus(suite.ctx, "backend_user1", false)

	assert.NoError(suite.T(), err)
	assert.NotNil(suite.T(), user)
	assert.Equal(suite.T(), "backend_user1", user.ID)
	assert.False(suite.T(), user.IsActive)

	// Проверяем что изменения сохранились в БД
	updatedUser, err := suite.repo.GetByID(suite.ctx, "backend_user1")
	assert.NoError(suite.T(), err)
	assert.False(suite.T(), updatedUser.IsActive)
}

func (suite *UserRepositoryTestSuite) TestUpdateActiveStatus_Deactivate() {
	// Активируем неактивного пользователя
	user, err := suite.repo.UpdateActiveStatus(suite.ctx, "backend_user3", true)

	assert.NoError(suite.T(), err)
	assert.NotNil(suite.T(), user)
	assert.Equal(suite.T(), "backend_user3", user.ID)
	assert.True(suite.T(), user.IsActive)

	// Проверяем что изменения сохранились
	updatedUser, err := suite.repo.GetByID(suite.ctx, "backend_user3")
	assert.NoError(suite.T(), err)
	assert.True(suite.T(), updatedUser.IsActive)
}

func (suite *UserRepositoryTestSuite) TestGetUserTeam_Success() {
	teamName, err := suite.repo.GetUserTeam(suite.ctx, "backend_user1")

	assert.NoError(suite.T(), err)
	assert.Equal(suite.T(), "backend", teamName)
}

func (suite *UserRepositoryTestSuite) TestGetUserTeam_UserNotFound() {
	teamName, err := suite.repo.GetUserTeam(suite.ctx, "nonexistent_user")

	assert.Error(suite.T(), err)
	assert.Equal(suite.T(), "", teamName)
	assert.Equal(suite.T(), domain.ErrUserNotFound, err)
}

func TestUserRepositoryTestSuite(t *testing.T) {
	if os.Getenv("RUN_INTEGRATION_TESTS") != "1" {
		t.Skip("Skipping integration test. Set RUN_INTEGRATION_TESTS=1 to run.")
	}
	suite.Run(t, new(UserRepositoryTestSuite))
}
