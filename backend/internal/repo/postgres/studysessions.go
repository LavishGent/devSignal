package postgres

import (
	"context"

	"devsignal/internal/domain"
	"devsignal/internal/repo"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

// StudySessionRepo is the postgres repo.StudySessionRepo.
type StudySessionRepo struct {
	pool *pgxpool.Pool
}

const studySessionColumns = `id, user_id, skill_id, topic, minutes, studied_on,
	resource, notes, created_at, updated_at`

// Create inserts a stamped study session.
func (r *StudySessionRepo) Create(ctx context.Context, s *domain.StudySession) error {
	_, err := r.pool.Exec(ctx, `
		INSERT INTO study_sessions (`+studySessionColumns+`)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)`,
		s.ID, s.UserID, s.SkillID, s.Topic, s.Minutes, s.StudiedOn,
		s.Resource, s.Notes, s.CreatedAt, s.UpdatedAt)
	return mapError(err)
}

// Get fetches one study session scoped to its owner.
func (r *StudySessionRepo) Get(ctx context.Context, userID, id uuid.UUID) (*domain.StudySession, error) {
	row := r.pool.QueryRow(ctx, `
		SELECT `+studySessionColumns+` FROM study_sessions
		WHERE user_id = $1 AND id = $2`, userID, id)
	return scanStudySession(row)
}

// List returns the user's study sessions, filtered and newest-updated first.
func (r *StudySessionRepo) List(ctx context.Context, userID uuid.UUID, f repo.StudySessionFilter) ([]domain.StudySession, error) {
	p := f.Clamp()
	rows, err := r.pool.Query(ctx, `
		SELECT `+studySessionColumns+` FROM study_sessions
		WHERE user_id = $1
		  AND ($2::uuid IS NULL OR skill_id = $2)
		  AND ($3 = '' OR topic ILIKE $4 OR resource ILIKE $4 OR notes ILIKE $4)
		ORDER BY updated_at DESC, id
		LIMIT $5 OFFSET $6`,
		userID, f.SkillID, f.Q, likePattern(f.Q), p.Limit, p.Offset)
	if err != nil {
		return nil, mapError(err)
	}
	defer rows.Close()

	out := []domain.StudySession{}
	for rows.Next() {
		s, err := scanStudySession(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, *s)
	}
	return out, mapError(rows.Err())
}

// Update rewrites the mutable columns of an owned study session.
func (r *StudySessionRepo) Update(ctx context.Context, s *domain.StudySession) error {
	tag, err := r.pool.Exec(ctx, `
		UPDATE study_sessions SET
			skill_id = $3, topic = $4, minutes = $5, studied_on = $6,
			resource = $7, notes = $8, updated_at = $9
		WHERE user_id = $1 AND id = $2`,
		s.UserID, s.ID, s.SkillID, s.Topic, s.Minutes, s.StudiedOn,
		s.Resource, s.Notes, s.UpdatedAt)
	if err != nil {
		return mapError(err)
	}
	if tag.RowsAffected() == 0 {
		return repo.ErrNotFound
	}
	return nil
}

// Delete removes an owned study session.
func (r *StudySessionRepo) Delete(ctx context.Context, userID, id uuid.UUID) error {
	tag, err := r.pool.Exec(ctx, `
		DELETE FROM study_sessions WHERE user_id = $1 AND id = $2`, userID, id)
	if err != nil {
		return mapError(err)
	}
	if tag.RowsAffected() == 0 {
		return repo.ErrNotFound
	}
	return nil
}

func scanStudySession(row rowScanner) (*domain.StudySession, error) {
	var s domain.StudySession
	err := row.Scan(&s.ID, &s.UserID, &s.SkillID, &s.Topic, &s.Minutes, &s.StudiedOn,
		&s.Resource, &s.Notes, &s.CreatedAt, &s.UpdatedAt)
	if err != nil {
		return nil, mapError(err)
	}
	return &s, nil
}
