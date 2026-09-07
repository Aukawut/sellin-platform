// Package excel อ่านไฟล์ Template ของฝ่ายขายแล้วแปลงเป็นโครงสร้างข้อมูลของ domain
//
// เป็น adapter ตัวเดียวในระบบที่รู้จักรูปแบบไฟล์ Excel — ทุกอย่างที่อยู่ลึกกว่านี้
// ทำงานกับ domain.Workbook เท่านั้น ไม่รู้ว่าข้อมูลมาจากไฟล์แบบไหน
package excel

import (
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/xuri/excelize/v2"

	"sellin-server/internal/core/domain"
)

// ชื่อ sheet ตาม Template ที่ตกลงกันว่าจะคงเดิม
const (
	SheetCustomer = "Data_Customer"
	SheetProduct  = "Data_Product"
	SheetSellIn   = "Data_Sell-In"
	SheetSellOut  = "Data_Sell-Out"
	SheetStock    = "Data_Stock_inventory"
	SheetTarget   = "Target_Sale"
	SheetTracking = "Tracking_Non-Sun Flower Seed"
)

// DCRegion คือภาคของศูนย์กระจายสินค้า ซึ่ง dashboard เดิม hardcode ไว้ใน JavaScript
// ที่นี่ใช้เป็นค่าตั้งต้นตอนนำเข้าเท่านั้น เพราะไฟล์ Template ไม่มีคอลัมน์ภาค
// เมื่อบันทึกลง dim_customers แล้ว admin แก้ไขผ่านหน้า master data ได้
var DCRegion = map[string]string{
	"BP018": "อีสาน", "BP022": "เหนือ", "BP015": "อีสาน", "BP010": "เหนือ",
	"BP010-1": "อีสาน", "BP024": "อีสาน", "TT091": "อีสาน", "BP069": "กลาง",
	"BP023": "ตะวันออก", "TT065": "กลาง", "BP007": "ตะวันตก", "BP009": "อีสาน",
	"BP070": "กลาง", "BP002": "กลาง", "BP011": "ใต้", "BP012": "อีสาน",
	"BP071": "เหนือ", "BP004": "กลาง",
}

type Reader struct{}

func NewReader() *Reader { return &Reader{} }

// Read อ่านไฟล์ทั้งไฟล์แล้วคืน Workbook พร้อมรายการปัญหาที่พบ
//
// error คืนเฉพาะกรณีที่เปิดไฟล์ไม่ได้เลย ส่วนปัญหาระดับข้อมูลจะอยู่ใน Workbook.Issues
// เพื่อให้ผู้ใช้เห็นภาพรวมทั้งไฟล์ในครั้งเดียว แทนที่จะแก้ทีละจุดแล้วอัปโหลดใหม่ซ้ำ ๆ
func (r *Reader) Read(src io.Reader) (*domain.Workbook, error) {
	f, err := excelize.OpenReader(src)
	if err != nil {
		return nil, fmt.Errorf("เปิดไฟล์ไม่สำเร็จ ไฟล์อาจเสียหายหรือไม่ใช่ .xlsx: %w", err)
	}
	defer f.Close()

	wb := &domain.Workbook{}
	present := map[string]bool{}
	for _, name := range f.GetSheetList() {
		present[name] = true
	}

	load := func(name string) *sheet {
		if !present[name] {
			return nil
		}
		// RawCellValue: true สำคัญมาก — ถ้าไม่ตั้ง excelize จะคืนค่าที่ผ่านการจัดรูปแบบตัวเลขของเซลล์แล้ว
		// ทำให้ยอดที่แสดงทศนิยม 2 ตำแหน่งถูกปัดตั้งแต่ตอนอ่าน และยอดรวมทั้งไฟล์คลาดจากต้นฉบับ
		raw, err := f.GetRows(name, excelize.Options{RawCellValue: true})
		if err != nil {
			wb.AddIssue(name, 0, domain.SeverityBlocking, "", "อ่าน sheet นี้ไม่สำเร็จ: "+err.Error())
			return nil
		}
		if len(raw) < 2 {
			wb.AddIssue(name, 0, domain.SeverityWarning, "", "sheet นี้ไม่มีข้อมูลใต้หัวตาราง")
			return nil
		}
		wb.SheetsRead = append(wb.SheetsRead, name)
		return newSheet(name, raw)
	}

	r.readCustomers(wb, load(SheetCustomer))
	r.readProducts(wb, load(SheetProduct))
	r.readSellIn(wb, load(SheetSellIn))
	r.readSellOut(wb, load(SheetSellOut))
	r.readStock(wb, load(SheetStock))
	r.readTargets(wb, load(SheetTarget))
	r.readTracking(wb, load(SheetTracking))

	r.validate(wb, present)
	return wb, nil
}

