package services

import (
	"context"
	"fmt"
	"io"
	"strings"
	"time"

	"sellin-server/internal/core/domain"
	"sellin-server/internal/core/ports"
)

// ExportService สร้างไฟล์ Excel ของข้อมูลตามขอบเขตที่ผู้ใช้กำลังดูอยู่
//
// โครงไฟล์ตามของ dashboard เดิม คือแผ่นสรุปตัวชี้วัดตามด้วยแผ่นข้อมูลดิบ
// แต่ตัวเลขในแผ่นสรุปมาจากการคำนวณจริง ไม่ได้อ่านจากข้อความบนหน้าจอเหมือนเดิม
type ExportService struct {
	rows      ports.ExportRepo
	analytics *AnalyticsService
	newBook   func() ports.SheetWriter
}

func NewExportService(rows ports.ExportRepo, analytics *AnalyticsService, newBook func() ports.SheetWriter) *ExportService {
	return &ExportService{rows: rows, analytics: analytics, newBook: newBook}
}

const summarySheet = "Summary"

// Filename ตั้งชื่อไฟล์ให้บอกได้ว่าเป็นข้อมูลขอบเขตไหน โดยไม่ต้องเปิดไฟล์ดู
func (s *ExportService) Filename(scope domain.Scope, now time.Time) string {
	name := "SalePerformance"
	if scope.DC != "" {
		name += "_" + scope.DC
	}
	if scope.Year != nil {
		name += fmt.Sprintf("_%d", *scope.Year)
	}
	if scope.Month != nil {
		name += fmt.Sprintf("-%02d", *scope.Month)
	}
	return fmt.Sprintf("%s_%s.xlsx", name, now.Format("20060102"))
}

func (s *ExportService) Write(ctx context.Context, scope domain.Scope, dst io.Writer, now time.Time) error {
	book := s.newBook()

	if err := s.writeSummary(ctx, scope, book, now); err != nil {
		return err
	}
	if err := s.writeData(ctx, scope, book); err != nil {
		return err
	}
	return book.WriteTo(dst, summarySheet)
}

// writeSummary เขียนตัวชี้วัดของทั้งสี่หน้าลงแผ่นเดียว
// เรียกใช้ตัวคำนวณชุดเดียวกับหน้าเว็บ ตัวเลขในไฟล์จึงตรงกับที่เห็นบนจอเสมอ
func (s *ExportService) writeSummary(ctx context.Context, scope domain.Scope, book ports.SheetWriter, now time.Time) error {
	sellIn, err := s.analytics.SellIn(ctx, scope)
	if err != nil {
		return err
	}
	sellOut, err := s.analytics.SellOut(ctx, SellOutQuery{Scope: scope})
	if err != nil {
		return err
	}
	stock, err := s.analytics.Stock(ctx, StockQuery{Scope: scope})
	if err != nil {
		return err
	}
	planning, err := s.analytics.Planning(ctx, scope)
	if err != nil {
		return err
	}

	rows := [][]any{
		{"Value Plus Worldwide — Sale Performance T-T"},
		{"ขอบเขตข้อมูล", scopeText(sellIn.Scope)},
		{"ส่งออกเมื่อ", now.Format("2 Jan 2006 15:04")},
		{},
		{"1. Sell-In Performance"},
	}
	rows = append(rows, kpiRows(sellIn.KPIs, map[string]string{
		"revenue": "Revenue", "orders": "จำนวนออเดอร์",
		"active_store": "ศูนย์กระจายสินค้า", "growth": "% เทียบยอดปีก่อนหน้า",
	})...)

	rows = append(rows, []any{}, []any{"2. Sell-Out Performance"})
	rows = append(rows, kpiRows(sellOut.KPIs, map[string]string{
		"target": "Target Sale", "revenue": "Revenue",
		"achievement": "% เทียบเป้าหมาย", "growth": "% เทียบยอดปีก่อนหน้า",
	})...)

	rows = append(rows, []any{}, []any{"3. Stock Inventory"})
	if stock.Scope.HasData {
		rows = append(rows, []any{"เดือนสแนปช็อต",
			fmt.Sprintf("%d/%d", stock.Scope.EffectiveMonth, stock.Scope.EffectiveYear)})
	}
	rows = append(rows, kpiRows(stock.KPIs, map[string]string{
		"beginning_stock": "สต๊อกต้นเดือน (ลัง)", "stock_in": "รับเข้าจาก Sell-In (ลัง)",
		"sell_through": "Sell-Through Rate", "stock_cover": "Stock Cover (เดือน)",
	})...)
	rows = append(rows, []any{"รายการที่ต้องเฝ้าระวัง", stock.AlertCount})

	rows = append(rows, []any{}, []any{"4. Sale Analysis Planning"})
	rows = append(rows, kpiRows(planning.KPIs, map[string]string{
		"actual": "ยอดขายจริง (Sell-Out)", "target": "Target Sale",
		"achievement": "% Achievement", "forecast": "คาดการณ์ยอดขายทั้งปี",
	})...)

	book.AddSheet(summarySheet, []string{"รายการ", "ค่า"}, rows)
	book.SetColumnWidths(summarySheet, map[string]float64{"A": 38, "B": 30})
	return nil
}

