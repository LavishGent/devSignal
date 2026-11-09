package domain

import "devsignal/internal/validation"

// Goal is one tracked career goal.
type Goal struct {
	BaseRecord
	Title       string `json:"title"`
	Description string `json:"description"`
	Category    string `json:"category"`
	Status      string `json:"status"`
	Priority    int    `json:"priority"`
	TargetDate  Date   `json:"targetDate"`
	CompletedOn Date   `json:"completedOn"`
	Notes       string `json:"notes"`
}

// Validate implements Validator.
func (g *Goal) Validate() validation.Errors {
	e := validation.New()
	validation.Required(e, "title", g.Title)
	validation.MaxLen(e, "title", g.Title, 300)
	validation.MaxLen(e, "description", g.Description, 5000)
	validation.Required(e, "category", g.Category)
	validation.Required(e, "status", g.Status)
	validation.IntRange(e, "priority", g.Priority, 1, 3)
	if g.Status != "completed" && !g.CompletedOn.IsZero() {
		e.Add("completedOn", "may only be set when status is completed")
	}
	validation.MaxLen(e, "notes", g.Notes, 5000)
	return e
}

// EnumFields implements Enumerated; the allowed values come from the lookup
// table at request time.
func (g *Goal) EnumFields() []EnumField {
	return []EnumField{
		{Field: "category", Kind: KindGoalCategory, Value: g.Category},
		{Field: "status", Kind: KindGoalStatus, Value: g.Status},
	}
}