func (r *Reader) readCustomers(wb *domain.Workbook, s *sheet) {
	if s == nil {
		return
	}
	if !s.has("Customer Code") {
		wb.AddIssue(s.name, 1, domain.SeverityBlocking, "Customer Code", "ไม่พบคอลัมน์ Customer Code")
		return
	}
	seen := map[string]bool{}
	s.each(func(row row) {
		code := row.str("Customer Code")
		if code == "" {
			return
		}
		if seen[code] {
			wb.AddIssue(s.name, row.no, domain.SeverityWarning, "Customer Code",
				fmt.Sprintf("รหัสศูนย์ %s ซ้ำกับแถวก่อนหน้า ระบบใช้แถวแรกเท่านั้น", code))
			return
		}
		seen[code] = true

		status := row.str("Customer Status", "Status")
		if status == "" {
			status = "Active"
		}
		wb.Customers = append(wb.Customers, domain.Customer{
			Code:      code,
			Name:      row.str("Customer", "Customer Name"),
			Status:    status,
			Provinces: splitProvinces(row.str("Region / Province (optional)", "Region / Province", "Province")),
			Region:    DCRegion[code],
		})
	})
}

// splitProvinces แยกรายชื่อจังหวัดที่คั่นด้วยคอมมา และตัดคำว่า "จังหวัด" ออก
// ให้เหลือแต่ชื่อ ตรงกับที่ dashboard เดิมทำตอน import
func splitProvinces(raw string) []string {
	if raw == "" {
		return nil
	}
	out := []string{}
	for _, p := range strings.Split(raw, ",") {
		p = strings.TrimSpace(strings.ReplaceAll(p, "\n", ""))
		p = strings.TrimSpace(strings.TrimPrefix(p, "จังหวัด"))
		if p != "" {
			out = append(out, p)
		}
	}
	return out
}

func (r *Reader) readProducts(wb *domain.Workbook, s *sheet) {
	if s == nil {
		return
	}
	if !s.has("Product Code") {
		wb.AddIssue(s.name, 1, domain.SeverityWarning, "Product Code",
			"ไม่พบคอลัมน์ Product Code — ตาราง Stock จะไม่แสดงสถานะสินค้า")
		return
	}
	seen := map[string]bool{}
	s.each(func(row row) {
		code := row.str("Product Code")
		if code == "" || seen[code] {
			return
		}
		seen[code] = true
		wb.Products = append(wb.Products, domain.Product{
			Code:        code,
			Description: row.str("Product Description", "Description"),
			GroupName:   row.str("Product Group"),
			Category:    row.str("Product Category"),
			Innerbox:    row.num("Innerbox"),
			PriceUnit:   row.num("Price Unit"),
			PriceCarton: row.num("Price Cartoon", "Price Carton"),
			Status:      row.str("Status"),
		})
	})
}

