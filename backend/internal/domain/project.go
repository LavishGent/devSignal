package domain

import "devsignal/internal/validation"

// Project statuses model a portfolio project's lifecycle.
var ProjectStatuses = []string{"idea", "active", "paused", "completed", "archived"}

// Project is one tracked personal or portfolio project.
type Project struct {
	BaseRecord
	Name           string `json:"name"`
	Description    string `json:"description"`
	Status         string `json:"status"`
	TechStack      string `json:"techStack"`
	RepoURL        string `json:"repoUrl"`
	DemoURL        string `json:"demoUrl"`
	StartedOn      Date   `json:"startedOn"`
	LastActivityOn Date   `json:"lastActivityOn"`
	Notes          string `json:"notes"`
}

// Validate implements Validator.
func (p *Project) Validate() validation.Errors {
	e := validation.New()
	validation.Required(e, "name", p.Name)
	validation.MaxLen(e, "name", p.Name, 200)
	validation.MaxLen(e, "description", p.Description, 5000)
	validation.Required(e, "status", p.Status)
	validation.OneOf(e, "status", p.Status, ProjectStatuses...)
	validation.MaxLen(e, "techStack", p.TechStack, 500)
	validation.WebURL(e, "repoUrl", p.RepoURL)
	validation.MaxLen(e, "repoUrl", p.RepoURL, 2000)
	validation.WebURL(e, "demoUrl", p.DemoURL)
	validation.MaxLen(e, "demoUrl", p.DemoURL, 2000)
	validation.DateOrder(e, "lastActivityOn", p.StartedOn.Time(), p.LastActivityOn.Time(), "startedOn")
	validation.MaxLen(e, "notes", p.Notes, 5000)
	return e
}
