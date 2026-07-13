package repo

import (
	"context"

	"devsignal/internal/domain"

	"github.com/google/uuid"
)

// StudySessionFilter narrows study-session lists. Q matches topic, resource,
// and notes case-insensitively. A nil SkillID matches all skills.
type StudySessionFilter struct {
	Q       string
	SkillID *uuid.UUID
	Page
}

// StudySessionRepo persists logged study time.
type StudySessionRepo interface {
	Create(ctx context.Context, s *domain.StudySession) error
	Get(ctx context.Context, userID, id uuid.UUID) (*domain.StudySession, error)
	List(ctx context.Context, userID uuid.UUID, f StudySessionFilter) ([]domain.StudySession, error)
	Update(ctx context.Context, s *domain.StudySession) error
	Delete(ctx context.Context, userID, id uuid.UUID) error
}
