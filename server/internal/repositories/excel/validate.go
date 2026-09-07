package excel

import (
	"fmt"
	"sort"

	"sellin-server/internal/core/domain"
)

// validate ตรวจสิ่งที่ต้องมองข้ามหลาย sheet พร้อมกัน จึงทำหลังอ่านครบทุกแผ่นแล้ว
//
// หลักการรายงาน: จัดกลุ่มปัญหาชนิดเดียวกันให้เหลือรายการเดียวพร้อมจำนวน
// แทนที่จะยิงทีละแถว เพราะไฟล์ที่ผิดรูปแบบมักผิดเป็นพันแถวและอ่านไม่ไหว
func (r *Reader) validate(wb *domain.Workbook, present map[string]bool) {
	r.validateRequiredSheets(wb, present)
	r.validateRevenue(wb)
	r.validateCustomerRefs(wb)
	r.validateProductRefs(wb)
	r.validateYearSpread(wb)
}

func (r *Reader) validateRequiredSheets(wb *domain.Workbook, present map[string]bool) {
	for _, name := range []string{SheetSellIn, SheetCustomer} {
		if !present[name] {
			wb.AddIssue(name, 0, domain.SeverityBlocking, "",
				"ไม่พบ sheet "+name+" ในไฟล์ — ชื่อ sheet ต้องตรงกับ Template")
		}
	}
	if len(wb.SheetsRead) == 0 {
		wb.AddIssue("", 0, domain.SeverityBlocking, "",
			"ไม่พบ sheet ที่ตรงกับ Template เลย ตรวจสอบว่าอัปโหลดไฟล์ถูกไฟล์หรือไม่")
	}
}

// validateRevenue จับกรณีที่ไฟล์อ่านได้แต่ยอดเป็นศูนย์ทั้งหมด ซึ่งเกือบทุกครั้งแปลว่า
// คอลัมน์ Revenue เป็นสูตรที่ไม่มีค่าแคชติดมากับไฟล์ ไม่ใช่ว่ายอดขายเป็นศูนย์จริง
func (r *Reader) validateRevenue(wb *domain.Workbook) {
	if len(wb.SellIn) > 0 {
		var total float64
		for _, row := range wb.SellIn {
			total += row.Revenue
		}
		if total == 0 {
			wb.AddIssue(SheetSellIn, 0, domain.SeverityBlocking, "Revenue",
				fmt.Sprintf("อ่านได้ %d แถว แต่ยอด Revenue รวมเป็น 0 — ตรวจสอบว่าคอลัมน์ Revenue เป็นตัวเลข ไม่ใช่สูตรที่ยังไม่ได้คำนวณ", len(wb.SellIn)))
		}
	}
	if len(wb.SellOut) > 0 {
		var total float64
		for _, row := range wb.SellOut {
			total += row.Revenue
		}
		if total == 0 {
			wb.AddIssue(SheetSellOut, 0, domain.SeverityBlocking, "Revenue",
				fmt.Sprintf("อ่านได้ %d แถว แต่ยอด Revenue รวมเป็น 0 — ตรวจสอบคอลัมน์ Revenue ในไฟล์", len(wb.SellOut)))
		}
	}
}

func (r *Reader) validateCustomerRefs(wb *domain.Workbook) {
	if len(wb.Customers) == 0 {
		return
	}
	known := make(map[string]bool, len(wb.Customers))
	for _, c := range wb.Customers {
		known[c.Code] = true
	}

	unknown := map[string]map[string]int{} // sheet -> code -> จำนวนแถว
	note := func(sheetName, code string) {
		if code == "" || known[code] {
			return
		}
		if unknown[sheetName] == nil {
			unknown[sheetName] = map[string]int{}
		}
		unknown[sheetName][code]++
	}

	for _, row := range wb.SellIn {
		note(SheetSellIn, row.CustomerCode)
	}
	for _, row := range wb.SellOut {
		note(SheetSellOut, row.CustomerCode)
	}
	for _, row := range wb.Stock {
		note(SheetStock, row.CustomerCode)
	}
	for _, row := range wb.Targets {
		note(SheetTarget, row.CustomerCode)
	}
	for _, row := range wb.Tracking {
		note(SheetTracking, row.CustomerCode)
	}

	for _, sheetName := range sortedKeys(unknown) {
		codes := unknown[sheetName]
		total := 0
		for _, n := range codes {
			total += n
		}
		wb.AddIssue(sheetName, 0, domain.SeverityWarning, "Customer Code",
			fmt.Sprintf("พบรหัสศูนย์ที่ไม่มีใน %s จำนวน %d รหัส (%s) รวม %d แถว — แถวเหล่านี้จะไม่ถูกจัดกลุ่มตามศูนย์",
				SheetCustomer, len(codes), joinSample(codes, 5), total))
	}
}

func (r *Reader) validateProductRefs(wb *domain.Workbook) {
	if len(wb.Products) == 0 {
		return
	}
	known := make(map[string]bool, len(wb.Products))
	for _, p := range wb.Products {
		known[p.Code] = true
	}
	missing := map[string]int{}
	for _, row := range wb.Stock {
		if row.ProductCode != "" && !known[row.ProductCode] {
			missing[row.ProductCode]++
		}
	}
	for _, row := range wb.SellOut {
		if row.ProductCode != "" && !known[row.ProductCode] {
			missing[row.ProductCode]++
		}
	}
	if len(missing) > 0 {
		wb.AddIssue(SheetProduct, 0, domain.SeverityWarning, "Product Code",
			fmt.Sprintf("มีสินค้า %d รหัสที่ปรากฏในข้อมูลขายหรือสต๊อก แต่ไม่มีใน %s (%s) — ตาราง Stock จะแสดงสถานะเป็น N/A",
				len(missing), SheetProduct, joinSample(missing, 5)))
	}
}

// validateYearSpread จับปีที่พิมพ์ผิด เช่น 2062 แทน 2026 ซึ่งจะลากแกนกราฟยาวจนอ่านไม่ได้
func (r *Reader) validateYearSpread(wb *domain.Workbook) {
	min, max, ok := wb.Years()
	if !ok || max-min <= 2 {
		return
	}
	wb.AddIssue("", 0, domain.SeverityWarning, "Year",
		fmt.Sprintf("ข้อมูลครอบคลุมปี %d ถึง %d ซึ่งห่างกัน %d ปี — ตรวจสอบว่ามีปีที่พิมพ์ผิดหรือไม่",
			min, max, max-min))
}

func sortedKeys(m map[string]map[string]int) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	sort.Strings(out)
	return out
}

// joinSample แสดงตัวอย่างรหัสไม่เกิน limit ตัว เรียงตามตัวอักษรให้ผลลัพธ์คงที่ทุกครั้ง
func joinSample(m map[string]int, limit int) string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	out := ""
	for i, k := range keys {
		if i == limit {
			out += fmt.Sprintf(" และอีก %d รหัส", len(keys)-limit)
			break
		}
		if i > 0 {
			out += ", "
		}
		out += k
	}
	return out
}
