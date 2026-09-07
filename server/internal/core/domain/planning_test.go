package domain

import "testing"

// TestForecastUsesMonthsWithData เฝ้าจุดที่พลาดง่ายที่สุดของสูตรคาดการณ์
// ต้องหารด้วยจำนวนเดือนที่มีข้อมูลจริง ไม่ใช่ 12 เสมอ
func TestForecastUsesMonthsWithData(t *testing.T) {
	// ข้อมูลจริงของไฟล์ปัจจุบัน: Sell-Out ปี 2569 มี 7 เดือน รวม 122.07 ล้านบาท
	total, avg, ok := Forecast(122_070_000, 7)
	if !ok {
		t.Fatal("ควรคำนวณได้")
	}
	wantAvg := 122_070_000.0 / 7
	if avg != wantAvg {
		t.Errorf("ค่าเฉลี่ยต่อเดือน ได้ %.2f ต้องการ %.2f", avg, wantAvg)
	}
	if total != wantAvg*12 {
		t.Errorf("ยอดคาดการณ์ทั้งปี ได้ %.2f ต้องการ %.2f", total, wantAvg*12)
	}

	// ถ้าหารด้วย 12 ผิด ๆ จะได้ค่าเฉลี่ยราว 10.17 ล้าน แทนที่จะเป็น 17.44 ล้าน
	if avg < 15_000_000 {
		t.Errorf("ค่าเฉลี่ยต่ำผิดปกติ (%.2f) — น่าจะหารด้วย 12 แทนจำนวนเดือนที่มีข้อมูล", avg)
	}

	if _, _, ok := Forecast(0, 0); ok {
		t.Error("ยังไม่มีเดือนไหนมีข้อมูล ต้องคืน ok=false")
	}
}

func TestForecastLineConnectsToActual(t *testing.T) {
	var actual [12]float64
	var present [12]bool
	for i := 0; i < 7; i++ {
		actual[i] = float64(100 + i)
		present[i] = true
	}

	line := ForecastLine(actual, present, 200)

	// เดือน 1–6 ต้องว่าง เพราะเส้นคาดการณ์ไม่ควรทับช่วงที่มียอดจริงแล้ว
	for i := 0; i < 6; i++ {
		if line[i] != nil {
			t.Errorf("เดือนที่ %d ควรว่าง แต่ได้ %v", i+1, *line[i])
		}
	}
	// เดือนที่ 7 คือจุดเชื่อม ต้องเท่ากับยอดจริงพอดี ไม่ใช่ค่าเฉลี่ย
	if line[6] == nil || *line[6] != actual[6] {
		t.Errorf("จุดเชื่อมที่เดือน 7 ต้องเท่ากับยอดจริง %v", actual[6])
	}
	for i := 7; i < 12; i++ {
		if line[i] == nil || *line[i] != 200 {
			t.Errorf("เดือนที่ %d ต้องเป็นค่าเฉลี่ย 200", i+1)
		}
	}
}

func TestForecastLineWithNoActualData(t *testing.T) {
	line := ForecastLine([12]float64{}, [12]bool{}, 500)
	for i, v := range line {
		if v != nil {
			t.Errorf("ไม่มียอดจริงเลย เดือนที่ %d ไม่ควรมีค่าคาดการณ์ แต่ได้ %v", i+1, *v)
		}
	}
}

