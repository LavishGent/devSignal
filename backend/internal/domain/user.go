package domain

import (
	"strings"
	"time"

	"devsignal/internal/validation"

	"github.com/google/uuid"
)

// User is an account holder. PasswordHash is excluded from JSON so it can
// never leak through an API response.
type User struct {
	ID           uuid.UUID `json:"id"`
	Email        string    `json:"email"`
	DisplayName  string    `json:"displayName"`
	PasswordHash string    `json:"-"`
	CreatedAt    time.Time `json:"createdAt"`
	UpdatedAt    time.Time `json:"updatedAt"`
}

// NormalizeEmail lowercases and trims an email for storage and lookup.
func NormalizeEmail(email string) string {
	return strings.ToLower(strings.TrimSpace(email))
}

// Validate implements Validator.
func (u *User) Validate() validation.Errors {
	e := validation.New()
	validation.Required(e, "email", u.Email)
	validation.Email(e, "email", u.Email)
	validation.MaxLen(e, "email", u.Email, 254)
	validation.Required(e, "displayName", u.DisplayName)
	validation.MaxLen(e, "displayName", u.DisplayName, 100)
	return e
}
