package ports

import (
	"context"
	"io"

	"github.com/google/uuid"

	"sellin-server/internal/core/domain"
)

// ชื่อคอลัมน์ที่อนุญาตให้ใช้จัดกลุ่ม ประกาศเป็นค่าคงที่เพื่อให้ adapter ตรวจได้
// ว่าค่าที่ส่งมาอยู่ในรายการนี้จริง ก่อนนำไปประกอบเป็น SQL
const (
	GroupByProductGroup    = "group_name"
	GroupByProductCategory = "category"
)

// Bucket คือผลรวมหนึ่งกลุ่ม ใช้ร่วมกันทุกที่ที่ query เป็นแบบ "รวมยอดแยกตามอะไรสักอย่าง"
// Secondary มีค่าเมื่อ query นั้นดึงสองตัวเลขพร้อมกัน เช่นเป้าหมายคู่กับยอดจริง
type Bucket struct {
	Key       string
	Label     string
	Value     float64
	Secondary float64
}

// DatasetMeta คือทุกอย่างที่ต้องรู้เพื่อสร้างตัวเลือกใน dropdown และหาปีฐาน
// ดึงมาครั้งเดียวต่อ request แล้วใช้ซ้ำทั้ง 4 หน้า
type DatasetMeta struct {
	Customers []domain.Customer
	Products  map[string]domain.Product

	SellInYears  []int
	SellOutYears []int
	StockYears   []int
	TargetYears  []int

	SellOutGroups []string
	StockGroups   []string
}

type MasterRepo interface {
	Meta(ctx context.Context, datasetID uuid.UUID) (DatasetMeta, error)
}

// ---------- Sell-In ----------

type SellInSummary struct {
	Revenue float64
	Cartons float64
	Orders  int
}

type SellInRepo interface {
	Summary(ctx context.Context, s domain.Scope) (SellInSummary, error)
	// RevenueByMonth คืนทั้งยอดและธงว่าเดือนนั้นมีแถวข้อมูลจริงหรือไม่
	// เพื่อให้กราฟเว้นช่วงที่ยังไม่มีข้อมูล แทนที่จะลากเส้นลงไปแตะศูนย์
	RevenueByMonth(ctx context.Context, datasetID uuid.UUID, dc string, year int) ([12]float64, [12]bool, error)
	RevenueByDay(ctx context.Context, datasetID uuid.UUID, dc string, year, month int) (map[int]float64, error)
	RevenueByDC(ctx context.Context, datasetID uuid.UUID, year, month *int) ([]Bucket, error)
	CartonsBy(ctx context.Context, s domain.Scope, groupBy string) ([]Bucket, error)
	// CartonsTotal ใช้โดยหน้า Stock สำหรับ KPI ยอดรับเข้าของเดือนสแนปช็อต
	CartonsTotal(ctx context.Context, datasetID uuid.UUID, dc string, year, month int) (float64, error)
	MonthsWithData(ctx context.Context, datasetID uuid.UUID, dc string, year int) ([]int, error)
	TrackingByGroup(ctx context.Context, s domain.Scope) ([]Bucket, error)
}

// ---------- Sell-Out ----------

type SellOutSummary struct {
	Revenue float64
	Cartons float64
}

type ProductAgg struct {
	Code        string
	Description string
	GroupName   string
	Category    string
	Cartons     float64
	Revenue     float64
}

