package postgres

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	"sellin-server/internal/core/domain"
	"sellin-server/internal/core/ports"
)

type SellInRepo struct{ pool *pgxpool.Pool }

func NewSellInRepo(pool *pgxpool.Pool) *SellInRepo { return &SellInRepo{pool: pool} }

var sellInSaleOnly = fmt.Sprintf(saleOnly, "status_sale")

func (r *SellInRepo) Summary(ctx context.Context, s domain.Scope) (ports.SellInSummary, error) {
	c := newCond(s.DatasetID).scope(s)
	c.raw(sellInSaleOnly)

	var out ports.SellInSummary
	// นับเลขที่เอกสารแบบไม่ซ้ำ คือจำนวน "ออเดอร์" ตามนิยามของ dashboard เดิม
	// ไม่ใช่จำนวนบรรทัดสินค้า ซึ่งหนึ่งออเดอร์มีได้หลายบรรทัด
	err := r.pool.QueryRow(ctx, `
		SELECT coalesce(sum(revenue), 0), coalesce(sum(cartons), 0),
		       count(DISTINCT nullif(doc_no, ''))
		FROM fact_sellin WHERE `+c.where(), c.args...).
		Scan(&out.Revenue, &out.Cartons, &out.Orders)
	return out, err
}

func (r *SellInRepo) RevenueByMonth(ctx context.Context, datasetID uuid.UUID, dc string, year int) ([12]float64, [12]bool, error) {
	c := newCond(datasetID).dc(dc)
	c.eq("year", year)
	c.raw(sellInSaleOnly)

	rows, err := r.pool.Query(ctx, `
		SELECT month, coalesce(sum(revenue), 0)
		FROM fact_sellin WHERE `+c.where()+`
		GROUP BY month`, c.args...)
	if err != nil {
		return [12]float64{}, [12]bool{}, err
	}
	defer rows.Close()

	values, present := map[int]float64{}, map[int]bool{}
	for rows.Next() {
		var m int
		var v float64
		if err := rows.Scan(&m, &v); err != nil {
			return [12]float64{}, [12]bool{}, err
		}
		if m >= 1 && m <= 12 {
			values[m], present[m] = v, true
		}
	}
	vals, has := monthArray(values, present)
	return vals, has, rows.Err()
}

func (r *SellInRepo) RevenueByDay(ctx context.Context, datasetID uuid.UUID, dc string, year, month int) (map[int]float64, error) {
	c := newCond(datasetID).dc(dc)
	c.eq("year", year)
	c.eq("month", month)
	c.raw(sellInSaleOnly)

	rows, err := r.pool.Query(ctx, `
		SELECT day, coalesce(sum(revenue), 0)
		FROM fact_sellin WHERE `+c.where()+`
		GROUP BY day`, c.args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := map[int]float64{}
	for rows.Next() {
		var d int
		var v float64
		if err := rows.Scan(&d, &v); err != nil {
			return nil, err
		}
		out[d] = v
	}
	return out, rows.Err()
}

// RevenueByDC จงใจไม่รับฟิลเตอร์ศูนย์ เพราะตารางจัดอันดับต้องแสดงทุกศูนย์เสมอ
// การเลือกศูนย์ในหน้าเว็บมีผลแค่ทำให้แถวนั้นถูกเน้น ไม่ได้ตัดศูนย์อื่นออก
func (r *SellInRepo) RevenueByDC(ctx context.Context, datasetID uuid.UUID, year, month *int) ([]ports.Bucket, error) {
	c := newCondAlias(datasetID, "f").yearMonth(year, month)
	c.raw(fmt.Sprintf(saleOnly, "f.status_sale"))

	return collectBuckets(ctx, r.pool, `
		SELECT f.customer_code, coalesce(cu.name, f.customer_code), coalesce(sum(f.revenue), 0), 0
		FROM fact_sellin f
		LEFT JOIN dim_customers cu ON cu.dataset_id = f.dataset_id AND cu.code = f.customer_code
		WHERE `+c.where()+`
		GROUP BY f.customer_code, cu.name`, c.args)
}

func (r *SellInRepo) CartonsBy(ctx context.Context, s domain.Scope, groupBy string) ([]ports.Bucket, error) {
	col, err := groupColumn(groupBy)
	if err != nil {
		return nil, err
	}
	c := newCond(s.DatasetID).scope(s)
	c.raw(sellInSaleOnly)

	return collectBuckets(ctx, r.pool, `
		SELECT coalesce(`+col+`, ''), coalesce(`+col+`, ''), coalesce(sum(cartons), 0), 0
		FROM fact_sellin WHERE `+c.where()+`
		GROUP BY 1`, c.args)
}

func (r *SellInRepo) CartonsTotal(ctx context.Context, datasetID uuid.UUID, dc string, year, month int) (float64, error) {
	c := newCond(datasetID).dc(dc)
	c.eq("year", year)
	c.eq("month", month)
	c.raw(sellInSaleOnly)

	var total float64
	err := r.pool.QueryRow(ctx,
		`SELECT coalesce(sum(cartons), 0) FROM fact_sellin WHERE `+c.where(), c.args...).Scan(&total)
	return total, err
}

func (r *SellInRepo) MonthsWithData(ctx context.Context, datasetID uuid.UUID, dc string, year int) ([]int, error) {
	c := newCond(datasetID).dc(dc)
	c.eq("year", year)
	return collectInts(ctx, r.pool,
		`SELECT DISTINCT month FROM fact_sellin WHERE `+c.where()+` ORDER BY month`, c.args)
}

func (r *SellInRepo) TrackingByGroup(ctx context.Context, s domain.Scope) ([]ports.Bucket, error) {
	c := newCond(s.DatasetID).scope(s)

	// เป้าหมายและยอดจริงมาคู่กันเสมอในกราฟนี้ จึงดึงพร้อมกันในรอบเดียว
	return collectBuckets(ctx, r.pool, `
		SELECT coalesce(group_name, ''), coalesce(group_name, ''),
		       coalesce(sum(target_qty), 0), coalesce(sum(cartons), 0)
		FROM fact_tracking_nsfs WHERE `+c.where()+`
		GROUP BY 1`, c.args)
}
