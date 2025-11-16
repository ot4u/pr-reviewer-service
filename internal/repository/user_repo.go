package repository

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"pr-reviewer-service/internal/database"
	"pr-reviewer-service/internal/domain"
)

// UserRepository реализует взаимодействие с данными пользователей в PostgreSQL.
type UserRepository struct {
	db      *sql.DB
	queries *database.Queries
}

// NewUserRepository создает новый экземпляр UserRepository.
func NewUserRepository(db *sql.DB, queries *database.Queries) domain.UserRepository {
	return &UserRepository{
		db:      db,
		queries: queries,
	}
}

// GetByID возвращает пользователя по ID.
func (r *UserRepository) GetByID(ctx context.Context, userID string) (*domain.User, error) {
	dbUser, err := r.queries.GetUserByID(ctx, userID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, domain.ErrUserNotFound
		}
		return nil, fmt.Errorf("failed to get user: %w", err)
	}

	return &domain.User{
		ID:       dbUser.UserID,
		Username: dbUser.Username,
		TeamName: dbUser.TeamName,
		IsActive: dbUser.IsActive,
	}, nil
}

// GetActiveUsersByTeam возвращает активных пользователей команды для назначения ревьюверов.
func (r *UserRepository) GetActiveUsersByTeam(ctx context.Context, teamName string, excludeUserID string) ([]*domain.User, error) {
	candidates, err := r.queries.GetActiveUsersByTeam(ctx, database.GetActiveUsersByTeamParams{
		TeamName: teamName,
		UserID:   excludeUserID,
	})
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return []*domain.User{}, nil
		}
		return nil, fmt.Errorf("failed to get active users: %w", err)
	}

	users := make([]*domain.User, 0, len(candidates))
	for _, dbUser := range candidates {
		users = append(users, &domain.User{
			ID:       dbUser.UserID,
			Username: dbUser.Username,
			TeamName: dbUser.TeamName,
			IsActive: dbUser.IsActive,
		})
	}

	return users, nil
}

// UpdateActiveStatus обновляет статус активности пользователя.
func (r *UserRepository) UpdateActiveStatus(ctx context.Context, userID string, isActive bool) (*domain.User, error) {
	dbUser, err := r.queries.UpdateUserActiveStatus(ctx, database.UpdateUserActiveStatusParams{
		UserID:   userID,
		IsActive: isActive,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to update user status: %w", err)
	}

	return &domain.User{
		ID:       dbUser.UserID,
		Username: dbUser.Username,
		TeamName: dbUser.TeamName,
		IsActive: dbUser.IsActive,
	}, nil
}

// GetUserTeam возвращает название команды пользователя.
func (r *UserRepository) GetUserTeam(ctx context.Context, userID string) (string, error) {
	team, err := r.queries.GetUserTeam(ctx, userID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return "", domain.ErrUserNotFound
		}
		return "", fmt.Errorf("failed to get user team: %w", err)
	}

	return team, nil
}
