package repo

import (
	"context"

	"devsignal/internal/domain"

	"github.com/google/uuid"
)

// SkillFilter narrows skill lists. Q matches name and notes
// case-insensitively.
type SkillFilter struct {
	Q        string
	Category string
	Page
}

// SkillRepo persists technical skills.
type SkillRepo interface {
	Create(ctx context.Context, s *domain.Skill) error
	Get(ctx context.Context, userID, id uuid.UUID) (*domain.Skill, error)
	List(ctx context.Context, userID uuid.UUID, f SkillFilter) ([]domain.Skill, error)
	Update(ctx context.Context, s *domain.Skill) error
	Delete(ctx context.Context, userID, id uuid.UUID) error
}
