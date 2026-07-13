package postgres

import (
	"context"

	"devsignal/internal/domain"
	"devsignal/internal/repo"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

// ProjectRepo is the postgres repo.ProjectRepo.
type ProjectRepo struct {
	pool *pgxpool.Pool
}

const projectColumns = `id, user_id, name, description, status, tech_stack,
	repo_url, demo_url, started_on, last_activity_on, notes, created_at, updated_at`

// Create inserts a stamped project.
func (r *ProjectRepo) Create(ctx context.Context, p *domain.Project) error {
	_, err := r.pool.Exec(ctx, `
		INSERT INTO projects (`+projectColumns+`)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13)`,
		p.ID, p.UserID, p.Name, p.Description, p.Status, p.TechStack,
		p.RepoURL, p.DemoURL, p.StartedOn, p.LastActivityOn, p.Notes,
		p.CreatedAt, p.UpdatedAt)
	return mapError(err)
}

// Get fetches one project scoped to its owner.
func (r *ProjectRepo) Get(ctx context.Context, userID, id uuid.UUID) (*domain.Project, error) {
	row := r.pool.QueryRow(ctx, `
		SELECT `+projectColumns+` FROM projects
		WHERE user_id = $1 AND id = $2`, userID, id)
	return scanProject(row)
}

// List returns the user's projects, filtered and newest-updated first.
func (r *ProjectRepo) List(ctx context.Context, userID uuid.UUID, f repo.ProjectFilter) ([]domain.Project, error) {
	pg := f.Clamp()
	rows, err := r.pool.Query(ctx, `
		SELECT `+projectColumns+` FROM projects
		WHERE user_id = $1
		  AND ($2 = '' OR status = $2)
		  AND ($3 = '' OR name ILIKE $4 OR description ILIKE $4 OR tech_stack ILIKE $4 OR notes ILIKE $4)
		ORDER BY updated_at DESC, id
		LIMIT $5 OFFSET $6`,
		userID, f.Status, f.Q, likePattern(f.Q), pg.Limit, pg.Offset)
	if err != nil {
		return nil, mapError(err)
	}
	defer rows.Close()

	out := []domain.Project{}
	for rows.Next() {
		p, err := scanProject(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, *p)
	}
	return out, mapError(rows.Err())
}

// Update rewrites the mutable columns of an owned project.
func (r *ProjectRepo) Update(ctx context.Context, p *domain.Project) error {
	tag, err := r.pool.Exec(ctx, `
		UPDATE projects SET
			name = $3, description = $4, status = $5, tech_stack = $6,
			repo_url = $7, demo_url = $8, started_on = $9,
			last_activity_on = $10, notes = $11, updated_at = $12
		WHERE user_id = $1 AND id = $2`,
		p.UserID, p.ID, p.Name, p.Description, p.Status, p.TechStack,
		p.RepoURL, p.DemoURL, p.StartedOn, p.LastActivityOn, p.Notes,
		p.UpdatedAt)
	if err != nil {
		return mapError(err)
	}
	if tag.RowsAffected() == 0 {
		return repo.ErrNotFound
	}
	return nil
}

// Delete removes an owned project.
func (r *ProjectRepo) Delete(ctx context.Context, userID, id uuid.UUID) error {
	tag, err := r.pool.Exec(ctx, `
		DELETE FROM projects WHERE user_id = $1 AND id = $2`, userID, id)
	if err != nil {
		return mapError(err)
	}
	if tag.RowsAffected() == 0 {
		return repo.ErrNotFound
	}
	return nil
}

func scanProject(row rowScanner) (*domain.Project, error) {
	var p domain.Project
	err := row.Scan(&p.ID, &p.UserID, &p.Name, &p.Description, &p.Status, &p.TechStack,
		&p.RepoURL, &p.DemoURL, &p.StartedOn, &p.LastActivityOn, &p.Notes,
		&p.CreatedAt, &p.UpdatedAt)
	if err != nil {
		return nil, mapError(err)
	}
	return &p, nil
}
