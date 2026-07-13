package repo

import (
	"context"

	"devsignal/internal/domain"

	"github.com/google/uuid"
)

// ContactFilter narrows contact lists. Q matches name, company, role, email,
// and notes case-insensitively.
type ContactFilter struct {
	Q            string
	Relationship string
	Page
}

// ContactRepo persists professional contacts.
type ContactRepo interface {
	Create(ctx context.Context, c *domain.Contact) error
	Get(ctx context.Context, userID, id uuid.UUID) (*domain.Contact, error)
	List(ctx context.Context, userID uuid.UUID, f ContactFilter) ([]domain.Contact, error)
	Update(ctx context.Context, c *domain.Contact) error
	Delete(ctx context.Context, userID, id uuid.UUID) error
}
