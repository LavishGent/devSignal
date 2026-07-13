package postgres

import (
	"context"

	"devsignal/internal/domain"
	"devsignal/internal/repo"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

// ContactRepo is the postgres repo.ContactRepo.
type ContactRepo struct {
	pool *pgxpool.Pool
}

const contactColumns = `id, user_id, name, company, role, relationship, email,
	phone, linkedin_url, last_contacted_on, next_followup_on, notes, created_at, updated_at`

// Create inserts a stamped contact.
func (r *ContactRepo) Create(ctx context.Context, c *domain.Contact) error {
	_, err := r.pool.Exec(ctx, `
		INSERT INTO contacts (`+contactColumns+`)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14)`,
		c.ID, c.UserID, c.Name, c.Company, c.Role, c.Relationship, c.Email,
		c.Phone, c.LinkedinURL, c.LastContactedOn, c.NextFollowupOn, c.Notes,
		c.CreatedAt, c.UpdatedAt)
	return mapError(err)
}

// Get fetches one contact scoped to its owner.
func (r *ContactRepo) Get(ctx context.Context, userID, id uuid.UUID) (*domain.Contact, error) {
	row := r.pool.QueryRow(ctx, `
		SELECT `+contactColumns+` FROM contacts
		WHERE user_id = $1 AND id = $2`, userID, id)
	return scanContact(row)
}

// List returns the user's contacts, filtered and newest-updated first.
func (r *ContactRepo) List(ctx context.Context, userID uuid.UUID, f repo.ContactFilter) ([]domain.Contact, error) {
	p := f.Clamp()
	rows, err := r.pool.Query(ctx, `
		SELECT `+contactColumns+` FROM contacts
		WHERE user_id = $1
		  AND ($2 = '' OR relationship = $2)
		  AND ($3 = '' OR name ILIKE $4 OR company ILIKE $4 OR role ILIKE $4 OR email ILIKE $4 OR notes ILIKE $4)
		ORDER BY updated_at DESC, id
		LIMIT $5 OFFSET $6`,
		userID, f.Relationship, f.Q, likePattern(f.Q), p.Limit, p.Offset)
	if err != nil {
		return nil, mapError(err)
	}
	defer rows.Close()

	out := []domain.Contact{}
	for rows.Next() {
		c, err := scanContact(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, *c)
	}
	return out, mapError(rows.Err())
}

// Update rewrites the mutable columns of an owned contact.
func (r *ContactRepo) Update(ctx context.Context, c *domain.Contact) error {
	tag, err := r.pool.Exec(ctx, `
		UPDATE contacts SET
			name = $3, company = $4, role = $5, relationship = $6,
			email = $7, phone = $8, linkedin_url = $9, last_contacted_on = $10,
			next_followup_on = $11, notes = $12, updated_at = $13
		WHERE user_id = $1 AND id = $2`,
		c.UserID, c.ID, c.Name, c.Company, c.Role, c.Relationship,
		c.Email, c.Phone, c.LinkedinURL, c.LastContactedOn, c.NextFollowupOn,
		c.Notes, c.UpdatedAt)
	if err != nil {
		return mapError(err)
	}
	if tag.RowsAffected() == 0 {
		return repo.ErrNotFound
	}
	return nil
}

// Delete removes an owned contact.
func (r *ContactRepo) Delete(ctx context.Context, userID, id uuid.UUID) error {
	tag, err := r.pool.Exec(ctx, `
		DELETE FROM contacts WHERE user_id = $1 AND id = $2`, userID, id)
	if err != nil {
		return mapError(err)
	}
	if tag.RowsAffected() == 0 {
		return repo.ErrNotFound
	}
	return nil
}

func scanContact(row rowScanner) (*domain.Contact, error) {
	var c domain.Contact
	err := row.Scan(&c.ID, &c.UserID, &c.Name, &c.Company, &c.Role, &c.Relationship, &c.Email,
		&c.Phone, &c.LinkedinURL, &c.LastContactedOn, &c.NextFollowupOn, &c.Notes,
		&c.CreatedAt, &c.UpdatedAt)
	if err != nil {
		return nil, mapError(err)
	}
	return &c, nil
}
