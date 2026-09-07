package excel

import (
	"encoding/json"
	"fmt"
	"math"
	"os"
	"path/filepath"
	"sort"
	"testing"

	"sellin-server/internal/core/domain"
)

// golden คือยอดรวมที่ dashboard เดิมแสดงอยู่จริง สกัดมาจาก const DATA ด้วย scripts/extract_golden.py
// ทุกตัวเลขในไฟล์นี้คือสิ่งที่ระบบใหม่ต้องคำนวณได้ตรงกัน มิฉะนั้นถือว่าพอร์ตพลาด
type golden struct {
	Counts struct {
		Customers, Products, SellIn, SellOut, Stock, Target, Tracking int
	} `json:"counts"`
	Customers struct {
		Active        int      `json:"active"`
		Codes         []string `json:"codes"`
		ProvinceCount int      `json:"province_count"`
	} `json:"customers"`
	SellIn struct {
		TotalRevenue      float64            `json:"total_revenue"`
		TotalCartons      float64            `json:"total_cartons"`
		UniqueDocs        int                `json:"unique_docs"`
		ByYearMonth       map[string]float64 `json:"by_year_month"`
		ByDC              map[string]float64 `json:"by_dc"`
		ByGroupCartons    map[string]float64 `json:"by_group_cartons"`
		ByCategoryCartons map[string]float64 `json:"by_category_cartons"`
	} `json:"sellin"`
	SellOut struct {
		TotalRevenue   float64            `json:"total_revenue"`
		TotalCartons   float64            `json:"total_cartons"`
		ByYearMonth    map[string]float64 `json:"by_year_month"`
		ByDC           map[string]float64 `json:"by_dc"`
		ByGroupCartons map[string]float64 `json:"by_group_cartons"`
	} `json:"sellout"`
	Stock struct {
		TotalBeginning    float64            `json:"total_beginning"`
		TotalSellIn       float64            `json:"total_sell_in"`
		TotalSellOut      float64            `json:"total_sell_out"`
		TotalEnding       float64            `json:"total_ending"`
		EndingByYearMonth map[string]float64 `json:"ending_by_year_month"`
	} `json:"stock"`
	Target struct {
		Total       float64            `json:"total"`
		ByYearMonth map[string]float64 `json:"by_year_month"`
	} `json:"target"`
	Tracking struct {
		TotalTargetQty float64 `json:"total_target_qty"`
		TotalCartons   float64 `json:"total_cartons"`
	} `json:"tracking"`
}

// tolerance เผื่อความต่างจากลำดับการบวกเลขทศนิยม ซึ่งต่างกันได้ระหว่าง Python กับ Go
const tolerance = 0.01

// missingFixtures คือข้อความที่บอกวิธีเตรียมไฟล์ทดสอบ
//
// ไฟล์ทั้งสองเป็นข้อมูลยอดขายจริงของบริษัท จึงไม่ถูกเก็บไว้ใน repo
// เมื่อไม่มีไฟล์ test จะข้ามพร้อมบอกวิธีเตรียม แทนที่จะล้มเหลวจนดูเหมือนโค้ดพัง
const missingFixtures = `ไม่พบไฟล์ทดสอบ — ข้าม golden test

ไฟล์เหล่านี้เป็นข้อมูลยอดขายจริงจึงไม่อยู่ใน repo เตรียมได้ด้วย (จากโฟลเดอร์ sellin-platform):
  cp "<โฟลเดอร์ที่เก็บไฟล์>/Data_Template_Sale_Performance_T-T_Final.xlsx" server/testdata/template.xlsx
  python3 scripts/extract_golden.py "<โฟลเดอร์ที่เก็บไฟล์>/dashboard_sellin_Final.html" server/testdata/golden_dashboard.json`

func loadFixtures(t *testing.T) (*domain.Workbook, golden) {
	t.Helper()

	raw, err := os.ReadFile(filepath.Join("..", "..", "..", "testdata", "golden_dashboard.json"))
	if err != nil {
		t.Skip(missingFixtures)
	}
	var g golden
	if err := json.Unmarshal(raw, &g); err != nil {
		t.Fatalf("แปลง golden fixture ไม่ได้: %v", err)
	}

	f, err := os.Open(filepath.Join("..", "..", "..", "testdata", "template.xlsx"))
	if err != nil {
		t.Skip(missingFixtures)
	}
	defer f.Close()

	wb, err := NewReader().Read(f)
	if err != nil {
		t.Fatalf("อ่านไฟล์ Excel ไม่สำเร็จ: %v", err)
	}
	return wb, g
}

