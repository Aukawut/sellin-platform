package postgres

import (
	"context"

	"github.com/jackc/pgx/v5/pgxpool"

	"sellin-server/internal/core/domain"
)

// ExportRepo ดึงแถวดิบตามขอบเขตที่เลือก ต่างจาก repository อื่นที่คืนเฉพาะยอดรวม
//
// จำกัดจำนวนแถวสูงสุดไว้ เพราะการส่งออกทั้งชุดโดยไม่กรองอะไรเลยอาจกินหน่วยความจำมาก
// ถ้าชนเพดานผู้ใช้จะเห็นในไฟล์ว่าข้อมูลถูกตัด ไม่ใช่ได้ไฟล์ที่ขาดไปเงียบ ๆ
type ExportRepo struct{ pool *pgxpool.Pool }

func NewExportRepo(pool *pgxpool.Pool) *ExportRepo { return &ExportRepo{pool: pool} }

const exportLimit = 200000

func (r *ExportRepo) SellIn(ctx context.Context, s domain.Scope) ([]domain.SellInRow, error) {
	c := newCond(s.DatasetID).scope(s)
	limit := c.next(exportLimit)

	rows, err := r.pool.Query(ctx, `
		SELECT year, month, coalesce(day, 0), coalesce(doc_no, ''), customer_code,
		       coalesce(product_code, ''), coalesce(product_desc, ''),
		       coalesce(group_name, ''), coalesce(category, ''),
		       coalesce(qty, 0), coalesce(unit, ''), coalesce(cartons, 0),
		       coalesce(price_unit, 0), coalesce(revenue, 0), coalesce(status_sale, '')
		FROM fact_sellin WHERE `+c.where()+`
		ORDER BY year, month, day, doc_no LIMIT `+limit, c.args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := []domain.SellInRow{}
	for rows.Next() {
		var x domain.SellInRow
		if err := rows.Scan(&x.Year, &x.Month, &x.Day, &x.DocNo, &x.CustomerCode,
			&x.ProductCode, &x.ProductDesc, &x.GroupName, &x.Category,
			&x.Qty, &x.Unit, &x.Cartons, &x.PriceUnit, &x.Revenue, &x.StatusSale); err != nil {
			return nil, err
		}
		out = append(out, x)
	}
	return out, rows.Err()
}

func (r *ExportRepo) SellOut(ctx context.Context, s domain.Scope) ([]domain.SellOutRow, error) {
	c := newCond(s.DatasetID).scope(s)
	limit := c.next(exportLimit)

	rows, err := r.pool.Query(ctx, `
		SELECT year, month, customer_code, coalesce(product_code, ''), coalesce(product_desc, ''),
		       coalesce(group_name, ''), coalesce(category, ''), coalesce(cartons, 0),
		       coalesce(unit, ''), coalesce(price_unit, 0), coalesce(revenue, 0), coalesce(status, '')
		FROM fact_sellout WHERE `+c.where()+`
		ORDER BY year, month, customer_code, product_code LIMIT `+limit, c.args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := []domain.SellOutRow{}
	for rows.Next() {
		var x domain.SellOutRow
		if err := rows.Scan(&x.Year, &x.Month, &x.CustomerCode, &x.ProductCode, &x.ProductDesc,
			&x.GroupName, &x.Category, &x.Cartons, &x.Unit, &x.PriceUnit, &x.Revenue, &x.Status); err != nil {
			return nil, err
		}
		out = append(out, x)
	}
	return out, rows.Err()
}

func (r *ExportRepo) Stock(ctx context.Context, s domain.Scope) ([]domain.StockRow, error) {
	c := newCond(s.DatasetID).scope(s)
	limit := c.next(exportLimit)

	rows, err := r.pool.Query(ctx, `
		SELECT year, month, customer_code, coalesce(product_code, ''), coalesce(product_desc, ''),
		       coalesce(group_name, ''), coalesce(category, ''),
		       coalesce(beginning, 0), coalesce(sell_in, 0), coalesce(sell_out, 0), coalesce(ending, 0)
		FROM fact_stock WHERE `+c.where()+`
		ORDER BY year, month, customer_code, product_code LIMIT `+limit, c.args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := []domain.StockRow{}
	for rows.Next() {
		var x domain.StockRow
		if err := rows.Scan(&x.Year, &x.Month, &x.CustomerCode, &x.ProductCode, &x.ProductDesc,
			&x.GroupName, &x.Category, &x.Beginning, &x.SellIn, &x.SellOut, &x.Ending); err != nil {
			return nil, err
		}
		out = append(out, x)
	}
	return out, rows.Err()
}

func (r *ExportRepo) Targets(ctx context.Context, s domain.Scope) ([]domain.TargetRow, error) {
	c := newCond(s.DatasetID).scope(s)
	limit := c.next(exportLimit)

	rows, err := r.pool.Query(ctx, `
		SELECT year, month, customer_code, coalesce(customer_name, ''),
		       coalesce(category, ''), coalesce(target_amount, 0)
		FROM fact_target WHERE `+c.where()+`
		ORDER BY year, month, customer_code LIMIT `+limit, c.args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := []domain.TargetRow{}
	for rows.Next() {
		var x domain.TargetRow
		if err := rows.Scan(&x.Year, &x.Month, &x.CustomerCode, &x.CustomerName,
			&x.Category, &x.TargetAmount); err != nil {
			return nil, err
		}
		out = append(out, x)
	}
	return out, rows.Err()
}

func (r *ExportRepo) Tracking(ctx context.Context, s domain.Scope) ([]domain.TrackingRow, error) {
	c := newCond(s.DatasetID).scope(s)
	limit := c.next(exportLimit)

	rows, err := r.pool.Query(ctx, `
		SELECT year, month, customer_code, coalesce(group_name, ''),
		       coalesce(target_qty, 0), coalesce(cartons, 0)
		FROM fact_tracking_nsfs WHERE `+c.where()+`
		ORDER BY year, month, customer_code LIMIT `+limit, c.args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := []domain.TrackingRow{}
	for rows.Next() {
		var x domain.TrackingRow
		if err := rows.Scan(&x.Year, &x.Month, &x.CustomerCode, &x.GroupName,
			&x.TargetQty, &x.Cartons); err != nil {
			return nil, err
		}
		out = append(out, x)
	}
	return out, rows.Err()
}

// Customers และ Products ไม่ถูกกรองตามขอบเขต เพราะเป็นข้อมูลอ้างอิงของทั้งชุด
// ต้องส่งออกครบเสมอ ไม่งั้นไฟล์ที่ได้จะนำกลับเข้าระบบไม่ได้
func (r *ExportRepo) Customers(ctx context.Context, s domain.Scope) ([]domain.Customer, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT code, name, status, provinces
		FROM dim_customers WHERE dataset_id = $1 ORDER BY code`, s.DatasetID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := []domain.Customer{}
	for rows.Next() {
		var c domain.Customer
		if err := rows.Scan(&c.Code, &c.Name, &c.Status, &c.Provinces); err != nil {
			return nil, err
		}
		out = append(out, c)
	}
	return out, rows.Err()
}

func (r *ExportRepo) Products(ctx context.Context, s domain.Scope) ([]domain.Product, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT code, coalesce(description, ''), coalesce(group_name, ''), coalesce(category, ''),
		       coalesce(innerbox, 0), coalesce(price_unit, 0), coalesce(price_carton, 0),
		       coalesce(status, '')
		FROM dim_products WHERE dataset_id = $1 ORDER BY code`, s.DatasetID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := []domain.Product{}
	for rows.Next() {
		var p domain.Product
		if err := rows.Scan(&p.Code, &p.Description, &p.GroupName, &p.Category,
			&p.Innerbox, &p.PriceUnit, &p.PriceCarton, &p.Status); err != nil {
			return nil, err
		}
		out = append(out, p)
	}
	return out, rows.Err()
}
