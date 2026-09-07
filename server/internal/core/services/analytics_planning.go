package services

import (
	"context"

	"sellin-server/internal/core/domain"
)

type PlanningPage struct {
	Scope       domain.ScopeInfo `json:"scope"`
	KPIs        []domain.KPI     `json:"kpis"`
	Trend       domain.Chart     `json:"trend"`
	CategoryBar domain.Chart     `json:"category_bar"`
	GroupTrend  domain.Chart     `json:"group_trend"`
	Regions     RegionShare      `json:"regions"`
	Plan        []domain.PlanRow `json:"plan"`
}

type RegionShare struct {
	Nodes []domain.MapNode `json:"nodes"`
	Total float64          `json:"total"`
	Best  string           `json:"best,omitempty"`
}

func (s *AnalyticsService) Planning(ctx context.Context, scope domain.Scope) (PlanningPage, error) {
	meta, err := s.requireReady(ctx, scope)
	if err != nil {
		return PlanningPage{}, err
	}

	baseYear, yearInferred, hasYear := domain.ResolveYear(scope, meta.SellOutYears)
	prevYear := baseYear - 1

	actual, err := s.sellOut.Summary(ctx, scope)
	if err != nil {
		return PlanningPage{}, err
	}
	targetTotal, err := s.target.Total(ctx, scope)
	if err != nil {
		return PlanningPage{}, err
	}

	page := PlanningPage{}

	page.KPIs = append(page.KPIs, domain.KPI{
		Key: "actual", Value: actual.Revenue, Format: domain.FormatTHB, Available: true,
	})
	page.KPIs = append(page.KPIs, domain.KPI{
		Key: "target", Value: targetTotal, Format: domain.FormatTHB, Available: true,
		Secondary: &domain.Secondary{Kind: "target_years", Text: joinYears(meta.TargetYears)},
	})

	ach := domain.KPI{Key: "achievement", Format: domain.FormatPercent}
	if targetTotal > 0 {
		ach.Value = actual.Revenue / targetTotal * 100
		ach.Available = true
		ach.Secondary = &domain.Secondary{Kind: "tier", Text: domain.AchievementTier(ach.Value, true)}
	}
	page.KPIs = append(page.KPIs, ach)

	// ---- KPI คาดการณ์ยอดสิ้นปี ----
	yearActual, monthsWithData, err := s.sellOut.YearTotal(ctx, scope.DatasetID, scope.DC, baseYear)
	if err != nil {
		return PlanningPage{}, err
	}
	forecastTotal, monthlyAvg, forecastOK := domain.Forecast(yearActual, monthsWithData)
	yearTarget, err := s.target.YearTotal(ctx, scope.DatasetID, scope.DC, baseYear)
	if err != nil {
		return PlanningPage{}, err
	}
	forecast := domain.KPI{
		Key: "forecast", Value: forecastTotal, Format: domain.FormatTHB, Available: forecastOK,
		Secondary: &domain.Secondary{Kind: "months_counted", Value: float64(monthsWithData)},
	}
	if forecastOK && yearTarget > 0 {
		// เทียบยอดคาดการณ์กับเป้าทั้งปี ไม่ใช่กับปีก่อนหน้า จึงไม่ใช่การเติบโต
		pct := forecastTotal / yearTarget * 100
		dir := domain.DirectionUp
		if forecastTotal < yearTarget {
			dir = domain.DirectionDown
		}
		forecast.Delta = &domain.Delta{
			Pct: pct, Direction: dir, Current: forecastTotal, Previous: yearTarget, BaseYear: baseYear,
		}
	}
	page.KPIs = append(page.KPIs, forecast)

	// ---- กราฟ Actual vs Target vs Forecast ----
	actualByMonth, present, err := s.sellOut.RevenueByMonth(ctx, scope.DatasetID, scope.DC, baseYear)
	if err != nil {
		return PlanningPage{}, err
	}
	targetByMonth, err := s.target.ByMonth(ctx, scope.DatasetID, scope.DC, baseYear)
	if err != nil {
		return PlanningPage{}, err
	}
	var targetPresent [12]bool
	for i, v := range targetByMonth {
		targetPresent[i] = v > 0
	}
	page.Trend = domain.Chart{
		Labels: monthLabels(),
		Tag:    trendTag(meta, scope.DC),
		Series: []domain.Series{
			{Key: "actual", Name: "Actual", Kind: "line",
				Points: domain.PointsWithGaps(actualByMonth[:], present[:])},
			{Key: "target", Name: "Target", Kind: "line",
				Points: domain.PointsWithGaps(targetByMonth[:], targetPresent[:])},
			{Key: "forecast", Name: "Forecast", Kind: "dashed",
				Points: domain.ForecastLine(actualByMonth, present, monthlyAvg)},
		},
	}

	// ---- Target vs Actual แยกตาม Product Category ----
	actualByCat, err := s.sellOut.RevenueBy(ctx, scope, "category")
	if err != nil {
		return PlanningPage{}, err
	}
	targetByCat, err := s.target.ByCategory(ctx, scope)
	if err != nil {
		return PlanningPage{}, err
	}
	page.CategoryBar = categoryChart(bucketsToMap(targetByCat), bucketsToMap(actualByCat))

	// ---- เทรนยอดขายแยกตามกลุ่มสินค้า ----
	byGroup, groupPresent, err := s.sellOut.RevenueByGroupMonthly(ctx, scope.DatasetID, scope.DC, baseYear)
	if err != nil {
		return PlanningPage{}, err
	}
	page.GroupTrend = groupTrendChart(byGroup, groupPresent, trendTag(meta, scope.DC))

	// ---- สัดส่วนรายภาค พร้อมการเทียบปีก่อน ----
	page.Regions, err = s.regionShare(ctx, scope, baseYear, prevYear)
	if err != nil {
		return PlanningPage{}, err
	}

	// ---- ตารางแผนรายศูนย์: ไม่สนใจฟิลเตอร์ศูนย์ เพราะต้องเห็นทุกศูนย์พร้อมกัน ----
	targetsByDC, err := s.target.ByDC(ctx, scope.DatasetID, scope.Year, scope.Month)
	if err != nil {
		return PlanningPage{}, err
	}
	actualsByDC, err := s.sellOut.RevenueByDC(ctx, scope.DatasetID, scope.Year, scope.Month)
	if err != nil {
		return PlanningPage{}, err
	}
	page.Plan = domain.BuildPlanRows(meta.Customers, bucketsToMap(targetsByDC), bucketsToMap(actualsByDC))

	effMonth := 0
	if scope.Month != nil {
		effMonth = *scope.Month
	}
	page.Scope = scopeInfo(scope, meta, baseYear, effMonth, yearInferred, false, hasYear)
	return page, nil
}

