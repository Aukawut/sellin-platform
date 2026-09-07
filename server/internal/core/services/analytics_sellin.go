package services

import (
	"context"
	"fmt"
	"time"

	"sellin-server/internal/core/domain"
	"sellin-server/internal/core/ports"
)

type SellInPage struct {
	Scope         domain.ScopeInfo   `json:"scope"`
	KPIs          []domain.KPI       `json:"kpis"`
	Trend         domain.Chart       `json:"trend"`
	Daily         domain.Chart       `json:"daily"`
	Tracking      domain.Chart       `json:"tracking"`
	DonutGroup    []domain.Slice     `json:"donut_group"`
	DonutCategory []domain.Slice     `json:"donut_category"`
	Ranking       []domain.RankEntry `json:"ranking"`
	Network       NetworkMap         `json:"network"`
}

// NetworkMap คือข้อมูลของแผนที่เครือข่ายศูนย์กระจายสินค้า
type NetworkMap struct {
	Nodes         []domain.MapNode `json:"nodes"`
	ActiveCount   int              `json:"active_count"`
	TotalCount    int              `json:"total_count"`
	ProvinceCount int              `json:"province_count"`
}

func (s *AnalyticsService) SellIn(ctx context.Context, scope domain.Scope) (SellInPage, error) {
	meta, err := s.requireReady(ctx, scope)
	if err != nil {
		return SellInPage{}, err
	}

	baseYear, yearInferred, hasYear := domain.ResolveYear(scope, meta.SellInYears)
	prevYear := baseYear - 1

	summary, err := s.sellIn.Summary(ctx, scope)
	if err != nil {
		return SellInPage{}, err
	}

	page := SellInPage{}

	// ---- KPI Revenue ----
	page.KPIs = append(page.KPIs, domain.KPI{
		Key: "revenue", Value: summary.Revenue, Format: domain.FormatTHB, Available: true,
	})

	// ---- KPI Order: นับเลขที่เอกสารไม่ซ้ำ พร้อมยอดเฉลี่ยต่อออเดอร์ ----
	orderKPI := domain.KPI{
		Key: "orders", Value: float64(summary.Orders), Format: domain.FormatNumber, Available: true,
	}
	if summary.Orders > 0 {
		orderKPI.Secondary = &domain.Secondary{
			Kind: "avg_per_order", Value: summary.Revenue / float64(summary.Orders),
		}
	}
	page.KPIs = append(page.KPIs, orderKPI)

	// ---- KPI Active Store: ความหมายเปลี่ยนตามว่าเลือกศูนย์เดียวหรือทุกศูนย์ ----
	page.KPIs = append(page.KPIs, activeStoreKPI(meta, scope.DC))

	// ---- KPI %YoY: เทียบกับปีก่อนหน้า โดยคงฟิลเตอร์ศูนย์และเดือนไว้ แต่ไม่คงฟิลเตอร์ปี ----
	growth := domain.KPI{Key: "growth", Format: domain.FormatPercent}
	if hasYear {
		cur, prev, err := s.sellInYearPair(ctx, scope, baseYear, prevYear)
		if err != nil {
			return SellInPage{}, err
		}
		if d := domain.GrowthDelta(cur, prev, baseYear, prevYear); d != nil && !d.IsNew {
			growth.Value, growth.Delta, growth.Available = d.Pct, d, true
		} else if d != nil {
			growth.Delta = d
		}
	}
	page.KPIs = append(page.KPIs, growth)

	// ---- กราฟแนวโน้ม: สนใจเฉพาะฟิลเตอร์ศูนย์ เพราะเป็นการเทียบสองปีเต็ม ----
	page.Trend, err = s.sellInTrend(ctx, scope, meta, baseYear, prevYear)
	if err != nil {
		return SellInPage{}, err
	}

	// ---- กราฟรายวัน ----
	daily, effMonth, monthInferred, err := s.sellInDaily(ctx, scope, baseYear)
	if err != nil {
		return SellInPage{}, err
	}
	page.Daily = daily

	// ---- กราฟติดตาม Non-Sun Flower Seed ----
	tracking, err := s.sellIn.TrackingByGroup(ctx, scope)
	if err != nil {
		return SellInPage{}, err
	}
	page.Tracking = trackingChart(tracking)

	// ---- กราฟวงแหวน: นับเป็นลัง ไม่ใช่เงิน ----
	page.DonutGroup, err = s.sellInDonut(ctx, scope, ports.GroupByProductGroup)
	if err != nil {
		return SellInPage{}, err
	}
	page.DonutCategory, err = s.sellInDonut(ctx, scope, ports.GroupByProductCategory)
	if err != nil {
		return SellInPage{}, err
	}

	// ---- ตารางจัดอันดับ: แสดงทุกศูนย์เสมอ ไม่ตัดตามฟิลเตอร์ศูนย์ ----
	ranking, err := s.sellIn.RevenueByDC(ctx, scope.DatasetID, scope.Year, scope.Month)
	if err != nil {
		return SellInPage{}, err
	}
	page.Ranking = bucketsToRanking(ranking)

	page.Network = buildNetwork(meta)
	page.Scope = scopeInfo(scope, meta, baseYear, effMonth, yearInferred, monthInferred, hasYear)
	return page, nil
}

