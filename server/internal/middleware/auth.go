package middleware

import (
	"strings"

	"github.com/gofiber/fiber/v2"

	"sellin-server/internal/appctx"
	"sellin-server/internal/core/domain"
	"sellin-server/internal/core/ports"
)

// RequireAuth ตรวจ access token จาก header Authorization แล้วโหลดผู้ใช้จริงจากฐานข้อมูล
//
// โหลดผู้ใช้ทุก request โดยตั้งใจ เพื่อให้การปิดบัญชีมีผลทันที
// ไม่ต้องรอ token เดิมหมดอายุ 15 นาที
func RequireAuth(issuer ports.TokenIssuer, users ports.UserRepo) fiber.Handler {
	return func(c *fiber.Ctx) error {
		raw := strings.TrimSpace(c.Get(fiber.HeaderAuthorization))
		if !strings.HasPrefix(strings.ToLower(raw), "bearer ") {
			return fiber.NewError(fiber.StatusUnauthorized, "กรุณาเข้าสู่ระบบก่อนใช้งาน")
		}

		claims, err := issuer.Parse(strings.TrimSpace(raw[7:]))
		if err != nil {
			return fiber.NewError(fiber.StatusUnauthorized, "เซสชันหมดอายุ กรุณาเข้าสู่ระบบใหม่")
		}

		u, err := users.ByID(c.UserContext(), claims.UserID)
		if err != nil {
			return fiber.NewError(fiber.StatusUnauthorized, "ไม่พบบัญชีผู้ใช้นี้")
		}
		if !u.IsActive {
			return fiber.NewError(fiber.StatusForbidden, "บัญชีนี้ถูกปิดการใช้งาน")
		}

		appctx.SetUser(c, u)
		return c.Next()
	}
}

// RequireAdmin ใช้ต่อจาก RequireAuth สำหรับเส้นทางที่เฉพาะ admin เท่านั้นที่เข้าได้
func RequireAdmin() fiber.Handler {
	return func(c *fiber.Ctx) error {
		if !appctx.User(c).IsAdmin() {
			return fiber.NewError(fiber.StatusForbidden, domain.ErrForbidden.Error())
		}
		return c.Next()
	}
}
