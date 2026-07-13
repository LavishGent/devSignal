package memory

import (
	"context"
	"strings"

	"devsignal/internal/domain"
	"devsignal/internal/repo"

	"github.com/google/uuid"
)

// ProjectRepo is the in-memory repo.ProjectRepo.
type ProjectRepo struct {
	*store[domain.Project]
}

// NewProjectRepo returns an empty in-memory project store.
func NewProjectRepo() *ProjectRepo {
	return &ProjectRepo{store: newStore(func(p *domain.Project) *domain.BaseRecord { return &p.BaseRecord })}
}

// List filters the user's projects by status and free text.
func (r *ProjectRepo) List(_ context.Context, userID uuid.UUID, f repo.ProjectFilter) ([]domain.Project, error) {
	q := strings.ToLower(strings.TrimSpace(f.Q))
	var out []domain.Project
	for _, p := range r.all(userID) {
		if f.Status != "" && p.Status != f.Status {
			continue
		}
		if q != "" && !containsFold(q, p.Name, p.Description, p.TechStack, p.Notes) {
			continue
		}
		out = append(out, p)
	}
	return paginate(out, f.Page), nil
}
