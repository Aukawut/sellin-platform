package services

import (
	"context"
	"fmt"

	"sellin-server/internal/core/domain"
	"sellin-server/internal/core/ports"
)

// AnalyticsService ประกอบคำตอบของ dashboard ทั้ง 4 หน้า
//
// หน้าที่ของมันคือสั่ง repository ให้รวมยอด แล้วส่งผลให้ domain ตัดสินความหมาย
// ไม่มีการวนลูปข้อมูลดิบที่นี่ — การรวมยอดเป็นงานของ SQL
type AnalyticsService struct {
	master  ports.MasterRepo
	sellIn  ports.SellInRepo
	sellOut ports.SellOutRepo
	target  ports.TargetRepo
	stock   ports.StockRepo
	sets    ports.DatasetRepo
}

func NewAnalyticsService(
	master ports.MasterRepo, sellIn ports.SellInRepo, sellOut ports.SellOutRepo,
	target ports.TargetRepo, stock ports.StockRepo, sets ports.DatasetRepo,
) *AnalyticsService {
	return &AnalyticsService{
		master: master, sellIn: sellIn, sellOut: sellOut,
		target: target, stock: stock, sets: sets,
	}
}

// FilterOptions คือตัวเลือกทั้งหมดที่มีจริงในชุดข้อมูลนี้
// สร้างจากข้อมูลที่นำเข้ามาจริง ไม่ใช่รายการตายตัวในโค้ด
type FilterOptions struct {
	Customers []FilterCustomer `json:"customers"`
	Years     []int            `json:"years"`
	Months    []int            `json:"months"`
	Groups    struct {
		SellOut []string `json:"sellout"`
		Stock   []string `json:"stock"`
	} `json:"product_groups"`
	StockStatuses []string `json:"stock_statuses"`
}

type FilterCustomer struct {
	Code      string   `json:"code"`
	Name      string   `json:"name"`
	Status    string   `json:"status"`
	Region    string   `json:"region,omitempty"`
	Provinces []string `json:"provinces,omitempty"`
}

func (s *AnalyticsService) Filters(ctx context.Context, datasetID domain.Scope) (FilterOptions, error) {
	meta, err := s.requireReady(ctx, datasetID)
	if err != nil {
		return FilterOptions{}, err
	}

	var out FilterOptions
	for _, c := range meta.Customers {
		// dropdown แสดงเฉพาะศูนย์ที่ยัง Active ตรงกับ dashboard เดิม
		if !c.IsActive() {
			continue
		}
		out.Customers = append(out.Customers, FilterCustomer{
			Code: c.Code, Name: c.Name, Status: c.Status,
			Region: c.Region, Provinces: c.Provinces,
		})
	}

	years := map[int]bool{}
	for _, list := range [][]int{meta.SellInYears, meta.SellOutYears, meta.StockYears, meta.TargetYears} {
		for _, y := range list {
			years[y] = true
		}
	}
	out.Years = domain.SortedInts(years)
	out.Months = []int{1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12}
	out.Groups.SellOut = meta.SellOutGroups
	out.Groups.Stock = meta.StockGroups
	out.StockStatuses = []string{
		domain.StockLabelNormal, domain.StockLabelOutOf,
		domain.StockLabelOverfull, domain.StockLabelLow, domain.StockLabelNoSales,
	}
	return out, nil
}

// requireReady กันไม่ให้หน้า dashboard อ่านชุดข้อมูลที่ยังนำเข้าไม่เสร็จหรือถูกลบไปแล้ว
func (s *AnalyticsService) requireReady(ctx context.Context, scope domain.Scope) (ports.DatasetMeta, error) {
	ds, err := s.sets.ByID(ctx, scope.DatasetID)
	if err != nil {
		return ports.DatasetMeta{}, err
	}
	if ds.IsDeleted() {
		return ports.DatasetMeta{}, domain.ErrDatasetNotFound
	}
	if ds.Status != domain.StatusReady {
		return ports.DatasetMeta{}, domain.ErrDatasetNotReady
	}
	return s.master.Meta(ctx, scope.DatasetID)
}

// scopeInfo อธิบายขอบเขตที่ตัวเลขชุดนี้ครอบคลุม รวมถึงปีและเดือนที่ระบบเลือกให้เอง
func scopeInfo(scope domain.Scope, meta ports.DatasetMeta, effYear, effMonth int, yearInferred, monthInferred, hasData bool) domain.ScopeInfo {
	info := domain.ScopeInfo{
		DatasetID:      scope.DatasetID.String(),
		DC:             scope.DC,
		Year:           scope.Year,
		Month:          scope.Month,
		EffectiveYear:  effYear,
		EffectiveMonth: effMonth,
		YearInferred:   yearInferred,
		MonthInferred:  monthInferred,
		HasData:        hasData,
	}
	if scope.DC != "" {
		for _, c := range meta.Customers {
			if c.Code == scope.DC {
				info.DCName = c.Name
				break
			}
		}
	}
	return info
}

func customerName(meta ports.DatasetMeta, code string) string {
	for _, c := range meta.Customers {
		if c.Code == code {
			return c.Name
		}
	}
	return code
}

// bucketsToMap ทำให้ผลลัพธ์จาก repository ใช้ค้นหาด้วยคีย์ได้สะดวก
func bucketsToMap(list []ports.Bucket) map[string]float64 {
	out := make(map[string]float64, len(list))
	for _, b := range list {
		out[b.Key] += b.Value
	}
	return out
}

func bucketsToRanking(list []ports.Bucket) []domain.RankEntry {
	entries := make([]domain.RankEntry, 0, len(list))
	for _, b := range list {
		entries = append(entries, domain.RankEntry{Code: b.Key, Name: b.Label, Value: b.Value})
	}
	return domain.RankByValue(entries)
}

// monthLabels คืนป้ายเดือนเป็นเลข 1–12 ให้ frontend แปลงเป็นชื่อเดือนไทยเอง
func monthLabels() []string {
	out := make([]string, 12)
	for i := range out {
		out[i] = fmt.Sprintf("%d", i+1)
	}
	return out
}
