package memory

import (
	"context"
	"strings"

	"devsignal/internal/domain"
	"devsignal/internal/repo"

	"github.com/google/uuid"
)

// InterviewRepo is the in-memory repo.InterviewRepo.
type InterviewRepo struct {
	*store[domain.Interview]
}

// NewInterviewRepo returns an empty in-memory interview store.
func NewInterviewRepo() *InterviewRepo {
	return &InterviewRepo{store: newStore(func(i *domain.Interview) *domain.BaseRecord { return &i.BaseRecord })}
}

// List filters the user's interviews by stage, outcome, application, and
// free text.
func (r *InterviewRepo) List(_ context.Context, userID uuid.UUID, f repo.InterviewFilter) ([]domain.Interview, error) {
	q := strings.ToLower(strings.TrimSpace(f.Q))
	var out []domain.Interview
	for _, i := range r.all(userID) {
		if f.Stage != "" && i.Stage != f.Stage {
			continue
		}
		if f.Outcome != "" && i.Outcome != f.Outcome {
			continue
		}
		if f.ApplicationID != uuid.Nil && i.ApplicationID != f.ApplicationID {
			continue
		}
		if q != "" && !containsFold(q, i.Interviewer, i.Location, i.PrepNotes, i.Feedback) {
			continue
		}
		out = append(out, i)
	}
	return paginate(out, f.Page), nil
}
