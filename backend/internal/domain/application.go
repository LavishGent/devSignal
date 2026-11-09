package domain

import "devsignal/internal/validation"

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

// EnumFields implements Enumerated; the allowed values come from the lookup
// table at request time.
func (a *Application) EnumFields() []EnumField {
	return []EnumField{
		{Field: "status", Kind: KindApplicationStatus, Value: a.Status},
		{Field: "workMode", Kind: KindWorkMode, Value: a.WorkMode},
	}
}