type SellOutRepo interface {
	Summary(ctx context.Context, s domain.Scope) (SellOutSummary, error)
	RevenueByMonth(ctx context.Context, datasetID uuid.UUID, dc string, year int) ([12]float64, [12]bool, error)
	RevenueByDC(ctx context.Context, datasetID uuid.UUID, year, month *int) ([]Bucket, error)
	RevenueBy(ctx context.Context, s domain.Scope, groupBy string) ([]Bucket, error)
	CartonsBy(ctx context.Context, s domain.Scope, groupBy string) ([]Bucket, error)
	// CartonsByForYear ใช้เทียบปีต่อปีในป้ายกำกับกราฟวงแหวน จึงรับปีแยกจาก scope
	CartonsByForYear(ctx context.Context, datasetID uuid.UUID, dc string, month *int, year int, groupBy string) ([]Bucket, error)
	Products(ctx context.Context, s domain.Scope, group string) ([]ProductAgg, error)
	RevenueByGroupMonthly(ctx context.Context, datasetID uuid.UUID, dc string, year int) (map[string][12]float64, [12]bool, error)
	RevenueByRegion(ctx context.Context, s domain.Scope) ([]Bucket, error)
	RevenueByRegionForYear(ctx context.Context, datasetID uuid.UUID, dc string, month *int, year int) ([]Bucket, error)
	MonthsWithData(ctx context.Context, datasetID uuid.UUID, dc string, year int) ([]int, error)
	YearTotal(ctx context.Context, datasetID uuid.UUID, dc string, year int) (float64, int, error)
}

// ---------- Target ----------

type TargetRepo interface {
	Total(ctx context.Context, s domain.Scope) (float64, error)
	ByMonth(ctx context.Context, datasetID uuid.UUID, dc string, year int) ([12]float64, error)
	ByDC(ctx context.Context, datasetID uuid.UUID, year, month *int) ([]Bucket, error)
	ByCategory(ctx context.Context, s domain.Scope) ([]Bucket, error)
	YearTotal(ctx context.Context, datasetID uuid.UUID, dc string, year int) (float64, error)
}

// ---------- Stock ----------

type StockTotals struct {
	Beginning float64
	SellIn    float64
	SellOut   float64
	Ending    float64
}

type StockMonthPoint struct {
	Month int
	StockTotals
}

type StockProductAgg struct {
	Code        string
	Description string
	GroupName   string
	StockTotals
}

type StockRepo interface {
	MonthsWithData(ctx context.Context, datasetID uuid.UUID, dc string, year int) ([]int, error)
	Snapshot(ctx context.Context, datasetID uuid.UUID, dc string, year, month int) (StockTotals, error)
	Monthly(ctx context.Context, datasetID uuid.UUID, dc string, year int) ([]StockMonthPoint, error)
	Products(ctx context.Context, datasetID uuid.UUID, dc string, year, month int, group string) ([]StockProductAgg, error)
	// RecentSellOut คืนยอดขายออกของเดือนล่าสุดไม่เกิน n เดือน โดยนับเฉพาะเดือนที่มีแถวข้อมูลจริง
	RecentSellOut(ctx context.Context, datasetID uuid.UUID, dc string, year, month, n int) ([]float64, error)
	EndingBy(ctx context.Context, datasetID uuid.UUID, dc string, year, month int, groupBy string) ([]Bucket, error)
}

// ExportRepo ดึงแถวดิบตามขอบเขตที่เลือก สำหรับส่งออกเป็นไฟล์ Excel
// ใช้ชนิดข้อมูลเดียวกับตอนนำเข้า ไฟล์ที่ส่งออกจึงนำกลับเข้าระบบได้โดยไม่ต้องแปลงรูปแบบ
type ExportRepo interface {
	SellIn(ctx context.Context, s domain.Scope) ([]domain.SellInRow, error)
	SellOut(ctx context.Context, s domain.Scope) ([]domain.SellOutRow, error)
	Stock(ctx context.Context, s domain.Scope) ([]domain.StockRow, error)
	Targets(ctx context.Context, s domain.Scope) ([]domain.TargetRow, error)
	Tracking(ctx context.Context, s domain.Scope) ([]domain.TrackingRow, error)
	Customers(ctx context.Context, s domain.Scope) ([]domain.Customer, error)
	Products(ctx context.Context, s domain.Scope) ([]domain.Product, error)
}

// SheetWriter สร้างไฟล์ Excel โดยที่ service ไม่ต้องรู้จักไลบรารีที่ใช้เขียนจริง
type SheetWriter interface {
	AddSheet(name string, header []string, rows [][]any)
	SetColumnWidths(sheet string, widths map[string]float64)
	WriteTo(dst io.Writer, firstSheet string) error
}
