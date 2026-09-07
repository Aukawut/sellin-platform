package postgres

import (
	"context"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	"sellin-server/internal/core/domain"
	"sellin-server/internal/core/ports"
)

type MasterRepo struct{ pool *pgxpool.Pool }

func NewMasterRepo(pool *pgxpool.Pool) *MasterRepo { return &MasterRepo{pool: pool} }

// Meta ดึงทุกอย่างที่ต้องใช้สร้างตัวเลือกใน dropdown และหาปีฐานของแต่ละหน้า
//
// ปีถูกดึงแยกตามแหล่งข้อมูล เพราะแต่ละ sheet มีช่วงปีไม่เท่ากัน
// เช่น Target มีแค่ปี 2569 ขณะที่ Sell-In มีทั้ง 2568 และ 2569
func (r *MasterRepo) Meta(ctx context.Context, datasetID uuid.UUID) (ports.DatasetMeta, error) {
	var m ports.DatasetMeta

	customers, err := r.customers(ctx, datasetID)
	if err != nil {
		return m, err
	}
	m.Customers = customers

	products, err := r.products(ctx, datasetID)
	if err != nil {
		return m, err
	}
	m.Products = products

	for _, src := range []struct {
		table string
		dest  *[]int
	}{
		{"fact_sellin", &m.SellInYears},
		{"fact_sellout", &m.SellOutYears},
		{"fact_stock", &m.StockYears},
		{"fact_target", &m.TargetYears},
	} {
		years, err := collectInts(ctx, r.pool,
			`SELECT DISTINCT year FROM `+src.table+` WHERE dataset_id = $1 ORDER BY year`,
			[]any{datasetID})
		if err != nil {
			return m, err
		}
		*src.dest = years
	}

	for _, src := range []struct {
		table string
		dest  *[]string
	}{
		{"fact_sellout", &m.SellOutGroups},
		{"fact_stock", &m.StockGroups},
	} {
		groups, err := collectStrings(ctx, r.pool,
			`SELECT DISTINCT group_name FROM `+src.table+`
			 WHERE dataset_id = $1 AND coalesce(group_name, '') <> ''
			 ORDER BY group_name`, []any{datasetID})
		if err != nil {
			return m, err
		}
		*src.dest = groups
	}

	return m, nil
}

func (r *MasterRepo) customers(ctx context.Context, datasetID uuid.UUID) ([]domain.Customer, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT code, name, status, provinces, coalesce(region, '')
		FROM dim_customers WHERE dataset_id = $1
		ORDER BY name`, datasetID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := []domain.Customer{}
	for rows.Next() {
		var c domain.Customer
		if err := rows.Scan(&c.Code, &c.Name, &c.Status, &c.Provinces, &c.Region); err != nil {
			return nil, err
		}
		out = append(out, c)
	}
	return out, rows.Err()
}

// products คืนเป็น map เพราะผู้ใช้หลักคือตาราง Stock ที่ต้องหาสถานะสินค้าทีละรหัส
func (r *MasterRepo) products(ctx context.Context, datasetID uuid.UUID) (map[string]domain.Product, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT code, coalesce(description, ''), coalesce(group_name, ''), coalesce(category, ''),
		       coalesce(innerbox, 0), coalesce(price_unit, 0), coalesce(price_carton, 0),
		       coalesce(status, '')
		FROM dim_products WHERE dataset_id = $1`, datasetID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := map[string]domain.Product{}
	for rows.Next() {
		var p domain.Product
		if err := rows.Scan(&p.Code, &p.Description, &p.GroupName, &p.Category,
			&p.Innerbox, &p.PriceUnit, &p.PriceCarton, &p.Status); err != nil {
			return nil, err
		}
		out[p.Code] = p
	}
	return out, rows.Err()
}
