package domain

import "context"

// User представляет сущность пользователя в системе.
type User struct {
	ID       string
	Username string
	TeamName string
	IsActive bool
}

// UserRepository определяет контракт для работы с хранилищем пользователей.
type UserRepository interface {
	GetByID(ctx context.Context, userID string) (*User, error)
	GetActiveUsersByTeam(ctx context.Context, teamName string, excludeUserID string) ([]*User, error)
	UpdateActiveStatus(ctx context.Context, userID string, isActive bool) (*User, error)
	GetUserTeam(ctx context.Context, userID string) (string, error)
}
