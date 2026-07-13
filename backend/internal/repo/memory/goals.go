package memory

import (
	"context"
	"strings"

	"devsignal/internal/domain"
	"devsignal/internal/repo"

	"github.com/google/uuid"
)

// GoalRepo is the in-memory repo.GoalRepo.
type GoalRepo struct {
	*store[domain.Goal]
}

// NewGoalRepo returns an empty in-memory goal store.
func NewGoalRepo() *GoalRepo {
	return &GoalRepo{store: newStore(func(g *domain.Goal) *domain.BaseRecord { return &g.BaseRecord })}
}

// List filters the user's goals by status, category, and free text.
func (r *GoalRepo) List(_ context.Context, userID uuid.UUID, f repo.GoalFilter) ([]domain.Goal, error) {
	q := strings.ToLower(strings.TrimSpace(f.Q))
	var out []domain.Goal
	for _, g := range r.all(userID) {
		if f.Status != "" && g.Status != f.Status {
			continue
		}
		if f.Category != "" && g.Category != f.Category {
			continue
		}
		if q != "" && !containsFold(q, g.Title, g.Description, g.Notes) {
			continue
		}
		out = append(out, g)
	}
	return paginate(out, f.Page), nil
}
