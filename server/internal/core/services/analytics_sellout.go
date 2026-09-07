package services

import (
	"context"
	"fmt"
	"sort"

	"sellin-server/internal/core/domain"
	"sellin-server/internal/core/ports"
)

type SellOutPage struct {
	Scope         domain.ScopeInfo   `json:"scope"`
	KPIs          []domain.KPI       `json:"kpis"`
	Trend         domain.Chart       `json:"trend"`
	YearCompare   domain.Chart       `json:"year_compare"`
	DonutGroup    []domain.Slice     `json:"donut_group"`
	DonutCategory []domain.Slice     `json:"donut_category"`
	Products      []SellOutProduct   `json:"products"`
	Ranking       []domain.RankEntry `json:"ranking"`
}

type SellOutProduct struct {
	Rank        int     `json:"rank"`
	Code        string  `json:"code"`
	Description string  `json:"description"`
	GroupName   string  `json:"group_name"`
	Category    string  `json:"category"`
	Cartons     float64 `json:"cartons"`
	Revenue     float64 `json:"revenue"`
	Share       float64 `json:"share"`
}

// SellOutQuery มีฟิลเตอร์เฉพาะหน้านี้เพิ่มจากฟิลเตอร์ระดับ global
type SellOutQuery struct {
	Scope domain.Scope
	Group string // "" = ทุกกลุ่มสินค้า
}

func (s *AnalyticsService) SellOut(ctx context.Context, q SellOutQuery) (SellOutPage, error) {
	scope := q.Scope
	meta, err := s.requireReady(ctx, scope)
	if err != nil {
		return SellOutPage{}, err
	}

	baseYear, yearInferred, hasYear := domain.ResolveYear(scope, meta.SellOutYears)
	prevYear := baseYear - 1

	summary, err := s.sellOut.Summary(ctx, scope)
	if err != nil {
		return SellOutPage{}, err
	}
	targetTotal, err := s.target.Total(ctx, scope)
	if err != nil {
		return SellOutPage{}, err
	}

	page := SellOutPage{}

	// ---- KPI Target Sale ----
	page.KPIs = append(page.KPIs, domain.KPI{
		Key: "target", Value: targetTotal, Format: domain.FormatTHB, Available: true,
		Secondary: &domain.Secondary{Kind: "target_years", Text: joinYears(meta.TargetYears)},
	})

	// ---- KPI Revenue ----
	page.KPIs = append(page.KPIs, domain.KPI{
		Key: "revenue", Value: summary.Revenue, Format: domain.FormatTHB, Available: true,
		Secondary: &domain.Secondary{Kind: "cartons", Value: summary.Cartons},
	})

	// ---- KPI % Achievement ----
	ach := domain.KPI{Key: "achievement", Format: domain.FormatPercent}
	if targetTotal > 0 {
		ach.Value = summary.Revenue / targetTotal * 100
		ach.Available = true
		ach.Secondary = &domain.Secondary{Kind: "tier", Text: domain.AchievementTier(ach.Value, true)}
	}
	page.KPIs = append(page.KPIs, ach)

	// ---- KPI %YoY ----
	growth := domain.KPI{Key: "growth", Format: domain.FormatPercent}
	if hasYear {
		cur, prev, err := s.sellOutYearPair(ctx, scope, baseYear, prevYear)
		if err != nil {
			return SellOutPage{}, err
		}
		if d := domain.GrowthDelta(cur, prev, baseYear, prevYear); d != nil {
			growth.Delta = d
			if !d.IsNew {
				growth.Value, growth.Available = d.Pct, true
			}
		}
	}
	page.KPIs = append(page.KPIs, growth)

	// ---- กราฟ Target vs Revenue รายเดือนของปีฐาน ----
	revenueByMonth, present, err := s.sellOut.RevenueByMonth(ctx, scope.DatasetID, scope.DC, baseYear)
	if err != nil {
		return SellOutPage{}, err
	}
	targetByMonth, err := s.target.ByMonth(ctx, scope.DatasetID, scope.DC, baseYear)
	if err != nil {
		return SellOutPage{}, err
	}
	page.Trend = domain.Chart{
		Labels: monthLabels(),
		Tag:    trendTag(meta, scope.DC),
		Series: []domain.Series{
			{Key: "target", Name: "Target Sale", Kind: "bar", Points: domain.Points(targetByMonth[:])},
			{Key: "revenue", Name: "Revenue", Kind: "line", Points: domain.PointsWithGaps(revenueByMonth[:], present[:])},
		},
	}

	// ---- กราฟเทียบสองปี ----
	prevByMonth, prevPresent, err := s.sellOut.RevenueByMonth(ctx, scope.DatasetID, scope.DC, prevYear)
	if err != nil {
		return SellOutPage{}, err
	}
	page.YearCompare = domain.Chart{
		Labels: monthLabels(),
		Tag:    trendTag(meta, scope.DC),
		Series: []domain.Series{
			{Key: fmt.Sprint(prevYear), Name: fmt.Sprint(prevYear), Kind: "bar",
				Points: domain.PointsWithGaps(prevByMonth[:], prevPresent[:])},
			{Key: fmt.Sprint(baseYear), Name: fmt.Sprint(baseYear), Kind: "bar",
				Points: domain.PointsWithGaps(revenueByMonth[:], present[:])},
		},
	}

	// ---- กราฟวงแหวนพร้อมป้ายเทียบปีก่อน ----
	page.DonutGroup, err = s.sellOutDonut(ctx, scope, ports.GroupByProductGroup, baseYear, prevYear)
	if err != nil {
		return SellOutPage{}, err
	}
	page.DonutCategory, err = s.sellOutDonut(ctx, scope, ports.GroupByProductCategory, baseYear, prevYear)
	if err != nil {
		return SellOutPage{}, err
	}

	// ---- ตารางสินค้า ----
	products, err := s.sellOut.Products(ctx, scope, q.Group)
	if err != nil {
		return SellOutPage{}, err
	}
	page.Products = rankProducts(products)

	// ---- ตารางจัดอันดับศูนย์ ----
	ranking, err := s.sellOut.RevenueByDC(ctx, scope.DatasetID, scope.Year, scope.Month)
	if err != nil {
		return SellOutPage{}, err
	}
	page.Ranking = bucketsToRanking(ranking)

	effMonth := 0
	if scope.Month != nil {
		effMonth = *scope.Month
	}
	page.Scope = scopeInfo(scope, meta, baseYear, effMonth, yearInferred, false, hasYear)
	return page, nil
}

