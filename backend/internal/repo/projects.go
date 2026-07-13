package repo

import (
	"context"

	"devsignal/internal/domain"

	"github.com/google/uuid"
)

// ProjectFilter narrows project lists. Q matches name, description, tech
// stack, and notes case-insensitively.
type ProjectFilter struct {
	Q      string
	Status string
	Page
}

// ProjectRepo persists personal projects.
type ProjectRepo interface {
	Create(ctx context.Context, p *domain.Project) error
	Get(ctx context.Context, userID, id uuid.UUID) (*domain.Project, error)
	List(ctx context.Context, userID uuid.UUID, f ProjectFilter) ([]domain.Project, error)
	Update(ctx context.Context, p *domain.Project) error
	Delete(ctx context.Context, userID, id uuid.UUID) error
}
