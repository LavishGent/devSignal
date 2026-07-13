package domain

import "devsignal/internal/validation"

// SkillCategories group technical skills for filtering and review.
var SkillCategories = []string{
	"language", "framework", "database", "cloud",
	"tooling", "system_design", "soft_skill", "other",
}

// Skill is one tracked technical skill with proficiency progress.
type Skill struct {
	BaseRecord
	Name              string `json:"name"`
	Category          string `json:"category"`
	Proficiency       int    `json:"proficiency"`
	TargetProficiency int    `json:"targetProficiency"`
	LastPracticedOn   Date   `json:"lastPracticedOn"`
	Notes             string `json:"notes"`
}

// Validate implements Validator.
func (s *Skill) Validate() validation.Errors {
	e := validation.New()
	validation.Required(e, "name", s.Name)
	validation.MaxLen(e, "name", s.Name, 200)
	validation.Required(e, "category", s.Category)
	validation.OneOf(e, "category", s.Category, SkillCategories...)
	validation.IntRange(e, "proficiency", s.Proficiency, 1, 5)
	validation.IntRange(e, "targetProficiency", s.TargetProficiency, 1, 5)
	validation.MaxLen(e, "notes", s.Notes, 5000)
	return e
}
