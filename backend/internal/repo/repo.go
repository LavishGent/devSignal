// Package repo defines the persistence contracts for DevSignal. Handlers and
// services depend only on these interfaces (polymorphism): the postgres
// sub-package implements them for production and the memory sub-package
// implements them for fast unit tests.
package repo

import (
	"context"
	"errors"
	"time"

	"devsignal/internal/domain"

	"github.com/google/uuid"
)

// ErrNotFound is returned when a record does not exist or is not visible to
// the requesting user. Callers translate it to HTTP 404.
var ErrNotFound = errors.New("record not found")

// ErrConflict is returned when a uniqueness rule is violated (e.g., email
// already registered). Callers translate it to HTTP 409.
var ErrConflict = errors.New("record conflicts with an existing record")

// Page bounds list queries so responses stay small under growth.
type Page struct {
	Limit  int
	Offset int
}

// Clamp applies the default and maximum page size.
func (p Page) Clamp() Page {
	if p.Limit <= 0 {
		p.Limit = 50
	}
	if p.Limit > 200 {
		p.Limit = 200
	}
	if p.Offset < 0 {
		p.Offset = 0
	}
	return p
}

// UserRepo persists accounts.
type UserRepo interface {
	Create(ctx context.Context, u *domain.User) error
	GetByEmail(ctx context.Context, email string) (*domain.User, error)
	GetByID(ctx context.Context, id uuid.UUID) (*domain.User, error)
}

// SessionRepo persists login sessions keyed by token hash.
type SessionRepo interface {
	Create(ctx context.Context, s *domain.Session) error
	GetByTokenHash(ctx context.Context, tokenHash string) (*domain.Session, error)
	DeleteByTokenHash(ctx context.Context, tokenHash string) error
	DeleteExpired(ctx context.Context, now time.Time) error
}
