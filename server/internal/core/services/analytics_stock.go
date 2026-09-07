package services

import (
	"context"
	"fmt"

	"sellin-server/internal/core/domain"
	"sellin-server/internal/core/ports"
)

type StockPage struct {
	Scope         domain.ScopeInfo      `json:"scope"`
	KPIs          []domain.KPI          `json:"kpis"`
	Trend         domain.Chart          `json:"trend"`
	Flow          domain.Chart          `json:"flow"`
	DonutGroup    []domain.Slice        `json:"donut_group"`
	DonutCategory []domain.Slice        `json:"donut_category"`
	Products      []domain.StockProduct `json:"products"`
	AlertCount    int                   `json:"alert_count"`
	TotalProducts int                   `json:"total_products"`
}

type StockQuery struct {
	Scope  domain.Scope
	Group  string // "" = ทุกกลุ่มสินค้า
	Status string // "" = ทุกสถานะ
}

// Stock สร้างหน้าสต๊อก ซึ่งต่างจากหน้าอื่นตรงที่ทุกตัวเลขเป็นค่า ณ เดือนเดียว
//
// สต๊อกเป็นค่า ณ จุดเวลา ไม่ใช่ยอดสะสม การรวมหลายเดือนเข้าด้วยกันจึงไม่มีความหมาย
// เมื่อผู้ใช้เลือก "ทุกเดือน" ระบบจึงเลือกเดือนล่าสุดที่มีข้อมูลให้แทนการรวม
func (s *AnalyticsService) Stock(ctx context.Context, q StockQuery) (StockPage, error) {
	scope := q.Scope
	meta, err := s.requireReady(ctx, scope)
	if err != nil {
		return StockPage{}, err
	}

	// ปีของหน้านี้ต้องเป็นปีที่มีข้อมูลสต๊อกจริง ไม่ใช่ปีที่มีข้อมูลขาย
	effYear := 0
	yearInferred := true
	if scope.Year != nil && containsInt(meta.StockYears, *scope.Year) {
		effYear, yearInferred = *scope.Year, false
	} else if y, inferred, ok := domain.ResolveYear(domain.Scope{}, meta.StockYears); ok {
		effYear, yearInferred = y, inferred
	}

	page := StockPage{}
	if effYear == 0 {
		page.Scope = scopeInfo(scope, meta, 0, 0, true, true, false)
		return page, nil
	}

	months, err := s.stock.MonthsWithData(ctx, scope.DatasetID, scope.DC, effYear)
	if err != nil {
		return StockPage{}, err
	}
	effMonth, monthInferred, hasData := domain.ResolveMonth(scope, months)
	page.Scope = scopeInfo(scope, meta, effYear, effMonth, yearInferred, monthInferred, hasData)
	if !hasData {
		return page, nil
	}

	snapshot, err := s.stock.Snapshot(ctx, scope.DatasetID, scope.DC, effYear, effMonth)
	if err != nil {
		return StockPage{}, err
	}

	// ---- KPI Beginning Stock ----
	page.KPIs = append(page.KPIs, domain.KPI{
		Key: "beginning_stock", Value: snapshot.Beginning, Format: domain.FormatCartons, Available: true,
	})

	// ---- KPI Stock In: มาจาก Sell-In ไม่ใช่คอลัมน์ sell_in ในตารางสต๊อก ----
	// dashboard เดิมใช้ Sum Cartoon จาก Data_Sell-In เพื่อสะท้อนยอดที่ "สั่งซื้อเข้ามา" จริง
	stockIn, err := s.sellIn.CartonsTotal(ctx, scope.DatasetID, scope.DC, effYear, effMonth)
	if err != nil {
		return StockPage{}, err
	}
	page.KPIs = append(page.KPIs, domain.KPI{
		Key: "stock_in", Value: stockIn, Format: domain.FormatCartons, Available: true,
		Secondary: &domain.Secondary{Kind: "source", Text: "sellin"},
	})

	// ---- KPI Sell-Through Rate ----
	rate, available, ok := domain.SellThroughRate(snapshot.Beginning, snapshot.SellIn, snapshot.SellOut)
	page.KPIs = append(page.KPIs, domain.KPI{
		Key: "sell_through", Value: rate, Format: domain.FormatPercent, Available: ok,
		Secondary: &domain.Secondary{Kind: "available", Value: available},
	})

	// ---- KPI Stock Cover ----
	recent, err := s.stock.RecentSellOut(ctx, scope.DatasetID, scope.DC, effYear, effMonth, 3)
	if err != nil {
		return StockPage{}, err
	}
	cover, counted, coverOK := domain.StockCover(snapshot.Ending, recent)
	page.KPIs = append(page.KPIs, domain.KPI{
		Key: "stock_cover", Value: cover, Format: domain.FormatMonths, Available: coverOK,
		Secondary: &domain.Secondary{Kind: "months_counted", Value: float64(counted)},
	})

	// ---- ตารางสถานะรายสินค้า ----
	products, err := s.stock.Products(ctx, scope.DatasetID, scope.DC, effYear, effMonth, q.Group)
	if err != nil {
		return StockPage{}, err
	}
	all := make([]domain.StockProduct, 0, len(products))
	for _, p := range products {
		status, label, coverM := domain.ClassifyStock(p.Beginning, p.SellIn, p.SellOut, p.Ending)
		item := domain.StockProduct{
			Code: p.Code, Description: p.Description, GroupName: p.GroupName,
			Beginning: p.Beginning, SellIn: p.SellIn, SellOut: p.SellOut, Ending: p.Ending,
			CoverMonths: coverM, Status: status, Label: label,
			ProductStatus: productStatus(meta, p.Code),
		}
		all = append(all, item)
	}
	domain.SortStockProducts(all)

	page.TotalProducts = len(all)
	for _, p := range all {
		if p.Status != domain.StockStatusOK {
			page.AlertCount++
		}
	}
	// การกรองสถานะทำหลังนับรายการเฝ้าระวัง เพื่อให้ตัวเลขสรุปไม่เปลี่ยนตามฟิลเตอร์ที่ผู้ใช้เลือกดู
	page.Products = filterByStatus(all, q.Status)

	// ---- กราฟรายเดือน ----
	monthly, err := s.stock.Monthly(ctx, scope.DatasetID, scope.DC, effYear)
	if err != nil {
		return StockPage{}, err
	}
	labels := make([]string, 0, len(monthly))
	beg := make([]float64, 0, len(monthly))
	end := make([]float64, 0, len(monthly))
	in := make([]float64, 0, len(monthly))
	out := make([]float64, 0, len(monthly))
	for _, m := range monthly {
		labels = append(labels, fmt.Sprintf("%d", m.Month))
		beg = append(beg, m.Beginning)
		end = append(end, m.Ending)
		in = append(in, m.SellIn)
		out = append(out, m.SellOut)
	}
	tag := trendTag(meta, scope.DC)
	page.Trend = domain.Chart{
		Labels: labels, Tag: tag,
		Series: []domain.Series{
			{Key: "beginning", Name: "Beginning Stock", Kind: "line", Points: domain.Points(beg)},
			{Key: "ending", Name: "Ending Stock", Kind: "line", Points: domain.Points(end)},
		},
	}
	page.Flow = domain.Chart{
		Labels: labels, Tag: tag,
		Series: []domain.Series{
			{Key: "sell_in", Name: "Sell-In", Kind: "bar", Points: domain.Points(in)},
			{Key: "sell_out", Name: "Sell-Out", Kind: "bar", Points: domain.Points(out)},
		},
	}

	// ---- กราฟวงแหวนของสต๊อกคงเหลือ ----
	page.DonutGroup, err = s.stockDonut(ctx, scope, effYear, effMonth, ports.GroupByProductGroup)
	if err != nil {
		return StockPage{}, err
	}
	page.DonutCategory, err = s.stockDonut(ctx, scope, effYear, effMonth, ports.GroupByProductCategory)
	if err != nil {
		return StockPage{}, err
	}

	return page, nil
}

func (s *AnalyticsService) stockDonut(ctx context.Context, scope domain.Scope, year, month int, groupBy string) ([]domain.Slice, error) {
	buckets, err := s.stock.EndingBy(ctx, scope.DatasetID, scope.DC, year, month, groupBy)
	if err != nil {
		return nil, err
	}
	return domain.MakeSlices(bucketsToMap(buckets)), nil
}

// productStatus คือสถานะจาก Data_Product ไม่ใช่สถานะสต๊อก
// dashboard เดิม hardcode ค่านี้ไว้ 97 รายการใน JavaScript จนสินค้าใหม่ขึ้น N/A เสมอ
func productStatus(meta ports.DatasetMeta, code string) string {
	if p, ok := meta.Products[code]; ok && p.Status != "" {
		return p.Status
	}
	return "N/A"
}

func filterByStatus(items []domain.StockProduct, status string) []domain.StockProduct {
	if status == "" {
		return items
	}
	out := make([]domain.StockProduct, 0, len(items))
	for _, p := range items {
		if p.Label == status {
			out = append(out, p)
		}
	}
	return out
}

func containsInt(list []int, v int) bool {
	for _, x := range list {
		if x == v {
			return true
		}
	}
	return false
}
