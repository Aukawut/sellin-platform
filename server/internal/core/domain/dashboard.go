package domain

// ชนิดข้อมูลในไฟล์นี้คือ "รูปร่างของคำตอบ" ที่ทุกหน้า dashboard ใช้ร่วมกัน
//
// หลักการ: backend ส่งตัวเลขและความหมาย ส่วนการจัดรูปแบบข้อความเป็นหน้าที่ของ frontend
// dashboard เดิมประกอบข้อความไทยไว้ในโค้ด JavaScript ซึ่งทำให้แก้คำพูดทีต้องแก้ตรรกะไปด้วย

// Format บอก frontend ว่าควรแสดงตัวเลขนี้เป็นอะไร
const (
	FormatTHB     = "thb"
	FormatNumber  = "number"
	FormatPercent = "percent"
	FormatCartons = "cartons"
	FormatMonths  = "months"
	FormatText    = "text"
)

const (
	DirectionUp   = "up"
	DirectionDown = "down"
	DirectionFlat = "flat"
)

type KPI struct {
	Key    string  `json:"key"`
	Value  float64 `json:"value"`
	Text   string  `json:"text,omitempty"`
	Format string  `json:"format"`
	// Available เป็น false เมื่อคำนวณไม่ได้ เช่นไม่มีข้อมูลปีก่อนหน้าให้เทียบ
	// frontend แสดง N/A แทนที่จะแสดงเลข 0 ซึ่งสื่อความหมายผิด
	Available bool       `json:"available"`
	Delta     *Delta     `json:"delta,omitempty"`
	Secondary *Secondary `json:"secondary,omitempty"`
}

// Delta คือการเปรียบเทียบกับช่วงก่อนหน้า ส่งค่าดิบไปให้ frontend ประกอบข้อความเอง
type Delta struct {
	Pct       float64 `json:"pct"`
	Direction string  `json:"direction"`
	Current   float64 `json:"current"`
	Previous  float64 `json:"previous"`
	BaseYear  int     `json:"base_year,omitempty"`
	PrevYear  int     `json:"prev_year,omitempty"`
	// IsNew = true เมื่อช่วงก่อนหน้าไม่มียอดเลยแต่ช่วงนี้มี ซึ่งคิดเป็นเปอร์เซ็นต์ไม่ได้
	IsNew bool `json:"is_new,omitempty"`
}

// Secondary คือบรรทัดรองใต้ตัวเลขหลักของการ์ด KPI
type Secondary struct {
	Kind  string  `json:"kind"`
	Value float64 `json:"value,omitempty"`
	Text  string  `json:"text,omitempty"`
}

type Series struct {
	Key    string     `json:"key"`
	Name   string     `json:"name"`
	Kind   string     `json:"kind,omitempty"` // line | bar | dashed
	Color  string     `json:"color,omitempty"`
	Points []*float64 `json:"points"`
}

type Chart struct {
	Labels []string `json:"labels"`
	Series []Series `json:"series"`
	Tag    string   `json:"tag,omitempty"`
	// Reference คือเส้นอ้างอิงแนวนอน เช่นเส้นค่าเฉลี่ยในกราฟยอดขายรายวัน
	// คำนวณที่นี่เพราะเป็นส่วนหนึ่งของความหมายของกราฟ ไม่ใช่การตกแต่ง
	Reference *float64 `json:"reference,omitempty"`
}

// Slice คือหนึ่งส่วนของกราฟวงแหวน พร้อมสัดส่วนที่คำนวณมาแล้ว
type Slice struct {
	Label string  `json:"label"`
	Value float64 `json:"value"`
	Pct   float64 `json:"pct"`
	YoY   *Delta  `json:"yoy,omitempty"`
}

type RankEntry struct {
	Rank  int     `json:"rank"`
	Code  string  `json:"code"`
	Name  string  `json:"name"`
	Value float64 `json:"value"`
	// Share คือสัดส่วนเทียบอันดับหนึ่ง ใช้กำหนดความยาวแถบในตาราง
	Share float64 `json:"share"`
}

