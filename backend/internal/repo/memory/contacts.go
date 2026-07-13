package memory

import (
	"context"
	"strings"

	"devsignal/internal/domain"
	"devsignal/internal/repo"

	"github.com/google/uuid"
)

// ContactRepo is the in-memory repo.ContactRepo.
type ContactRepo struct {
	*store[domain.Contact]
}

// NewContactRepo returns an empty in-memory contact store.
func NewContactRepo() *ContactRepo {
	return &ContactRepo{store: newStore(func(c *domain.Contact) *domain.BaseRecord { return &c.BaseRecord })}
}

// List filters the user's contacts by relationship and free text.
func (r *ContactRepo) List(_ context.Context, userID uuid.UUID, f repo.ContactFilter) ([]domain.Contact, error) {
	q := strings.ToLower(strings.TrimSpace(f.Q))
	var out []domain.Contact
	for _, c := range r.all(userID) {
		if f.Relationship != "" && c.Relationship != f.Relationship {
			continue
		}
		if q != "" && !containsFold(q, c.Name, c.Company, c.Role, c.Email, c.Notes) {
			continue
		}
		out = append(out, c)
	}
	return paginate(out, f.Page), nil
}
