package postgres

import (
	"context"

	"devsignal/internal/domain"
	"devsignal/internal/repo"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

// ApplicationRepo is the postgres repo.ApplicationRepo.
type ApplicationRepo struct {
	pool *pgxpool.Pool
}

const applicationColumns = `id, user_id, company, position, status, applied_on,
	location, work_mode, url, salary_min, salary_max, notes, created_at, updated_at`

// Create inserts a stamped application.
func (r *ApplicationRepo) Create(ctx context.Context, a *domain.Application) error {
	_, err := r.pool.Exec(ctx, `
		INSERT INTO applications (`+applicationColumns+`)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14)`,
		a.ID, a.UserID, a.Company, a.Position, a.Status, a.AppliedOn,
		a.Location, a.WorkMode, a.URL, a.SalaryMin, a.SalaryMax, a.Notes,
		a.CreatedAt, a.UpdatedAt)
	return mapError(err)
}

// Get fetches one application scoped to its owner.
func (r *ApplicationRepo) Get(ctx context.Context, userID, id uuid.UUID) (*domain.Application, error) {
	row := r.pool.QueryRow(ctx, `
		SELECT `+applicationColumns+` FROM applications
		WHERE user_id = $1 AND id = $2`, userID, id)
	return scanApplication(row)
}

// List returns the user's applications, filtered and newest-updated first.
func (r *ApplicationRepo) List(ctx context.Context, userID uuid.UUID, f repo.ApplicationFilter) ([]domain.Application, error) {
	p := f.Clamp()
	rows, err := r.pool.Query(ctx, `
		SELECT `+applicationColumns+` FROM applications
		WHERE user_id = $1
		  AND ($2 = '' OR status = $2)
		  AND ($3 = '' OR company ILIKE $4 OR position ILIKE $4 OR location ILIKE $4 OR notes ILIKE $4)
		ORDER BY updated_at DESC, id
		LIMIT $5 OFFSET $6`,
		userID, f.Status, f.Q, likePattern(f.Q), p.Limit, p.Offset)
	if err != nil {
		return nil, mapError(err)
	}
	defer rows.Close()

	out := []domain.Application{}
	for rows.Next() {
		a, err := scanApplication(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, *a)
	}
	return out, mapError(rows.Err())
}

// Update rewrites the mutable columns of an owned application.
func (r *ApplicationRepo) Update(ctx context.Context, a *domain.Application) error {
	tag, err := r.pool.Exec(ctx, `
		UPDATE applications SET
			company = $3, position = $4, status = $5, applied_on = $6,
			location = $7, work_mode = $8, url = $9, salary_min = $10,
			salary_max = $11, notes = $12, updated_at = $13
		WHERE user_id = $1 AND id = $2`,
		a.UserID, a.ID, a.Company, a.Position, a.Status, a.AppliedOn,
		a.Location, a.WorkMode, a.URL, a.SalaryMin, a.SalaryMax, a.Notes,
		a.UpdatedAt)
	if err != nil {
		return mapError(err)
	}
	if tag.RowsAffected() == 0 {
		return repo.ErrNotFound
	}
	return nil
}

// Delete removes an owned application (interviews cascade in the schema).
func (r *ApplicationRepo) Delete(ctx context.Context, userID, id uuid.UUID) error {
	tag, err := r.pool.Exec(ctx, `
		DELETE FROM applications WHERE user_id = $1 AND id = $2`, userID, id)
	if err != nil {
		return mapError(err)
	}
	if tag.RowsAffected() == 0 {
		return repo.ErrNotFound
	}
	return nil
}

func scanApplication(row rowScanner) (*domain.Application, error) {
	var a domain.Application
	err := row.Scan(&a.ID, &a.UserID, &a.Company, &a.Position, &a.Status, &a.AppliedOn,
		&a.Location, &a.WorkMode, &a.URL, &a.SalaryMin, &a.SalaryMax, &a.Notes,
		&a.CreatedAt, &a.UpdatedAt)
	if err != nil {
		return nil, mapError(err)
	}
	return &a, nil
}
