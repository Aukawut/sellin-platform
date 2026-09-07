package domain

import "strings"

// Regions คือภาคทั้งหมดที่ระบบรู้จัก
//
// ตรงกับที่ dashboard เดิมใช้ทั้งบนแผนที่และในกราฟสัดส่วน
// การเพิ่มภาคใหม่ต้องเพิ่มสีบนหน้าเว็บด้วย จึงคุมรายการไว้ที่นี่ที่เดียว
var Regions = []string{"เหนือ", "อีสาน", "กลาง", "ตะวันออก", "ตะวันตก", "ใต้"}

func IsValidRegion(r string) bool {
	for _, x := range Regions {
		if x == r {
			return true
		}
	}
	return false
}

func RegionsText() string { return strings.Join(Regions, ", ") }
