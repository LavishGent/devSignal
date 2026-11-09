package postgres

import (
	"context"

	"devsignal/internal/domain"

	"github.com/jackc/pgx/v5/pgxpool"
)

// LookupRepo is the postgres repo.LookupRepo over the lookup_values table.
type LookupRepo struct {
	pool *pgxpool.Pool
}

// All returns every option grouped by kind in display order.
func (r *LookupRepo) All(ctx context.Context) (domain.Lookups, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT kind, value, label FROM lookup_values
		ORDER BY kind, sort_order, value`)
	if err != nil {
		return nil, mapError(err)
	}
	defer rows.Close()

	out := domain.Lookups{}
	for rows.Next() {
		var kind string
		var o domain.Option
		if err := rows.Scan(&kind, &o.Value, &o.Label); err != nil {
			return nil, err
		}
		out[kind] = append(out[kind], o)
	}
	return out, mapError(rows.Err())
}
