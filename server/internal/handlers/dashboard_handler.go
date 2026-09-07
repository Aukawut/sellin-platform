package handlers

import (
	"strings"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"

	"sellin-server/internal/core/domain"
	"sellin-server/internal/core/services"
)

type DashboardHandler struct {
	analytics *services.AnalyticsService
}

func NewDashboardHandler(a *services.AnalyticsService) *DashboardHandler {
	return &DashboardHandler{analytics: a}
}

func (h *DashboardHandler) Register(r fiber.Router) {
	r.Get("/datasets/:id/filters", h.filters)
	r.Get("/dashboard/sellin", h.sellIn)
	r.Get("/dashboard/sellout", h.sellOut)
	r.Get("/dashboard/stock", h.stock)
	r.Get("/dashboard/planning", h.planning)
}

// parseScope อ่านฟิลเตอร์ระดับ global จาก query string
// ยอมรับทั้งค่าว่างและคำว่า ALL สำหรับ "ทุก..." ตรงกับที่ dropdown ส่งมา
func parseScope(c *fiber.Ctx) (domain.Scope, error) {
	raw := strings.TrimSpace(c.Query("dataset_id"))
	if raw == "" {
		raw = c.Params("id")
	}
	id, err := uuid.Parse(raw)
	if err != nil {
		return domain.Scope{}, fiber.NewError(fiber.StatusBadRequest, "ต้องระบุ dataset_id ที่ถูกต้อง")
	}

	scope, err := domain.ParseScope(id, c.Query("dc"), c.Query("year"), c.Query("month"))
	if err != nil {
		return domain.Scope{}, fiber.NewError(fiber.StatusBadRequest, err.Error())
	}
	return scope, nil
}

// optionalFilter แปลง ALL และค่าว่างให้เป็นสตริงว่าง ซึ่งหมายถึงไม่กรอง
func optionalFilter(c *fiber.Ctx, name string) string {
	v := strings.TrimSpace(c.Query(name))
	if strings.EqualFold(v, domain.AllToken) {
		return ""
	}
	return v
}

func (h *DashboardHandler) filters(c *fiber.Ctx) error {
	scope, err := parseScope(c)
	if err != nil {
		return err
	}
	out, err := h.analytics.Filters(c.UserContext(), scope)
	if err != nil {
		return err
	}
	return c.JSON(out)
}

func (h *DashboardHandler) sellIn(c *fiber.Ctx) error {
	scope, err := parseScope(c)
	if err != nil {
		return err
	}
	page, err := h.analytics.SellIn(c.UserContext(), scope)
	if err != nil {
		return err
	}
	return c.JSON(page)
}

func (h *DashboardHandler) sellOut(c *fiber.Ctx) error {
	scope, err := parseScope(c)
	if err != nil {
		return err
	}
	page, err := h.analytics.SellOut(c.UserContext(), services.SellOutQuery{
		Scope: scope,
		Group: optionalFilter(c, "group"),
	})
	if err != nil {
		return err
	}
	return c.JSON(page)
}

func (h *DashboardHandler) stock(c *fiber.Ctx) error {
	scope, err := parseScope(c)
	if err != nil {
		return err
	}
	page, err := h.analytics.Stock(c.UserContext(), services.StockQuery{
		Scope:  scope,
		Group:  optionalFilter(c, "group"),
		Status: optionalFilter(c, "status"),
	})
	if err != nil {
		return err
	}
	return c.JSON(page)
}

func (h *DashboardHandler) planning(c *fiber.Ctx) error {
	scope, err := parseScope(c)
	if err != nil {
		return err
	}
	page, err := h.analytics.Planning(c.UserContext(), scope)
	if err != nil {
		return err
	}
	return c.JSON(page)
}
