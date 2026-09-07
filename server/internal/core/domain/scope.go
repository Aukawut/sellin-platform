package domain

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/google/uuid"
)

// AllToken คือค่าที่ dropdown ส่งมาเมื่อผู้ใช้เลือก "ทุก..." ตรงกับพฤติกรรมของ dashboard เดิม
const AllToken = "ALL"

// Scope คือฟิลเตอร์ระดับ global ที่มีผลข้ามทั้ง 4 หน้าพร้อมกัน
// ค่าว่างหรือ nil แปลว่า "ทุก" ไม่ใช่ "ไม่มี"
type Scope struct {
	DatasetID uuid.UUID
	DC        string // "" = ทุกศูนย์กระจายสินค้า
	Year      *int   // nil = ทุกปี
	Month     *int   // nil = ทุกเดือน
}

func (s Scope) AllDC() bool    { return s.DC == "" }
func (s Scope) AllYear() bool  { return s.Year == nil }
func (s Scope) AllMonth() bool { return s.Month == nil }

// ParseScope แปลงค่าจาก query string โดยยอมรับทั้ง "ALL", ค่าว่าง และตัวเลข
func ParseScope(datasetID uuid.UUID, dc, year, month string) (Scope, error) {
	s := Scope{DatasetID: datasetID}

	if v := strings.TrimSpace(dc); v != "" && !strings.EqualFold(v, AllToken) {
		s.DC = v
	}

	y, err := parseOptionalInt(year, "year")
	if err != nil {
		return s, err
	}
	if y != nil && (*y < 1900 || *y > 3000) {
		return s, fmt.Errorf("ปี %d อยู่นอกช่วงที่รองรับ", *y)
	}
	s.Year = y

	m, err := parseOptionalInt(month, "month")
	if err != nil {
		return s, err
	}
	if m != nil && (*m < 1 || *m > 12) {
		return s, fmt.Errorf("เดือนต้องอยู่ระหว่าง 1 ถึง 12 (ได้ %d)", *m)
	}
	s.Month = m

	return s, nil
}

func parseOptionalInt(raw, field string) (*int, error) {
	v := strings.TrimSpace(raw)
	if v == "" || strings.EqualFold(v, AllToken) {
		return nil, nil
	}
	n, err := strconv.Atoi(v)
	if err != nil {
		return nil, fmt.Errorf("%s ต้องเป็นตัวเลขหรือ ALL (ได้ %q)", field, raw)
	}
	return &n, nil
}

// Resolved บอกว่าปีและเดือนที่ระบบ "ใช้จริง" คืออะไร หลังเติมค่าแทนตัวเลือก ทุกปี/ทุกเดือน
//
// จำเป็นเพราะหน้า Stock และกราฟรายวันไม่ได้รวมทุกเดือนเมื่อผู้ใช้เลือก "ทุกเดือน"
// แต่เลือกเดือนล่าสุดที่มีข้อมูลจริงแทน frontend ต้องได้ค่านี้กลับไปเพื่อขึ้นป้ายให้ถูก
type Resolved struct {
	Year          int  `json:"year"`
	Month         int  `json:"month"`
	MonthInferred bool `json:"month_inferred"` // true = ระบบเลือกเดือนให้ ไม่ใช่ผู้ใช้เลือก
	YearInferred  bool `json:"year_inferred"`
	HasData       bool `json:"has_data"`
}

// ResolveYear คืนปีฐานที่ใช้คำนวณ: ปีที่ผู้ใช้เลือก หรือปีล่าสุดที่มีข้อมูลถ้าเลือกทุกปี
//
// ต้นฉบับ hardcode 2025/2026 ไว้ในโค้ด ทำให้ไฟล์ปีถัดไปได้กราฟว่าง ที่นี่คำนวณจากข้อมูลจริงเสมอ
func ResolveYear(s Scope, availableYears []int) (year int, inferred bool, ok bool) {
	if s.Year != nil {
		return *s.Year, false, true
	}
	if len(availableYears) == 0 {
		return 0, true, false
	}
	max := availableYears[0]
	for _, y := range availableYears[1:] {
		if y > max {
			max = y
		}
	}
	return max, true, true
}

// ResolveMonth เลือกเดือนสแนปช็อต: เดือนที่ผู้ใช้เลือกถ้ามีข้อมูลจริง
// มิฉะนั้นใช้เดือนล่าสุดที่มีข้อมูลของปี/ศูนย์นั้น
//
// ห้ามแทนด้วยเดือนปัจจุบันตามนาฬิกาเซิร์ฟเวอร์ เพราะข้อมูลมักตามหลังปฏิทินหลายเดือน
func ResolveMonth(s Scope, monthsWithData []int) (month int, inferred bool, ok bool) {
	if len(monthsWithData) == 0 {
		return 0, true, false
	}
	if s.Month != nil {
		for _, m := range monthsWithData {
			if m == *s.Month {
				return m, false, true
			}
		}
	}
	max := monthsWithData[0]
	for _, m := range monthsWithData[1:] {
		if m > max {
			max = m
		}
	}
	return max, true, true
}