func (s *AnalyticsService) regionShare(ctx context.Context, scope domain.Scope, baseYear, prevYear int) (RegionShare, error) {
	current, err := s.sellOut.RevenueByRegion(ctx, scope)
	if err != nil {
		return RegionShare{}, err
	}
	curYear, err := s.sellOut.RevenueByRegionForYear(ctx, scope.DatasetID, scope.DC, scope.Month, baseYear)
	if err != nil {
		return RegionShare{}, err
	}
	prevYearData, err := s.sellOut.RevenueByRegionForYear(ctx, scope.DatasetID, scope.DC, scope.Month, prevYear)
	if err != nil {
		return RegionShare{}, err
	}

	values := bucketsToMap(current)
	curMap, prevMap := bucketsToMap(curYear), bucketsToMap(prevYearData)

	out := RegionShare{}
	for _, v := range values {
		out.Total += v
	}
	slices := domain.MakeSlices(values)
	for _, sl := range slices {
		out.Nodes = append(out.Nodes, domain.MapNode{
			Code: sl.Label, Name: sl.Label, Value: sl.Value, Pct: sl.Pct,
			YoY: domain.GrowthDelta(curMap[sl.Label], prevMap[sl.Label], baseYear, prevYear),
		})
	}
	if len(slices) > 0 && out.Total > 0 {
		out.Best = slices[0].Label
	}
	return out, nil
}

func categoryChart(targets, actuals map[string]float64) domain.Chart {
	keys := map[string]bool{}
	for k := range targets {
		keys[k] = true
	}
	for k := range actuals {
		keys[k] = true
	}

	labels := domain.SortedKeys(keys)
	t := make([]float64, len(labels))
	a := make([]float64, len(labels))
	for i, k := range labels {
		t[i], a[i] = targets[k], actuals[k]
	}
	return domain.Chart{
		Labels: labels,
		Series: []domain.Series{
			{Key: "target", Name: "Target", Kind: "bar", Points: domain.Points(t)},
			{Key: "actual", Name: "Actual", Kind: "bar", Points: domain.Points(a)},
		},
	}
}

func groupTrendChart(byGroup map[string][12]float64, present [12]bool, tag string) domain.Chart {
	chart := domain.Chart{Labels: monthLabels(), Tag: tag}
	for _, name := range domain.SortedKeys(byGroup) {
		values := byGroup[name]
		chart.Series = append(chart.Series, domain.Series{
			Key: name, Name: name, Kind: "line",
			Points: domain.PointsWithGaps(values[:], present[:]),
		})
	}
	return chart
}
