package memory

import (
	"context"
	"sort"
	"sync"

	"devsignal/internal/domain"
	"devsignal/internal/repo"

	"github.com/google/uuid"
)

// store is the generic core shared by every in-memory module repository.
// Module repos embed it (inheritance) and add their own List filtering.
// The base accessor exposes each record's embedded BaseRecord.
type store[T any] struct {
	mu    sync.RWMutex
	items map[uuid.UUID]T
	base  func(*T) *domain.BaseRecord
}

func newStore[T any](base func(*T) *domain.BaseRecord) *store[T] {
	return &store[T]{items: map[uuid.UUID]T{}, base: base}
}

// Create stores a stamped record.
func (s *store[T]) Create(_ context.Context, v *T) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.items[s.base(v).ID] = *v
	return nil
}

// Get returns a record only when it exists and belongs to userID.
func (s *store[T]) Get(_ context.Context, userID, id uuid.UUID) (*T, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	item, ok := s.items[id]
	if !ok || !s.base(&item).IsOwnedBy(userID) {
		return nil, repo.ErrNotFound
	}
	return &item, nil
}

// Update replaces a record when it exists and the owner matches.
func (s *store[T]) Update(_ context.Context, v *T) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	b := s.base(v)
	existing, ok := s.items[b.ID]
	if !ok || !s.base(&existing).IsOwnedBy(b.UserID) {
		return repo.ErrNotFound
	}
	s.items[b.ID] = *v
	return nil
}

// Delete removes a record when it exists and belongs to userID.
func (s *store[T]) Delete(_ context.Context, userID, id uuid.UUID) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	item, ok := s.items[id]
	if !ok || !s.base(&item).IsOwnedBy(userID) {
		return repo.ErrNotFound
	}
	delete(s.items, id)
	return nil
}

// all returns the user's records ordered by most recently updated, matching
// the postgres ORDER BY. Module List methods filter and paginate the result.
func (s *store[T]) all(userID uuid.UUID) []T {
	s.mu.RLock()
	defer s.mu.RUnlock()
	var out []T
	for _, item := range s.items {
		if s.base(&item).IsOwnedBy(userID) {
			out = append(out, item)
		}
	}
	sort.Slice(out, func(i, j int) bool {
		bi, bj := s.base(&out[i]), s.base(&out[j])
		if !bi.UpdatedAt.Equal(bj.UpdatedAt) {
			return bi.UpdatedAt.After(bj.UpdatedAt)
		}
		return bi.ID.String() < bj.ID.String() // stable tiebreak
	})
	return out
}

// paginate applies a clamped Page to a filtered slice.
func paginate[T any](items []T, p repo.Page) []T {
	p = p.Clamp()
	if p.Offset >= len(items) {
		return []T{}
	}
	end := min(p.Offset+p.Limit, len(items))
	return items[p.Offset:end]
}
