package domain

import (
	"time"

	"github.com/google/uuid"
)

// Session is a server-side login session. Only a SHA-256 hash of the bearer
// token is stored, so a leaked database cannot be replayed as live sessions.
type Session struct {
	ID        uuid.UUID
	UserID    uuid.UUID
	TokenHash string
	ExpiresAt time.Time
	CreatedAt time.Time
}

// Expired reports whether the session is past its expiry.
func (s *Session) Expired(now time.Time) bool {
	return !now.Before(s.ExpiresAt)
}
