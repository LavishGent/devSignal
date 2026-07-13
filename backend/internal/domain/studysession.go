package domain

import (
	"devsignal/internal/validation"

	"github.com/google/uuid"
)

// StudySession is one logged block of study time, optionally linked to a
// tracked skill.
type StudySession struct {
	BaseRecord
	SkillID   *uuid.UUID `json:"skillId"`
	Topic     string     `json:"topic"`
	Minutes   int        `json:"minutes"`
	StudiedOn Date       `json:"studiedOn"`
	Resource  string     `json:"resource"`
	Notes     string     `json:"notes"`
}

// Validate implements Validator.
func (s *StudySession) Validate() validation.Errors {
	e := validation.New()
	validation.Required(e, "topic", s.Topic)
	validation.MaxLen(e, "topic", s.Topic, 300)
	validation.IntRange(e, "minutes", s.Minutes, 1, 1440)
	if s.StudiedOn.IsZero() {
		e.Add("studiedOn", "is required")
	}
	validation.MaxLen(e, "resource", s.Resource, 2000)
	validation.MaxLen(e, "notes", s.Notes, 5000)
	return e
}