// kpiRows แปลง KPI เป็นแถวสองคอลัมน์ โดยเขียนตัวเลขเป็นตัวเลขจริง ไม่ใช่ข้อความที่จัดรูปแบบแล้ว
// เพื่อให้ผู้รับไฟล์นำไปคำนวณต่อใน Excel ได้ทันที
func kpiRows(kpis []domain.KPI, labels map[string]string) [][]any {
	out := make([][]any, 0, len(kpis))
	for _, k := range kpis {
		label := labels[k.Key]
		if label == "" {
			label = k.Key
		}
		switch {
		case k.Text != "":
			out = append(out, []any{label, k.Text})
		case !k.Available:
			out = append(out, []any{label, "N/A"})
		default:
			out = append(out, []any{label, k.Value})
		}
	}
	return out
}

func scopeText(s domain.ScopeInfo) string {
	dc := s.DCName
	if dc == "" {
		dc = "ทุกศูนย์กระจายสินค้า"
	}
	year := "ทุกปี"
	if s.Year != nil {
		year = fmt.Sprintf("%d", *s.Year)
	}
	month := "ทุกเดือน"
	if s.Month != nil {
		month = fmt.Sprintf("เดือน %d", *s.Month)
	}
	return fmt.Sprintf("%s · %s · %s", dc, year, month)
}

