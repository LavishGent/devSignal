// Package search implements cross-module search. It reuses each module
// repository's text filtering (one interface, many implementations), so the
// same code path serves postgres in production and memory stores in tests.
package search

import (
	"context"
	"fmt"
	"time"

	"devsignal/internal/repo"

	"github.com/google/uuid"
)

// Item is one search hit.
type Item struct {
	ID        string    `json:"id"`
	Title     string    `json:"title"`
	Subtitle  string    `json:"subtitle"`
	UpdatedAt time.Time `json:"updatedAt"`
}

// Group is all hits within one module.
type Group struct {
	Module string `json:"module"` // frontend route segment
	Label  string `json:"label"`
	Items  []Item `json:"items"`
}

// perModuleLimit keeps the global result set scannable.
const perModuleLimit = 10

// Service fans a query out across every module repository.
type Service struct {
	Applications   repo.ApplicationRepo
	Contacts       repo.ContactRepo
	Interviews     repo.InterviewRepo
	Skills         repo.SkillRepo
	StudySessions  repo.StudySessionRepo
	Certifications repo.CertificationRepo
	Projects       repo.ProjectRepo
	Goals          repo.GoalRepo
}

// Search returns grouped, multi-row matches across all modules. Groups with
// no hits are omitted.
func (s *Service) Search(ctx context.Context, userID uuid.UUID, q string) ([]Group, error) {
	page := repo.Page{Limit: perModuleLimit}
	var groups []Group

	add := func(module, label string, items []Item, err error) error {
		if err != nil {
			return fmt.Errorf("searching %s: %w", module, err)
		}
		if len(items) > 0 {
			groups = append(groups, Group{Module: module, Label: label, Items: items})
		}
		return nil
	}

	apps, err := s.Applications.List(ctx, userID, repo.ApplicationFilter{Q: q, Page: page})
	items := make([]Item, 0, len(apps))
	for _, a := range apps {
		items = append(items, Item{ID: a.ID.String(), Title: a.Company + " — " + a.Position, Subtitle: "Status: " + a.Status, UpdatedAt: a.UpdatedAt})
	}
	if err := add("applications", "Applications", items, err); err != nil {
		return nil, err
	}

	contacts, err := s.Contacts.List(ctx, userID, repo.ContactFilter{Q: q, Page: page})
	items = make([]Item, 0, len(contacts))
	for _, c := range contacts {
		sub := c.Relationship
		if c.Company != "" {
			sub += " at " + c.Company
		}
		items = append(items, Item{ID: c.ID.String(), Title: c.Name, Subtitle: sub, UpdatedAt: c.UpdatedAt})
	}
	if err := add("contacts", "Contacts", items, err); err != nil {
		return nil, err
	}

	ivs, err := s.Interviews.List(ctx, userID, repo.InterviewFilter{Q: q, Page: page})
	items = make([]Item, 0, len(ivs))
	for _, iv := range ivs {
		items = append(items, Item{ID: iv.ID.String(), Title: iv.Stage + " interview", Subtitle: iv.ScheduledAt.Format("Jan 2, 2006 3:04 PM"), UpdatedAt: iv.UpdatedAt})
	}
	if err := add("interviews", "Interviews", items, err); err != nil {
		return nil, err
	}

	skills, err := s.Skills.List(ctx, userID, repo.SkillFilter{Q: q, Page: page})
	items = make([]Item, 0, len(skills))
	for _, sk := range skills {
		items = append(items, Item{ID: sk.ID.String(), Title: sk.Name, Subtitle: fmt.Sprintf("%s · %d/5", sk.Category, sk.Proficiency), UpdatedAt: sk.UpdatedAt})
	}
	if err := add("skills", "Skills", items, err); err != nil {
		return nil, err
	}

	sessions, err := s.StudySessions.List(ctx, userID, repo.StudySessionFilter{Q: q, Page: page})
	items = make([]Item, 0, len(sessions))
	for _, ss := range sessions {
		items = append(items, Item{ID: ss.ID.String(), Title: ss.Topic, Subtitle: fmt.Sprintf("%d min on %s", ss.Minutes, ss.StudiedOn), UpdatedAt: ss.UpdatedAt})
	}
	if err := add("study-sessions", "Study sessions", items, err); err != nil {
		return nil, err
	}

	certs, err := s.Certifications.List(ctx, userID, repo.CertificationFilter{Q: q, Page: page})
	items = make([]Item, 0, len(certs))
	for _, c := range certs {
		items = append(items, Item{ID: c.ID.String(), Title: c.Name, Subtitle: c.Issuer + " · " + c.Status, UpdatedAt: c.UpdatedAt})
	}
	if err := add("certifications", "Certifications", items, err); err != nil {
		return nil, err
	}

	projects, err := s.Projects.List(ctx, userID, repo.ProjectFilter{Q: q, Page: page})
	items = make([]Item, 0, len(projects))
	for _, p := range projects {
		items = append(items, Item{ID: p.ID.String(), Title: p.Name, Subtitle: "Status: " + p.Status, UpdatedAt: p.UpdatedAt})
	}
	if err := add("projects", "Projects", items, err); err != nil {
		return nil, err
	}

	goals, err := s.Goals.List(ctx, userID, repo.GoalFilter{Q: q, Page: page})
	items = make([]Item, 0, len(goals))
	for _, g := range goals {
		items = append(items, Item{ID: g.ID.String(), Title: g.Title, Subtitle: g.Category + " · " + g.Status, UpdatedAt: g.UpdatedAt})
	}
	if err := add("goals", "Goals", items, err); err != nil {
		return nil, err
	}

	return groups, nil
}
