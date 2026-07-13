// Package memory provides in-memory repository implementations. They back
// unit tests and satisfy the same interfaces as the postgres package, which
// keeps handlers testable without a database.
package memory

import (
	"context"
	"sync"

	"devsignal/internal/domain"
	"devsignal/internal/repo"

	"github.com/google/uuid"
)

// UserRepo is a thread-safe in-memory repo.UserRepo.
type UserRepo struct {
	mu    sync.RWMutex
	byID  map[uuid.UUID]domain.User
	byKey map[string]uuid.UUID // normalized email → id
}

// NewUserRepo returns an empty in-memory user store.
func NewUserRepo() *UserRepo {
	return &UserRepo{
		byID:  map[uuid.UUID]domain.User{},
		byKey: map[string]uuid.UUID{},
	}
}

// Create stores a user, enforcing email uniqueness like the database does.
func (r *UserRepo) Create(_ context.Context, u *domain.User) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	key := domain.NormalizeEmail(u.Email)
	if _, taken := r.byKey[key]; taken {
		return repo.ErrConflict
	}
	r.byID[u.ID] = *u
	r.byKey[key] = u.ID
	return nil
}

// GetByEmail looks a user up by normalized email.
func (r *UserRepo) GetByEmail(_ context.Context, email string) (*domain.User, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	id, ok := r.byKey[domain.NormalizeEmail(email)]
	if !ok {
		return nil, repo.ErrNotFound
	}
	u := r.byID[id]
	return &u, nil
}

// GetByID looks a user up by id.
func (r *UserRepo) GetByID(_ context.Context, id uuid.UUID) (*domain.User, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	u, ok := r.byID[id]
	if !ok {
		return nil, repo.ErrNotFound
	}
	return &u, nil
}
