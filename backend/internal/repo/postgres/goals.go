package postgres

import (
	"context"

	"devsignal/internal/domain"
	"devsignal/internal/repo"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

// GoalRepo is the postgres repo.GoalRepo.
type GoalRepo struct {
	pool *pgxpool.Pool
}

const goalColumns = `id, user_id, title, description, category, status, priority,
	target_date, completed_on, notes, created_at, updated_at`

// Create inserts a stamped goal.
func (r *GoalRepo) Create(ctx context.Context, g *domain.Goal) error {
	_, err := r.pool.Exec(ctx, `
		INSERT INTO goals (`+goalColumns+`)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12)`,
		g.ID, g.UserID, g.Title, g.Description, g.Category, g.Status, g.Priority,
		g.TargetDate, g.CompletedOn, g.Notes, g.CreatedAt, g.UpdatedAt)
	return mapError(err)
}

// Get fetches one goal scoped to its owner.
func (r *GoalRepo) Get(ctx context.Context, userID, id uuid.UUID) (*domain.Goal, error) {
	row := r.pool.QueryRow(ctx, `
		SELECT `+goalColumns+` FROM goals
		WHERE user_id = $1 AND id = $2`, userID, id)
	return scanGoal(row)
}

// List returns the user's goals, filtered and newest-updated first.
func (r *GoalRepo) List(ctx context.Context, userID uuid.UUID, f repo.GoalFilter) ([]domain.Goal, error) {
	p := f.Clamp()
	rows, err := r.pool.Query(ctx, `
		SELECT `+goalColumns+` FROM goals
		WHERE user_id = $1
		  AND ($2 = '' OR status = $2)
		  AND ($3 = '' OR category = $3)
		  AND ($4 = '' OR title ILIKE $5 OR description ILIKE $5 OR notes ILIKE $5)
		ORDER BY updated_at DESC, id
		LIMIT $6 OFFSET $7`,
		userID, f.Status, f.Category, f.Q, likePattern(f.Q), p.Limit, p.Offset)
	if err != nil {
		return nil, mapError(err)
	}
	defer rows.Close()

	out := []domain.Goal{}
	for rows.Next() {
		g, err := scanGoal(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, *g)
	}
	return out, mapError(rows.Err())
}

// Update rewrites the mutable columns of an owned goal.
func (r *GoalRepo) Update(ctx context.Context, g *domain.Goal) error {
	tag, err := r.pool.Exec(ctx, `
		UPDATE goals SET
			title = $3, description = $4, category = $5, status = $6,
			priority = $7, target_date = $8, completed_on = $9, notes = $10,
			updated_at = $11
		WHERE user_id = $1 AND id = $2`,
		g.UserID, g.ID, g.Title, g.Description, g.Category, g.Status,
		g.Priority, g.TargetDate, g.CompletedOn, g.Notes, g.UpdatedAt)
	if err != nil {
		return mapError(err)
	}
	if tag.RowsAffected() == 0 {
		return repo.ErrNotFound
	}
	return nil
}

// Delete removes an owned goal.
func (r *GoalRepo) Delete(ctx context.Context, userID, id uuid.UUID) error {
	tag, err := r.pool.Exec(ctx, `
		DELETE FROM goals WHERE user_id = $1 AND id = $2`, userID, id)
	if err != nil {
		return mapError(err)
	}
	if tag.RowsAffected() == 0 {
		return repo.ErrNotFound
	}
	return nil
}

func scanGoal(row rowScanner) (*domain.Goal, error) {
	var g domain.Goal
	err := row.Scan(&g.ID, &g.UserID, &g.Title, &g.Description, &g.Category, &g.Status,
		&g.Priority, &g.TargetDate, &g.CompletedOn, &g.Notes,
		&g.CreatedAt, &g.UpdatedAt)
	if err != nil {
		return nil, mapError(err)
	}
	return &g, nil
}
