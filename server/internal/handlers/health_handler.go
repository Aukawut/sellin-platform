package handlers

import (
	"context"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/jackc/pgx/v5/pgxpool"
)

type HealthHandler struct {
	pool    *pgxpool.Pool
	version string
	started time.Time
}

func NewHealthHandler(pool *pgxpool.Pool, version string) *HealthHandler {
	return &HealthHandler{pool: pool, version: version, started: time.Now()}
}

func (h *HealthHandler) Register(app *fiber.App) {
	// liveness: ตอบเสมอถ้าโปรเซสยังอยู่ ใช้ให้ Docker รู้ว่าต้อง restart หรือยัง
	app.Get("/healthz", func(c *fiber.Ctx) error {
		return c.JSON(fiber.Map{
			"status":  "ok",
			"version": h.version,
			"uptime":  time.Since(h.started).Round(time.Second).String(),
		})
	})

	// readiness: ตอบ ok ต่อเมื่อฐานข้อมูลใช้งานได้จริง ใช้ตอนตัดสินใจส่ง traffic เข้ามา
	app.Get("/readyz", func(c *fiber.Ctx) error {
		ctx, cancel := context.WithTimeout(c.UserContext(), 3*time.Second)
		defer cancel()
		if err := h.pool.Ping(ctx); err != nil {
			return c.Status(fiber.StatusServiceUnavailable).
				JSON(fiber.Map{"status": "degraded", "database": "unreachable"})
		}
		return c.JSON(fiber.Map{"status": "ok", "database": "ok"})
	})
}
