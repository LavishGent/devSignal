package auth

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"time"

	"devsignal/internal/domain"
	"devsignal/internal/repo"

	"github.com/google/uuid"
)

// SessionManager mints, verifies, and revokes login sessions. Its state is
// unexported (encapsulation): callers can only use the session lifecycle it
// exposes. Tokens are 256-bit random values handed to the browser in an
// HttpOnly cookie; only their SHA-256 hash is persisted.
type SessionManager struct {
	sessions repo.SessionRepo
	ttl      time.Duration
	now      func() time.Time // injectable clock for tests
}

// NewSessionManager wires a manager to a session store.
func NewSessionManager(sessions repo.SessionRepo, ttl time.Duration) *SessionManager {
	return &SessionManager{sessions: sessions, ttl: ttl, now: time.Now}
}

// WithClock overrides the clock; tests use it to control expiry.
func (m *SessionManager) WithClock(now func() time.Time) *SessionManager {
	m.now = now
	return m
}

// TTL returns the configured session lifetime (used for cookie Max-Age).
func (m *SessionManager) TTL() time.Duration { return m.ttl }

// Issue creates a session for the user and returns the bearer token exactly
// once. The token itself is never stored.
func (m *SessionManager) Issue(ctx context.Context, userID uuid.UUID) (string, error) {
	raw := make([]byte, 32)
	if _, err := rand.Read(raw); err != nil {
		return "", fmt.Errorf("generating session token: %w", err)
	}
	token := base64.RawURLEncoding.EncodeToString(raw)

	now := m.now()
	s := &domain.Session{
		ID:        uuid.New(),
		UserID:    userID,
		TokenHash: HashToken(token),
		ExpiresAt: now.Add(m.ttl),
		CreatedAt: now,
	}
	if err := m.sessions.Create(ctx, s); err != nil {
		return "", fmt.Errorf("storing session: %w", err)
	}
	return token, nil
}

// Verify resolves a bearer token to a live session, deleting it when expired.
func (m *SessionManager) Verify(ctx context.Context, token string) (*domain.Session, error) {
	if token == "" {
		return nil, repo.ErrNotFound
	}
	s, err := m.sessions.GetByTokenHash(ctx, HashToken(token))
	if err != nil {
		return nil, err
	}
	if s.Expired(m.now()) {
		_ = m.sessions.DeleteByTokenHash(ctx, s.TokenHash)
		return nil, repo.ErrNotFound
	}
	return s, nil
}

// Revoke deletes the session for a token (logout). Unknown tokens are a no-op.
func (m *SessionManager) Revoke(ctx context.Context, token string) error {
	return m.sessions.DeleteByTokenHash(ctx, HashToken(token))
}

// HashToken derives the storage key for a bearer token.
func HashToken(token string) string {
	sum := sha256.Sum256([]byte(token))
	return hex.EncodeToString(sum[:])
}
