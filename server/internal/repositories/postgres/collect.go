package postgres

import (
	"context"
	"github.com/jackc/pgx/v5/pgxpool"

	"sellin-server/internal/core/ports"
)

// collectBuckets อ่านผลลัพธ์รูปแบบ (key, label, value, secondary) ซึ่งเป็นรูปแบบที่ query ส่วนใหญ่ใช้
func collectBuckets(ctx context.Context, pool *pgxpool.Pool, sql string, args []any) ([]ports.Bucket, error) {
	rows, err := pool.Query(ctx, sql, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := []ports.Bucket{}
	for rows.Next() {
		var b ports.Bucket
		if err := rows.Scan(&b.Key, &b.Label, &b.Value, &b.Secondary); err != nil {
			return nil, err
		}
		out = append(out, b)
	}
	return out, rows.Err()
}

func collectInts(ctx context.Context, pool *pgxpool.Pool, sql string, args []any) ([]int, error) {
	rows, err := pool.Query(ctx, sql, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := []int{}
	for rows.Next() {
		var v int
		if err := rows.Scan(&v); err != nil {
			return nil, err
		}
		out = append(out, v)
	}
	return out, rows.Err()
}

func collectStrings(ctx context.Context, pool *pgxpool.Pool, sql string, args []any) ([]string, error) {
	rows, err := pool.Query(ctx, sql, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := []string{}
	for rows.Next() {
		var v string
		if err := rows.Scan(&v); err != nil {
			return nil, err
		}
		out = append(out, v)
	}
	return out, rows.Err()
}
