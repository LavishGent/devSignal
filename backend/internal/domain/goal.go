package domain

import "devsignal/internal/validation"

// GoalCategories classify what kind of career progress a goal drives.
var GoalCategories = []string{
	"job_search", "skill", "certification", "project", "networking", "other",
}

// GoalStatuses model a goal's lifecycle.
var GoalStatuses = []string{"not_started", "in_progress", "completed", "abandoned"}

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
	validation.OneOf(e, "category", g.Category, GoalCategories...)
	validation.Required(e, "status", g.Status)
	validation.OneOf(e, "status", g.Status, GoalStatuses...)
	validation.IntRange(e, "priority", g.Priority, 1, 3)
	if g.Status != "completed" && !g.CompletedOn.IsZero() {
		e.Add("completedOn", "may only be set when status is completed")
	}
	validation.MaxLen(e, "notes", g.Notes, 5000)
	return e
}
