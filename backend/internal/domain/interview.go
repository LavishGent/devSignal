package domain

import (
	"time"

	"devsignal/internal/validation"

	"github.com/google/uuid"
)

// Interview is one interview event linked to an application.
type Interview struct {
	BaseRecord
	ApplicationID   uuid.UUID `json:"applicationId"`
	Stage           string    `json:"stage"`
	ScheduledAt     time.Time `json:"scheduledAt"`
	DurationMinutes int       `json:"durationMinutes"`
	Interviewer     string    `json:"interviewer"`
	Location        string    `json:"location"`
	Outcome         string    `json:"outcome"`
	PrepNotes       string    `json:"prepNotes"`
	Feedback        string    `json:"feedback"`
}

// Validate implements Validator.
func (i *Interview) Validate() validation.Errors {
	e := validation.New()
	if i.ApplicationID == uuid.Nil {
		e.Add("applicationId", "is required")
	}
	validation.Required(e, "stage", i.Stage)
	if i.ScheduledAt.IsZero() {
		e.Add("scheduledAt", "is required")
	}
	validation.IntRange(e, "durationMinutes", i.DurationMinutes, 0, 1440)
	validation.MaxLen(e, "interviewer", i.Interviewer, 200)
	validation.MaxLen(e, "location", i.Location, 500)
	validation.Required(e, "outcome", i.Outcome)
	validation.MaxLen(e, "prepNotes", i.PrepNotes, 5000)
	validation.MaxLen(e, "feedback", i.Feedback, 5000)
	return e
}

// EnumFields implements Enumerated; the allowed values come from the lookup
// table at request time.
func (i *Interview) EnumFields() []EnumField {
	return []EnumField{
		{Field: "stage", Kind: KindInterviewStage, Value: i.Stage},
		{Field: "outcome", Kind: KindInterviewOutcome, Value: i.Outcome},
	}
}
