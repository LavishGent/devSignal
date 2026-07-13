package postgres

import (
	"context"

	"devsignal/internal/domain"
	"devsignal/internal/repo"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

// SkillRepo is the postgres repo.SkillRepo.
type SkillRepo struct {
	pool *pgxpool.Pool
}

const skillColumns = `id, user_id, name, category, proficiency, target_proficiency,
	last_practiced_on, notes, created_at, updated_at`

// Create inserts a stamped skill.
func (r *SkillRepo) Create(ctx context.Context, s *domain.Skill) error {
	_, err := r.pool.Exec(ctx, `
		INSERT INTO skills (`+skillColumns+`)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)`,
		s.ID, s.UserID, s.Name, s.Category, s.Proficiency, s.TargetProficiency,
		s.LastPracticedOn, s.Notes, s.CreatedAt, s.UpdatedAt)
	return mapError(err)
}

// Get fetches one skill scoped to its owner.
func (r *SkillRepo) Get(ctx context.Context, userID, id uuid.UUID) (*domain.Skill, error) {
	row := r.pool.QueryRow(ctx, `
		SELECT `+skillColumns+` FROM skills
		WHERE user_id = $1 AND id = $2`, userID, id)
	return scanSkill(row)
}

// List returns the user's skills, filtered and newest-updated first.
func (r *SkillRepo) List(ctx context.Context, userID uuid.UUID, f repo.SkillFilter) ([]domain.Skill, error) {
	p := f.Clamp()
	rows, err := r.pool.Query(ctx, `
		SELECT `+skillColumns+` FROM skills
		WHERE user_id = $1
		  AND ($2 = '' OR category = $2)
		  AND ($3 = '' OR name ILIKE $4 OR notes ILIKE $4)
		ORDER BY updated_at DESC, id
		LIMIT $5 OFFSET $6`,
		userID, f.Category, f.Q, likePattern(f.Q), p.Limit, p.Offset)
	if err != nil {
		return nil, mapError(err)
	}
	defer rows.Close()

	out := []domain.Skill{}
	for rows.Next() {
		s, err := scanSkill(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, *s)
	}
	return out, mapError(rows.Err())
}

// Update rewrites the mutable columns of an owned skill.
func (r *SkillRepo) Update(ctx context.Context, s *domain.Skill) error {
	tag, err := r.pool.Exec(ctx, `
		UPDATE skills SET
			name = $3, category = $4, proficiency = $5, target_proficiency = $6,
			last_practiced_on = $7, notes = $8, updated_at = $9
		WHERE user_id = $1 AND id = $2`,
		s.UserID, s.ID, s.Name, s.Category, s.Proficiency, s.TargetProficiency,
		s.LastPracticedOn, s.Notes, s.UpdatedAt)
	if err != nil {
		return mapError(err)
	}
	if tag.RowsAffected() == 0 {
		return repo.ErrNotFound
	}
	return nil
}

// Delete removes an owned skill.
func (r *SkillRepo) Delete(ctx context.Context, userID, id uuid.UUID) error {
	tag, err := r.pool.Exec(ctx, `
		DELETE FROM skills WHERE user_id = $1 AND id = $2`, userID, id)
	if err != nil {
		return mapError(err)
	}
	if tag.RowsAffected() == 0 {
		return repo.ErrNotFound
	}
	return nil
}

func scanSkill(row rowScanner) (*domain.Skill, error) {
	var s domain.Skill
	err := row.Scan(&s.ID, &s.UserID, &s.Name, &s.Category, &s.Proficiency, &s.TargetProficiency,
		&s.LastPracticedOn, &s.Notes, &s.CreatedAt, &s.UpdatedAt)
	if err != nil {
		return nil, mapError(err)
	}
	return &s, nil
}
