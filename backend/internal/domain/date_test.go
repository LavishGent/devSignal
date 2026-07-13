package domain

import (
	"encoding/json"
	"testing"
	"time"
)

func TestParseDateRoundTrip(t *testing.T) {
	d, err := ParseDate("2026-07-12")
	if err != nil {
		t.Fatalf("ParseDate: %v", err)
	}
	if d.String() != "2026-07-12" {
		t.Errorf("String() = %q", d.String())
	}
	if d.IsZero() {
		t.Error("parsed date should not be zero")
	}
}

func TestParseDateEmptyAndInvalid(t *testing.T) {
	d, err := ParseDate("")
	if err != nil || !d.IsZero() {
		t.Error("empty string should parse to the zero Date without error")
	}
	if _, err := ParseDate("07/12/2026"); err == nil {
		t.Error("US-format date should be rejected")
	}
	if _, err := ParseDate("2026-13-40"); err == nil {
		t.Error("impossible date should be rejected")
	}
}

func TestDateJSON(t *testing.T) {
	d := NewDate(2026, time.July, 12)
	b, err := json.Marshal(d)
	if err != nil || string(b) != `"2026-07-12"` {
		t.Errorf("Marshal = %s, %v", b, err)
	}

	var zero Date
	b, _ = json.Marshal(zero)
	if string(b) != "null" {
		t.Errorf("zero Date should marshal to null, got %s", b)
	}

	var back Date
	if err := json.Unmarshal([]byte(`"2026-07-12"`), &back); err != nil || back.String() != "2026-07-12" {
		t.Errorf("Unmarshal round-trip failed: %v", err)
	}
	var fromNull Date
	if err := json.Unmarshal([]byte(`null`), &fromNull); err != nil || !fromNull.IsZero() {
		t.Error("null should unmarshal to the zero Date")
	}
	var bad Date
	if err := json.Unmarshal([]byte(`123`), &bad); err == nil {
		t.Error("numeric JSON should be rejected")
	}
}

func TestDateComparisons(t *testing.T) {
	early := NewDate(2026, time.January, 1)
	late := NewDate(2026, time.June, 1)
	if !early.Before(late) || late.Before(early) {
		t.Error("Before comparison wrong")
	}
	var zero Date
	if zero.Before(late) || late.Before(zero) {
		t.Error("zero dates must never compare Before")
	}
}

func TestDaysUntilAndSince(t *testing.T) {
	now := time.Date(2026, 7, 12, 15, 45, 0, 0, time.UTC) // mid-afternoon
	in3 := NewDate(2026, time.July, 15)
	if got := in3.DaysUntil(now); got != 3 {
		t.Errorf("DaysUntil = %d, want 3", got)
	}
	ago10 := NewDate(2026, time.July, 2)
	if got := ago10.DaysSince(now); got != 10 {
		t.Errorf("DaysSince = %d, want 10", got)
	}
}

func TestDateScan(t *testing.T) {
	var d Date
	if err := d.Scan(time.Date(2026, 7, 12, 0, 0, 0, 0, time.UTC)); err != nil || d.String() != "2026-07-12" {
		t.Errorf("Scan(time.Time) failed: %v %q", err, d.String())
	}
	var n Date
	if err := n.Scan(nil); err != nil || !n.IsZero() {
		t.Error("Scan(nil) should yield zero Date")
	}
	var bad Date
	if err := bad.Scan(42); err == nil {
		t.Error("Scan(int) should error")
	}
}

func TestDateValue(t *testing.T) {
	v, err := NewDate(2026, time.July, 12).Value()
	if err != nil || v == nil {
		t.Fatalf("Value: %v %v", v, err)
	}
	var zero Date
	v, err = zero.Value()
	if err != nil || v != nil {
		t.Error("zero Date should store as NULL")
	}
}
