package postgres

import (
	"context"

	"devsignal/internal/domain"
	"devsignal/internal/repo"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

// CertificationRepo is the postgres repo.CertificationRepo.
type CertificationRepo struct {
	pool *pgxpool.Pool
}

// NewCertificationRepo builds the repo on the shared pool.
func NewCertificationRepo(pool *pgxpool.Pool) *CertificationRepo {
	return &CertificationRepo{pool: pool}
}

const certificationColumns = `id, user_id, name, issuer, status, earned_on,
	expires_on, credential_id, credential_url, notes, created_at, updated_at`

// Create inserts a stamped certification.
func (r *CertificationRepo) Create(ctx context.Context, c *domain.Certification) error {
	_, err := r.pool.Exec(ctx, `
		INSERT INTO certifications (`+certificationColumns+`)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12)`,
		c.ID, c.UserID, c.Name, c.Issuer, c.Status, c.EarnedOn,
		c.ExpiresOn, c.CredentialID, c.CredentialURL, c.Notes,
		c.CreatedAt, c.UpdatedAt)
	return mapError(err)
}

// Get fetches one certification scoped to its owner.
func (r *CertificationRepo) Get(ctx context.Context, userID, id uuid.UUID) (*domain.Certification, error) {
	row := r.pool.QueryRow(ctx, `
		SELECT `+certificationColumns+` FROM certifications
		WHERE user_id = $1 AND id = $2`, userID, id)
	return scanCertification(row)
}

// List returns the user's certifications, filtered and newest-updated first.
func (r *CertificationRepo) List(ctx context.Context, userID uuid.UUID, f repo.CertificationFilter) ([]domain.Certification, error) {
	p := f.Clamp()
	rows, err := r.pool.Query(ctx, `
		SELECT `+certificationColumns+` FROM certifications
		WHERE user_id = $1
		  AND ($2 = '' OR status = $2)
		  AND ($3 = '' OR name ILIKE $4 OR issuer ILIKE $4 OR notes ILIKE $4)
		ORDER BY updated_at DESC, id
		LIMIT $5 OFFSET $6`,
		userID, f.Status, f.Q, likePattern(f.Q), p.Limit, p.Offset)
	if err != nil {
		return nil, mapError(err)
	}
	defer rows.Close()

	out := []domain.Certification{}
	for rows.Next() {
		c, err := scanCertification(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, *c)
	}
	return out, mapError(rows.Err())
}

// Update rewrites the mutable columns of an owned certification.
func (r *CertificationRepo) Update(ctx context.Context, c *domain.Certification) error {
	tag, err := r.pool.Exec(ctx, `
		UPDATE certifications SET
			name = $3, issuer = $4, status = $5, earned_on = $6,
			expires_on = $7, credential_id = $8, credential_url = $9,
			notes = $10, updated_at = $11
		WHERE user_id = $1 AND id = $2`,
		c.UserID, c.ID, c.Name, c.Issuer, c.Status, c.EarnedOn,
		c.ExpiresOn, c.CredentialID, c.CredentialURL, c.Notes,
		c.UpdatedAt)
	if err != nil {
		return mapError(err)
	}
	if tag.RowsAffected() == 0 {
		return repo.ErrNotFound
	}
	return nil
}

// Delete removes an owned certification.
func (r *CertificationRepo) Delete(ctx context.Context, userID, id uuid.UUID) error {
	tag, err := r.pool.Exec(ctx, `
		DELETE FROM certifications WHERE user_id = $1 AND id = $2`, userID, id)
	if err != nil {
		return mapError(err)
	}
	if tag.RowsAffected() == 0 {
		return repo.ErrNotFound
	}
	return nil
}

func scanCertification(row rowScanner) (*domain.Certification, error) {
	var c domain.Certification
	err := row.Scan(&c.ID, &c.UserID, &c.Name, &c.Issuer, &c.Status, &c.EarnedOn,
		&c.ExpiresOn, &c.CredentialID, &c.CredentialURL, &c.Notes,
		&c.CreatedAt, &c.UpdatedAt)
	if err != nil {
		return nil, mapError(err)
	}
	return &c, nil
}
