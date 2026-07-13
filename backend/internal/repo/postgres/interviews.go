package postgres

import (
	"context"

	"devsignal/internal/domain"
	"devsignal/internal/repo"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

// InterviewRepo is the postgres repo.InterviewRepo.
type InterviewRepo struct {
	pool *pgxpool.Pool
}

const interviewColumns = `id, user_id, application_id, stage, scheduled_at,
	duration_minutes, interviewer, location, outcome, prep_notes, feedback,
	created_at, updated_at`

// Create inserts a stamped interview.
func (r *InterviewRepo) Create(ctx context.Context, i *domain.Interview) error {
	_, err := r.pool.Exec(ctx, `
		INSERT INTO interviews (`+interviewColumns+`)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13)`,
		i.ID, i.UserID, i.ApplicationID, i.Stage, i.ScheduledAt,
		i.DurationMinutes, i.Interviewer, i.Location, i.Outcome, i.PrepNotes, i.Feedback,
		i.CreatedAt, i.UpdatedAt)
	return mapError(err)
}

// Get fetches one interview scoped to its owner.
func (r *InterviewRepo) Get(ctx context.Context, userID, id uuid.UUID) (*domain.Interview, error) {
	row := r.pool.QueryRow(ctx, `
		SELECT `+interviewColumns+` FROM interviews
		WHERE user_id = $1 AND id = $2`, userID, id)
	return scanInterview(row)
}

// List returns the user's interviews, filtered and newest-updated first.
func (r *InterviewRepo) List(ctx context.Context, userID uuid.UUID, f repo.InterviewFilter) ([]domain.Interview, error) {
	p := f.Clamp()
	rows, err := r.pool.Query(ctx, `
		SELECT `+interviewColumns+` FROM interviews
		WHERE user_id = $1
		  AND ($2 = '' OR stage = $2)
		  AND ($3 = '' OR outcome = $3)
		  AND ($4 = '00000000-0000-0000-0000-000000000000'::uuid OR application_id = $4)
		  AND ($5 = '' OR interviewer ILIKE $6 OR location ILIKE $6 OR prep_notes ILIKE $6 OR feedback ILIKE $6)
		ORDER BY updated_at DESC, id
		LIMIT $7 OFFSET $8`,
		userID, f.Stage, f.Outcome, f.ApplicationID, f.Q, likePattern(f.Q), p.Limit, p.Offset)
	if err != nil {
		return nil, mapError(err)
	}
	defer rows.Close()

	out := []domain.Interview{}
	for rows.Next() {
		i, err := scanInterview(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, *i)
	}
	return out, mapError(rows.Err())
}

// Update rewrites the mutable columns of an owned interview.
func (r *InterviewRepo) Update(ctx context.Context, i *domain.Interview) error {
	tag, err := r.pool.Exec(ctx, `
		UPDATE interviews SET
			application_id = $3, stage = $4, scheduled_at = $5,
			duration_minutes = $6, interviewer = $7, location = $8,
			outcome = $9, prep_notes = $10, feedback = $11, updated_at = $12
		WHERE user_id = $1 AND id = $2`,
		i.UserID, i.ID, i.ApplicationID, i.Stage, i.ScheduledAt,
		i.DurationMinutes, i.Interviewer, i.Location,
		i.Outcome, i.PrepNotes, i.Feedback, i.UpdatedAt)
	if err != nil {
		return mapError(err)
	}
	if tag.RowsAffected() == 0 {
		return repo.ErrNotFound
	}
	return nil
}

// Delete removes an owned interview.
func (r *InterviewRepo) Delete(ctx context.Context, userID, id uuid.UUID) error {
	tag, err := r.pool.Exec(ctx, `
		DELETE FROM interviews WHERE user_id = $1 AND id = $2`, userID, id)
	if err != nil {
		return mapError(err)
	}
	if tag.RowsAffected() == 0 {
		return repo.ErrNotFound
	}
	return nil
}

func scanInterview(row rowScanner) (*domain.Interview, error) {
	var i domain.Interview
	err := row.Scan(&i.ID, &i.UserID, &i.ApplicationID, &i.Stage, &i.ScheduledAt,
		&i.DurationMinutes, &i.Interviewer, &i.Location, &i.Outcome, &i.PrepNotes, &i.Feedback,
		&i.CreatedAt, &i.UpdatedAt)
	if err != nil {
		return nil, mapError(err)
	}
	return &i, nil
}
