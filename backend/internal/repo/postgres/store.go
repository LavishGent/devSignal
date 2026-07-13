// Package postgres implements the repo interfaces against PostgreSQL using
// pgx. Every statement is parameterized — user input is never concatenated
// into SQL — and every module query is scoped to the owning user.
package postgres

import (
	"errors"
	"strings"

	"devsignal/internal/repo"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

// Store bundles all postgres repositories over one connection pool.
type Store struct {
	Users          *UserRepo
	Sessions       *SessionRepo
	Applications   *ApplicationRepo
	Contacts       *ContactRepo
	Interviews     *InterviewRepo
	Skills         *SkillRepo
	StudySessions  *StudySessionRepo
	Certifications *CertificationRepo
	Projects       *ProjectRepo
	Goals          *GoalRepo
}

// NewStore builds every repository on the shared pool.
func NewStore(pool *pgxpool.Pool) *Store {
	return &Store{
		Users:          &UserRepo{pool: pool},
		Sessions:       &SessionRepo{pool: pool},
		Applications:   &ApplicationRepo{pool: pool},
		Contacts:       &ContactRepo{pool: pool},
		Interviews:     &InterviewRepo{pool: pool},
		Skills:         &SkillRepo{pool: pool},
		StudySessions:  &StudySessionRepo{pool: pool},
		Certifications: &CertificationRepo{pool: pool},
		Projects:       &ProjectRepo{pool: pool},
		Goals:          &GoalRepo{pool: pool},
	}
}

// mapError converts pgx errors to the repo sentinels handlers understand.
func mapError(err error) error {
	if err == nil {
		return nil
	}
	if errors.Is(err, pgx.ErrNoRows) {
		return repo.ErrNotFound
	}
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) && pgErr.Code == "23505" { // unique_violation
		return repo.ErrConflict
	}
	return err
}

// likePattern turns free text into a safe ILIKE pattern: LIKE wildcards in
// the user's input are escaped, then the text is wrapped in %...%.
func likePattern(q string) string {
	r := strings.NewReplacer(`\`, `\\`, `%`, `\%`, `_`, `\_`)
	return "%" + r.Replace(q) + "%"
}
