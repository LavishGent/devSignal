// Package domain holds DevSignal's entities and their validation rules.
//
// Object-oriented structure:
//   - Encapsulation: types like Date keep state unexported behind methods.
//   - Inheritance: every career record embeds BaseRecord, sharing identity,
//     ownership, and timestamp behavior.
//   - Polymorphism: every entity implements the Validator interface, so
//     handlers validate any record through one contract.
package domain

import (
	"time"

	"devsignal/internal/validation"

	"github.com/google/uuid"
)

// Validator is implemented by every entity that can validate itself.
type Validator interface {
	Validate() validation.Errors
}

// NewID mints a record identifier. Centralized so entities that do not embed
// BaseRecord (like User) mint ids the same way.
func NewID() uuid.UUID { return uuid.New() }

// Compile-time proof that every entity satisfies Validator; adding an entity
// without validation rules fails the build here.
var _ = []Validator{
	(*User)(nil), (*Application)(nil), (*Contact)(nil), (*Interview)(nil),
	(*Skill)(nil), (*StudySession)(nil), (*Certification)(nil),
	(*Project)(nil), (*Goal)(nil),
}

// BaseRecord carries the identity, ownership, and audit fields shared by all
// career records. Entities embed it (struct embedding is Go's inheritance
// mechanism), so shared behavior lives in exactly one place.
type BaseRecord struct {
	ID        uuid.UUID `json:"id"`
	UserID    uuid.UUID `json:"userId"`
	CreatedAt time.Time `json:"createdAt"`
	UpdatedAt time.Time `json:"updatedAt"`
}

// StampNew assigns a fresh id and owner and sets both audit timestamps.
// Called exactly once, when a record is created.
func (b *BaseRecord) StampNew(userID uuid.UUID, now time.Time) {
	b.ID = uuid.New()
	b.UserID = userID
	b.CreatedAt = now.UTC()
	b.UpdatedAt = now.UTC()
}

// Touch refreshes the modification timestamp. Called on every update.
func (b *BaseRecord) Touch(now time.Time) {
	b.UpdatedAt = now.UTC()
}

// IsOwnedBy reports whether the record belongs to the given user. Handlers
// use this as a second ownership check on top of user-scoped queries.
func (b *BaseRecord) IsOwnedBy(userID uuid.UUID) bool {
	return b.UserID != uuid.Nil && b.UserID == userID
}
