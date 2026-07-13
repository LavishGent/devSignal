package memory

import (
	"context"
	"strings"

	"devsignal/internal/domain"
	"devsignal/internal/repo"

	"github.com/google/uuid"
)

// StudySessionRepo is the in-memory repo.StudySessionRepo.
type StudySessionRepo struct {
	*store[domain.StudySession]
}

// NewStudySessionRepo returns an empty in-memory study-session store.
func NewStudySessionRepo() *StudySessionRepo {
	return &StudySessionRepo{store: newStore(func(s *domain.StudySession) *domain.BaseRecord { return &s.BaseRecord })}
}

// List filters the user's study sessions by skill and free text.
func (r *StudySessionRepo) List(_ context.Context, userID uuid.UUID, f repo.StudySessionFilter) ([]domain.StudySession, error) {
	q := strings.ToLower(strings.TrimSpace(f.Q))
	var out []domain.StudySession
	for _, s := range r.all(userID) {
		if f.SkillID != nil && (s.SkillID == nil || *s.SkillID != *f.SkillID) {
			continue
		}
		if q != "" && !containsFold(q, s.Topic, s.Resource, s.Notes) {
			continue
		}
		out = append(out, s)
	}
	return paginate(out, f.Page), nil
}
