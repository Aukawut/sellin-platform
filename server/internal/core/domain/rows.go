package domain

import "time"

// ชนิดข้อมูลในไฟล์นี้คือ "แถวดิบที่อ่านออกมาจาก Excel แล้ว" ยังไม่ผ่านการรวมยอดใด ๆ
// group_name และ category ถูกเก็บไว้กับแถวโดยตั้งใจ เพราะไฟล์ต้นทางมีค่าเหล่านี้ติดมากับทุกแถว
// และบางแถวไม่ตรงกับ Data_Product การ join เอาอย่างเดียวจะทำให้ยอดไม่ตรงกับ dashboard เดิม

type Customer struct {
	Code      string
	Name      string
	Status    string
	Provinces []string
	Region    string
}

func (c Customer) IsActive() bool { return c.Status == "Active" }

type Product struct {
	Code        string
	Description string
	GroupName   string
	Category    string
	Innerbox    float64
	PriceUnit   float64
	PriceCarton float64
	Status      string
}

type SellInRow struct {
	SaleDate     *time.Time
	Year         int
	Month        int
	Day          int
	DocNo        string
	CustomerCode string
	ProductCode  string
	ProductDesc  string
	GroupName    string
	Category     string
	Qty          float64
	Unit         string
	Innerbox     float64
	Cartons      float64 // Sum Cartoon: ค่าที่ normalize หน่วยแล้ว ห้ามคำนวณใหม่จาก Qty
	PriceUnit    float64
	Revenue      float64
	StatusSale   string
}

type SellOutRow struct {
	Year         int
	Month        int
	CustomerCode string
	ProductCode  string
	ProductDesc  string
	GroupName    string
	Category     string
	Cartons      float64
	Unit         string
	PriceUnit    float64
	Revenue      float64
	Status       string
}

type StockRow struct {
	Year         int
	Month        int
	CustomerCode string
	ProductCode  string
	ProductDesc  string
	GroupName    string
	Category     string
	Beginning    float64
	SellIn       float64
	SellOut      float64
	Ending       float64
}

type TargetRow struct {
	Year         int
	Month        int
	CustomerCode string
	CustomerName string
	Category     string
	TargetAmount float64
}

type TrackingRow struct {
	Year         int
	Month        int
	CustomerCode string
	GroupName    string
	TargetQty    float64
	Cartons      float64
}

// Workbook คือผลลัพธ์ของการอ่านไฟล์ Excel หนึ่งไฟล์ ก่อนเขียนลงฐานข้อมูล
type Workbook struct {
	Customers []Customer
	Products  []Product
	SellIn    []SellInRow
	SellOut   []SellOutRow
	Stock     []StockRow
	Targets   []TargetRow
	Tracking  []TrackingRow

	SheetsRead []string
	Issues     []ImportIssue
}

func (w *Workbook) AddIssue(sheet string, row int, sev Severity, field, msg string) {
	w.Issues = append(w.Issues, ImportIssue{
		Sheet: sheet, RowNo: row, Severity: sev, Field: field, Message: msg,
	})
}

func (w *Workbook) RowCounts() map[string]int {
	return map[string]int{
		"Data_Customer":                len(w.Customers),
		"Data_Product":                 len(w.Products),
		"Data_Sell-In":                 len(w.SellIn),
		"Data_Sell-Out":                len(w.SellOut),
		"Data_Stock_inventory":         len(w.Stock),
		"Target_Sale":                  len(w.Targets),
		"Tracking_Non-Sun Flower Seed": len(w.Tracking),
	}
}

// Years คืนปีทั้งหมดที่พบใน fact tables ใช้ตั้งค่า year_min / year_max ของ dataset
func (w *Workbook) Years() (min, max int, ok bool) {
	track := func(y int) {
		if y <= 0 {
			return
		}
		if !ok || y < min {
			min = y
		}
		if !ok || y > max {
			max = y
		}
		ok = true
	}
	for _, r := range w.SellIn {
		track(r.Year)
	}
	for _, r := range w.SellOut {
		track(r.Year)
	}
	for _, r := range w.Stock {
		track(r.Year)
	}
	for _, r := range w.Targets {
		track(r.Year)
	}
	return min, max, ok
}

func (w *Workbook) HasBlockingIssue() bool {
	for _, i := range w.Issues {
		if i.Severity == SeverityBlocking {
			return true
		}
	}
	return false
}
