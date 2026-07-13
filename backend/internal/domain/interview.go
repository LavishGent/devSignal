package domain

import (
	"time"

	"devsignal/internal/validation"

	"github.com/google/uuid"
)

// Interview stages model the steps of a hiring loop.
var InterviewStages = []string{
	"phone_screen", "technical", "system_design", "behavioral", "onsite", "final",
}

// InterviewOutcomes describe how an interview resolved.
var InterviewOutcomes = []string{"pending", "passed", "failed", "cancelled"}

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
	validation.OneOf(e, "stage", i.Stage, InterviewStages...)
	if i.ScheduledAt.IsZero() {
		e.Add("scheduledAt", "is required")
	}
	validation.IntRange(e, "durationMinutes", i.DurationMinutes, 0, 1440)
	validation.MaxLen(e, "interviewer", i.Interviewer, 200)
	validation.MaxLen(e, "location", i.Location, 500)
	validation.Required(e, "outcome", i.Outcome)
	validation.OneOf(e, "outcome", i.Outcome, InterviewOutcomes...)
	validation.MaxLen(e, "prepNotes", i.PrepNotes, 5000)
	validation.MaxLen(e, "feedback", i.Feedback, 5000)
	return e
}
