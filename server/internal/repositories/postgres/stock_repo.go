package postgres

import (
	"context"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	"sellin-server/internal/core/ports"
)

type StockRepo struct{ pool *pgxpool.Pool }

func NewStockRepo(pool *pgxpool.Pool) *StockRepo { return &StockRepo{pool: pool} }

const stockMeasures = `coalesce(sum(beginning), 0), coalesce(sum(sell_in), 0),
	coalesce(sum(sell_out), 0), coalesce(sum(ending), 0)`

func (r *StockRepo) MonthsWithData(ctx context.Context, datasetID uuid.UUID, dc string, year int) ([]int, error) {
	c := newCond(datasetID).dc(dc)
	c.eq("year", year)
	return collectInts(ctx, r.pool,
		`SELECT DISTINCT month FROM fact_stock WHERE `+c.where()+` ORDER BY month`, c.args)
}

// Snapshot คืนยอดของ "เดือนเดียว" ไม่ใช่ผลรวมทั้งปี
// หน้า Stock ทั้งหน้าอ่านสถานะ ณ เดือนใดเดือนหนึ่งเสมอ เพราะสต๊อกเป็นค่า ณ จุดเวลา ไม่ใช่ยอดสะสม
func (r *StockRepo) Snapshot(ctx context.Context, datasetID uuid.UUID, dc string, year, month int) (ports.StockTotals, error) {
	c := newCond(datasetID).dc(dc)
	c.eq("year", year)
	c.eq("month", month)

	var t ports.StockTotals
	err := r.pool.QueryRow(ctx,
		`SELECT `+stockMeasures+` FROM fact_stock WHERE `+c.where(), c.args...).
		Scan(&t.Beginning, &t.SellIn, &t.SellOut, &t.Ending)
	return t, err
}

func (r *StockRepo) Monthly(ctx context.Context, datasetID uuid.UUID, dc string, year int) ([]ports.StockMonthPoint, error) {
	c := newCond(datasetID).dc(dc)
	c.eq("year", year)

	rows, err := r.pool.Query(ctx, `
		SELECT month, `+stockMeasures+`
		FROM fact_stock WHERE `+c.where()+`
		GROUP BY month ORDER BY month`, c.args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := []ports.StockMonthPoint{}
	for rows.Next() {
		var p ports.StockMonthPoint
		if err := rows.Scan(&p.Month, &p.Beginning, &p.SellIn, &p.SellOut, &p.Ending); err != nil {
			return nil, err
		}
		out = append(out, p)
	}
	return out, rows.Err()
}

func (r *StockRepo) Products(ctx context.Context, datasetID uuid.UUID, dc string, year, month int, group string) ([]ports.StockProductAgg, error) {
	c := newCond(datasetID).dc(dc)
	c.eq("year", year)
	c.eq("month", month)
	if group != "" {
		c.eq("group_name", group)
	}

	rows, err := r.pool.Query(ctx, `
		SELECT product_code, coalesce(max(product_desc), ''), coalesce(max(group_name), ''),
		       `+stockMeasures+`
		FROM fact_stock WHERE `+c.where()+`
		GROUP BY product_code`, c.args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := []ports.StockProductAgg{}
	for rows.Next() {
		var p ports.StockProductAgg
		if err := rows.Scan(&p.Code, &p.Description, &p.GroupName,
			&p.Beginning, &p.SellIn, &p.SellOut, &p.Ending); err != nil {
			return nil, err
		}
		out = append(out, p)
	}
	return out, rows.Err()
}

// RecentSellOut คืนยอดขายออกย้อนหลังไม่เกิน n เดือนนับจากเดือนที่ระบุ
//
// นับเฉพาะเดือนที่มีแถวข้อมูลจริง เดือนที่ไม่มีแถวเลยจะถูกข้าม ไม่นับเป็นศูนย์
// ถ้านับเป็นศูนย์ ค่าเฉลี่ยจะต่ำผิดปกติแล้วทำให้ Stock Cover สูงเกินจริง
func (r *StockRepo) RecentSellOut(ctx context.Context, datasetID uuid.UUID, dc string, year, month, n int) ([]float64, error) {
	c := newCond(datasetID).dc(dc)
	// นับเดือนแบบต่อเนื่องข้ามปี เพื่อให้ ม.ค. ย้อนไปหา ธ.ค. ปีก่อนได้ถูกต้อง
	base := year*12 + (month - 1)
	from := c.next(base - (n - 1))
	to := c.next(base)

	rows, err := r.pool.Query(ctx, `
		SELECT (year * 12 + month - 1) AS idx, coalesce(sum(sell_out), 0)
		FROM fact_stock
		WHERE `+c.where()+` AND (year * 12 + month - 1) BETWEEN `+from+` AND `+to+`
		GROUP BY idx ORDER BY idx DESC`, c.args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := []float64{}
	for rows.Next() {
		var idx int
		var v float64
		if err := rows.Scan(&idx, &v); err != nil {
			return nil, err
		}
		out = append(out, v)
	}
	return out, rows.Err()
}

func (r *StockRepo) EndingBy(ctx context.Context, datasetID uuid.UUID, dc string, year, month int, groupBy string) ([]ports.Bucket, error) {
	col, err := groupColumn(groupBy)
	if err != nil {
		return nil, err
	}
	c := newCond(datasetID).dc(dc)
	c.eq("year", year)
	c.eq("month", month)

	return collectBuckets(ctx, r.pool, `
		SELECT coalesce(`+col+`, ''), coalesce(`+col+`, ''), coalesce(sum(ending), 0), 0
		FROM fact_stock WHERE `+c.where()+` GROUP BY 1`, c.args)
}