func (r *Reader) readSellIn(wb *domain.Workbook, s *sheet) {
	if s == nil {
		return
	}
	for _, required := range []string{"Year", "Month", "Customer Code", "Revenue"} {
		if !s.has(required) {
			wb.AddIssue(s.name, 1, domain.SeverityBlocking, required,
				"ไม่พบคอลัมน์ "+required+" ซึ่งจำเป็นต่อการคำนวณทุกหน้า")
			return
		}
	}
	skipped := 0
	s.each(func(row row) {
		if row.isBlank() {
			skipped++
			return
		}
		year := row.intVal("Year")
		if year == 0 {
			skipped++
			return
		}
		rec := domain.SellInRow{
			Year:         year,
			Month:        row.intVal("Month"),
			Day:          row.intVal("Day"),
			DocNo:        row.str("No-Document", "No Document", "Document", "Doc No"),
			CustomerCode: row.str("Customer Code"),
			ProductCode:  row.str("Product Code"),
			ProductDesc:  row.str("Product Description"),
			GroupName:    row.str("Product Group"),
			Category:     row.str("Product Category"),
			Qty:          row.num("Qty", "Quantity"),
			Unit:         row.str("Unit"),
			Innerbox:     row.num("Innerbox"),
			Cartons:      row.num("Sum Cartoon", "Sum Carton"),
			PriceUnit:    row.num("Price Unit"),
			Revenue:      row.num("Revenue"),
			StatusSale:   row.str("Status Sale", "Status"),
		}
		if d, ok := buildDate(row, rec.Year, rec.Month, rec.Day); ok {
			rec.SaleDate = &d
		}
		checkMonth(wb, s.name, row.no, rec.Month)
		wb.SellIn = append(wb.SellIn, rec)
	})
	noteSkipped(wb, s.name, skipped)
}

// buildDate ประกอบวันที่จากคอลัมน์ ปี/เดือน/วัน เป็นหลัก เพราะสามคอลัมน์นี้คือค่าที่ dashboard ใช้จริง
// ส่วนคอลัมน์ Date ใช้เป็นทางสำรองเมื่อคอลัมน์ Day ว่าง
func buildDate(row row, y, m, d int) (time.Time, bool) {
	if y > 0 && m >= 1 && m <= 12 && d >= 1 && d <= 31 {
		return time.Date(y, time.Month(m), d, 0, 0, 0, 0, time.UTC), true
	}
	return row.date("Date", "Sale Date")
}

func (r *Reader) readSellOut(wb *domain.Workbook, s *sheet) {
	if s == nil {
		return
	}
	for _, required := range []string{"Year", "Month", "Customer Code", "Revenue"} {
		if !s.has(required) {
			wb.AddIssue(s.name, 1, domain.SeverityBlocking, required, "ไม่พบคอลัมน์ "+required)
			return
		}
	}
	skipped := 0
	s.each(func(row row) {
		if row.isBlank() {
			skipped++
			return
		}
		year := row.intVal("Year")
		if year == 0 {
			skipped++
			return
		}
		rec := domain.SellOutRow{
			Year:         year,
			Month:        row.intVal("Month"),
			CustomerCode: row.str("Customer Code"),
			ProductCode:  row.str("Product Code"),
			ProductDesc:  row.str("Product Description"),
			GroupName:    row.str("Product Group"),
			Category:     row.str("Product Category"),
			Cartons:      row.num("Sum Cartoon", "Sum Carton"),
			Unit:         row.str("Unit"),
			PriceUnit:    row.num("Price Unit"),
			Revenue:      row.num("Revenue"),
			Status:       row.str("Status"),
		}
		checkMonth(wb, s.name, row.no, rec.Month)
		wb.SellOut = append(wb.SellOut, rec)
	})
	noteSkipped(wb, s.name, skipped)
}

