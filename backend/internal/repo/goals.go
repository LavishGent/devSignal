package repo

import (
	"context"

	"devsignal/internal/domain"

	"github.com/google/uuid"
)

// GoalFilter narrows goal lists. Q matches title, description, and notes
// case-insensitively.
type GoalFilter struct {
	Q        string
	Status   string
	Category string
	Page
}

// GoalRepo persists career goals.
type GoalRepo interface {
	Create(ctx context.Context, g *domain.Goal) error
	Get(ctx context.Context, userID, id uuid.UUID) (*domain.Goal, error)
	List(ctx context.Context, userID uuid.UUID, f GoalFilter) ([]domain.Goal, error)
	Update(ctx context.Context, g *domain.Goal) error
	Delete(ctx context.Context, userID, id uuid.UUID) error
}
