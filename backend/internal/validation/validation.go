// Package validation provides field-level input validation shared by every
// domain model. A model collects problems into an Errors map; an empty map
// means the value is valid.
package validation

import (
	"fmt"
	"net/mail"
	"net/url"
	"slices"
	"strings"
	"time"
	"unicode/utf8"
)

// Errors maps a field name to a human-readable problem with that field.
type Errors map[string]string

// New returns an empty, ready-to-use Errors map.
func New() Errors { return Errors{} }

// Add records a problem for a field. The first problem recorded for a field
// wins so callers can order checks from most to least specific.
func (e Errors) Add(field, msg string) {
	if _, exists := e[field]; !exists {
		e[field] = msg
	}
}

// Ok reports whether no problems were recorded.
func (e Errors) Ok() bool { return len(e) == 0 }

// Required records a problem when value is empty or only whitespace.
func Required(e Errors, field, value string) {
	if strings.TrimSpace(value) == "" {
		e.Add(field, "is required")
	}
}

// MaxLen records a problem when value exceeds max characters.
func MaxLen(e Errors, field, value string, max int) {
	if utf8.RuneCountInString(value) > max {
		e.Add(field, fmt.Sprintf("must be %d characters or fewer", max))
	}
}

// OneOf records a problem when value is not one of the allowed choices.
// Empty values are ignored; pair with Required for mandatory enums.
func OneOf(e Errors, field, value string, allowed ...string) {
	if value == "" || slices.Contains(allowed, value) {
		return
	}
	e.Add(field, "must be one of: "+strings.Join(allowed, ", "))
}

// IntRange records a problem when v falls outside [min, max].
func IntRange(e Errors, field string, v, min, max int) {
	if v < min || v > max {
		e.Add(field, fmt.Sprintf("must be between %d and %d", min, max))
	}
}

// MinInt records a problem when v is below min.
func MinInt(e Errors, field string, v, min int) {
	if v < min {
		e.Add(field, fmt.Sprintf("must be at least %d", min))
	}
}

// Email records a problem when value is not a parseable email address.
// Empty values are ignored; pair with Required for mandatory emails.
func Email(e Errors, field, value string) {
	if value == "" {
		return
	}
	addr, err := mail.ParseAddress(value)
	if err != nil || addr.Address != value {
		e.Add(field, "must be a valid email address")
	}
}

// WebURL records a problem when value is not an absolute http(s) URL.
// Empty values are ignored.
func WebURL(e Errors, field, value string) {
	if value == "" {
		return
	}
	u, err := url.Parse(value)
	if err != nil || (u.Scheme != "http" && u.Scheme != "https") || u.Host == "" {
		e.Add(field, "must be a valid http or https URL")
	}
}

// DateOrder records a problem against laterField when both times are set and
// later is before earlier.
func DateOrder(e Errors, laterField string, earlier, later time.Time, earlierName string) {
	if earlier.IsZero() || later.IsZero() {
		return
	}
	if later.Before(earlier) {
		e.Add(laterField, "must not be before "+earlierName)
	}
}
