// Package recommend turns a user's career data into prioritized, rule-based
// next steps.
//
// Every rule implements the Rule interface (polymorphism): the engine treats
// them uniformly and new rules are added without touching engine code. Rules
// share BaseRule via embedding (inheritance) and keep their thresholds
// unexported (encapsulation).
package recommend

import (
	"sort"
	"time"

	"devsignal/internal/snapshot"

	"github.com/google/uuid"
)

// Priorities order recommendations from urgent to nice-to-do.
const (
	PriorityHigh   = 1
	PriorityMedium = 2
	PriorityLow    = 3
)

// Recommendation is one actionable suggestion shown to the user.
type Recommendation struct {
	Rule     string `json:"rule"`
	Priority int    `json:"priority"` // 1 = high, 3 = low
	Title    string `json:"title"`
	Detail   string `json:"detail"`
	Module   string `json:"module"`             // frontend route segment, e.g. "contacts"
	RecordID string `json:"recordId,omitempty"` // subject record, when there is one
}

// Rule inspects a snapshot and produces zero or more recommendations.
type Rule interface {
	Name() string
	Evaluate(s *snapshot.Snapshot, now time.Time) []Recommendation
}

// BaseRule carries the identity shared by every rule implementation.
type BaseRule struct {
	name   string
	module string
}

// Name implements the Rule interface for every embedding rule.
func (b BaseRule) Name() string { return b.name }

// rec builds a Recommendation stamped with the rule's identity.
func (b BaseRule) rec(priority int, title, detail string, recordID uuid.UUID) Recommendation {
	r := Recommendation{
		Rule:     b.name,
		Priority: priority,
		Title:    title,
		Detail:   detail,
		Module:   b.module,
	}
	if recordID != uuid.Nil {
		r.RecordID = recordID.String()
	}
	return r
}

// Engine runs a fixed rule set over a snapshot.
type Engine struct {
	rules []Rule
}

// NewEngine assembles the production rule set.
func NewEngine() *Engine {
	return &Engine{rules: []Rule{
		ContactFollowUpRule{BaseRule{"contact_follow_up", "contacts"}},
		InterviewPrepRule{BaseRule{"interview_prep", "interviews"}},
		ApplicationFollowUpRule{BaseRule{"application_follow_up", "applications"}},
		StaleSkillRule{BaseRule{"stale_skill", "skills"}},
		OverdueGoalRule{BaseRule{"overdue_goal", "goals"}},
		InactiveProjectRule{BaseRule{"inactive_project", "projects"}},
		CertificationExpiryRule{BaseRule{"certification_expiry", "certifications"}},
		StudyGapRule{BaseRule{"study_gap", "study-sessions"}},
	}}
}

// Run evaluates every rule and returns recommendations ordered by priority,
// then module, then title, so output is stable for a given snapshot.
func (e *Engine) Run(s *snapshot.Snapshot, now time.Time) []Recommendation {
	out := []Recommendation{}
	for _, rule := range e.rules {
		out = append(out, rule.Evaluate(s, now)...)
	}
	sort.Slice(out, func(i, j int) bool {
		a, b := out[i], out[j]
		if a.Priority != b.Priority {
			return a.Priority < b.Priority
		}
		if a.Module != b.Module {
			return a.Module < b.Module
		}
		return a.Title < b.Title
	})
	return out
}