func (r *Reader) readStock(wb *domain.Workbook, s *sheet) {
	if s == nil {
		return
	}
	if !s.has("Year") || !s.has("Customer Code") {
		wb.AddIssue(s.name, 1, domain.SeverityBlocking, "", "ไม่พบคอลัมน์ Year หรือ Customer Code")
		return
	}

	// จุดที่ dashboard เดิมพลาด: โค้ดอ่านคอลัมน์ชื่อ "Stock on Hand" แต่หัวคอลัมน์จริงคือ "Ending Stock"
	// ทำให้สต๊อกคงเหลือเป็น 0 ทุกแถวหลัง import และ KPI Stock Cover กับสถานะ "หมดสต๊อก" ผิดทั้งหมด
	// ที่นี่รับทั้งสองชื่อ และเตือนถ้าหาไม่เจอสักชื่อ แทนที่จะเงียบแล้วให้ตัวเลขผิด
	if !s.has("Ending Stock", "Stock on Hand", "Ending") {
		wb.AddIssue(s.name, 1, domain.SeverityBlocking, "Ending Stock",
			"ไม่พบคอลัมน์สต๊อกคงเหลือ (รองรับชื่อ Ending Stock หรือ Stock on Hand)")
		return
	}

	skipped := 0
	s.each(func(row row) {
		if row.isBlank() {
			skipped++
			return
		}
		year := row.intVal("Year")
		if year == 0 {
			skipped++
			return
		}
		rec := domain.StockRow{
			Year:         year,
			Month:        row.intVal("Month"),
			CustomerCode: row.str("Customer Code"),
			ProductCode:  row.str("Product Code"),
			ProductDesc:  row.str("Product Description"),
			GroupName:    row.str("Product Group"),
			Category:     row.str("Product Category"),
			Beginning:    row.num("Beginning Stock", "Beginning"),
			SellIn:       row.num("Sell-In", "Sell In"),
			SellOut:      row.num("Sell-Out", "Sell Out"),
			Ending:       row.num("Ending Stock", "Stock on Hand", "Ending"),
		}
		checkMonth(wb, s.name, row.no, rec.Month)
		wb.Stock = append(wb.Stock, rec)
	})
	noteSkipped(wb, s.name, skipped)
}

func (r *Reader) readTargets(wb *domain.Workbook, s *sheet) {
	if s == nil {
		return
	}
	if !s.has("Target") {
		wb.AddIssue(s.name, 1, domain.SeverityWarning, "Target",
			"ไม่พบคอลัมน์ Target — หน้า Sell-Out และ Planning จะไม่มีเป้าหมายให้เทียบ")
		return
	}
	skipped := 0
	s.each(func(row row) {
		if row.isBlank() {
			skipped++
			return
		}
		year := row.intVal("Year")
		if year == 0 {
			skipped++
			return
		}
		rec := domain.TargetRow{
			Year:         year,
			Month:        row.intVal("Month"),
			CustomerCode: row.str("Customer Code"),
			CustomerName: row.str("Customer"),
			Category:     row.str("Product Category"),
			TargetAmount: row.num("Target"),
		}
		checkMonth(wb, s.name, row.no, rec.Month)
		wb.Targets = append(wb.Targets, rec)
	})
	noteSkipped(wb, s.name, skipped)
}

func (r *Reader) readTracking(wb *domain.Workbook, s *sheet) {
	if s == nil {
		return
	}
	skipped := 0
	s.each(func(row row) {
		if row.isBlank() {
			skipped++
			return
		}
		year := row.intVal("Year")
		if year == 0 {
			skipped++
			return
		}
		wb.Tracking = append(wb.Tracking, domain.TrackingRow{
			Year:         year,
			Month:        row.intVal("Month"),
			CustomerCode: row.str("Customer Code"),
			GroupName:    row.str("Product Group"),
			TargetQty:    row.num("Target_Qty", "Target Qty"),
			Cartons:      row.num("Sum Cartoon", "Sum Carton"),
		})
	})
	noteSkipped(wb, s.name, skipped)
}

func checkMonth(wb *domain.Workbook, sheetName string, rowNo, month int) {
	if month < 1 || month > 12 {
		wb.AddIssue(sheetName, rowNo, domain.SeverityWarning, "Month",
			fmt.Sprintf("เดือน %d อยู่นอกช่วง 1–12 แถวนี้จะไม่ปรากฏในกราฟรายเดือน", month))
	}
}

func noteSkipped(wb *domain.Workbook, sheetName string, skipped int) {
	if skipped > 0 {
		wb.AddIssue(sheetName, 0, domain.SeverityInfo, "",
			fmt.Sprintf("ข้ามแถวว่างหรือแถวที่ไม่มีค่าปี %d แถว", skipped))
	}
}
