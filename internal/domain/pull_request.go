package domain

import (
	"context"
	"time"
)

// PullRequest представляет сущность пул-реквеста в системе.
type PullRequest struct {
	ID                string
	Name              string
	AuthorID          string
	Status            string
	AssignedReviewers []string
	MergedAt          *time.Time
}

// PRRepository определяет контракт для работы с хранилищем пул-реквестов.
type PRRepository interface {
	CreateWithReviewers(ctx context.Context, pr *PullRequest, reviewerIDs []string) error
	GetByID(ctx context.Context, prID string) (*PullRequest, error)
	Merge(ctx context.Context, prID string) (*PullRequest, error)
	ReassignReviewer(ctx context.Context, prID, oldReviewerID, newReviewerID string) error
	GetUserAssignedPRs(ctx context.Context, userID string) ([]*PullRequest, error)
	IsUserReviewer(ctx context.Context, prID, userID string) (bool, error)
	ExistsPr(ctx context.Context, prID string) (bool, error)
	GetReviewers(ctx context.Context, prID string) ([]string, error)
}
