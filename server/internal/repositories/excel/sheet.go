package excel

import (
	"strconv"
	"strings"
	"time"
)

// normalizeHeader ทำให้ชื่อหัวคอลัมน์เทียบกันได้: ตัดช่องว่างหัวท้าย ลดเป็นตัวพิมพ์เล็ก
// และยุบช่องว่างซ้อนให้เหลือช่องเดียว ตรงกับ buildAliasMap() ของ dashboard เดิม
//
// เพิ่มเติมจากต้นฉบับ: ตัด non-breaking space และ zero-width space ที่มักติดมากับไฟล์ที่ก๊อปมาจากเว็บ
func normalizeHeader(s string) string {
	s = strings.Map(func(r rune) rune {
		switch r {
		case '\u00A0', '\u200B', '\uFEFF': // nbsp, zero-width space, BOM
			return ' '
		case '\n', '\r', '\t':
			return ' '
		}
		return r
	}, s)
	return strings.Join(strings.Fields(strings.ToLower(s)), " ")
}

// sheet ห่อข้อมูลดิบของ worksheet หนึ่งแผ่น พร้อมแผนที่จากชื่อหัวคอลัมน์ไปยังตำแหน่งคอลัมน์
type sheet struct {
	name    string
	header  []string
	aliases map[string]int
	rows    [][]string
}

func newSheet(name string, raw [][]string) *sheet {
	s := &sheet{name: name, aliases: map[string]int{}}
	if len(raw) == 0 {
		return s
	}
	s.header = raw[0]
	for i, h := range s.header {
		key := normalizeHeader(h)
		if key == "" {
			continue
		}
		// หัวคอลัมน์ซ้ำ: ยึดคอลัมน์แรกที่เจอ เหมือนพฤติกรรมของ sheet_to_json เดิม
		if _, exists := s.aliases[key]; !exists {
			s.aliases[key] = i
		}
	}
	s.rows = raw[1:]
	return s
}

// has บอกว่ามีคอลัมน์ที่ตรงกับชื่อใดชื่อหนึ่งใน candidates หรือไม่
func (s *sheet) has(candidates ...string) bool {
	for _, c := range candidates {
		if _, ok := s.aliases[normalizeHeader(c)]; ok {
			return true
		}
	}
	return false
}

// row ให้ตัวช่วยอ่านค่าจากแถวเดียว โดยอ้างชื่อคอลัมน์แทนตำแหน่ง
type row struct {
	s     *sheet
	cells []string
	no    int // เลขแถวตามที่เห็นใน Excel (นับรวมหัวตาราง) เพื่อให้ผู้ใช้ตามไปดูได้
}

func (s *sheet) each(fn func(r row)) {
	for i, cells := range s.rows {
		fn(row{s: s, cells: cells, no: i + 2})
	}
}

// raw คืนค่าดิบของคอลัมน์แรกที่พบและไม่ว่าง ตรงกับ val() ของต้นฉบับที่ข้ามค่าว่างไปหา candidate ถัดไป
func (r row) raw(candidates ...string) string {
	for _, c := range candidates {
		idx, ok := r.s.aliases[normalizeHeader(c)]
		if !ok || idx >= len(r.cells) {
			continue
		}
		if v := strings.TrimSpace(r.cells[idx]); v != "" {
			return v
		}
	}
	return ""
}

func (r row) str(candidates ...string) string { return r.raw(candidates...) }

// num แปลงเป็นตัวเลขแบบผ่อนปรน: ตัดคอมมา สัญลักษณ์สกุลเงิน และช่องว่างออกก่อน
// ค่าที่แปลงไม่ได้คืน 0 เหมือน toNum() เดิม เพื่อไม่ให้แถวเดียวทำให้ทั้งไฟล์ล้ม
func (r row) num(candidates ...string) float64 {
	v := r.raw(candidates...)
	if v == "" {
		return 0
	}
	return parseNum(v)
}

func parseNum(v string) float64 {
	cleaned := strings.Map(func(c rune) rune {
		switch c {
		case ',', ' ', '\u00A0', '฿', '$':
			return -1
		}
		return c
	}, v)
	if cleaned == "" {
		return 0
	}
	// ตัวเลขติดลบในวงเล็บตามแบบบัญชี เช่น (1,200)
	if strings.HasPrefix(cleaned, "(") && strings.HasSuffix(cleaned, ")") {
		if n, err := strconv.ParseFloat(strings.Trim(cleaned, "()"), 64); err == nil {
			return -n
		}
	}
	n, err := strconv.ParseFloat(cleaned, 64)
	if err != nil {
		return 0
	}
	return n
}

func (r row) intVal(candidates ...string) int { return int(r.num(candidates...)) }

// isBlank บอกว่าแถวนี้ไม่มีค่าอะไรเลย ใช้ข้ามแถวว่างท้ายตารางที่ Excel มักทิ้งไว้
func (r row) isBlank() bool {
	for _, c := range r.cells {
		if strings.TrimSpace(c) != "" {
			return false
		}
	}
	return true
}

// date พยายามอ่านคอลัมน์วันที่หลายรูปแบบ ถ้าไม่สำเร็จให้ผู้เรียกไปประกอบจาก ปี/เดือน/วัน แทน
func (r row) date(candidates ...string) (time.Time, bool) {
	v := r.raw(candidates...)
	if v == "" {
		return time.Time{}, false
	}
	layouts := []string{
		"2006-01-02 15:04:05", "2006-01-02T15:04:05Z07:00", "2006-01-02",
		"02/01/2006", "1/2/2006", "01-02-06", "2/1/2006 15:04:05",
	}
	for _, l := range layouts {
		if t, err := time.Parse(l, v); err == nil {
			return t, true
		}
	}
	// serial number ของ Excel นับจาก 1899-12-30 (รวม leap-year bug ของปี 1900 ไว้แล้ว)
	if n := parseNum(v); n > 0 && n < 300000 {
		return time.Date(1899, 12, 30, 0, 0, 0, 0, time.UTC).AddDate(0, 0, int(n)), true
	}
	return time.Time{}, false
}
