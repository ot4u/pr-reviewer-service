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

type TeamRepositoryTestSuite struct {
	suite.Suite
	db      *sql.DB
	queries *database.Queries
	repo    domain.TeamRepository
	ctx     context.Context
}

func (suite *TeamRepositoryTestSuite) SetupSuite() {
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
}

func (suite *TeamRepositoryTestSuite) TearDownTest() {
	suite.cleanDatabase()
}

func (suite *TeamRepositoryTestSuite) TearDownSuite() {
	if suite.db != nil {
		suite.db.Close()
	}
}

func (suite *TeamRepositoryTestSuite) cleanDatabase() {
	tables := []string{"reviewers", "pull_requests", "users", "teams"}
	for _, table := range tables {
		_, err := suite.db.ExecContext(suite.ctx, fmt.Sprintf("DELETE FROM %s", table))
		if err != nil {
			log.Printf("Failed to clean table %s: %v", table, err)
		}
	}
}

func (suite *TeamRepositoryTestSuite) TestCreateTeam() {
	team := &domain.Team{
		Name: "backend",
		Members: []*domain.User{
			{
				ID:       "user1",
				Username: "Alice",
				TeamName: "backend",
				IsActive: true,
			},
			{
				ID:       "user2",
				Username: "Bob",
				TeamName: "backend",
				IsActive: true,
			},
		},
	}

	err := suite.repo.Create(suite.ctx, team)
	assert.NoError(suite.T(), err)

	exists, err := suite.repo.ExistsTeam(suite.ctx, "backend")
	assert.NoError(suite.T(), err)
	assert.True(suite.T(), exists)

	retrievedTeam, err := suite.repo.GetByName(suite.ctx, "backend")
	assert.NoError(suite.T(), err)
	assert.Equal(suite.T(), 2, len(retrievedTeam.Members))
	assert.Equal(suite.T(), "backend", retrievedTeam.Name)
}

func (suite *TeamRepositoryTestSuite) TestCreateTeam_AlreadyExists() {
	team1 := &domain.Team{
		Name: "backend",
		Members: []*domain.User{
			{ID: "user1", Username: "Alice", TeamName: "backend", IsActive: true},
		},
	}
	err := suite.repo.Create(suite.ctx, team1)
	assert.NoError(suite.T(), err)

	team2 := &domain.Team{
		Name: "backend",
		Members: []*domain.User{
			{ID: "user2", Username: "Bob", TeamName: "backend", IsActive: true},
		},
	}
	err = suite.repo.Create(suite.ctx, team2)
	assert.Error(suite.T(), err)
}

func (suite *TeamRepositoryTestSuite) TestGetTeam_NotFound() {
	team, err := suite.repo.GetByName(suite.ctx, "nonexistent")
	assert.NoError(suite.T(), err)
	assert.NotNil(suite.T(), team)
	assert.Equal(suite.T(), "nonexistent", team.Name)
	assert.Empty(suite.T(), team.Members)
}

func (suite *TeamRepositoryTestSuite) TestExistsTeam_False() {
	exists, err := suite.repo.ExistsTeam(suite.ctx, "nonexistent")
	assert.NoError(suite.T(), err)
	assert.False(suite.T(), exists)
}

func (suite *TeamRepositoryTestSuite) TestGetAllUsersByTeam() {
	team := &domain.Team{
		Name: "frontend",
		Members: []*domain.User{
			{ID: "user3", Username: "Charlie", TeamName: "frontend", IsActive: true},
			{ID: "user4", Username: "David", TeamName: "frontend", IsActive: false},
		},
	}
	err := suite.repo.Create(suite.ctx, team)
	assert.NoError(suite.T(), err)

	users, err := suite.repo.GetAllUsersByTeam(suite.ctx, "frontend")
	assert.NoError(suite.T(), err)
	assert.Equal(suite.T(), 2, len(users))

	assert.Equal(suite.T(), "user3", users[0].ID)
	assert.Equal(suite.T(), "Charlie", users[0].Username)
	assert.Equal(suite.T(), "frontend", users[0].TeamName)
	assert.True(suite.T(), users[0].IsActive)

	assert.Equal(suite.T(), "user4", users[1].ID)
	assert.Equal(suite.T(), "David", users[1].Username)
	assert.False(suite.T(), users[1].IsActive)
}

func (suite *TeamRepositoryTestSuite) TestGetAllUsersByTeam_Empty() {
	users, err := suite.repo.GetAllUsersByTeam(suite.ctx, "empty_team")
	assert.NoError(suite.T(), err)
	assert.Empty(suite.T(), users)
}

