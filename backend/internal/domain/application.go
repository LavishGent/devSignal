package domain

import "devsignal/internal/validation"

// Application statuses model the job-search pipeline.
var ApplicationStatuses = []string{
	"wishlist", "applied", "phone_screen", "interviewing",
	"offer", "accepted", "rejected", "withdrawn",
}

// WorkModes describe where a role is performed.
var WorkModes = []string{"remote", "hybrid", "onsite"}

// Application is one tracked job application.
type Application struct {
	BaseRecord
	Company   string `json:"company"`
	Position  string `json:"position"`
	Status    string `json:"status"`
	AppliedOn Date   `json:"appliedOn"`
	Location  string `json:"location"`
	WorkMode  string `json:"workMode"`
	URL       string `json:"url"`
	SalaryMin *int   `json:"salaryMin"`
	SalaryMax *int   `json:"salaryMax"`
	Notes     string `json:"notes"`
}

// Validate implements Validator.
func (a *Application) Validate() validation.Errors {
	e := validation.New()
	validation.Required(e, "company", a.Company)
	validation.MaxLen(e, "company", a.Company, 200)
	validation.Required(e, "position", a.Position)
	validation.MaxLen(e, "position", a.Position, 200)
	validation.Required(e, "status", a.Status)
	validation.OneOf(e, "status", a.Status, ApplicationStatuses...)
	validation.OneOf(e, "workMode", a.WorkMode, WorkModes...)
	validation.MaxLen(e, "location", a.Location, 200)
	validation.WebURL(e, "url", a.URL)
	validation.MaxLen(e, "url", a.URL, 2000)
	if a.SalaryMin != nil {
		validation.MinInt(e, "salaryMin", *a.SalaryMin, 0)
	}
	if a.SalaryMax != nil {
		validation.MinInt(e, "salaryMax", *a.SalaryMax, 0)
	}
	if a.SalaryMin != nil && a.SalaryMax != nil && *a.SalaryMax < *a.SalaryMin {
		e.Add("salaryMax", "must not be less than salaryMin")
	}
	validation.MaxLen(e, "notes", a.Notes, 5000)
	return e
}
