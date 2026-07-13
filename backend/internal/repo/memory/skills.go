package memory

import (
	"context"
	"strings"

	"devsignal/internal/domain"
	"devsignal/internal/repo"

	"github.com/google/uuid"
)

// SkillRepo is the in-memory repo.SkillRepo.
type SkillRepo struct {
	*store[domain.Skill]
}

// NewSkillRepo returns an empty in-memory skill store.
func NewSkillRepo() *SkillRepo {
	return &SkillRepo{store: newStore(func(s *domain.Skill) *domain.BaseRecord { return &s.BaseRecord })}
}

// List filters the user's skills by category and free text.
func (r *SkillRepo) List(_ context.Context, userID uuid.UUID, f repo.SkillFilter) ([]domain.Skill, error) {
	q := strings.ToLower(strings.TrimSpace(f.Q))
	var out []domain.Skill
	for _, s := range r.all(userID) {
		if f.Category != "" && s.Category != f.Category {
			continue
		}
		if q != "" && !containsFold(q, s.Name, s.Notes) {
			continue
		}
		out = append(out, s)
	}
	return paginate(out, f.Page), nil
}
