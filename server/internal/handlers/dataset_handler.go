package handlers

import (
	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"

	"sellin-server/internal/appctx"
	"sellin-server/internal/core/ports"
	"sellin-server/internal/core/services"
)

type DatasetHandler struct {
	imports *services.ImportService
	files   ports.FileStore
	repo    ports.DatasetRepo
}

func NewDatasetHandler(imports *services.ImportService, files ports.FileStore, repo ports.DatasetRepo) *DatasetHandler {
	return &DatasetHandler{imports: imports, files: files, repo: repo}
}

func (h *DatasetHandler) Register(r fiber.Router) {
	r.Get("/datasets", h.list)
	r.Post("/datasets", h.upload)
	r.Get("/datasets/:id", h.detail)
	r.Get("/datasets/:id/issues", h.issues)
	r.Get("/datasets/:id/file", h.download)
	r.Delete("/datasets/:id", h.remove)
	r.Post("/datasets/:id/restore", h.restore)
}

func (h *DatasetHandler) list(c *fiber.Ctx) error {
	f := ports.DatasetFilter{
		Deleted: c.QueryBool("deleted", false),
		Limit:   c.QueryInt("limit", 50),
		Offset:  c.QueryInt("offset", 0),
	}
	// ค่าเริ่มต้นคือเห็นไฟล์ของทุกคนในองค์กร ส่งพารามิเตอร์นี้เมื่ออยากดูเฉพาะของตัวเอง
	if c.QueryBool("mine", false) {
		id := appctx.User(c).ID
		f.OwnerID = &id
	}

	items, err := h.imports.List(c.UserContext(), appctx.User(c), f)
	if err != nil {
		return err
	}
	return c.JSON(fiber.Map{"datasets": items})
}

func (h *DatasetHandler) upload(c *fiber.Ctx) error {
	fh, err := c.FormFile("file")
	if err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "กรุณาแนบไฟล์ Excel ในฟิลด์ชื่อ file")
	}
	src, err := fh.Open()
	if err != nil {
		return err
	}
	defer src.Close()

	res, err := h.imports.Upload(c.UserContext(), appctx.User(c),
		fh.Filename, c.FormValue("label"), src, appctx.ClientIP(c))
	if err != nil {
		return err
	}

	// ตอบกลับทันทีแล้วอ่านไฟล์เบื้องหลัง หน้าเว็บติดตามสถานะผ่าน GET /datasets/:id
	h.imports.ProcessAsync(res.Dataset.ID)
	return c.Status(fiber.StatusAccepted).JSON(res)
}

func (h *DatasetHandler) detail(c *fiber.Ctx) error {
	id, err := parseID(c)
	if err != nil {
		return err
	}
	summary, err := h.imports.Summary(c.UserContext(), id)
	if err != nil {
		return err
	}
	ds, err := h.repo.ByID(c.UserContext(), id)
	if err != nil {
		return err
	}
	ds.CanDelete = ds.CanBeDeletedBy(appctx.User(c))
	return c.JSON(fiber.Map{"dataset": ds, "import": summary})
}

func (h *DatasetHandler) issues(c *fiber.Ctx) error {
	id, err := parseID(c)
	if err != nil {
		return err
	}
	items, total, err := h.repo.Issues(c.UserContext(), id,
		c.QueryInt("limit", 100), c.QueryInt("offset", 0))
	if err != nil {
		return err
	}
	return c.JSON(fiber.Map{"issues": items, "total": total})
}

func (h *DatasetHandler) download(c *fiber.Ctx) error {
	id, err := parseID(c)
	if err != nil {
		return err
	}
	ds, err := h.repo.ByID(c.UserContext(), id)
	if err != nil {
		return err
	}
	key, err := h.repo.FileKey(c.UserContext(), id)
	if err != nil {
		return err
	}
	f, err := h.files.Open(c.UserContext(), key)
	if err != nil {
		return err
	}

	c.Set(fiber.HeaderContentType, "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet")
	c.Set(fiber.HeaderContentDisposition,
		`attachment; filename*=UTF-8''`+urlEncode(ds.OriginalFilename))
	return c.SendStream(f)
}

func (h *DatasetHandler) remove(c *fiber.Ctx) error {
	id, err := parseID(c)
	if err != nil {
		return err
	}
	if err := h.imports.Delete(c.UserContext(), appctx.User(c), id, appctx.ClientIP(c)); err != nil {
		return err
	}
	return c.JSON(fiber.Map{"ok": true, "restorable": true})
}

func (h *DatasetHandler) restore(c *fiber.Ctx) error {
	id, err := parseID(c)
	if err != nil {
		return err
	}
	if err := h.imports.Restore(c.UserContext(), appctx.User(c), id, appctx.ClientIP(c)); err != nil {
		return err
	}
	return c.JSON(fiber.Map{"ok": true})
}

func parseID(c *fiber.Ctx) (uuid.UUID, error) {
	id, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return uuid.Nil, fiber.NewError(fiber.StatusBadRequest, "รหัสชุดข้อมูลไม่ถูกต้อง")
	}
	return id, nil
}

// urlEncode เข้ารหัสชื่อไฟล์สำหรับ header Content-Disposition
// จำเป็นเพราะชื่อไฟล์เป็นภาษาไทยและมีช่องว่าง
func urlEncode(s string) string {
	const hex = "0123456789ABCDEF"
	out := make([]byte, 0, len(s)*3)
	for i := 0; i < len(s); i++ {
		ch := s[i]
		if (ch >= 'a' && ch <= 'z') || (ch >= 'A' && ch <= 'Z') ||
			(ch >= '0' && ch <= '9') || ch == '-' || ch == '_' || ch == '.' || ch == '~' {
			out = append(out, ch)
			continue
		}
		out = append(out, '%', hex[ch>>4], hex[ch&0x0F])
	}
	return string(out)
}
