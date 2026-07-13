package repo

import (
	"context"

	"devsignal/internal/domain"

	"github.com/google/uuid"
)

// ApplicationFilter narrows application lists. Q matches company, position,
// location, and notes case-insensitively.
type ApplicationFilter struct {
	Q      string
	Status string
	Page
}

// ApplicationRepo persists job applications.
type ApplicationRepo interface {
	Create(ctx context.Context, a *domain.Application) error
	Get(ctx context.Context, userID, id uuid.UUID) (*domain.Application, error)
	List(ctx context.Context, userID uuid.UUID, f ApplicationFilter) ([]domain.Application, error)
	Update(ctx context.Context, a *domain.Application) error
	Delete(ctx context.Context, userID, id uuid.UUID) error
}
