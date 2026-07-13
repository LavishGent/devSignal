package repo

import (
	"context"

	"devsignal/internal/domain"

	"github.com/google/uuid"
)

// CertificationFilter narrows certification lists. Q matches name, issuer,
// and notes case-insensitively.
type CertificationFilter struct {
	Q      string
	Status string
	Page
}

// CertificationRepo persists professional certifications.
type CertificationRepo interface {
	Create(ctx context.Context, c *domain.Certification) error
	Get(ctx context.Context, userID, id uuid.UUID) (*domain.Certification, error)
	List(ctx context.Context, userID uuid.UUID, f CertificationFilter) ([]domain.Certification, error)
	Update(ctx context.Context, c *domain.Certification) error
	Delete(ctx context.Context, userID, id uuid.UUID) error
}
