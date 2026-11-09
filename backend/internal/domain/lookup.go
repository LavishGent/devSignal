package domain

import "devsignal/internal/validation"

// Fixed-choice fields (statuses, categories, stages) are not hard-coded in
// the application. Their allowed values live in the lookup_values table so an
// operator can add an option with a database insert and every dropdown and
// validator picks it up without a code change. The Go defaults below are the
// initial seed and the data source for the in-memory repositories used in
// tests; at runtime the database is authoritative.

// Lookup kinds name each fixed-choice list. The string is the lookup_values
// "kind" column and the key of the /api/v1/options response.
const (
	KindApplicationStatus   = "application_status"
	KindWorkMode            = "work_mode"
	KindContactRelationship = "contact_relationship"
	KindInterviewStage      = "interview_stage"
	KindInterviewOutcome    = "interview_outcome"
	KindSkillCategory       = "skill_category"
	KindCertificationStatus = "certification_status"
	KindProjectStatus       = "project_status"
	KindGoalCategory        = "goal_category"
	KindGoalStatus          = "goal_status"
)

// LookupKinds lists every kind the application expects to find.
var LookupKinds = []string{
	KindApplicationStatus, KindWorkMode, KindContactRelationship,
	KindInterviewStage, KindInterviewOutcome, KindSkillCategory,
	KindCertificationStatus, KindProjectStatus, KindGoalCategory, KindGoalStatus,
}

// Option is one selectable value with its display label.
type Option struct {
	Value string `json:"value"`
	Label string `json:"label"`
}

// Lookups maps a kind to its ordered options.
type Lookups map[string][]Option

// Values returns the raw values of a kind in display order.
func (l Lookups) Values(kind string) []string {
	opts := l[kind]
	out := make([]string, len(opts))
	for i, o := range opts {
		out[i] = o.Value
	}
	return out
}

// EnumField ties an entity field to the lookup kind that governs it.
type EnumField struct {
	Field string // JSON field name, used as the validation error key
	Kind  string // lookup kind
	Value string // current value on the entity
}

// Enumerated is implemented by every entity with fixed-choice fields. The
// handler validates those fields against the lookup repository, so the set
// of allowed values is owned by the database rather than by this package.
type Enumerated interface {
	EnumFields() []EnumField
}

// ValidateEnums records a problem for every enumerated field whose value is
// not in the lookups. Empty values pass; whether a field may be empty is a
// structural rule and belongs in the entity's Validate method. A kind that
// is missing from the lookups rejects every non-empty value, so a
// misconfigured database fails closed.
func ValidateEnums(e validation.Errors, ent Enumerated, l Lookups) {
	for _, f := range ent.EnumFields() {
		if f.Value == "" {
			continue
		}
		validation.OneOf(e, f.Field, f.Value, l.Values(f.Kind)...)
	}
}

// DefaultLookups returns the seed option set. Migration 00011 inserts the
// same rows into lookup_values.
func DefaultLookups() Lookups {
	opts := func(values ...string) []Option {
		out := make([]Option, len(values))
		for i, v := range values {
			out[i] = Option{Value: v, Label: labelOf(v)}
		}
		return out
	}
	return Lookups{
		KindApplicationStatus:   opts("wishlist", "applied", "phone_screen", "interviewing", "offer", "accepted", "rejected", "withdrawn"),
		KindWorkMode:            opts("remote", "hybrid", "onsite"),
		KindContactRelationship: opts("recruiter", "hiring_manager", "engineer", "mentor", "peer", "other"),
		KindInterviewStage:      opts("phone_screen", "technical", "system_design", "behavioral", "onsite", "final"),
		KindInterviewOutcome:    opts("pending", "passed", "failed", "cancelled"),
		KindSkillCategory:       opts("language", "framework", "database", "cloud", "tooling", "system_design", "soft_skill", "other"),
		KindCertificationStatus: opts("planned", "in_progress", "earned", "expired"),
		KindProjectStatus:       opts("idea", "active", "paused", "completed", "archived"),
		KindGoalCategory:        opts("job_search", "skill", "certification", "project", "networking", "other"),
		KindGoalStatus:          opts("not_started", "in_progress", "completed", "abandoned"),
	}
}

// labelOf turns "phone_screen" into "Phone screen" for the seed labels.
func labelOf(value string) string {
	b := []byte(value)
	for i := range b {
		if b[i] == '_' {
			b[i] = ' '
		}
	}
	if len(b) > 0 && b[0] >= 'a' && b[0] <= 'z' {
		b[0] -= 'a' - 'A'
	}
	return string(b)
}
