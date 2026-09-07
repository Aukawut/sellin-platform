package handlers

import (
	"time"

	"github.com/gofiber/fiber/v2"

	"sellin-server/internal/appctx"
	"sellin-server/internal/core/domain"
	"sellin-server/internal/core/services"
)

const refreshCookieName = "sellin_refresh"

type AuthHandler struct {
	auth   *services.AuthService
	secure bool // ตั้ง Secure บน cookie เฉพาะตอนรันจริงที่มี HTTPS
	ttl    time.Duration
}

func NewAuthHandler(auth *services.AuthService, secure bool, refreshTTL time.Duration) *AuthHandler {
	return &AuthHandler{auth: auth, secure: secure, ttl: refreshTTL}
}

func (h *AuthHandler) Register(r fiber.Router) {
	r.Post("/auth/login", h.login)
	r.Post("/auth/refresh", h.refresh)
	r.Post("/auth/logout", h.logout)
}

type loginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

func (h *AuthHandler) login(c *fiber.Ctx) error {
	var req loginRequest
	if err := c.BodyParser(&req); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "รูปแบบข้อมูลที่ส่งมาไม่ถูกต้อง")
	}
	if req.Email == "" || req.Password == "" {
		return fiber.NewError(fiber.StatusBadRequest, "กรุณากรอกอีเมลและรหัสผ่าน")
	}

	sess, err := h.auth.Login(c.UserContext(), req.Email, req.Password,
		c.Get(fiber.HeaderUserAgent), appctx.ClientIP(c))
	if err != nil {
		return err
	}
	return h.respondWithSession(c, sess)
}

func (h *AuthHandler) refresh(c *fiber.Ctx) error {
	raw := c.Cookies(refreshCookieName)
	if raw == "" {
		return fiber.NewError(fiber.StatusUnauthorized, "ไม่พบเซสชัน กรุณาเข้าสู่ระบบใหม่")
	}

	sess, err := h.auth.Refresh(c.UserContext(), raw,
		c.Get(fiber.HeaderUserAgent), appctx.ClientIP(c))
	if err != nil {
		h.clearRefreshCookie(c)
		return err
	}
	return h.respondWithSession(c, sess)
}

func (h *AuthHandler) logout(c *fiber.Ctx) error {
	if raw := c.Cookies(refreshCookieName); raw != "" {
		_ = h.auth.Logout(c.UserContext(), raw, appctx.ClientIP(c))
	}
	h.clearRefreshCookie(c)
	return c.JSON(fiber.Map{"ok": true})
}

// Me คืนโปรไฟล์ของผู้ใช้ที่ล็อกอินอยู่ ใช้ตอนหน้าเว็บโหลดครั้งแรกเพื่อรู้ว่ายังล็อกอินอยู่หรือไม่
func (h *AuthHandler) Me(c *fiber.Ctx) error {
	return c.JSON(fiber.Map{"user": appctx.User(c)})
}

func (h *AuthHandler) respondWithSession(c *fiber.Ctx, sess services.Session) error {
	// refresh token ไม่เคยถูกส่งใน body — อยู่ใน HttpOnly cookie เท่านั้น
	// เพื่อไม่ให้ JavaScript ในหน้าเว็บอ่านได้แม้มีช่องโหว่ XSS
	c.Cookie(&fiber.Cookie{
		Name:     refreshCookieName,
		Value:    sess.RefreshToken,
		Path:     "/api/v1/auth",
		Expires:  sess.RefreshExpires,
		HTTPOnly: true,
		Secure:   h.secure,
		SameSite: fiber.CookieSameSiteStrictMode,
	})

	return c.JSON(fiber.Map{
		"user":         sess.User,
		"access_token": sess.AccessToken,
		"expires_at":   sess.AccessExpires,
	})
}

func (h *AuthHandler) clearRefreshCookie(c *fiber.Ctx) {
	c.Cookie(&fiber.Cookie{
		Name:     refreshCookieName,
		Value:    "",
		Path:     "/api/v1/auth",
		Expires:  time.Unix(0, 0),
		HTTPOnly: true,
		Secure:   h.secure,
		SameSite: fiber.CookieSameSiteStrictMode,
	})
}

// CreateUser เปิดให้เฉพาะ admin เพราะระบบนี้ไม่เปิดให้สมัครเอง
func (h *AuthHandler) CreateUser(c *fiber.Ctx) error {
	var req struct {
		Email       string `json:"email"`
		DisplayName string `json:"display_name"`
		Password    string `json:"password"`
		Role        string `json:"role"`
	}
	if err := c.BodyParser(&req); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "รูปแบบข้อมูลที่ส่งมาไม่ถูกต้อง")
	}

	created, err := h.auth.CreateUser(c.UserContext(), appctx.User(c), domain.User{
		Email:       req.Email,
		DisplayName: req.DisplayName,
		Role:        domain.Role(req.Role),
	}, req.Password)
	if err != nil {
		return err
	}
	return c.Status(fiber.StatusCreated).JSON(fiber.Map{"user": created})
}

// ChangeOwnPassword ให้ผู้ใช้ทุกระดับเปลี่ยนรหัสผ่านของตัวเอง โดยต้องยืนยันรหัสเดิม
func (h *AuthHandler) ChangeOwnPassword(c *fiber.Ctx) error {
	var req struct {
		CurrentPassword string `json:"current_password"`
		NewPassword     string `json:"new_password"`
	}
	if err := c.BodyParser(&req); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "รูปแบบข้อมูลที่ส่งมาไม่ถูกต้อง")
	}
	if err := h.auth.ChangeOwnPassword(c.UserContext(), appctx.User(c),
		req.CurrentPassword, req.NewPassword); err != nil {
		return err
	}
	return c.JSON(fiber.Map{"ok": true})
}
