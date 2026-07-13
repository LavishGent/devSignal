package auth

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"devsignal/internal/repo"
	"devsignal/internal/repo/memory"

	"github.com/google/uuid"
)

const testCost = 10 // minimum cost keeps tests fast

func TestHashPasswordAndMatch(t *testing.T) {
	p, err := HashPassword("correct horse battery", testCost)
	if err != nil {
		t.Fatalf("HashPassword: %v", err)
	}
	if !p.Matches("correct horse battery") {
		t.Error("correct password should match")
	}
	if p.Matches("wrong password") {
		t.Error("wrong password must not match")
	}
	if strings.Contains(p.Hash(), "correct horse") {
		t.Error("hash must not contain the plaintext")
	}
}

func TestHashPasswordPolicy(t *testing.T) {
	if _, err := HashPassword("short", testCost); !errors.Is(err, ErrPasswordPolicy) {
		t.Error("short password should violate policy")
	}
	if _, err := HashPassword(strings.Repeat("x", 73), testCost); !errors.Is(err, ErrPasswordPolicy) {
		t.Error("over-long password should violate policy (bcrypt 72-byte limit)")
	}
	if _, err := HashPassword(strings.Repeat("x", 8), testCost); err != nil {
		t.Errorf("8-char password should be accepted: %v", err)
	}
}

func TestZeroPasswordMatchesNothing(t *testing.T) {
	var p Password
	if p.Matches("") || p.Matches("anything") {
		t.Error("zero Password must never match")
	}
}

func TestPasswordFromHashRoundTrip(t *testing.T) {
	p, _ := HashPassword("password123", testCost)
	restored := PasswordFromHash(p.Hash())
	if !restored.Matches("password123") {
		t.Error("restored hash should still match")
	}
}

func TestSessionLifecycle(t *testing.T) {
	ctx := context.Background()
	store := memory.NewSessionRepo()
	m := NewSessionManager(store, time.Hour)
	userID := uuid.New()

	token, err := m.Issue(ctx, userID)
	if err != nil {
		t.Fatalf("Issue: %v", err)
	}
	if len(token) < 40 {
		t.Errorf("token looks too short to be 256-bit: %q", token)
	}

	s, err := m.Verify(ctx, token)
	if err != nil {
		t.Fatalf("Verify: %v", err)
	}
	if s.UserID != userID {
		t.Error("session should belong to the issuing user")
	}

	if err := m.Revoke(ctx, token); err != nil {
		t.Fatalf("Revoke: %v", err)
	}
	if _, err := m.Verify(ctx, token); !errors.Is(err, repo.ErrNotFound) {
		t.Error("revoked session must not verify")
	}
}

func TestVerifyRejectsExpiredAndGarbage(t *testing.T) {
	ctx := context.Background()
	store := memory.NewSessionRepo()

	current := time.Date(2026, 7, 12, 12, 0, 0, 0, time.UTC)
	m := NewSessionManager(store, time.Hour).WithClock(func() time.Time { return current })

	token, _ := m.Issue(ctx, uuid.New())

	current = current.Add(2 * time.Hour) // move past expiry
	if _, err := m.Verify(ctx, token); !errors.Is(err, repo.ErrNotFound) {
		t.Error("expired session must not verify")
	}
	// Expired session should also have been cleaned up.
	if _, err := store.GetByTokenHash(ctx, HashToken(token)); !errors.Is(err, repo.ErrNotFound) {
		t.Error("expired session should be deleted on verify")
	}

	if _, err := m.Verify(ctx, ""); !errors.Is(err, repo.ErrNotFound) {
		t.Error("empty token must not verify")
	}
	if _, err := m.Verify(ctx, "made-up-token"); !errors.Is(err, repo.ErrNotFound) {
		t.Error("unknown token must not verify")
	}
}

func TestTokensAreUniqueAndHashedAtRest(t *testing.T) {
	ctx := context.Background()
	store := memory.NewSessionRepo()
	m := NewSessionManager(store, time.Hour)

	t1, _ := m.Issue(ctx, uuid.New())
	t2, _ := m.Issue(ctx, uuid.New())
	if t1 == t2 {
		t.Error("two issued tokens must differ")
	}
	// The raw token must not be usable as a storage key: only its hash is.
	if _, err := store.GetByTokenHash(ctx, t1); !errors.Is(err, repo.ErrNotFound) {
		t.Error("raw token must not appear in the session store")
	}
}
