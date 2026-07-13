package validation

import (
	"testing"
	"time"
)

func TestErrorsAddFirstProblemWins(t *testing.T) {
	e := New()
	e.Add("name", "is required")
	e.Add("name", "second problem")
	if e["name"] != "is required" {
		t.Errorf("expected first problem to win, got %q", e["name"])
	}
}

func TestErrorsOk(t *testing.T) {
	e := New()
	if !e.Ok() {
		t.Error("empty Errors should be Ok")
	}
	e.Add("f", "bad")
	if e.Ok() {
		t.Error("non-empty Errors should not be Ok")
	}
}

func TestRequired(t *testing.T) {
	cases := []struct {
		value string
		valid bool
	}{
		{"hello", true},
		{"", false},
		{"   ", false},
		{"\t\n", false},
		{" x ", true},
	}
	for _, c := range cases {
		e := New()
		Required(e, "f", c.value)
		if e.Ok() != c.valid {
			t.Errorf("Required(%q): valid=%v, want %v", c.value, e.Ok(), c.valid)
		}
	}
}

func TestMaxLen(t *testing.T) {
	e := New()
	MaxLen(e, "f", "abc", 3)
	if !e.Ok() {
		t.Error("exactly max length should pass")
	}
	MaxLen(e, "g", "abcd", 3)
	if e.Ok() {
		t.Error("over max length should fail")
	}
	e2 := New()
	MaxLen(e2, "f", "héllo", 5) // multibyte counts runes, not bytes
	if !e2.Ok() {
		t.Error("rune counting should treat héllo as 5 characters")
	}
}

func TestOneOf(t *testing.T) {
	e := New()
	OneOf(e, "status", "applied", "applied", "rejected")
	if !e.Ok() {
		t.Error("allowed value should pass")
	}
	OneOf(e, "status2", "bogus", "applied", "rejected")
	if _, found := e["status2"]; !found {
		t.Error("disallowed value should fail")
	}
	e2 := New()
	OneOf(e2, "status", "", "applied")
	if !e2.Ok() {
		t.Error("empty value should be ignored")
	}
}

func TestIntRange(t *testing.T) {
	for _, c := range []struct {
		v     int
		valid bool
	}{{1, true}, {5, true}, {0, false}, {6, false}} {
		e := New()
		IntRange(e, "p", c.v, 1, 5)
		if e.Ok() != c.valid {
			t.Errorf("IntRange(%d): valid=%v, want %v", c.v, e.Ok(), c.valid)
		}
	}
}

func TestMinInt(t *testing.T) {
	e := New()
	MinInt(e, "salary", -1, 0)
	if e.Ok() {
		t.Error("below min should fail")
	}
	e2 := New()
	MinInt(e2, "salary", 0, 0)
	if !e2.Ok() {
		t.Error("at min should pass")
	}
}

func TestEmail(t *testing.T) {
	cases := []struct {
		value string
		valid bool
	}{
		{"user@example.com", true},
		{"", true}, // optional when empty
		{"not-an-email", false},
		{"User Name <user@example.com>", false}, // display-name form rejected
		{"user@", false},
	}
	for _, c := range cases {
		e := New()
		Email(e, "email", c.value)
		if e.Ok() != c.valid {
			t.Errorf("Email(%q): valid=%v, want %v", c.value, e.Ok(), c.valid)
		}
	}
}

func TestWebURL(t *testing.T) {
	cases := []struct {
		value string
		valid bool
	}{
		{"https://example.com/jobs/1", true},
		{"http://example.com", true},
		{"", true}, // optional when empty
		{"ftp://example.com", false},
		{"javascript:alert(1)", false},
		{"example.com", false}, // no scheme
	}
	for _, c := range cases {
		e := New()
		WebURL(e, "url", c.value)
		if e.Ok() != c.valid {
			t.Errorf("WebURL(%q): valid=%v, want %v", c.value, e.Ok(), c.valid)
		}
	}
}

func TestDateOrder(t *testing.T) {
	earlier := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	later := time.Date(2026, 6, 1, 0, 0, 0, 0, time.UTC)

	e := New()
	DateOrder(e, "expiresOn", earlier, later, "earnedOn")
	if !e.Ok() {
		t.Error("correctly ordered dates should pass")
	}

	e2 := New()
	DateOrder(e2, "expiresOn", later, earlier, "earnedOn")
	if e2.Ok() {
		t.Error("reversed dates should fail")
	}

	e3 := New()
	DateOrder(e3, "expiresOn", time.Time{}, earlier, "earnedOn")
	if !e3.Ok() {
		t.Error("zero earlier date should be ignored")
	}
}
