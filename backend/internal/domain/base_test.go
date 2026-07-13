package domain

import (
	"testing"
	"time"

	"github.com/google/uuid"
)

func TestStampNewSetsIdentityAndTimestamps(t *testing.T) {
	owner := uuid.New()
	now := time.Date(2026, 7, 12, 10, 30, 0, 0, time.FixedZone("CDT", -5*3600))

	var b BaseRecord
	b.StampNew(owner, now)

	if b.ID == uuid.Nil {
		t.Error("StampNew should assign a non-nil ID")
	}
	if b.UserID != owner {
		t.Error("StampNew should assign the owner")
	}
	if !b.CreatedAt.Equal(now) || !b.UpdatedAt.Equal(now) {
		t.Error("StampNew should set both timestamps to now")
	}
	if b.CreatedAt.Location() != time.UTC {
		t.Error("timestamps should be normalized to UTC")
	}
}

func TestTouchOnlyMovesUpdatedAt(t *testing.T) {
	var b BaseRecord
	created := time.Date(2026, 7, 1, 0, 0, 0, 0, time.UTC)
	b.StampNew(uuid.New(), created)

	later := created.Add(48 * time.Hour)
	b.Touch(later)

	if !b.CreatedAt.Equal(created) {
		t.Error("Touch must not change CreatedAt")
	}
	if !b.UpdatedAt.Equal(later) {
		t.Error("Touch should move UpdatedAt")
	}
}

func TestIsOwnedBy(t *testing.T) {
	owner := uuid.New()
	var b BaseRecord
	b.StampNew(owner, time.Now())

	if !b.IsOwnedBy(owner) {
		t.Error("record should be owned by its owner")
	}
	if b.IsOwnedBy(uuid.New()) {
		t.Error("record should not be owned by a stranger")
	}
	var unstamped BaseRecord
	if unstamped.IsOwnedBy(uuid.Nil) {
		t.Error("a record with no owner must never match the nil user")
	}
}

func TestUserValidate(t *testing.T) {
	valid := User{Email: "dev@example.com", DisplayName: "Darrell"}
	if e := valid.Validate(); !e.Ok() {
		t.Errorf("valid user should pass, got %v", e)
	}

	cases := []struct {
		name  string
		user  User
		field string
	}{
		{"missing email", User{DisplayName: "D"}, "email"},
		{"bad email", User{Email: "nope", DisplayName: "D"}, "email"},
		{"missing display name", User{Email: "dev@example.com"}, "displayName"},
	}
	for _, c := range cases {
		if e := c.user.Validate(); e.Ok() || e[c.field] == "" {
			t.Errorf("%s: expected problem on %q, got %v", c.name, c.field, e)
		}
	}
}

func TestNormalizeEmail(t *testing.T) {
	if got := NormalizeEmail("  Dev@Example.COM "); got != "dev@example.com" {
		t.Errorf("NormalizeEmail = %q", got)
	}
}

func TestSessionExpired(t *testing.T) {
	now := time.Date(2026, 7, 12, 12, 0, 0, 0, time.UTC)
	s := Session{ExpiresAt: now.Add(time.Hour)}
	if s.Expired(now) {
		t.Error("session expiring in an hour is not expired")
	}
	if !s.Expired(now.Add(time.Hour)) {
		t.Error("session at exactly its expiry is expired")
	}
}
