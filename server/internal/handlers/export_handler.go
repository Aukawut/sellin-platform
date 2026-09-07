package handlers

import (
	"bytes"
	"time"

	"github.com/gofiber/fiber/v2"

	"sellin-server/internal/core/services"
)

type ExportHandler struct {
	svc *services.ExportService
}

func NewExportHandler(export *services.ExportService) *ExportHandler {
	return &ExportHandler{svc: export}
}

func (h *ExportHandler) Register(r fiber.Router) {
	r.Get("/export", h.download)
}

// download สร้างไฟล์ Excel ตามขอบเขตที่ผู้ใช้กำลังดูอยู่
//
// เขียนลงบัฟเฟอร์ให้เสร็จก่อนส่ง เพราะถ้าสตรีมออกไปแล้วเกิดข้อผิดพลาดกลางทาง
// ผู้ใช้จะได้ไฟล์ที่เปิดไม่ขึ้นโดยไม่มีข้อความบอกว่าเกิดอะไรขึ้น
func (h *ExportHandler) download(c *fiber.Ctx) error {
	scope, err := parseScope(c)
	if err != nil {
		return err
	}

	buf := new(bytes.Buffer)
	now := time.Now()
	if err := h.svc.Write(c.UserContext(), scope, buf, now); err != nil {
		return err
	}

	filename := h.svc.Filename(scope, now)
	c.Set(fiber.HeaderContentType,
		"application/vnd.openxmlformats-officedocument.spreadsheetml.sheet")
	c.Set(fiber.HeaderContentDisposition, `attachment; filename*=UTF-8''`+urlEncode(filename))
	return c.Send(buf.Bytes())
}
