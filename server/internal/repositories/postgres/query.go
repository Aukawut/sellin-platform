package postgres

import (
	"fmt"
	"strings"

	"github.com/google/uuid"

	"sellin-server/internal/core/domain"
	"sellin-server/internal/core/ports"
)

// saleOnly กรองเฉพาะรายการที่เป็นการขายจริง
//
// ปัจจุบันไฟล์ Template มีแต่ค่า Sale ทั้งหมด แต่คอลัมน์นี้มีอยู่จริงในไฟล์
// การกรองไว้ตั้งแต่ต้นทำให้เมื่อมีรายการคืนของหรือยกเลิกเข้ามาในอนาคต ยอดจะไม่บวกเกินโดยไม่มีใครรู้
//
// แถวที่ไม่มีค่าสถานะถือเป็นการขาย เพื่อให้ผลลัพธ์ตรงกับ dashboard เดิมซึ่งไม่กรองอะไรเลย
const saleOnly = `coalesce(nullif(%s, ''), 'Sale') = 'Sale'`

// cond ประกอบเงื่อนไข WHERE พร้อมหมายเลขพารามิเตอร์ให้อัตโนมัติ
// ทุกค่าที่มาจากผู้ใช้เดินทางเป็นพารามิเตอร์เสมอ ไม่เคยถูกต่อเป็นสตริง SQL
type cond struct {
	sql   []string
	args  []any
	alias string // ชื่อย่อของตารางหลัก เติมให้คอลัมน์ตอนที่ query มีการ join
}

func newCond(datasetID uuid.UUID) *cond { return newCondAlias(datasetID, "") }

// newCondAlias ใช้เมื่อ query ต้อง join ตารางอื่น ซึ่งชื่อคอลัมน์อาจซ้ำกันจนกำกวม
func newCondAlias(datasetID uuid.UUID, alias string) *cond {
	c := &cond{alias: alias}
	c.eq("dataset_id", datasetID)
	return c
}

// col เติม alias ให้ชื่อคอลัมน์ ชื่อทั้งหมดที่ผ่านทางนี้เป็นค่าคงที่ในโค้ด ไม่ได้มาจากภายนอก
func (c *cond) col(name string) string {
	if c.alias == "" {
		return name
	}
	return c.alias + "." + name
}

func (c *cond) eq(col string, v any) {
	c.args = append(c.args, v)
	c.sql = append(c.sql, fmt.Sprintf("%s = $%d", c.col(col), len(c.args)))
}

func (c *cond) raw(sql string) { c.sql = append(c.sql, sql) }

// scope เติมเงื่อนไขจากฟิลเตอร์ระดับ global โดยข้ามตัวที่ผู้ใช้เลือก "ทุก..."
func (c *cond) scope(s domain.Scope) *cond {
	if !s.AllDC() {
		c.eq("customer_code", s.DC)
	}
	if !s.AllYear() {
		c.eq("year", *s.Year)
	}
	if !s.AllMonth() {
		c.eq("month", *s.Month)
	}
	return c
}

// yearMonth เติมเฉพาะปีและเดือน ใช้กับ query ที่จงใจไม่สนใจฟิลเตอร์ศูนย์
// เช่นตารางจัดอันดับ ซึ่งต้องแสดงทุกศูนย์เสมอเพื่อให้เทียบกันได้
func (c *cond) yearMonth(year, month *int) *cond {
	if year != nil {
		c.eq("year", *year)
	}
	if month != nil {
		c.eq("month", *month)
	}
	return c
}

func (c *cond) dc(code string) *cond {
	if code != "" {
		c.eq("customer_code", code)
	}
	return c
}

func (c *cond) where() string { return strings.Join(c.sql, " AND ") }

func (c *cond) next(v any) string {
	c.args = append(c.args, v)
	return fmt.Sprintf("$%d", len(c.args))
}

// groupColumn ตรวจว่าชื่อคอลัมน์ที่ขอมาอยู่ในรายการที่อนุญาต ก่อนนำไปประกอบเป็น SQL
// เป็นด่านเดียวที่กันไม่ให้ชื่อคอลัมน์จากภายนอกหลุดเข้าไปในคำสั่ง
func groupColumn(name string) (string, error) {
	switch name {
	case ports.GroupByProductGroup, ports.GroupByProductCategory:
		return name, nil
	default:
		return "", fmt.Errorf("จัดกลุ่มตามคอลัมน์ %q ไม่ได้", name)
	}
}

// monthArray แปลงผลลัพธ์รายเดือนเป็น array 12 ช่อง พร้อมธงว่าเดือนไหนมีข้อมูลจริง
func monthArray(values map[int]float64, present map[int]bool) ([12]float64, [12]bool) {
	var out [12]float64
	var has [12]bool
	for m := 1; m <= 12; m++ {
		out[m-1] = values[m]
		has[m-1] = present[m]
	}
	return out, has
}
