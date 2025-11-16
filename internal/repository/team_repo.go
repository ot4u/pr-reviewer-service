package repository

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"pr-reviewer-service/internal/database"
	"pr-reviewer-service/internal/domain"
)

// TeamRepository реализует взаимодействие с данными команд в PostgreSQL.
type TeamRepository struct {
	db      *sql.DB
	queries *database.Queries
}

// NewTeamRepository создает новый экземпляр TeamRepository.
func NewTeamRepository(db *sql.DB, queries *database.Queries) domain.TeamRepository {
	return &TeamRepository{
		db:      db,
		queries: queries,
	}
}

// Create создает команду и обновляет/создает пользователей.
func (r *TeamRepository) Create(ctx context.Context, team *domain.Team) error {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}

	defer func() {
		if err != nil {
			_ = tx.Rollback()
		}
	}()

	txQueries := r.queries.WithTx(tx)

	// 1. Создаем команду
	_, err = txQueries.CreateTeam(ctx, team.Name)
	if err != nil {
		return fmt.Errorf("failed to create team: %w", err)
	}

	// 2. Создаем/обновляем пользователей команды
	for _, member := range team.Members {
		_, err := txQueries.UpsertUser(ctx, database.UpsertUserParams{
			UserID:   member.ID,
			Username: member.Username,
			TeamName: team.Name,
			IsActive: member.IsActive,
		})
		if err != nil {
			return fmt.Errorf("failed to upsert user %s: %w", member.ID, err)
		}
	}

	// 3. Коммитим транзакцию
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	return nil
}

// GetByName возвращает команду по названию.
func (r *TeamRepository) GetByName(ctx context.Context, teamName string) (*domain.Team, error) {
	// Получаем всех пользователей команды
	users, err := r.GetAllUsersByTeam(ctx, teamName)
	if err != nil {
		return nil, fmt.Errorf("failed to get team members: %w", err)
	}

	return &domain.Team{
		Name:    teamName,
		Members: users,
	}, nil
}

// Exists проверяет существование команды.
func (r *TeamRepository) ExistsTeam(ctx context.Context, teamName string) (bool, error) {
	count, err := r.queries.TeamExists(ctx, teamName)
	if err != nil {
		return false, fmt.Errorf("failed to check team existence: %w", err)
	}
	return count > 0, nil
}

// GetAllUsersByTeam возвращает всех пользователей команды.
func (r *TeamRepository) GetAllUsersByTeam(ctx context.Context, teamName string) ([]*domain.User, error) {
	dbUsers, err := r.queries.GetAllUsersByTeam(ctx, teamName)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return []*domain.User{}, nil
		}
		return nil, fmt.Errorf("failed to get all team users: %w", err)
	}

	users := make([]*domain.User, 0, len(dbUsers))
	for _, dbUser := range dbUsers {
		users = append(users, &domain.User{
			ID:       dbUser.UserID,
			Username: dbUser.Username,
			TeamName: dbUser.TeamName,
			IsActive: dbUser.IsActive,
		})
	}

	return users, nil
}

// GetActiveUsersFromTeam возвращает активных пользователей команды
func (r *TeamRepository) GetActiveUsersFromTeam(ctx context.Context, teamName string) ([]*domain.User, error) {
	dbUsers, err := r.queries.GetActiveUsersFromTeam(ctx, teamName)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return []*domain.User{}, nil
		}
		return nil, fmt.Errorf("failed to get active users from team: %w", err)
	}

	users := make([]*domain.User, 0, len(dbUsers))
	for _, dbUser := range dbUsers {
		users = append(users, &domain.User{
			ID:       dbUser.UserID,
			Username: dbUser.Username,
			TeamName: dbUser.TeamName,
			IsActive: dbUser.IsActive,
		})
	}

	return users, nil
}

// GetOpenPRsWithTeamReviewers возвращает ID открытых PR с ревьюверами из указанной команды
func (r *TeamRepository) GetOpenPRsWithTeamReviewers(ctx context.Context, teamName string) ([]string, error) {
	prIDs, err := r.queries.GetOpenPRsWithTeamReviewers(ctx, teamName)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return []string{}, nil
		}
		return nil, fmt.Errorf("failed to get open PRs with team reviewers: %w", err)
	}

	return prIDs, nil
}

// GetPRReviewersFromTeam возвращает ревьюверов из указанной команды для конкретного PR
func (r *TeamRepository) GetPRReviewersFromTeam(ctx context.Context, prID, teamName string) ([]string, error) {
	reviewerIDs, err := r.queries.GetPRReviewersFromTeam(ctx, database.GetPRReviewersFromTeamParams{
		PullRequestID: prID,
		TeamName:      teamName,
	})
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return []string{}, nil
		}
		return nil, fmt.Errorf("failed to get PR reviewers from team: %w", err)
	}

	return reviewerIDs, nil
}

// GetAllTeams возвращает все команды
func (r *TeamRepository) GetAllTeams(ctx context.Context) ([]*domain.Team, error) {
	teamNames, err := r.queries.GetAllTeams(ctx)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return []*domain.Team{}, nil
		}
		return nil, fmt.Errorf("failed to get all teams: %w", err)
	}

	teams := make([]*domain.Team, 0, len(teamNames))
	for _, teamName := range teamNames {
		teams = append(teams, &domain.Team{
			Name: teamName,
		})
	}

	return teams, nil
}
