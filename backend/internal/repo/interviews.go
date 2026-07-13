package repo

import (
	"context"

	"devsignal/internal/domain"

	"github.com/google/uuid"
)

// InterviewFilter narrows interview lists. Q matches interviewer, location,
// prep notes, and feedback case-insensitively. ApplicationID of uuid.Nil
// matches all applications.
type InterviewFilter struct {
	Q             string
	Stage         string
	Outcome       string
	ApplicationID uuid.UUID
	Page
}

// InterviewRepo persists interview events.
type InterviewRepo interface {
	Create(ctx context.Context, i *domain.Interview) error
	Get(ctx context.Context, userID, id uuid.UUID) (*domain.Interview, error)
	List(ctx context.Context, userID uuid.UUID, f InterviewFilter) ([]domain.Interview, error)
	Update(ctx context.Context, i *domain.Interview) error
	Delete(ctx context.Context, userID, id uuid.UUID) error
}
