package excel

import "testing"

func TestNormalizeHeader(t *testing.T) {
	cases := map[string]string{
		"  Customer Code  ": "customer code",
		"CUSTOMER   CODE":   "customer code",
		"Sum Cartoon":       "sum cartoon",
		"Product\nGroup":    "product group",
		"\ufeffYear":        "year",
		"Target Qty":        "target qty",
		"":                  "",
	}
	for in, want := range cases {
		if got := normalizeHeader(in); got != want {
			t.Errorf("normalizeHeader(%q) = %q ต้องการ %q", in, got, want)
		}
	}
}

func TestParseNum(t *testing.T) {
	cases := map[string]float64{
		"1234.5":       1234.5,
		"1,234.50":     1234.5,
		"฿1,234":       1234,
		"$ 99.99":      99.99,
		"(1,200)":      -1200, // ตัวเลขติดลบแบบบัญชี
		"":             0,
		"ไม่ใช่ตัวเลข": 0,
		"0":            0,
	}
	for in, want := range cases {
		if got := parseNum(in); got != want {
			t.Errorf("parseNum(%q) = %v ต้องการ %v", in, got, want)
		}
	}
}

// TestEndingStockAlias เฝ้าคอลัมน์ที่ dashboard เดิมอ่านผิดโดยตรง
// ไฟล์จริงใช้หัวคอลัมน์ "Ending Stock" ส่วนโค้ดเดิมมองหา "Stock on Hand"
// ระบบใหม่ต้องอ่านได้ทั้งสองชื่อ
func TestEndingStockAlias(t *testing.T) {
	for _, header := range []string{"Ending Stock", "Stock on Hand", "ending stock"} {
		s := newSheet(SheetStock, [][]string{
			{"Year", "Month", "Customer Code", header},
			{"2026", "8", "BP002", "1250.75"},
		})
		if !s.has("Ending Stock", "Stock on Hand", "Ending") {
			t.Fatalf("หัวคอลัมน์ %q ควรถูกจับคู่ได้", header)
		}
		var got float64
		s.each(func(r row) { got = r.num("Ending Stock", "Stock on Hand", "Ending") })
		if got != 1250.75 {
			t.Errorf("หัวคอลัมน์ %q: อ่านได้ %v ต้องการ 1250.75", header, got)
		}
	}
}

// TestValFallsThroughEmptyCells ยืนยันพฤติกรรมเดิมของ val(): ข้ามคอลัมน์ที่ว่างไปหา candidate ถัดไป
func TestValFallsThroughEmptyCells(t *testing.T) {
	s := newSheet("x", [][]string{
		{"Sum Cartoon", "Sum Carton"},
		{"", "42"},
	})
	var got float64
	s.each(func(r row) { got = r.num("Sum Cartoon", "Sum Carton") })
	if got != 42 {
		t.Errorf("อ่านได้ %v ต้องการ 42 (ต้องข้ามคอลัมน์ว่างไปใช้ชื่อสำรอง)", got)
	}
}

func TestSplitProvinces(t *testing.T) {
	got := splitProvinces("จังหวัดร้อยเอ็ด , จังหวัดกาฬสินธุ์,\nจังหวัดมหาสารคาม")
	want := []string{"ร้อยเอ็ด", "กาฬสินธุ์", "มหาสารคาม"}
	if len(got) != len(want) {
		t.Fatalf("ได้ %d จังหวัด ต้องการ %d: %v", len(got), len(want), got)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("จังหวัดที่ %d: ได้ %q ต้องการ %q", i, got[i], want[i])
		}
	}
}

func TestBlankRowDetection(t *testing.T) {
	s := newSheet("x", [][]string{
		{"Year", "Month"},
		{"", "  "},
		{"2026", "8"},
	})
	blanks := 0
	s.each(func(r row) {
		if r.isBlank() {
			blanks++
		}
	})
	if blanks != 1 {
		t.Errorf("นับแถวว่างได้ %d ต้องการ 1", blanks)
	}
}
