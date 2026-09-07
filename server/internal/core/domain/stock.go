package domain

import "sort"

// สถานะสต๊อกรายสินค้า ตรงกับที่ dashboard เดิมใช้ทุกประการ
const (
	StockStatusOK   = "ok"
	StockStatusWarn = "warn"
	StockStatusRisk = "risk"
)

// ป้ายสถานะเป็นภาษาไทยและถูกใช้เป็นค่าของ dropdown กรองสถานะด้วย
// จึงต้องคงข้อความให้ตรงกันทั้งสองที่
const (
	StockLabelNormal   = "ปกติ"
	StockLabelOutOf    = "หมดสต๊อก"
	StockLabelOverfull = "สต๊อกเกิน"
	StockLabelLow      = "สต๊อกต่ำ"
	StockLabelNoSales  = "ไม่มีการขายออก"
)

// StockProduct คือหนึ่งแถวในตาราง "สถานะสต๊อกรายสินค้า"
type StockProduct struct {
	Code        string   `json:"code"`
	Description string   `json:"description"`
	GroupName   string   `json:"group_name"`
	Beginning   float64  `json:"beginning"`
	SellIn      float64  `json:"sell_in"`
	SellOut     float64  `json:"sell_out"`
	Ending      float64  `json:"ending"`
	CoverMonths *float64 `json:"cover_months"`
	Status      string   `json:"status"`
	Label       string   `json:"label"`
	// ProductStatus มาจาก Data_Product ไม่ใช่สถานะสต๊อก
	// สินค้า Non-Active ที่ยังมีของค้างคือสิ่งที่ต้องเคลียร์ออก จึงต้องเห็นคู่กัน
	ProductStatus string `json:"product_status"`
}

// ClassifyStock ตัดสินสถานะของสินค้าหนึ่งรายการจากตัวเลขสต๊อกของเดือนสแนปช็อต
//
// ลำดับการตรวจสำคัญมากและคัดลอกมาจาก dashboard เดิมตรง ๆ:
// ของหมดถือว่าเสี่ยงที่สุด แล้วจึงดูสต๊อกเกิน แล้วสต๊อกต่ำ
// ส่วนสินค้าที่ยังมีของแต่ไม่มีการขายออกเลยถือเป็นข้อควรระวัง ไม่ใช่ปกติ
func ClassifyStock(beginning, sellIn, sellOut, ending float64) (status, label string, cover *float64) {
	if sellOut > 0 {
		c := ending / sellOut
		cover = &c
	}

	switch {
	case ending <= 0 && (beginning > 0 || sellIn > 0):
		return StockStatusRisk, StockLabelOutOf, cover
	case cover != nil && *cover > 3:
		return StockStatusWarn, StockLabelOverfull, cover
	case cover != nil && *cover < 0.5:
		return StockStatusRisk, StockLabelLow, cover
	case cover == nil && ending > 0:
		return StockStatusWarn, StockLabelNoSales, cover
	default:
		return StockStatusOK, StockLabelNormal, cover
	}
}

// SortStockProducts เรียงให้รายการที่ต้องจัดการก่อนอยู่บนสุด
// ภายในระดับความเสี่ยงเดียวกันเรียงตามปริมาณคงเหลือจากมากไปน้อย
func SortStockProducts(items []StockProduct) {
	priority := map[string]int{StockStatusRisk: 0, StockStatusWarn: 1, StockStatusOK: 2}
	sort.SliceStable(items, func(i, j int) bool {
		pi, pj := priority[items[i].Status], priority[items[j].Status]
		if pi != pj {
			return pi < pj
		}
		if items[i].Ending != items[j].Ending {
			return items[i].Ending > items[j].Ending
		}
		return items[i].Code < items[j].Code
	})
}

// SellThroughRate คือสัดส่วนที่ขายออกได้จากของที่พร้อมขายทั้งหมดในเดือนนั้น
// ตัวหารคือสต๊อกต้นเดือนบวกของที่รับเข้ามา ไม่ใช่แค่ของที่รับเข้า
func SellThroughRate(beginning, sellIn, sellOut float64) (rate float64, available float64, ok bool) {
	available = beginning + sellIn
	if available <= 0 {
		return 0, available, false
	}
	return sellOut / available * 100, available, true
}

// StockCover ประมาณว่าของที่เหลืออยู่จะขายได้อีกกี่เดือน
//
// ใช้ค่าเฉลี่ยการขายออกของเดือนล่าสุดที่ "มีข้อมูลจริง" ไม่เกิน 3 เดือน
// โดยข้ามเดือนที่ไม่มีแถวเลย ไม่นับเป็นศูนย์ ซึ่งจะทำให้ค่าเฉลี่ยต่ำผิดปกติ
func StockCover(ending float64, recentSellOut []float64) (months float64, monthsCounted int, ok bool) {
	if len(recentSellOut) == 0 {
		return 0, 0, false
	}
	sum := 0.0
	for _, v := range recentSellOut {
		sum += v
	}
	avg := sum / float64(len(recentSellOut))
	if avg <= 0 {
		return 0, len(recentSellOut), false
	}
	return ending / avg, len(recentSellOut), true
}
