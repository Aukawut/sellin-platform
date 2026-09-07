package postgres

import (
	"context"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	"sellin-server/internal/core/domain"
	"sellin-server/internal/core/ports"
)

type TargetRepo struct{ pool *pgxpool.Pool }

func NewTargetRepo(pool *pgxpool.Pool) *TargetRepo { return &TargetRepo{pool: pool} }

func (r *TargetRepo) Total(ctx context.Context, s domain.Scope) (float64, error) {
	c := newCond(s.DatasetID).scope(s)
	var total float64
	err := r.pool.QueryRow(ctx,
		`SELECT coalesce(sum(target_amount), 0) FROM fact_target WHERE `+c.where(), c.args...).Scan(&total)
	return total, err
}

func (r *TargetRepo) ByMonth(ctx context.Context, datasetID uuid.UUID, dc string, year int) ([12]float64, error) {
	c := newCond(datasetID).dc(dc)
	c.eq("year", year)

	rows, err := r.pool.Query(ctx, `
		SELECT month, coalesce(sum(target_amount), 0)
		FROM fact_target WHERE `+c.where()+` GROUP BY month`, c.args...)
	if err != nil {
		return [12]float64{}, err
	}
	defer rows.Close()

	var out [12]float64
	for rows.Next() {
		var m int
		var v float64
		if err := rows.Scan(&m, &v); err != nil {
			return [12]float64{}, err
		}
		if m >= 1 && m <= 12 {
			out[m-1] = v
		}
	}
	return out, rows.Err()
}

func (r *TargetRepo) ByDC(ctx context.Context, datasetID uuid.UUID, year, month *int) ([]ports.Bucket, error) {
	c := newCondAlias(datasetID, "f").yearMonth(year, month)
	return collectBuckets(ctx, r.pool, `
		SELECT f.customer_code, coalesce(cu.name, max(f.customer_name), f.customer_code),
		       coalesce(sum(f.target_amount), 0), 0
		FROM fact_target f
		LEFT JOIN dim_customers cu ON cu.dataset_id = f.dataset_id AND cu.code = f.customer_code
		WHERE `+c.where()+`
		GROUP BY f.customer_code, cu.name`, c.args)
}

func (r *TargetRepo) ByCategory(ctx context.Context, s domain.Scope) ([]ports.Bucket, error) {
	c := newCond(s.DatasetID).scope(s)
	return collectBuckets(ctx, r.pool, `
		SELECT coalesce(category, ''), coalesce(category, ''), coalesce(sum(target_amount), 0), 0
		FROM fact_target WHERE `+c.where()+` GROUP BY 1`, c.args)
}

func (r *TargetRepo) YearTotal(ctx context.Context, datasetID uuid.UUID, dc string, year int) (float64, error) {
	c := newCond(datasetID).dc(dc)
	c.eq("year", year)
	var total float64
	err := r.pool.QueryRow(ctx,
		`SELECT coalesce(sum(target_amount), 0) FROM fact_target WHERE `+c.where(), c.args...).Scan(&total)
	return total, err
}
