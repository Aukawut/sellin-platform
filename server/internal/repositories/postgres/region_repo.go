package postgres

import (
	"context"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

type RegionRepo struct{ pool *pgxpool.Pool }

func NewRegionRepo(pool *pgxpool.Pool) *RegionRepo { return &RegionRepo{pool: pool} }

func (r *RegionRepo) All(ctx context.Context) (map[string]string, error) {
	rows, err := r.pool.Query(ctx, `SELECT code, region FROM dc_regions`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := map[string]string{}
	for rows.Next() {
		var code, region string
		if err := rows.Scan(&code, &region); err != nil {
			return nil, err
		}
		out[code] = region
	}
	return out, rows.Err()
}

// Set บันทึกภาคของศูนย์ แล้วอัปเดตชุดข้อมูลที่นำเข้าไปแล้วให้ตรงกันทันที
//
// ถ้าไม่อัปเดตย้อนหลัง ผู้ใช้จะแก้ภาคแล้วเห็นแผนที่เหมือนเดิมจนกว่าจะอัปโหลดไฟล์ใหม่
// ซึ่งอ่านได้เหมือนว่าการแก้ไม่มีผล
func (r *RegionRepo) Set(ctx context.Context, code, region string, by uuid.UUID) error {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	if _, err := tx.Exec(ctx, `
		INSERT INTO dc_regions (code, region, updated_by) VALUES ($1, $2, $3)
		ON CONFLICT (code) DO UPDATE
		SET region = excluded.region, updated_at = now(), updated_by = excluded.updated_by`,
		code, region, by); err != nil {
		return err
	}

	if _, err := tx.Exec(ctx,
		`UPDATE dim_customers SET region = $2 WHERE code = $1`, code, region); err != nil {
		return err
	}

	return tx.Commit(ctx)
}

// Codes คืนรหัสศูนย์ทั้งหมดที่เคยปรากฏในชุดข้อมูลที่ยังไม่ถูกลบ พร้อมชื่อและภาคปัจจุบัน
// ใช้สร้างหน้าจัดการภาคของ admin ซึ่งต้องเห็นศูนย์ที่ยังไม่มีภาคกำกับด้วย
func (r *RegionRepo) Codes(ctx context.Context) ([]RegionRow, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT c.code, max(c.name), coalesce(max(g.region), ''), bool_or(c.status = 'Active')
		FROM dim_customers c
		JOIN v_active_datasets d ON d.id = c.dataset_id
		LEFT JOIN dc_regions g ON g.code = c.code
		GROUP BY c.code
		ORDER BY c.code`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := []RegionRow{}
	for rows.Next() {
		var x RegionRow
		if err := rows.Scan(&x.Code, &x.Name, &x.Region, &x.Active); err != nil {
			return nil, err
		}
		out = append(out, x)
	}
	return out, rows.Err()
}

type RegionRow struct {
	Code   string `json:"code"`
	Name   string `json:"name"`
	Region string `json:"region"`
	Active bool   `json:"active"`
}