// writeData เขียนแผ่นข้อมูลดิบ โดยใช้ชื่อหัวคอลัมน์ชุดเดียวกับไฟล์ Template ต้นทาง
// ไฟล์ที่ส่งออกจึงนำกลับเข้าระบบได้ทันทีโดยไม่ต้องแก้หัวตาราง
func (s *ExportService) writeData(ctx context.Context, scope domain.Scope, book ports.SheetWriter) error {
	sellIn, err := s.rows.SellIn(ctx, scope)
	if err != nil {
		return err
	}
	rows := make([][]any, 0, len(sellIn))
	for _, r := range sellIn {
		rows = append(rows, []any{r.Year, r.Month, r.Day, r.DocNo, r.CustomerCode,
			r.ProductCode, r.ProductDesc, r.GroupName, r.Category,
			r.Qty, r.Unit, r.Cartons, r.PriceUnit, r.Revenue, r.StatusSale})
	}
	book.AddSheet("Data_Sell-In", []string{
		"Year", "Month", "Day", "No-Document", "Customer Code", "Product Code",
		"Product Description", "Product Group", "Product Category",
		"Qty", "Unit", "Sum Cartoon", "Price Unit", "Revenue", "Status Sale",
	}, rows)

	sellOut, err := s.rows.SellOut(ctx, scope)
	if err != nil {
		return err
	}
	rows = make([][]any, 0, len(sellOut))
	for _, r := range sellOut {
		rows = append(rows, []any{r.Year, r.Month, r.CustomerCode, r.ProductCode, r.ProductDesc,
			r.GroupName, r.Category, r.Cartons, r.Unit, r.PriceUnit, r.Revenue, r.Status})
	}
	book.AddSheet("Data_Sell-Out", []string{
		"Year", "Month", "Customer Code", "Product Code", "Product Description",
		"Product Group", "Product Category", "Sum Cartoon", "Unit", "Price Unit", "Revenue", "Status",
	}, rows)

	stock, err := s.rows.Stock(ctx, scope)
	if err != nil {
		return err
	}
	rows = make([][]any, 0, len(stock))
	for _, r := range stock {
		rows = append(rows, []any{r.Year, r.Month, r.CustomerCode, r.ProductCode, r.ProductDesc,
			r.GroupName, r.Category, r.Beginning, r.SellIn, r.SellOut, r.Ending})
	}
	book.AddSheet("Data_Stock_inventory", []string{
		"Year", "Month", "Customer Code", "Product Code", "Product Description",
		"Product Group", "Product Category", "Beginning Stock", "Sell-In", "Sell-Out", "Ending Stock",
	}, rows)

	targets, err := s.rows.Targets(ctx, scope)
	if err != nil {
		return err
	}
	rows = make([][]any, 0, len(targets))
	for _, r := range targets {
		rows = append(rows, []any{r.Month, r.Year, r.CustomerName, r.CustomerCode, r.Category, r.TargetAmount})
	}
	book.AddSheet("Target_Sale", []string{
		"Month", "Year", "Customer", "Customer Code", "Product Category", "Target",
	}, rows)

	// ข้อมูลอ้างอิงต้องส่งออกครบทั้งชุด ไม่กรองตามขอบเขต
	// เพราะ importer ต้องการ Data_Customer เสมอ ไฟล์ที่ขาดไปจะนำกลับเข้าระบบไม่ได้
	customers, err := s.rows.Customers(ctx, scope)
	if err != nil {
		return err
	}
	rows = make([][]any, 0, len(customers))
	for _, c := range customers {
		rows = append(rows, []any{c.Name, c.Code, c.Status, strings.Join(c.Provinces, " , ")})
	}
	book.AddSheet("Data_Customer", []string{
		"Customer", "Customer Code", "Customer Status", "Region / Province (optional)",
	}, rows)

	products, err := s.rows.Products(ctx, scope)
	if err != nil {
		return err
	}
	rows = make([][]any, 0, len(products))
	for i, p := range products {
		rows = append(rows, []any{i + 1, p.Code, p.Description, p.GroupName, p.Category,
			p.Innerbox, p.PriceUnit, p.PriceCarton, p.Status})
	}
	book.AddSheet("Data_Product", []string{
		"No.", "Product Code", "Product Description", "Product Group", "Product Category",
		"Innerbox", "Price Unit", "Price Cartoon", "Status",
	}, rows)

	tracking, err := s.rows.Tracking(ctx, scope)
	if err != nil {
		return err
	}
	rows = make([][]any, 0, len(tracking))
	for _, r := range tracking {
		pct := 0.0
		if r.TargetQty > 0 {
			pct = r.Cartons / r.TargetQty
		}
		rows = append(rows, []any{r.Month, r.Year, r.CustomerCode, r.GroupName, r.TargetQty, r.Cartons, pct})
	}
	book.AddSheet("Tracking_Non-Sun Flower Seed", []string{
		"Month", "Year", "Customer Code", "Product Group", "Target_Qty", "Sum Cartoon", "% Of Target",
	}, rows)

	return nil
}