func (s *AnalyticsService) sellOutYearPair(ctx context.Context, scope domain.Scope, baseYear, prevYear int) (float64, float64, error) {
	cur := scope
	cur.Year = &baseYear
	curSum, err := s.sellOut.Summary(ctx, cur)
	if err != nil {
		return 0, 0, err
	}
	prev := scope
	prev.Year = &prevYear
	prevSum, err := s.sellOut.Summary(ctx, prev)
	if err != nil {
		return 0, 0, err
	}
	return curSum.Revenue, prevSum.Revenue, nil
}

// sellOutDonut เพิ่มป้ายเทียบปีก่อนให้แต่ละส่วนของวงแหวน
//
// สัดส่วนคำนวณจากขอบเขตที่ผู้ใช้เลือก แต่ป้ายเทียบปีใช้ปีฐานกับปีก่อนหน้าเสมอ
// โดยคงฟิลเตอร์ศูนย์และเดือนไว้ ตรงกับพฤติกรรมของ dashboard เดิม
func (s *AnalyticsService) sellOutDonut(ctx context.Context, scope domain.Scope, groupBy string, baseYear, prevYear int) ([]domain.Slice, error) {
	current, err := s.sellOut.CartonsBy(ctx, scope, groupBy)
	if err != nil {
		return nil, err
	}
	slices := domain.MakeSlices(bucketsToMap(current))

	curYear, err := s.sellOut.CartonsByForYear(ctx, scope.DatasetID, scope.DC, scope.Month, baseYear, groupBy)
	if err != nil {
		return nil, err
	}
	prevYearData, err := s.sellOut.CartonsByForYear(ctx, scope.DatasetID, scope.DC, scope.Month, prevYear, groupBy)
	if err != nil {
		return nil, err
	}
	curMap, prevMap := bucketsToMap(curYear), bucketsToMap(prevYearData)

	for i := range slices {
		slices[i].YoY = domain.GrowthDelta(curMap[slices[i].Label], prevMap[slices[i].Label], baseYear, prevYear)
	}
	return slices, nil
}

func rankProducts(items []ports.ProductAgg) []SellOutProduct {
	total := 0.0
	for _, p := range items {
		total += p.Revenue
	}

	sort.SliceStable(items, func(i, j int) bool {
		if items[i].Revenue != items[j].Revenue {
			return items[i].Revenue > items[j].Revenue
		}
		return items[i].Code < items[j].Code
	})

	out := make([]SellOutProduct, 0, len(items))
	for i, p := range items {
		share := 0.0
		if total > 0 {
			share = p.Revenue / total * 100
		}
		out = append(out, SellOutProduct{
			Rank: i + 1, Code: p.Code, Description: p.Description,
			GroupName: p.GroupName, Category: p.Category,
			Cartons: p.Cartons, Revenue: p.Revenue, Share: share,
		})
	}
	return out
}

func joinYears(years []int) string {
	out := ""
	for i, y := range years {
		if i > 0 {
			out += ", "
		}
		out += fmt.Sprint(y)
	}
	return out
}
