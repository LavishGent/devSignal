package domain

import (
	"database/sql/driver"
	"encoding/json"
	"fmt"
	"time"
)

// Date is a calendar date without a time component, used for fields like
// appliedOn or targetDate. Its state is unexported (encapsulation): callers
// construct one through NewDate or ParseDate and interact through methods,
// so an invalid or half-initialized date cannot be created from outside.
type Date struct {
	t     time.Time
	valid bool
}

const dateLayout = "2006-01-02"

// NewDate builds a Date from year, month, and day.
func NewDate(year int, month time.Month, day int) Date {
	return Date{t: time.Date(year, month, day, 0, 0, 0, 0, time.UTC), valid: true}
}

// DateOf truncates a time to its calendar date (UTC).
func DateOf(t time.Time) Date {
	if t.IsZero() {
		return Date{}
	}
	y, m, d := t.UTC().Date()
	return NewDate(y, m, d)
}

// ParseDate parses "YYYY-MM-DD". An empty string yields the zero Date.
func ParseDate(s string) (Date, error) {
	if s == "" {
		return Date{}, nil
	}
	t, err := time.Parse(dateLayout, s)
	if err != nil {
		return Date{}, fmt.Errorf("invalid date %q: expected YYYY-MM-DD", s)
	}
	return Date{t: t, valid: true}, nil
}

// IsZero reports whether the date is unset.
func (d Date) IsZero() bool { return !d.valid }

// Time returns the underlying UTC midnight time, or the zero time if unset.
func (d Date) Time() time.Time {
	if !d.valid {
		return time.Time{}
	}
	return d.t
}

// String renders "YYYY-MM-DD", or "" when unset.
func (d Date) String() string {
	if !d.valid {
		return ""
	}
	return d.t.Format(dateLayout)
}

// Before reports whether d is earlier than other; unset dates are never before.
func (d Date) Before(other Date) bool {
	return d.valid && other.valid && d.t.Before(other.t)
}

// DaysUntil returns whole days from now's date to d (negative when past).
func (d Date) DaysUntil(now time.Time) int {
	if !d.valid {
		return 0
	}
	return int(d.t.Sub(DateOf(now).Time()).Hours() / 24)
}

// DaysSince returns whole days from d to now's date (negative when future).
func (d Date) DaysSince(now time.Time) int {
	return -d.DaysUntil(now)
}

// MarshalJSON renders the date as "YYYY-MM-DD" or null when unset.
func (d Date) MarshalJSON() ([]byte, error) {
	if !d.valid {
		return []byte("null"), nil
	}
	return json.Marshal(d.String())
}

// UnmarshalJSON accepts "YYYY-MM-DD", "", or null.
func (d *Date) UnmarshalJSON(b []byte) error {
	var s *string
	if err := json.Unmarshal(b, &s); err != nil {
		return fmt.Errorf("date must be a string in YYYY-MM-DD format")
	}
	if s == nil {
		*d = Date{}
		return nil
	}
	parsed, err := ParseDate(*s)
	if err != nil {
		return err
	}
	*d = parsed
	return nil
}

// Value implements driver.Valuer so a Date can be written to a DATE column;
// unset dates are stored as NULL.
func (d Date) Value() (driver.Value, error) {
	if !d.valid {
		return nil, nil
	}
	return d.t, nil
}

// Scan implements sql.Scanner so a DATE column (or NULL) can be read back.
func (d *Date) Scan(src any) error {
	switch v := src.(type) {
	case nil:
		*d = Date{}
		return nil
	case time.Time:
		*d = DateOf(v)
		return nil
	case string:
		parsed, err := ParseDate(v)
		if err != nil {
			return err
		}
		*d = parsed
		return nil
	default:
		return fmt.Errorf("cannot scan %T into Date", src)
	}
}
