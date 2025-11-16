package usecase

import (
	"context"

	"pr-reviewer-service/internal/domain"
)

// PRUseCase реализует бизнес-логику для работы с Pull Request'ами.
type PRUseCase struct {
	prRepo   domain.PRRepository
	userRepo domain.UserRepository
}

// NewPRUseCase создает новый экземпляр PRUseCase.
func NewPRUseCase(prRepo domain.PRRepository, userRepo domain.UserRepository) domain.PRUseCase {
	return &PRUseCase{
		prRepo:   prRepo,
		userRepo: userRepo,
	}
}

// CreatePR создает PR и автоматически назначает ревьюверов.
func (uc *PRUseCase) CreatePR(ctx context.Context, prID, prName, authorID string) (*domain.PullRequest, error) {
	// Валидация входных данных
	if prID == "" {
		return nil, domain.ErrInvalidPRID
	}
	if prName == "" {
		return nil, domain.ErrInvalidPRName
	}
	if authorID == "" {
		return nil, domain.ErrInvalidUserID
	}

	// 1. Проверяем, что автор существует
	author, err := uc.userRepo.GetByID(ctx, authorID)
	if err != nil {
		return nil, domain.ErrPRAuthorNotFound
	}

	// 2. Проверяем, что PR не существует
	exists, err := uc.prRepo.ExistsPr(ctx, prID)
	if err != nil {
		return nil, err
	}
	if exists {
		return nil, domain.ErrPRAlreadyExists
	}

	// 3. Находим активных пользователей в команде автора (исключая самого автора)
	candidates, err := uc.userRepo.GetActiveUsersByTeam(ctx, author.TeamName, authorID)
	if err != nil {
		return nil, err
	}

	// 4. Проверяем наличие кандидатов
	if len(candidates) == 0 {
		return nil, domain.ErrNoReviewerCandidate
	}

	// 6. Создаем PR с ревьюверами
	pr := &domain.PullRequest{
		ID:       prID,
		Name:     prName,
		AuthorID: authorID,
		Status:   "OPEN",
	}

	//Преобразуем в []string
	reviewerIDs := make([]string, len(candidates))
	for i, u := range candidates {
		reviewerIDs[i] = u.ID
	}

	err = uc.prRepo.CreateWithReviewers(ctx, pr, reviewerIDs)
	if err != nil {
		return nil, err
	}

	pr.AssignedReviewers = reviewerIDs
	return pr, nil
}

// MergePR помечает PR как MERGED.
func (uc *PRUseCase) MergePR(ctx context.Context, prID string) (*domain.PullRequest, error) {
	// 1. Проверяем, что PR существует
	exists, err := uc.prRepo.ExistsPr(ctx, prID)
	if err != nil {
		return nil, err
	}
	if !exists {
		return nil, domain.ErrPRNotFound
	}

	// 2. Выполняем мердж (идемпотентная операция)
	return uc.prRepo.Merge(ctx, prID)
}

// ReassignReviewer заменяет ревьювера на случайного из той же команды.
func (uc *PRUseCase) ReassignReviewer(ctx context.Context, prID, oldReviewerID string) (*domain.PullRequest, string, error) {
	// 1. Получаем PR и проверяем существование
	pr, err := uc.prRepo.GetByID(ctx, prID)
	if err != nil {
		return nil, "", domain.ErrPRNotFound
	}

	// 2. Нельзя менять ревьюверов у MERGED PR
	if pr.Status == "MERGED" {
		return nil, "", domain.ErrPRAlreadyMerged
	}

	// 3. Проверяем что старый ревьювер назначен на PR
	isAssigned, err := uc.prRepo.IsUserReviewer(ctx, prID, oldReviewerID)
	if err != nil {
		return nil, "", err
	}
	if !isAssigned {
		return nil, "", domain.ErrReviewerNotAssigned
	}

	// 4. Находим команду старого ревьювера
	oldReviewer, err := uc.userRepo.GetByID(ctx, oldReviewerID)
	if err != nil {
		return nil, "", domain.ErrUserNotFound
	}

	// 5. Находим случайную замену из той же команды (исключая автора PR)
	candidates, err := uc.userRepo.GetActiveUsersByTeam(ctx, oldReviewer.TeamName, pr.AuthorID)
	if err != nil {
		return nil, "", err
	}

	// 6. Проверяем наличие кандидатов для замены
	if len(candidates) == 0 {
		return nil, "", domain.ErrNoReviewerCandidate
	}

	// 7. Выбираем первого случайного кандидата
	newReviewer := candidates[0]

	// 8. Выполняем замену
	err = uc.prRepo.ReassignReviewer(ctx, prID, oldReviewerID, newReviewer.ID)
	if err != nil {
		return nil, "", err
	}

	// 9. Получаем обновленный PR
	updatedPR, err := uc.prRepo.GetByID(ctx, prID)
	if err != nil {
		return nil, "", err
	}

	return updatedPR, newReviewer.ID, nil
}