func TestReadTemplate_RowCounts(t *testing.T) {
	wb, g := loadFixtures(t)

	cases := []struct {
		name string
		got  int
		want int
	}{
		{"Data_Customer", len(wb.Customers), g.Counts.Customers},
		{"Data_Product", len(wb.Products), g.Counts.Products},
		{"Data_Sell-In", len(wb.SellIn), g.Counts.SellIn},
		{"Data_Sell-Out", len(wb.SellOut), g.Counts.SellOut},
		{"Data_Stock_inventory", len(wb.Stock), g.Counts.Stock},
		{"Target_Sale", len(wb.Targets), g.Counts.Target},
		{"Tracking_Non-Sun Flower Seed", len(wb.Tracking), g.Counts.Tracking},
	}
	for _, c := range cases {
		if c.got != c.want {
			t.Errorf("%s: อ่านได้ %d แถว ต้องการ %d แถว", c.name, c.got, c.want)
		}
	}
}

func TestReadTemplate_SellInTotals(t *testing.T) {
	wb, g := loadFixtures(t)

	var revenue, cartons float64
	docs := map[string]bool{}
	byYM := map[string]float64{}
	byDC := map[string]float64{}
	byGroup := map[string]float64{}
	byCat := map[string]float64{}

	for _, r := range wb.SellIn {
		revenue += r.Revenue
		cartons += r.Cartons
		docs[r.DocNo] = true
		byYM[fmt.Sprintf("%d-%02d", r.Year, r.Month)] += r.Revenue
		byDC[r.CustomerCode] += r.Revenue
		byGroup[r.GroupName] += r.Cartons
		byCat[r.Category] += r.Cartons
	}

	closeEnough(t, "Sell-In revenue รวม", revenue, g.SellIn.TotalRevenue)
	closeEnough(t, "Sell-In cartons รวม", cartons, g.SellIn.TotalCartons)
	if len(docs) != g.SellIn.UniqueDocs {
		t.Errorf("จำนวนเลขที่เอกสารไม่ซ้ำ: ได้ %d ต้องการ %d", len(docs), g.SellIn.UniqueDocs)
	}
	compareMaps(t, "Sell-In revenue รายเดือน", byYM, g.SellIn.ByYearMonth)
	compareMaps(t, "Sell-In revenue รายศูนย์", byDC, g.SellIn.ByDC)
	compareMaps(t, "Sell-In cartons ราย Product Group", byGroup, g.SellIn.ByGroupCartons)
	compareMaps(t, "Sell-In cartons ราย Product Category", byCat, g.SellIn.ByCategoryCartons)
}

func TestReadTemplate_SellOutTotals(t *testing.T) {
	wb, g := loadFixtures(t)

	var revenue, cartons float64
	byYM := map[string]float64{}
	byDC := map[string]float64{}
	byGroup := map[string]float64{}

	for _, r := range wb.SellOut {
		revenue += r.Revenue
		cartons += r.Cartons
		byYM[fmt.Sprintf("%d-%02d", r.Year, r.Month)] += r.Revenue
		byDC[r.CustomerCode] += r.Revenue
		byGroup[r.GroupName] += r.Cartons
	}

	closeEnough(t, "Sell-Out revenue รวม", revenue, g.SellOut.TotalRevenue)
	closeEnough(t, "Sell-Out cartons รวม", cartons, g.SellOut.TotalCartons)
	compareMaps(t, "Sell-Out revenue รายเดือน", byYM, g.SellOut.ByYearMonth)
	compareMaps(t, "Sell-Out revenue รายศูนย์", byDC, g.SellOut.ByDC)
	compareMaps(t, "Sell-Out cartons ราย Product Group", byGroup, g.SellOut.ByGroupCartons)
}

// TestReadTemplate_StockEndingColumn คือ test ที่เฝ้าจุดที่ dashboard เดิมพลาดโดยตรง
//
// โค้ดเดิมอ่านคอลัมน์ชื่อ "Stock on Hand" แต่หัวคอลัมน์จริงคือ "Ending Stock"
// ทำให้สต๊อกคงเหลือเป็นศูนย์ทุกแถวหลัง import ถ้ามีใครแก้ alias ให้ผิดอีก test นี้จะจับได้ทันที
func TestReadTemplate_StockEndingColumn(t *testing.T) {
	wb, g := loadFixtures(t)

	var beg, in, out, end float64
	endByYM := map[string]float64{}
	for _, r := range wb.Stock {
		beg += r.Beginning
		in += r.SellIn
		out += r.SellOut
		end += r.Ending
		endByYM[fmt.Sprintf("%d-%02d", r.Year, r.Month)] += r.Ending
	}

	if end == 0 {
		t.Fatal("สต๊อกคงเหลือรวมเป็น 0 — คอลัมน์ Ending Stock ถูกอ่านผิด (นี่คือ bug เดิมของ dashboard)")
	}
	closeEnough(t, "Beginning Stock รวม", beg, g.Stock.TotalBeginning)
	closeEnough(t, "Sell-In ในตารางสต๊อกรวม", in, g.Stock.TotalSellIn)
	closeEnough(t, "Sell-Out ในตารางสต๊อกรวม", out, g.Stock.TotalSellOut)
	closeEnough(t, "Ending Stock รวม", end, g.Stock.TotalEnding)
	compareMaps(t, "Ending Stock รายเดือน", endByYM, g.Stock.EndingByYearMonth)
}