// MapNode คือหนึ่งหมุดบนแผนที่ประเทศไทย ทั้งของศูนย์กระจายสินค้าและของภาค
//
// จงใจไม่มีพิกัด x/y เพราะพิกัดเป็นคุณสมบัติของภาพ SVG ไม่ใช่ของข้อมูล
// frontend เป็นเจ้าของทั้งภาพและตารางพิกัด แล้วจับคู่กับหมุดเหล่านี้ด้วย Code
type MapNode struct {
	Code      string   `json:"code"`
	Name      string   `json:"name"`
	Value     float64  `json:"value"`
	Pct       float64  `json:"pct"`
	Provinces []string `json:"provinces,omitempty"`
	Status    string   `json:"status,omitempty"`
	YoY       *Delta   `json:"yoy,omitempty"`
}

// ScopeInfo บอก frontend ว่าตัวเลขที่ได้มาครอบคลุมขอบเขตไหน
// รวมถึงเดือน/ปีที่ระบบเลือกให้เองเมื่อผู้ใช้เลือก "ทุกเดือน"
type ScopeInfo struct {
	DatasetID string `json:"dataset_id"`
	DC        string `json:"dc"`
	DCName    string `json:"dc_name"`
	Year      *int   `json:"year"`
	Month     *int   `json:"month"`

	EffectiveYear  int  `json:"effective_year"`
	EffectiveMonth int  `json:"effective_month"`
	MonthInferred  bool `json:"month_inferred"`
	YearInferred   bool `json:"year_inferred"`
	HasData        bool `json:"has_data"`
}

func ptr(v float64) *float64 { return &v }

// Points แปลง slice ของ float64 เป็น slice ของ pointer โดยไม่มีค่าว่าง
func Points(vals []float64) []*float64 {
	out := make([]*float64, len(vals))
	for i, v := range vals {
		out[i] = ptr(v)
	}
	return out
}

// PointsWithGaps แปลงเป็น pointer โดยเปลี่ยนเดือนที่ไม่มีข้อมูลเป็น null
// ต่างจากศูนย์ตรงที่กราฟจะเว้นช่วงไว้ ไม่ใช่ลากเส้นลงไปแตะแกน
func PointsWithGaps(vals []float64, present []bool) []*float64 {
	out := make([]*float64, len(vals))
	for i, v := range vals {
		if i < len(present) && present[i] {
			out[i] = ptr(v)
		}
	}
	return out
}

// GrowthDelta คำนวณการเติบโตเทียบช่วงก่อนหน้า
//
// คืน nil เมื่อไม่มีฐานให้เทียบและช่วงนี้ก็ไม่มียอด ซึ่ง dashboard เดิมแสดงเป็น N/A
// ส่วนกรณีที่ช่วงก่อนไม่มียอดแต่ช่วงนี้มี ถือเป็น "ใหม่" ไม่ใช่การเติบโตอนันต์
func GrowthDelta(current, previous float64, baseYear, prevYear int) *Delta {
	switch {
	case previous > 0:
		pct := (current - previous) / previous * 100
		dir := DirectionUp
		if pct < 0 {
			dir = DirectionDown
		} else if pct == 0 {
			dir = DirectionFlat
		}
		return &Delta{
			Pct: pct, Direction: dir, Current: current, Previous: previous,
			BaseYear: baseYear, PrevYear: prevYear,
		}
	case current > 0:
		return &Delta{
			Direction: DirectionUp, Current: current, Previous: 0,
			BaseYear: baseYear, PrevYear: prevYear, IsNew: true,
		}
	default:
		return nil
	}
}

// MakeSlices เรียงจากมากไปน้อยและคำนวณสัดส่วนให้เรียบร้อย ตรงกับพฤติกรรมกราฟวงแหวนเดิม
func MakeSlices(values map[string]float64) []Slice {
	total := 0.0
	for _, v := range values {
		total += v
	}
	out := make([]Slice, 0, len(values))
	for k, v := range values {
		pct := 0.0
		if total > 0 {
			pct = v / total * 100
		}
		out = append(out, Slice{Label: k, Value: v, Pct: pct})
	}
	SortSlicesDesc(out)
	return out
}
