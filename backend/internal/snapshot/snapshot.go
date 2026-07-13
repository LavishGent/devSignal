// Package snapshot loads one user's complete career data in a single pass.
// The recommendation engine, report builders, and dashboard all consume the
// same Snapshot, so their logic stays pure and unit-testable.
package snapshot

import (
	"context"
	"fmt"

	"devsignal/internal/domain"
	"devsignal/internal/repo"

	"github.com/google/uuid"
)

// Snapshot is a point-in-time copy of everything the user tracks.
type Snapshot struct {
	Applications   []domain.Application
	Contacts       []domain.Contact
	Interviews     []domain.Interview
	Skills         []domain.Skill
	StudySessions  []domain.StudySession
	Certifications []domain.Certification
	Projects       []domain.Project
	Goals          []domain.Goal
}

// ApplicationByID indexes applications for cross-record lookups.
func (s *Snapshot) ApplicationByID() map[uuid.UUID]domain.Application {
	m := make(map[uuid.UUID]domain.Application, len(s.Applications))
	for _, a := range s.Applications {
		m[a.ID] = a
	}
	return m
}

// SkillByID indexes skills for study-session lookups.
func (s *Snapshot) SkillByID() map[uuid.UUID]domain.Skill {
	m := make(map[uuid.UUID]domain.Skill, len(s.Skills))
	for _, sk := range s.Skills {
		m[sk.ID] = sk
	}
	return m
}

// maxRecords bounds each module load; it matches the repository page cap.
// Insights are computed over the user's most recently updated records.
var maxPage = repo.Page{Limit: 200}

// Loader assembles snapshots from the module repositories.
type Loader struct {
	Applications   repo.ApplicationRepo
	Contacts       repo.ContactRepo
	Interviews     repo.InterviewRepo
	Skills         repo.SkillRepo
	StudySessions  repo.StudySessionRepo
	Certifications repo.CertificationRepo
	Projects       repo.ProjectRepo
	Goals          repo.GoalRepo
}

// Load reads every module for the user.
func (l Loader) Load(ctx context.Context, userID uuid.UUID) (*Snapshot, error) {
	var (
		s   Snapshot
		err error
	)
	if s.Applications, err = l.Applications.List(ctx, userID, repo.ApplicationFilter{Page: maxPage}); err != nil {
		return nil, fmt.Errorf("loading applications: %w", err)
	}
	if s.Contacts, err = l.Contacts.List(ctx, userID, repo.ContactFilter{Page: maxPage}); err != nil {
		return nil, fmt.Errorf("loading contacts: %w", err)
	}
	if s.Interviews, err = l.Interviews.List(ctx, userID, repo.InterviewFilter{Page: maxPage}); err != nil {
		return nil, fmt.Errorf("loading interviews: %w", err)
	}
	if s.Skills, err = l.Skills.List(ctx, userID, repo.SkillFilter{Page: maxPage}); err != nil {
		return nil, fmt.Errorf("loading skills: %w", err)
	}
	if s.StudySessions, err = l.StudySessions.List(ctx, userID, repo.StudySessionFilter{Page: maxPage}); err != nil {
		return nil, fmt.Errorf("loading study sessions: %w", err)
	}
	if s.Certifications, err = l.Certifications.List(ctx, userID, repo.CertificationFilter{Page: maxPage}); err != nil {
		return nil, fmt.Errorf("loading certifications: %w", err)
	}
	if s.Projects, err = l.Projects.List(ctx, userID, repo.ProjectFilter{Page: maxPage}); err != nil {
		return nil, fmt.Errorf("loading projects: %w", err)
	}
	if s.Goals, err = l.Goals.List(ctx, userID, repo.GoalFilter{Page: maxPage}); err != nil {
		return nil, fmt.Errorf("loading goals: %w", err)
	}
	return &s, nil
}
