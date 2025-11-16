package usecase

import (
	"context"

	"pr-reviewer-service/internal/domain"
)

// UserUseCase реализует бизнес-логику для работы с пользователями.
type UserUseCase struct {
	userRepo domain.UserRepository
	prRepo   domain.PRRepository
}

// NewUserUseCase создает новый экземпляр UserUseCase.
func NewUserUseCase(userRepo domain.UserRepository, prRepo domain.PRRepository) domain.UserUseCase {
	return &UserUseCase{
		userRepo: userRepo,
		prRepo:   prRepo,
	}
}

// SetUserActive устанавливает флаг активности пользователя.
func (uc *UserUseCase) SetUserActive(ctx context.Context, userID string, isActive bool) (*domain.User, error) {
	// Проверяем, что пользователь существует
	_, err := uc.userRepo.GetByID(ctx, userID)
	if err != nil {
		return nil, domain.ErrUserNotFound
	}

	return uc.userRepo.UpdateActiveStatus(ctx, userID, isActive)
}

// GetUserReviewPRs возвращает PR, где пользователь назначен ревьювером.
func (uc *UserUseCase) GetUserReviewPRs(ctx context.Context, userID string) ([]*domain.PullRequest, error) {
	// Проверяем, что пользователь существует
	_, err := uc.userRepo.GetByID(ctx, userID)
	if err != nil {
		return nil, domain.ErrUserNotFound
	}

	return uc.prRepo.GetUserAssignedPRs(ctx, userID)
}
