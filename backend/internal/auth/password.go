// Package auth implements password hashing and session management.
//
// Password demonstrates encapsulation: the bcrypt hash is unexported, so the
// only operations available are the safe ones — create from a plaintext,
// restore from a stored hash, compare, and export for storage.
package auth

import (
	"fmt"

	"golang.org/x/crypto/bcrypt"
)

// MinPasswordLength is the shortest password accepted at registration.
const MinPasswordLength = 8

// maxPasswordLength guards bcrypt's 72-byte input limit.
const maxPasswordLength = 72

// ErrPasswordPolicy is returned when a candidate password violates policy.
var ErrPasswordPolicy = fmt.Errorf("password must be between %d and %d characters", MinPasswordLength, maxPasswordLength)

// Password wraps a bcrypt hash. The zero value matches nothing.
type Password struct {
	hash []byte
}

// HashPassword enforces the password policy and derives a bcrypt hash at the
// given cost. The plaintext is never stored.
func HashPassword(plain string, cost int) (Password, error) {
	if len(plain) < MinPasswordLength || len(plain) > maxPasswordLength {
		return Password{}, ErrPasswordPolicy
	}
	h, err := bcrypt.GenerateFromPassword([]byte(plain), cost)
	if err != nil {
		return Password{}, fmt.Errorf("hashing password: %w", err)
	}
	return Password{hash: h}, nil
}

// PasswordFromHash restores a Password from its stored hash string.
func PasswordFromHash(hash string) Password {
	return Password{hash: []byte(hash)}
}

// Matches reports whether the plaintext corresponds to the stored hash,
// in constant time with respect to the hash comparison.
func (p Password) Matches(plain string) bool {
	if len(p.hash) == 0 {
		return false
	}
	return bcrypt.CompareHashAndPassword(p.hash, []byte(plain)) == nil
}

// Hash exports the bcrypt hash for persistence.
func (p Password) Hash() string { return string(p.hash) }