// sellInYearPair ดึงยอดของปีฐานและปีก่อนหน้าภายใต้ฟิลเตอร์เดียวกัน ยกเว้นปี
func (s *AnalyticsService) sellInYearPair(ctx context.Context, scope domain.Scope, baseYear, prevYear int) (float64, float64, error) {
	cur := scope
	cur.Year = &baseYear
	curSum, err := s.sellIn.Summary(ctx, cur)
	if err != nil {
		return 0, 0, err
	}

	prev := scope
	prev.Year = &prevYear
	prevSum, err := s.sellIn.Summary(ctx, prev)
	if err != nil {
		return 0, 0, err
	}
	return curSum.Revenue, prevSum.Revenue, nil
}

func (s *AnalyticsService) sellInTrend(ctx context.Context, scope domain.Scope, meta ports.DatasetMeta, baseYear, prevYear int) (domain.Chart, error) {
	base, basePresent, err := s.sellIn.RevenueByMonth(ctx, scope.DatasetID, scope.DC, baseYear)
	if err != nil {
		return domain.Chart{}, err
	}
	prev, prevPresent, err := s.sellIn.RevenueByMonth(ctx, scope.DatasetID, scope.DC, prevYear)
	if err != nil {
		return domain.Chart{}, err
	}

	return domain.Chart{
		Labels: monthLabels(),
		Tag:    trendTag(meta, scope.DC),
		Series: []domain.Series{
			{Key: fmt.Sprint(prevYear), Name: fmt.Sprint(prevYear), Kind: "line",
				Points: domain.PointsWithGaps(prev[:], prevPresent[:])},
			{Key: fmt.Sprint(baseYear), Name: fmt.Sprint(baseYear), Kind: "line",
				Points: domain.PointsWithGaps(base[:], basePresent[:])},
		},
	}, nil
}

// sellInDaily วาดยอดรายวันของเดือนเดียว
//
// เมื่อผู้ใช้เลือก "ทุกเดือน" ระบบเลือกเดือนล่าสุดที่มีข้อมูลให้ ไม่ใช่รวมทุกเดือนเข้าด้วยกัน
// เพราะกราฟรายวันที่รวมหลายเดือนจะอ่านไม่ได้ความ
func (s *AnalyticsService) sellInDaily(ctx context.Context, scope domain.Scope, baseYear int) (domain.Chart, int, bool, error) {
	months, err := s.sellIn.MonthsWithData(ctx, scope.DatasetID, scope.DC, baseYear)
	if err != nil {
		return domain.Chart{}, 0, false, err
	}
	month, inferred, ok := domain.ResolveMonth(scope, months)
	if !ok {
		return domain.Chart{Labels: []string{}, Series: []domain.Series{}}, 0, true, nil
	}

	byDay, err := s.sellIn.RevenueByDay(ctx, scope.DatasetID, scope.DC, baseYear, month)
	if err != nil {
		return domain.Chart{}, 0, false, err
	}

	days := daysInMonth(baseYear, month)
	labels := make([]string, days)
	values := make([]float64, days)
	var sum float64
	var counted int
	for d := 1; d <= days; d++ {
		labels[d-1] = fmt.Sprintf("%d", d)
		values[d-1] = byDay[d]
		// เส้นค่าเฉลี่ยนับเฉพาะวันที่มียอดจริง วันที่ไม่มีการสั่งซื้อไม่ถูกนับเป็นศูนย์
		// ไม่งั้นวันหยุดยาวจะดึงค่าเฉลี่ยลงจนเส้นอ้างอิงไม่มีความหมาย
		if byDay[d] > 0 {
			sum += byDay[d]
			counted++
		}
	}

	chart := domain.Chart{
		Labels: labels,
		Tag:    fmt.Sprintf("%d/%d", month, baseYear),
		Series: []domain.Series{{Key: "revenue", Name: "Revenue", Kind: "bar", Points: domain.Points(values)}},
	}
	if counted > 0 {
		avg := sum / float64(counted)
		chart.Reference = &avg
	}
	return chart, month, inferred, nil
}

