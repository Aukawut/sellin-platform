package domain

import "sort"

// PlanRow คือหนึ่งแถวในตาราง "แผนปฏิบัติการรายศูนย์"
type PlanRow struct {
	Code        string   `json:"code"`
	Name        string   `json:"name"`
	Target      float64  `json:"target"`
	Actual      float64  `json:"actual"`
	Achievement *float64 `json:"achievement"`
	Gap         float64  `json:"gap"`
	HasTarget   bool     `json:"has_target"`
	// Tier ใช้เลือกสีของ pill: reached / near / behind / no_target
	Tier string `json:"tier"`
}

const (
	TierReached  = "reached"
	TierNear     = "near"
	TierBehind   = "behind"
	TierNoTarget = "no_target"
)

// AchievementTier แบ่งระดับตามเกณฑ์เดียวกับ achColor() ของ dashboard เดิม
func AchievementTier(pct float64, hasTarget bool) string {
	switch {
	case !hasTarget:
		return TierNoTarget
	case pct >= 100:
		return TierReached
	case pct >= 80:
		return TierNear
	default:
		return TierBehind
	}
}

// BuildPlanRows ประกอบตารางแผนจากเป้าหมายและยอดจริงรายศูนย์
//
// แสดงศูนย์ที่ Active ทุกแห่งแม้ยังไม่มียอด เพราะตารางนี้ใช้วางแผน ไม่ใช่แค่รายงานผล
// ศูนย์ที่ไม่มีเป้าหมายถูกจัดไว้ท้ายตารางเสมอ ไม่ปะปนกับศูนย์ที่ทำได้ 0%
func BuildPlanRows(customers []Customer, targetByDC, actualByDC map[string]float64) []PlanRow {
	rows := make([]PlanRow, 0, len(customers))
	for _, c := range customers {
		if !c.IsActive() {
			continue
		}
		target := targetByDC[c.Code]
		actual := actualByDC[c.Code]
		row := PlanRow{
			Code: c.Code, Name: c.Name,
			Target: target, Actual: actual,
			Gap: target - actual, HasTarget: target > 0,
		}
		if row.HasTarget {
			pct := actual / target * 100
			row.Achievement = &pct
			row.Tier = AchievementTier(pct, true)
		} else {
			row.Tier = TierNoTarget
		}
		rows = append(rows, row)
	}

	sort.SliceStable(rows, func(i, j int) bool {
		a, b := rows[i], rows[j]
		if a.HasTarget != b.HasTarget {
			return a.HasTarget
		}
		if a.HasTarget && *a.Achievement != *b.Achievement {
			return *a.Achievement > *b.Achievement
		}
		if a.Actual != b.Actual {
			return a.Actual > b.Actual
		}
		return a.Name < b.Name
	})
	return rows
}

// Forecast ประมาณยอดทั้งปีจากอัตราขายเฉลี่ยของเดือนที่มีข้อมูลจริง
//
// หารด้วยจำนวนเดือนที่มีข้อมูล ไม่ใช่ 12 เสมอ — ถ้าเพิ่งมีข้อมูล 8 เดือน
// การหารด้วย 12 จะทำให้ค่าเฉลี่ยต่ำกว่าความจริงหนึ่งในสาม แล้วคาดการณ์ต่ำตามไปด้วย
func Forecast(yearActual float64, monthsWithData int) (total, monthlyAverage float64, ok bool) {
	if monthsWithData <= 0 {
		return 0, 0, false
	}
	monthlyAverage = yearActual / float64(monthsWithData)
	return monthlyAverage * 12, monthlyAverage, true
}

// ForecastLine สร้างเส้นคาดการณ์ที่เริ่มจากเดือนสุดท้ายที่มียอดจริง
//
// จุดที่เดือนสุดท้ายถูกตั้งให้เท่ากับยอดจริง เพื่อให้เส้นคาดการณ์ต่อกับเส้นจริงพอดี
// ไม่ลอยแยกเป็นสองเส้นที่ไม่เชื่อมกัน
func ForecastLine(actualByMonth [12]float64, present [12]bool, monthlyAverage float64) []*float64 {
	lastActual := -1
	for i := 11; i >= 0; i-- {
		if present[i] {
			lastActual = i
			break
		}
	}

	out := make([]*float64, 12)
	if lastActual < 0 {
		return out
	}
	out[lastActual] = ptr(actualByMonth[lastActual])
	for i := lastActual + 1; i < 12; i++ {
		out[i] = ptr(monthlyAverage)
	}
	return out
}
