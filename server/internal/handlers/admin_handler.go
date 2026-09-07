package handlers

import (
	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"

	"sellin-server/internal/appctx"
	"sellin-server/internal/core/domain"
	"sellin-server/internal/core/services"
	"sellin-server/internal/repositories/postgres"
)

// AdminHandler ดูแลเส้นทางที่เฉพาะ admin เข้าได้ — จัดการผู้ใช้และภาคของศูนย์กระจายสินค้า
type AdminHandler struct {
	auth    *services.AuthService
	regions *postgres.RegionRepo
}

func NewAdminHandler(auth *services.AuthService, regions *postgres.RegionRepo) *AdminHandler {
	return &AdminHandler{auth: auth, regions: regions}
}

func (h *AdminHandler) Register(r fiber.Router) {
	r.Get("/users", h.listUsers)
	r.Post("/users/:id/active", h.setActive)
	r.Post("/users/:id/password", h.resetPassword)
	r.Get("/regions", h.listRegions)
	r.Post("/regions/:code", h.setRegion)
}

func (h *AdminHandler) listUsers(c *fiber.Ctx) error {
	users, err := h.auth.ListUsers(c.UserContext(), appctx.User(c))
	if err != nil {
		return err
	}
	return c.JSON(fiber.Map{"users": users})
}

func (h *AdminHandler) setActive(c *fiber.Ctx) error {
	id, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "รหัสผู้ใช้ไม่ถูกต้อง")
	}
	var req struct {
		Active bool `json:"active"`
	}
	if err := c.BodyParser(&req); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "รูปแบบข้อมูลที่ส่งมาไม่ถูกต้อง")
	}
	if err := h.auth.SetActive(c.UserContext(), appctx.User(c), id, req.Active); err != nil {
		return err
	}
	return c.JSON(fiber.Map{"ok": true})
}

func (h *AdminHandler) resetPassword(c *fiber.Ctx) error {
	id, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "รหัสผู้ใช้ไม่ถูกต้อง")
	}
	var req struct {
		Password string `json:"password"`
	}
	if err := c.BodyParser(&req); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "รูปแบบข้อมูลที่ส่งมาไม่ถูกต้อง")
	}
	if err := h.auth.ResetPassword(c.UserContext(), appctx.User(c), id, req.Password); err != nil {
		return err
	}
	// การตั้งรหัสใหม่เพิกถอนเซสชันเดิมทั้งหมด จึงบอกกลับไปให้หน้าเว็บแจ้งผู้ใช้ได้ถูก
	return c.JSON(fiber.Map{"ok": true, "sessions_revoked": true})
}

func (h *AdminHandler) listRegions(c *fiber.Ctx) error {
	rows, err := h.regions.Codes(c.UserContext())
	if err != nil {
		return err
	}
	return c.JSON(fiber.Map{"regions": rows, "options": domain.Regions})
}

func (h *AdminHandler) setRegion(c *fiber.Ctx) error {
	code := c.Params("code")
	if code == "" {
		return fiber.NewError(fiber.StatusBadRequest, "ต้องระบุรหัสศูนย์กระจายสินค้า")
	}
	var req struct {
		Region string `json:"region"`
	}
	if err := c.BodyParser(&req); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "รูปแบบข้อมูลที่ส่งมาไม่ถูกต้อง")
	}
	if !domain.IsValidRegion(req.Region) {
		return fiber.NewError(fiber.StatusBadRequest,
			"ภาคต้องเป็นค่าใดค่าหนึ่งใน: "+domain.RegionsText())
	}

	if err := h.regions.Set(c.UserContext(), code, req.Region, appctx.User(c).ID); err != nil {
		return err
	}
	return c.JSON(fiber.Map{"ok": true})
}