func (s *AnalyticsService) sellInDonut(ctx context.Context, scope domain.Scope, groupBy string) ([]domain.Slice, error) {
	buckets, err := s.sellIn.CartonsBy(ctx, scope, groupBy)
	if err != nil {
		return nil, err
	}
	return domain.MakeSlices(bucketsToMap(buckets)), nil
}

func activeStoreKPI(meta ports.DatasetMeta, dc string) domain.KPI {
	active, total := 0, len(meta.Customers)
	for _, c := range meta.Customers {
		if c.IsActive() {
			active++
		}
	}

	if dc == "" {
		return domain.KPI{
			Key: "active_store", Value: float64(active), Format: domain.FormatNumber, Available: true,
			Text:      fmt.Sprintf("%d/%d", active, total),
			Secondary: &domain.Secondary{Kind: "active_total", Value: float64(total)},
		}
	}
	for _, c := range meta.Customers {
		if c.Code == dc {
			return domain.KPI{
				Key: "active_store", Format: domain.FormatText, Available: true,
				Text:      c.Name,
				Secondary: &domain.Secondary{Kind: "customer_status", Text: c.Status},
			}
		}
	}
	return domain.KPI{Key: "active_store", Format: domain.FormatText, Text: dc, Available: true}
}

func trackingChart(buckets []ports.Bucket) domain.Chart {
	labels := make([]string, 0, len(buckets))
	targets := make([]float64, 0, len(buckets))
	actuals := make([]float64, 0, len(buckets))
	for _, b := range buckets {
		labels = append(labels, b.Label)
		targets = append(targets, b.Value)
		actuals = append(actuals, b.Secondary)
	}
	return domain.Chart{
		Labels: labels,
		Series: []domain.Series{
			{Key: "target", Name: "Target_Qty", Kind: "bar", Points: domain.Points(targets)},
			{Key: "actual", Name: "Sum Cartoon", Kind: "bar", Points: domain.Points(actuals)},
		},
	}
}

func buildNetwork(meta ports.DatasetMeta) NetworkMap {
	out := NetworkMap{TotalCount: len(meta.Customers)}
	provinces := map[string]bool{}
	for _, c := range meta.Customers {
		if !c.IsActive() {
			continue
		}
		out.ActiveCount++
		for _, p := range c.Provinces {
			provinces[p] = true
		}
		out.Nodes = append(out.Nodes, domain.MapNode{
			Code: c.Code, Name: c.Name, Provinces: c.Provinces,
			Status: c.Status, Value: float64(len(c.Provinces)),
		})
	}
	out.ProvinceCount = len(provinces)
	return out
}

func trendTag(meta ports.DatasetMeta, dc string) string {
	if dc == "" {
		return ""
	}
	return customerName(meta, dc)
}

// daysInMonth ใช้วิธีเดียวกับ new Date(y, m, 0).getDate() ของ JavaScript
// คือถอยจากวันที่ 1 ของเดือนถัดไปมาหนึ่งวัน ซึ่งรองรับปีอธิกสุรทินโดยอัตโนมัติ
func daysInMonth(year, month int) int {
	return time.Date(year, time.Month(month)+1, 0, 0, 0, 0, 0, time.UTC).Day()
}