func TestReadTemplate_TargetAndTracking(t *testing.T) {
	wb, g := loadFixtures(t)

	var target float64
	byYM := map[string]float64{}
	for _, r := range wb.Targets {
		target += r.TargetAmount
		byYM[fmt.Sprintf("%d-%02d", r.Year, r.Month)] += r.TargetAmount
	}
	closeEnough(t, "Target รวม", target, g.Target.Total)
	compareMaps(t, "Target รายเดือน", byYM, g.Target.ByYearMonth)

	var qty, cartons float64
	for _, r := range wb.Tracking {
		qty += r.TargetQty
		cartons += r.Cartons
	}
	closeEnough(t, "Tracking target qty รวม", qty, g.Tracking.TotalTargetQty)
	closeEnough(t, "Tracking cartons รวม", cartons, g.Tracking.TotalCartons)
}

func TestReadTemplate_Customers(t *testing.T) {
	wb, g := loadFixtures(t)

	active := 0
	provinces := map[string]bool{}
	codes := make([]string, 0, len(wb.Customers))
	for _, c := range wb.Customers {
		codes = append(codes, c.Code)
		if c.IsActive() {
			active++
			for _, p := range c.Provinces {
				provinces[p] = true
			}
		}
	}
	sort.Strings(codes)

	if active != g.Customers.Active {
		t.Errorf("ศูนย์ Active: ได้ %d ต้องการ %d", active, g.Customers.Active)
	}
	if len(provinces) != g.Customers.ProvinceCount {
		t.Errorf("จำนวนจังหวัดที่ครอบคลุม: ได้ %d ต้องการ %d", len(provinces), g.Customers.ProvinceCount)
	}
	if len(codes) != len(g.Customers.Codes) {
		t.Fatalf("จำนวนรหัสศูนย์: ได้ %d ต้องการ %d", len(codes), len(g.Customers.Codes))
	}
	for i := range codes {
		if codes[i] != g.Customers.Codes[i] {
			t.Errorf("รหัสศูนย์ตำแหน่งที่ %d: ได้ %q ต้องการ %q", i, codes[i], g.Customers.Codes[i])
		}
	}

	// ภาคไม่ได้อยู่ในไฟล์ Excel จึงต้องถูกเติมจากตารางค่าตั้งต้นตอนนำเข้า
	for _, c := range wb.Customers {
		if c.Region == "" {
			t.Errorf("ศูนย์ %s (%s) ไม่มีภาคกำกับ", c.Code, c.Name)
		}
	}
}

// TestReadTemplate_NoBlockingIssues ยืนยันว่าไฟล์ Template ที่ถูกต้องผ่านการตรวจโดยไม่มีปัญหาระดับ blocking
func TestReadTemplate_NoBlockingIssues(t *testing.T) {
	wb, _ := loadFixtures(t)

	for _, issue := range wb.Issues {
		if issue.Severity == domain.SeverityBlocking {
			t.Errorf("ไฟล์ Template มาตรฐานไม่ควรมีปัญหาระดับ blocking แต่พบ: [%s] %s", issue.Sheet, issue.Message)
		}
	}
	for _, issue := range wb.Issues {
		t.Logf("issue [%s] %s แถว %d: %s", issue.Severity, issue.Sheet, issue.RowNo, issue.Message)
	}
}

func closeEnough(t *testing.T, label string, got, want float64) {
	t.Helper()
	if math.Abs(got-want) > tolerance {
		t.Errorf("%s: ได้ %.4f ต้องการ %.4f (ต่างกัน %.4f)", label, got, want, got-want)
	}
}

func compareMaps(t *testing.T, label string, got, want map[string]float64) {
	t.Helper()
	if len(got) != len(want) {
		t.Errorf("%s: จำนวนกลุ่มไม่ตรง ได้ %d ต้องการ %d", label, len(got), len(want))
	}
	keys := make([]string, 0, len(want))
	for k := range want {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	for _, k := range keys {
		if math.Abs(got[k]-want[k]) > tolerance {
			t.Errorf("%s [%s]: ได้ %.4f ต้องการ %.4f", label, k, got[k], want[k])
		}
	}
	for k := range got {
		if _, ok := want[k]; !ok {
			t.Errorf("%s: พบกลุ่มเกินมา %q (%.4f)", label, k, got[k])
		}
	}
}
