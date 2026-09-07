package domain

import "sort"

// การเรียงทุกจุดในระบบผูกตัวตัดสินเสมอกันไว้ด้วย เพื่อให้ผลลัพธ์คงที่ทุกครั้งที่เรียก
// ไม่งั้นสองรายการที่ยอดเท่ากันจะสลับตำแหน่งกันไปมาระหว่าง request

func SortSlicesDesc(s []Slice) {
	sort.SliceStable(s, func(i, j int) bool {
		if s[i].Value != s[j].Value {
			return s[i].Value > s[j].Value
		}
		return s[i].Label < s[j].Label
	})
}

// RankByValue เรียงจากมากไปน้อย เติมเลขอันดับ และคำนวณสัดส่วนเทียบอันดับหนึ่ง
func RankByValue(entries []RankEntry) []RankEntry {
	sort.SliceStable(entries, func(i, j int) bool {
		if entries[i].Value != entries[j].Value {
			return entries[i].Value > entries[j].Value
		}
		return entries[i].Name < entries[j].Name
	})

	max := 0.0
	if len(entries) > 0 {
		max = entries[0].Value
	}
	for i := range entries {
		entries[i].Rank = i + 1
		if max > 0 {
			entries[i].Share = entries[i].Value / max
		}
	}
	return entries
}

func SortedKeys[V any](m map[string]V) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	sort.Strings(out)
	return out
}

func SortedInts(m map[int]bool) []int {
	out := make([]int, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	sort.Ints(out)
	return out
}
