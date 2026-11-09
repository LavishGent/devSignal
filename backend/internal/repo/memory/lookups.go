package memory

import (
	"context"

	"devsignal/internal/domain"
)

// LookupRepo is the in-memory repo.LookupRepo. Tests may replace the option
// set to simulate an operator adding or removing choices.
type LookupRepo struct {
	lookups domain.Lookups
}

// NewLookupRepo starts with the default seed options.
func NewLookupRepo() *LookupRepo { return &LookupRepo{lookups: domain.DefaultLookups()} }

// Set replaces the option set.
func (r *LookupRepo) Set(l domain.Lookups) { r.lookups = l }

// All implements repo.LookupRepo.
func (r *LookupRepo) All(_ context.Context) (domain.Lookups, error) { return r.lookups, nil }
