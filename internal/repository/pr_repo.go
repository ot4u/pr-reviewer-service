package repository

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"pr-reviewer-service/internal/database"
	"pr-reviewer-service/internal/domain"
)

// PRRepository реализует взаимодействие с данными Pull Request'ов в PostgreSQL.
type PRRepository struct {
	db      *sql.DB
	queries *database.Queries
}

// NewPRRepository создает новый экземпляр PRRepository.
func NewPRRepository(db *sql.DB, queries *database.Queries) domain.PRRepository {
	return &PRRepository{
		db:      db,
		queries: queries,
	}
}

// CreateWithReviewers создает PR и назначает до 2 ревьюверов.
func (r *PRRepository) CreateWithReviewers(ctx context.Context, pr *domain.PullRequest, reviewerIDs []string) error {
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

	// 1. Создаем PR
	_, err = txQueries.CreatePullRequest(ctx, database.CreatePullRequestParams{
		PullRequestID:   pr.ID,
		PullRequestName: pr.Name,
		AuthorID:        pr.AuthorID,
	})
	if err != nil {
		return fmt.Errorf("failed to create PR: %w", err)
	}

	// 2. Назначаем ревьюверов
	for i := 0; i < len(reviewerIDs); i++ {
		reviewerID := reviewerIDs[i]
		err = txQueries.AssignReviewer(ctx, database.AssignReviewerParams{
			PullRequestID: pr.ID,
			UserID:        reviewerID,
		})
		if err != nil {
			return fmt.Errorf("failed to assign reviewer %s: %w", reviewerID, err)
		}
	}

	// 5. Коммитим транзакцию
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	return nil
}

// GetByID возвращает PR по ID.
func (r *PRRepository) GetByID(ctx context.Context, prID string) (*domain.PullRequest, error) {
	dbPR, err := r.queries.GetPullRequestByID(ctx, prID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, domain.ErrPRNotFound
		}
		return nil, fmt.Errorf("failed to get PR: %w", err)
	}

	reviewers, err := r.GetReviewers(ctx, prID)
	if err != nil {
		return nil, err
	}

	// Конвертируем NullTime → *time.Time
	var mergedAt *time.Time
	if dbPR.MergedAt.Valid {
		mergedAt = &dbPR.MergedAt.Time
	}

	return &domain.PullRequest{
		ID:                dbPR.PullRequestID,
		Name:              dbPR.PullRequestName,
		AuthorID:          dbPR.AuthorID,
		Status:            dbPR.Status,
		MergedAt:          mergedAt,
		AssignedReviewers: reviewers,
	}, nil
}

// Merge изменяет статус PR на MERGED.
func (r *PRRepository) Merge(ctx context.Context, prID string) (*domain.PullRequest, error) {
	dbPR, err := r.queries.MergePullRequest(ctx, prID)
	if err != nil {
		return nil, fmt.Errorf("failed to merge PR: %w", err)
	}

	reviewers, err := r.GetReviewers(ctx, prID)
	if err != nil {
		return nil, err
	}

	// Конвертируем NullTime → *time.Time
	var mergedAt *time.Time
	if dbPR.MergedAt.Valid {
		mergedAt = &dbPR.MergedAt.Time
	}

	return &domain.PullRequest{
		ID:                dbPR.PullRequestID,
		Name:              dbPR.PullRequestName,
		AuthorID:          dbPR.AuthorID,
		Status:            dbPR.Status,
		MergedAt:          mergedAt,
		AssignedReviewers: reviewers,
	}, nil
}

// ReassignReviewer заменяет ревьювера на нового из той же команды.
func (r *PRRepository) ReassignReviewer(ctx context.Context, prID, oldReviewerID, newReviewerID string) error {
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

	// 1. Удаляем старого ревьювера
	err = txQueries.RemoveReviewer(ctx, database.RemoveReviewerParams{
		PullRequestID: prID,
		UserID:        oldReviewerID,
	})
	if err != nil {
		return fmt.Errorf("failed to remove reviewer: %w", err)
	}

	// 2. Добавляем нового ревьювера
	err = txQueries.AssignReviewer(ctx, database.AssignReviewerParams{
		PullRequestID: prID,
		UserID:        newReviewerID,
	})
	if err != nil {
		return fmt.Errorf("failed to assign new reviewer: %w", err)
	}

	// 5. Коммитим транзакцию
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	return nil
}

// GetUserAssignedPRs возвращает PR, где пользователь назначен ревьювером.
func (r *PRRepository) GetUserAssignedPRs(ctx context.Context, userID string) ([]*domain.PullRequest, error) {
	dbPRs, err := r.queries.GetUserAssignedPRs(ctx, userID)
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		return nil, fmt.Errorf("failed to get user assigned PRs: %w", err)
	}

	prs := make([]*domain.PullRequest, 0, len(dbPRs))
	for _, dbPR := range dbPRs {
		prs = append(prs, &domain.PullRequest{
			ID:       dbPR.PullRequestID,
			Name:     dbPR.PullRequestName,
			AuthorID: dbPR.AuthorID,
			Status:   dbPR.Status,
		})
	}

	return prs, nil
}

// prExists проверяет существование PR.
func (r *PRRepository) ExistsPr(ctx context.Context, prID string) (bool, error) {
	count, err := r.queries.PRExists(ctx, prID)
	if err != nil {
		return false, fmt.Errorf("failed to check pr exists: %w", err)
	}
	return count > 0, nil
}

// getReviewers возвращает список ревьюверов PR.
func (r *PRRepository) GetReviewers(ctx context.Context, prID string) ([]string, error) {
	dbReviewers, err := r.queries.GetPRReviewers(ctx, prID)
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		return nil, fmt.Errorf("failed to get reviewers: %w", err)
	}

	reviewers := make([]string, 0, len(dbReviewers))
	reviewers = append(reviewers, dbReviewers...)

	return reviewers, nil
}

// IsUserReviewer проверяет, является ли пользователь ревьювером PR.
func (r *PRRepository) IsUserReviewer(ctx context.Context, prID, userID string) (bool, error) {
	count, err := r.queries.IsUserReviewer(ctx, database.IsUserReviewerParams{
		PullRequestID: prID,
		UserID:        userID,
	})
	if err != nil {
		return false, fmt.Errorf("failed to check reviewer assignment: %w", err)
	}
	return count > 0, nil
}
