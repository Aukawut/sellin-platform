package domain

import "testing"

func TestClassifyStock(t *testing.T) {
	cases := []struct {
		name                         string
		beg, sellIn, sellOut, ending float64
		wantStatus, wantLabel        string
		wantCover                    *float64
	}{
		{
			name: "ของหมดทั้งที่เคยมีของต้นเดือน",
			beg:  100, sellIn: 0, sellOut: 100, ending: 0,
			wantStatus: StockStatusRisk, wantLabel: StockLabelOutOf, wantCover: f(0),
		},
		{
			name: "ของหมดทั้งที่เพิ่งรับเข้ามา",
			beg:  0, sellIn: 50, sellOut: 50, ending: 0,
			wantStatus: StockStatusRisk, wantLabel: StockLabelOutOf, wantCover: f(0),
		},
		{
			name: "สต๊อกเกินสามเดือน",
			beg:  500, sellIn: 100, sellOut: 100, ending: 500,
			wantStatus: StockStatusWarn, wantLabel: StockLabelOverfull, wantCover: f(5),
		},
		{
			name: "สต๊อกต่ำกว่าครึ่งเดือน",
			beg:  100, sellIn: 0, sellOut: 100, ending: 40,
			wantStatus: StockStatusRisk, wantLabel: StockLabelLow, wantCover: f(0.4),
		},
		{
			name: "มีของแต่ไม่มีการขายออกเลย",
			beg:  200, sellIn: 0, sellOut: 0, ending: 200,
			wantStatus: StockStatusWarn, wantLabel: StockLabelNoSales, wantCover: nil,
		},
		{
			name: "ปกติ อยู่ระหว่างครึ่งเดือนถึงสามเดือน",
			beg:  200, sellIn: 100, sellOut: 100, ending: 200,
			wantStatus: StockStatusOK, wantLabel: StockLabelNormal, wantCover: f(2),
		},
		{
			name: "ไม่มีอะไรเลยทั้งเดือน ถือว่าปกติ ไม่ใช่ของหมด",
			beg:  0, sellIn: 0, sellOut: 0, ending: 0,
			wantStatus: StockStatusOK, wantLabel: StockLabelNormal, wantCover: nil,
		},
		{
			name: "ขอบเขตพอดีสามเดือน ยังไม่ถือว่าเกิน",
			beg:  300, sellIn: 0, sellOut: 100, ending: 300,
			wantStatus: StockStatusOK, wantLabel: StockLabelNormal, wantCover: f(3),
		},
		{
			name: "ขอบเขตพอดีครึ่งเดือน ยังไม่ถือว่าต่ำ",
			beg:  100, sellIn: 0, sellOut: 100, ending: 50,
			wantStatus: StockStatusOK, wantLabel: StockLabelNormal, wantCover: f(0.5),
		},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			status, label, cover := ClassifyStock(c.beg, c.sellIn, c.sellOut, c.ending)
			if status != c.wantStatus || label != c.wantLabel {
				t.Errorf("ได้ %s/%s ต้องการ %s/%s", status, label, c.wantStatus, c.wantLabel)
			}
			switch {
			case c.wantCover == nil && cover != nil:
				t.Errorf("cover ควรเป็น nil แต่ได้ %v", *cover)
			case c.wantCover != nil && cover == nil:
				t.Errorf("cover ควรเป็น %v แต่ได้ nil", *c.wantCover)
			case c.wantCover != nil && *cover != *c.wantCover:
				t.Errorf("cover ได้ %v ต้องการ %v", *cover, *c.wantCover)
			}
		})
	}
}

func TestSortStockProducts(t *testing.T) {
	items := []StockProduct{
		{Code: "C", Status: StockStatusOK, Ending: 900},
		{Code: "A", Status: StockStatusWarn, Ending: 100},
		{Code: "B", Status: StockStatusRisk, Ending: 5},
		{Code: "D", Status: StockStatusWarn, Ending: 400},
	}
	SortStockProducts(items)

	want := []string{"B", "D", "A", "C"}
	for i, code := range want {
		if items[i].Code != code {
			t.Errorf("ตำแหน่งที่ %d: ได้ %s ต้องการ %s (ลำดับที่ได้ทั้งหมด: %v)",
				i, items[i].Code, code, codes(items))
		}
	}
}

func TestSellThroughRate(t *testing.T) {
	rate, avail, ok := SellThroughRate(100, 300, 200)
	if !ok || avail != 400 || rate != 50 {
		t.Errorf("ได้ rate=%v avail=%v ok=%v ต้องการ 50/400/true", rate, avail, ok)
	}

	// ไม่มีของให้ขายเลย คำนวณอัตราไม่ได้ ต้องไม่คืน 0% ซึ่งสื่อว่าขายไม่ออก
	if _, _, ok := SellThroughRate(0, 0, 0); ok {
		t.Error("เมื่อไม่มีของพร้อมขาย ต้องคืน ok=false")
	}
}

func TestStockCover(t *testing.T) {
	// เฉลี่ยขายออก 3 เดือน = (100+200+300)/3 = 200 คงเหลือ 600 → 3 เดือน
	months, counted, ok := StockCover(600, []float64{100, 200, 300})
	if !ok || months != 3 || counted != 3 {
		t.Errorf("ได้ %v เดือน จาก %d เดือน ok=%v ต้องการ 3/3/true", months, counted, ok)
	}

	// มีข้อมูลแค่เดือนเดียว ต้องหารด้วย 1 ไม่ใช่ 3
	months, counted, _ = StockCover(600, []float64{200})
	if months != 3 || counted != 1 {
		t.Errorf("ได้ %v เดือน จาก %d เดือน ต้องการ 3/1", months, counted)
	}

	if _, _, ok := StockCover(600, nil); ok {
		t.Error("ไม่มีข้อมูลย้อนหลังเลย ต้องคืน ok=false")
	}
	if _, _, ok := StockCover(600, []float64{0, 0}); ok {
		t.Error("ขายออกเฉลี่ยเป็นศูนย์ ต้องคืน ok=false ไม่ใช่หารด้วยศูนย์")
	}
}

func f(v float64) *float64 { return &v }

func codes(items []StockProduct) []string {
	out := make([]string, len(items))
	for i, it := range items {
		out[i] = it.Code
	}
	return out
}
