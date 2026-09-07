package postgres

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"

	"sellin-server/internal/core/domain"
)

// querier ครอบทั้ง pool และ transaction ให้ deleteFacts เรียกได้จากทั้งสองบริบท
type querier interface {
	Exec(ctx context.Context, sql string, args ...any) (pgconn.CommandTag, error)
}

// factTables เรียงตามลำดับที่ปลอดภัยต่อการลบ และใช้เป็นรายการเดียวกันทั้งตอนลบและตอนล้าง
var factTables = []string{
	"fact_tracking_nsfs", "fact_target", "fact_stock",
	"fact_sellout", "fact_sellin", "dim_products", "dim_customers",
}

// ReplaceFacts เขียนข้อมูลทั้งชุดของ dataset ใน transaction เดียว
//
// ล้างของเดิมก่อนเสมอ เพื่อให้การนำเข้าซ้ำ (เช่นระบบล่มกลางคันแล้วสั่งใหม่) ไม่ทิ้งข้อมูลซ้อน
// ถ้าล้มเหลวกลางคัน transaction จะถูก rollback ทั้งหมด — ไม่มีสถานะครึ่ง ๆ กลาง ๆ
func (r *DatasetRepo) ReplaceFacts(ctx context.Context, id uuid.UUID, wb *domain.Workbook) error {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	for _, t := range factTables {
		if _, err := tx.Exec(ctx, fmt.Sprintf(`DELETE FROM %s WHERE dataset_id = $1`, t), id); err != nil {
			return fmt.Errorf("ล้างข้อมูลเดิมของ %s ไม่สำเร็จ: %w", t, err)
		}
	}

	copyInto := func(table string, cols []string, n int, fn func(i int) ([]any, error)) error {
		if n == 0 {
			return nil
		}
		if _, err := tx.CopyFrom(ctx, pgx.Identifier{table}, cols, pgx.CopyFromSlice(n, fn)); err != nil {
			return fmt.Errorf("เขียนข้อมูลลง %s ไม่สำเร็จ: %w", table, err)
		}
		return nil
	}

	if err := copyInto("dim_customers",
		[]string{"dataset_id", "code", "name", "status", "provinces", "region"},
		len(wb.Customers), func(i int) ([]any, error) {
			c := wb.Customers[i]
			provinces := c.Provinces
			if provinces == nil {
				provinces = []string{}
			}
			return []any{id, c.Code, c.Name, c.Status, provinces, nullIfEmpty(c.Region)}, nil
		}); err != nil {
		return err
	}

	if err := copyInto("dim_products",
		[]string{"dataset_id", "code", "description", "group_name", "category",
			"innerbox", "price_unit", "price_carton", "status"},
		len(wb.Products), func(i int) ([]any, error) {
			p := wb.Products[i]
			return []any{id, p.Code, p.Description, p.GroupName, p.Category,
				p.Innerbox, p.PriceUnit, p.PriceCarton, nullIfEmpty(p.Status)}, nil
		}); err != nil {
		return err
	}

	if err := copyInto("fact_sellin",
		[]string{"dataset_id", "sale_date", "year", "month", "day", "doc_no",
			"customer_code", "product_code", "product_desc", "group_name", "category",
			"qty", "unit", "innerbox", "cartons", "price_unit", "revenue", "status_sale"},
		len(wb.SellIn), func(i int) ([]any, error) {
			s := wb.SellIn[i]
			return []any{id, s.SaleDate, s.Year, s.Month, s.Day, s.DocNo,
				s.CustomerCode, s.ProductCode, s.ProductDesc, s.GroupName, s.Category,
				s.Qty, s.Unit, s.Innerbox, s.Cartons, s.PriceUnit, s.Revenue, s.StatusSale}, nil
		}); err != nil {
		return err
	}

	if err := copyInto("fact_sellout",
		[]string{"dataset_id", "year", "month", "customer_code", "product_code",
			"product_desc", "group_name", "category", "cartons", "unit",
			"price_unit", "revenue", "status"},
		len(wb.SellOut), func(i int) ([]any, error) {
			s := wb.SellOut[i]
			return []any{id, s.Year, s.Month, s.CustomerCode, s.ProductCode,
				s.ProductDesc, s.GroupName, s.Category, s.Cartons, s.Unit,
				s.PriceUnit, s.Revenue, s.Status}, nil
		}); err != nil {
		return err
	}

	if err := copyInto("fact_stock",
		[]string{"dataset_id", "year", "month", "customer_code", "product_code",
			"product_desc", "group_name", "category",
			"beginning", "sell_in", "sell_out", "ending"},
		len(wb.Stock), func(i int) ([]any, error) {
			s := wb.Stock[i]
			return []any{id, s.Year, s.Month, s.CustomerCode, s.ProductCode,
				s.ProductDesc, s.GroupName, s.Category,
				s.Beginning, s.SellIn, s.SellOut, s.Ending}, nil
		}); err != nil {
		return err
	}

	if err := copyInto("fact_target",
		[]string{"dataset_id", "year", "month", "customer_code", "customer_name",
			"category", "target_amount"},
		len(wb.Targets), func(i int) ([]any, error) {
			s := wb.Targets[i]
			return []any{id, s.Year, s.Month, s.CustomerCode, s.CustomerName,
				s.Category, s.TargetAmount}, nil
		}); err != nil {
		return err
	}

	if err := copyInto("fact_tracking_nsfs",
		[]string{"dataset_id", "year", "month", "customer_code", "group_name",
			"target_qty", "cartons"},
		len(wb.Tracking), func(i int) ([]any, error) {
			s := wb.Tracking[i]
			return []any{id, s.Year, s.Month, s.CustomerCode, s.GroupName,
				s.TargetQty, s.Cartons}, nil
		}); err != nil {
		return err
	}

	return tx.Commit(ctx)
}

func (r *DatasetRepo) deleteFacts(ctx context.Context, q querier, id uuid.UUID) error {
	for _, t := range factTables {
		if _, err := q.Exec(ctx, fmt.Sprintf(`DELETE FROM %s WHERE dataset_id = $1`, t), id); err != nil {
			return fmt.Errorf("ล้างข้อมูลของ %s ไม่สำเร็จ: %w", t, err)
		}
	}
	return nil
}

func nullIfEmpty(s string) any {
	if s == "" {
		return nil
	}
	return s
}
