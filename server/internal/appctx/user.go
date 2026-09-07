// Package appctx เก็บและอ่านผู้ใช้ที่ล็อกอินอยู่จาก context ของ request
package appctx

import (
	"github.com/gofiber/fiber/v2"

	"sellin-server/internal/core/domain"
)

const userKey = "auth.user"

func SetUser(c *fiber.Ctx, u domain.User) { c.Locals(userKey, u) }

// User คืนผู้ใช้ที่ middleware ตรวจแล้ว handler ที่อยู่หลัง RequireAuth เรียกได้เสมอ
func User(c *fiber.Ctx) domain.User {
	if u, ok := c.Locals(userKey).(domain.User); ok {
		return u
	}
	return domain.User{}
}

// ClientIP อ่าน IP จริงของผู้ใช้ โดยเชื่อ X-Forwarded-For เฉพาะเมื่อ Fiber ถูกตั้งค่า proxy ไว้แล้ว
func ClientIP(c *fiber.Ctx) string { return c.IP() }