func (suite *TeamRepositoryTestSuite) TestCreateTeam_UpdateExistingUsers() {
	team1 := &domain.Team{
		Name: "mobile",
		Members: []*domain.User{
			{ID: "user5", Username: "Eve", TeamName: "mobile", IsActive: true},
		},
	}
	err := suite.repo.Create(suite.ctx, team1)
	assert.NoError(suite.T(), err)

	team2 := &domain.Team{
		Name: "web",
		Members: []*domain.User{
			{ID: "user5", Username: "Eve", TeamName: "web", IsActive: true},
		},
	}
	err = suite.repo.Create(suite.ctx, team2)
	assert.NoError(suite.T(), err)

	users, err := suite.repo.GetAllUsersByTeam(suite.ctx, "web")
	assert.NoError(suite.T(), err)
	assert.Equal(suite.T(), 1, len(users))
	assert.Equal(suite.T(), "user5", users[0].ID)
	assert.Equal(suite.T(), "Eve", users[0].Username)
	assert.Equal(suite.T(), "web", users[0].TeamName)
	assert.True(suite.T(), users[0].IsActive)
}

func (suite *TeamRepositoryTestSuite) TestCreateTeam_UpsertPreservesUserState() {
	// Создаем пользователя в первой команде (активный)
	team1 := &domain.Team{
		Name: "team_a",
		Members: []*domain.User{
			{ID: "user8", Username: "OriginalName", TeamName: "team_a", IsActive: true},
		},
	}
	err := suite.repo.Create(suite.ctx, team1)
	assert.NoError(suite.T(), err)

	// Перемещаем пользователя во вторую команду
	// Должны сохраниться: username и is_active (только team_name обновится)
	team2 := &domain.Team{
		Name: "team_b",
		Members: []*domain.User{
			{ID: "user8", Username: "NewName", TeamName: "team_b", IsActive: false}, // Эти значения игнорируются
		},
	}
	err = suite.repo.Create(suite.ctx, team2)
	assert.NoError(suite.T(), err)

	// Проверяем что сохранились оригинальные username и is_active
	users, err := suite.repo.GetAllUsersByTeam(suite.ctx, "team_b")
	assert.NoError(suite.T(), err)
	assert.Equal(suite.T(), 1, len(users))
	assert.Equal(suite.T(), "user8", users[0].ID)
	assert.Equal(suite.T(), "OriginalName", users[0].Username) // username сохранился
	assert.Equal(suite.T(), "team_b", users[0].TeamName)       // team_name обновился
	assert.True(suite.T(), users[0].IsActive)                  // is_active сохранился (не изменился на false)
}

func (suite *TeamRepositoryTestSuite) TestCreateTeam_TransactionRollbackOnError() {
	team1 := &domain.Team{
		Name: "duplicate_team",
		Members: []*domain.User{
			{ID: "user6", Username: "Frank", TeamName: "duplicate_team", IsActive: true},
		},
	}
	err := suite.repo.Create(suite.ctx, team1)
	assert.NoError(suite.T(), err)

	team2 := &domain.Team{
		Name: "duplicate_team",
		Members: []*domain.User{
			{ID: "user7", Username: "Grace", TeamName: "duplicate_team", IsActive: true},
		},
	}
	err = suite.repo.Create(suite.ctx, team2)
	assert.Error(suite.T(), err)

	users, err := suite.repo.GetAllUsersByTeam(suite.ctx, "duplicate_team")
	assert.NoError(suite.T(), err)
	assert.Equal(suite.T(), 1, len(users))
	assert.Equal(suite.T(), "user6", users[0].ID)
}

func (suite *TeamRepositoryTestSuite) TestCreateTeam_EmptyMembers() {
	team := &domain.Team{
		Name:    "empty_team",
		Members: []*domain.User{},
	}

	err := suite.repo.Create(suite.ctx, team)
	assert.NoError(suite.T(), err)

	exists, err := suite.repo.ExistsTeam(suite.ctx, "empty_team")
	assert.NoError(suite.T(), err)
	assert.True(suite.T(), exists)

	users, err := suite.repo.GetAllUsersByTeam(suite.ctx, "empty_team")
	assert.NoError(suite.T(), err)
	assert.Empty(suite.T(), users)
}

func TestTeamRepositoryTestSuite(t *testing.T) {
	if os.Getenv("RUN_INTEGRATION_TESTS") != "1" {
		t.Skip("Skipping integration test. Set RUN_INTEGRATION_TESTS=1 to run.")
	}
	suite.Run(t, new(TeamRepositoryTestSuite))
}
