package memory

import (
	"context"
	"strings"

	"devsignal/internal/domain"
	"devsignal/internal/repo"

	"github.com/google/uuid"
)

// ApplicationRepo is the in-memory repo.ApplicationRepo.
type ApplicationRepo struct {
	*store[domain.Application]
}

// NewApplicationRepo returns an empty in-memory application store.
func NewApplicationRepo() *ApplicationRepo {
	return &ApplicationRepo{store: newStore(func(a *domain.Application) *domain.BaseRecord { return &a.BaseRecord })}
}

// List filters the user's applications by status and free text.
func (r *ApplicationRepo) List(_ context.Context, userID uuid.UUID, f repo.ApplicationFilter) ([]domain.Application, error) {
	q := strings.ToLower(strings.TrimSpace(f.Q))
	var out []domain.Application
	for _, a := range r.all(userID) {
		if f.Status != "" && a.Status != f.Status {
			continue
		}
		if q != "" && !containsFold(q, a.Company, a.Position, a.Location, a.Notes) {
			continue
		}
		out = append(out, a)
	}
	return paginate(out, f.Page), nil
}

// containsFold reports whether any field contains q case-insensitively.
func containsFold(q string, fields ...string) bool {
	for _, f := range fields {
		if strings.Contains(strings.ToLower(f), q) {
			return true
		}
	}
	return false
}
