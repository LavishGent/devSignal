package memory

import (
	"context"
	"strings"

	"devsignal/internal/domain"
	"devsignal/internal/repo"

	"github.com/google/uuid"
)

// CertificationRepo is the in-memory repo.CertificationRepo.
type CertificationRepo struct {
	*store[domain.Certification]
}

// NewCertificationRepo returns an empty in-memory certification store.
func NewCertificationRepo() *CertificationRepo {
	return &CertificationRepo{store: newStore(func(c *domain.Certification) *domain.BaseRecord { return &c.BaseRecord })}
}

// List filters the user's certifications by status and free text.
func (r *CertificationRepo) List(_ context.Context, userID uuid.UUID, f repo.CertificationFilter) ([]domain.Certification, error) {
	q := strings.ToLower(strings.TrimSpace(f.Q))
	var out []domain.Certification
	for _, c := range r.all(userID) {
		if f.Status != "" && c.Status != f.Status {
			continue
		}
		if q != "" && !containsFold(q, c.Name, c.Issuer, c.Notes) {
			continue
		}
		out = append(out, c)
	}
	return paginate(out, f.Page), nil
}
