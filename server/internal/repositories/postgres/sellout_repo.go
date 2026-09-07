package postgres

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	"sellin-server/internal/core/domain"
	"sellin-server/internal/core/ports"
)

type SellOutRepo struct{ pool *pgxpool.Pool }

func NewSellOutRepo(pool *pgxpool.Pool) *SellOutRepo { return &SellOutRepo{pool: pool} }

var sellOutSaleOnly = fmt.Sprintf(saleOnly, "status")

func (r *SellOutRepo) Summary(ctx context.Context, s domain.Scope) (ports.SellOutSummary, error) {
	c := newCond(s.DatasetID).scope(s)
	c.raw(sellOutSaleOnly)

	var out ports.SellOutSummary
	err := r.pool.QueryRow(ctx, `
		SELECT coalesce(sum(revenue), 0), coalesce(sum(cartons), 0)
		FROM fact_sellout WHERE `+c.where(), c.args...).Scan(&out.Revenue, &out.Cartons)
	return out, err
}

func (r *SellOutRepo) RevenueByMonth(ctx context.Context, datasetID uuid.UUID, dc string, year int) ([12]float64, [12]bool, error) {
	c := newCond(datasetID).dc(dc)
	c.eq("year", year)
	c.raw(sellOutSaleOnly)

	rows, err := r.pool.Query(ctx, `
		SELECT month, coalesce(sum(revenue), 0)
		FROM fact_sellout WHERE `+c.where()+` GROUP BY month`, c.args...)
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

func (r *SellOutRepo) RevenueByDC(ctx context.Context, datasetID uuid.UUID, year, month *int) ([]ports.Bucket, error) {
	c := newCondAlias(datasetID, "f").yearMonth(year, month)
	c.raw(fmt.Sprintf(saleOnly, "f.status"))

	return collectBuckets(ctx, r.pool, `
		SELECT f.customer_code, coalesce(cu.name, f.customer_code), coalesce(sum(f.revenue), 0), 0
		FROM fact_sellout f
		LEFT JOIN dim_customers cu ON cu.dataset_id = f.dataset_id AND cu.code = f.customer_code
		WHERE `+c.where()+`
		GROUP BY f.customer_code, cu.name`, c.args)
}

func (r *SellOutRepo) RevenueBy(ctx context.Context, s domain.Scope, groupBy string) ([]ports.Bucket, error) {
	return r.aggregateBy(ctx, s, groupBy, "revenue")
}

func (r *SellOutRepo) CartonsBy(ctx context.Context, s domain.Scope, groupBy string) ([]ports.Bucket, error) {
	return r.aggregateBy(ctx, s, groupBy, "cartons")
}

// aggregateBy รับชื่อคอลัมน์ค่าจากภายในแพ็กเกจเท่านั้น ส่วนคอลัมน์จัดกลุ่มผ่านการตรวจของ groupColumn
func (r *SellOutRepo) aggregateBy(ctx context.Context, s domain.Scope, groupBy, measure string) ([]ports.Bucket, error) {
	col, err := groupColumn(groupBy)
	if err != nil {
		return nil, err
	}
	c := newCond(s.DatasetID).scope(s)
	c.raw(sellOutSaleOnly)

	return collectBuckets(ctx, r.pool, `
		SELECT coalesce(`+col+`, ''), coalesce(`+col+`, ''), coalesce(sum(`+measure+`), 0), 0
		FROM fact_sellout WHERE `+c.where()+` GROUP BY 1`, c.args)
}

func (r *SellOutRepo) CartonsByForYear(ctx context.Context, datasetID uuid.UUID, dc string, month *int, year int, groupBy string) ([]ports.Bucket, error) {
	col, err := groupColumn(groupBy)
	if err != nil {
		return nil, err
	}
	c := newCond(datasetID).dc(dc)
	c.eq("year", year)
	if month != nil {
		c.eq("month", *month)
	}
	c.raw(sellOutSaleOnly)

	return collectBuckets(ctx, r.pool, `
		SELECT coalesce(`+col+`, ''), coalesce(`+col+`, ''), coalesce(sum(cartons), 0), 0
		FROM fact_sellout WHERE `+c.where()+` GROUP BY 1`, c.args)
}

func (r *SellOutRepo) Products(ctx context.Context, s domain.Scope, group string) ([]ports.ProductAgg, error) {
	c := newCond(s.DatasetID).scope(s)
	c.raw(sellOutSaleOnly)
	if group != "" {
		c.eq("group_name", group)
	}

	rows, err := r.pool.Query(ctx, `
		SELECT product_code,
		       coalesce(max(product_desc), ''), coalesce(max(group_name), ''), coalesce(max(category), ''),
		       coalesce(sum(cartons), 0), coalesce(sum(revenue), 0)
		FROM fact_sellout WHERE `+c.where()+`
		GROUP BY product_code`, c.args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := []ports.ProductAgg{}
	for rows.Next() {
		var p ports.ProductAgg
		if err := rows.Scan(&p.Code, &p.Description, &p.GroupName, &p.Category, &p.Cartons, &p.Revenue); err != nil {
			return nil, err
		}
		out = append(out, p)
	}
	return out, rows.Err()
}

// RevenueByGroupMonthly คืนธงเดือนที่มีข้อมูลมาด้วย
// เพื่อให้กราฟเว้นช่วงหลังเดือนสุดท้ายที่มียอด แทนที่จะลากเส้นลงไปแตะศูนย์
// ซึ่งจะอ่านเหมือนว่ายอดขายตกเป็นศูนย์จริง ๆ
func (r *SellOutRepo) RevenueByGroupMonthly(ctx context.Context, datasetID uuid.UUID, dc string, year int) (map[string][12]float64, [12]bool, error) {
	c := newCond(datasetID).dc(dc)
	c.eq("year", year)
	c.raw(sellOutSaleOnly)

	rows, err := r.pool.Query(ctx, `
		SELECT coalesce(group_name, ''), month, coalesce(sum(revenue), 0)
		FROM fact_sellout WHERE `+c.where()+`
		GROUP BY 1, month`, c.args...)
	if err != nil {
		return nil, [12]bool{}, err
	}
	defer rows.Close()

	out := map[string][12]float64{}
	var present [12]bool
	for rows.Next() {
		var g string
		var m int
		var v float64
		if err := rows.Scan(&g, &m, &v); err != nil {
			return nil, [12]bool{}, err
		}
		if m < 1 || m > 12 {
			continue
		}
		arr := out[g]
		arr[m-1] = v
		out[g] = arr
		present[m-1] = true
	}
	return out, present, rows.Err()
}

// RevenueByRegion จัดกลุ่มตามภาคของศูนย์กระจายสินค้า ซึ่งเก็บอยู่ใน dim_customers
// dashboard เดิม hardcode แผนที่ภาคไว้ใน JavaScript ที่นี่อ่านจากข้อมูลจึงแก้ไขได้
func (r *SellOutRepo) RevenueByRegion(ctx context.Context, s domain.Scope) ([]ports.Bucket, error) {
	c := newCondAlias(s.DatasetID, "f").scope(s)
	c.raw(fmt.Sprintf(saleOnly, "f.status"))
	return r.regionQuery(ctx, c)
}

func (r *SellOutRepo) RevenueByRegionForYear(ctx context.Context, datasetID uuid.UUID, dc string, month *int, year int) ([]ports.Bucket, error) {
	c := newCondAlias(datasetID, "f").dc(dc)
	c.eq("year", year)
	if month != nil {
		c.eq("month", *month)
	}
	c.raw(fmt.Sprintf(saleOnly, "f.status"))
	return r.regionQuery(ctx, c)
}

func (r *SellOutRepo) regionQuery(ctx context.Context, c *cond) ([]ports.Bucket, error) {
	unknown := c.next("ไม่ระบุภาค")
	return collectBuckets(ctx, r.pool, `
		SELECT coalesce(cu.region, `+unknown+`), coalesce(cu.region, `+unknown+`),
		       coalesce(sum(f.revenue), 0), 0
		FROM fact_sellout f
		LEFT JOIN dim_customers cu ON cu.dataset_id = f.dataset_id AND cu.code = f.customer_code
		WHERE `+c.where()+`
		GROUP BY 1`, c.args)
}

func (r *SellOutRepo) MonthsWithData(ctx context.Context, datasetID uuid.UUID, dc string, year int) ([]int, error) {
	c := newCond(datasetID).dc(dc)
	c.eq("year", year)
	return collectInts(ctx, r.pool,
		`SELECT DISTINCT month FROM fact_sellout WHERE `+c.where()+` ORDER BY month`, c.args)
}

// YearTotal คืนยอดทั้งปีพร้อมจำนวนเดือนที่มีข้อมูล ซึ่งเป็นตัวหารของสูตรคาดการณ์
func (r *SellOutRepo) YearTotal(ctx context.Context, datasetID uuid.UUID, dc string, year int) (float64, int, error) {
	c := newCond(datasetID).dc(dc)
	c.eq("year", year)
	c.raw(sellOutSaleOnly)

	var total float64
	var months int
	err := r.pool.QueryRow(ctx, `
		SELECT coalesce(sum(revenue), 0), count(DISTINCT month)
		FROM fact_sellout WHERE `+c.where(), c.args...).Scan(&total, &months)
	return total, months, err
}
