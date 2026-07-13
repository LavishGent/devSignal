package postgres

import (
	"context"
	"time"

	"devsignal/internal/domain"

	"github.com/jackc/pgx/v5/pgxpool"
)

// SessionRepo is the postgres repo.SessionRepo.
type SessionRepo struct {
	pool *pgxpool.Pool
}

// Create stores a session.
func (r *SessionRepo) Create(ctx context.Context, s *domain.Session) error {
	_, err := r.pool.Exec(ctx, `
		INSERT INTO sessions (id, user_id, token_hash, expires_at, created_at)
		VALUES ($1, $2, $3, $4, $5)`,
		s.ID, s.UserID, s.TokenHash, s.ExpiresAt, s.CreatedAt)
	return mapError(err)
}

// GetByTokenHash resolves a session by its token hash.
func (r *SessionRepo) GetByTokenHash(ctx context.Context, tokenHash string) (*domain.Session, error) {
	var s domain.Session
	err := r.pool.QueryRow(ctx, `
		SELECT id, user_id, token_hash, expires_at, created_at
		FROM sessions WHERE token_hash = $1`, tokenHash).
		Scan(&s.ID, &s.UserID, &s.TokenHash, &s.ExpiresAt, &s.CreatedAt)
	if err != nil {
		return nil, mapError(err)
	}
	return &s, nil
}

// DeleteByTokenHash revokes a session.
func (r *SessionRepo) DeleteByTokenHash(ctx context.Context, tokenHash string) error {
	_, err := r.pool.Exec(ctx, `DELETE FROM sessions WHERE token_hash = $1`, tokenHash)
	return mapError(err)
}

// DeleteExpired clears out dead sessions (run opportunistically).
func (r *SessionRepo) DeleteExpired(ctx context.Context, now time.Time) error {
	_, err := r.pool.Exec(ctx, `DELETE FROM sessions WHERE expires_at <= $1`, now)
	return mapError(err)
}