// TestBuildPlanRowsOrdering ยืนยันว่าศูนย์ที่ไม่มีเป้าหมายตกไปท้ายตารางเสมอ
// ไม่ปะปนกับศูนย์ที่มีเป้าแต่ทำได้ 0% ซึ่งเป็นคนละความหมาย
func TestBuildPlanRowsOrdering(t *testing.T) {
	customers := []Customer{
		{Code: "A", Name: "ศูนย์ A", Status: "Active"},
		{Code: "B", Name: "ศูนย์ B", Status: "Active"},
		{Code: "C", Name: "ศูนย์ C", Status: "Active"},
		{Code: "D", Name: "ศูนย์ D", Status: "Active"},
		{Code: "Z", Name: "ศูนย์ที่ปิดแล้ว", Status: "Non-Active"},
	}
	targets := map[string]float64{"A": 100, "B": 100, "C": 100}
	actuals := map[string]float64{"A": 120, "B": 0, "C": 85, "D": 999}

	rows := BuildPlanRows(customers, targets, actuals)

	if len(rows) != 4 {
		t.Fatalf("ต้องแสดงเฉพาะศูนย์ Active 4 แห่ง แต่ได้ %d", len(rows))
	}
	want := []string{"A", "C", "B", "D"}
	for i, code := range want {
		if rows[i].Code != code {
			t.Errorf("ตำแหน่งที่ %d: ได้ %s ต้องการ %s", i, rows[i].Code, code)
		}
	}
	if rows[3].HasTarget || rows[3].Tier != TierNoTarget {
		t.Error("ศูนย์ D ไม่มีเป้าหมาย ต้องอยู่ท้ายตารางและมี tier เป็น no_target")
	}
	if rows[0].Tier != TierReached {
		t.Errorf("ศูนย์ A ทำได้ 120%% ต้องเป็น %s ได้ %s", TierReached, rows[0].Tier)
	}
	if rows[1].Tier != TierNear {
		t.Errorf("ศูนย์ C ทำได้ 85%% ต้องเป็น %s ได้ %s", TierNear, rows[1].Tier)
	}
	if rows[2].Tier != TierBehind {
		t.Errorf("ศูนย์ B ทำได้ 0%% ต้องเป็น %s ได้ %s", TierBehind, rows[2].Tier)
	}
	if rows[1].Gap != 15 {
		t.Errorf("ศูนย์ C ควรขาดอีก 15 แต่ได้ %v", rows[1].Gap)
	}
}

func TestAchievementTierBoundaries(t *testing.T) {
	cases := map[float64]string{
		100.0: TierReached, 100.1: TierReached,
		99.9: TierNear, 80.0: TierNear,
		79.9: TierBehind, 0.0: TierBehind,
	}
	for pct, want := range cases {
		if got := AchievementTier(pct, true); got != want {
			t.Errorf("%.1f%%: ได้ %s ต้องการ %s", pct, got, want)
		}
	}
	if got := AchievementTier(150, false); got != TierNoTarget {
		t.Errorf("ไม่มีเป้าหมาย ต้องได้ %s ไม่ว่าเปอร์เซ็นต์จะเป็นเท่าไร ได้ %s", TierNoTarget, got)
	}
}

func TestGrowthDelta(t *testing.T) {
	d := GrowthDelta(120, 100, 2569, 2568)
	if d == nil || d.Pct != 20 || d.Direction != DirectionUp {
		t.Errorf("โตขึ้น 20%% แต่ได้ %+v", d)
	}

	d = GrowthDelta(80, 100, 2569, 2568)
	if d == nil || d.Pct != -20 || d.Direction != DirectionDown {
		t.Errorf("ลดลง 20%% แต่ได้ %+v", d)
	}

	// ปีก่อนไม่มียอดแต่ปีนี้มี คิดเป็นเปอร์เซ็นต์ไม่ได้ ต้องบอกว่า "ใหม่"
	d = GrowthDelta(500, 0, 2569, 2568)
	if d == nil || !d.IsNew {
		t.Errorf("ควรถูกทำเครื่องหมายว่าเป็นของใหม่ ได้ %+v", d)
	}

	// ไม่มีทั้งสองปี ไม่มีอะไรให้เทียบ
	if d := GrowthDelta(0, 0, 2569, 2568); d != nil {
		t.Errorf("ไม่มีข้อมูลทั้งสองปี ควรคืน nil ได้ %+v", d)
	}
}
