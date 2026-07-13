package memory

import (
	"context"
	"sync"
	"time"

	"devsignal/internal/domain"
	"devsignal/internal/repo"
)

// SessionRepo is a thread-safe in-memory repo.SessionRepo.
type SessionRepo struct {
	mu      sync.RWMutex
	byToken map[string]domain.Session
}

// NewSessionRepo returns an empty in-memory session store.
func NewSessionRepo() *SessionRepo {
	return &SessionRepo{byToken: map[string]domain.Session{}}
}

// Create stores a session.
func (r *SessionRepo) Create(_ context.Context, s *domain.Session) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.byToken[s.TokenHash] = *s
	return nil
}

// GetByTokenHash retrieves a session by its token hash.
func (r *SessionRepo) GetByTokenHash(_ context.Context, tokenHash string) (*domain.Session, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	s, ok := r.byToken[tokenHash]
	if !ok {
		return nil, repo.ErrNotFound
	}
	return &s, nil
}

// DeleteByTokenHash revokes a session; deleting a missing session is not an error.
func (r *SessionRepo) DeleteByTokenHash(_ context.Context, tokenHash string) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	delete(r.byToken, tokenHash)
	return nil
}

// DeleteExpired removes all sessions past their expiry.
func (r *SessionRepo) DeleteExpired(_ context.Context, now time.Time) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	for k, s := range r.byToken {
		if s.Expired(now) {
			delete(r.byToken, k)
		}
	}
	return nil
}
