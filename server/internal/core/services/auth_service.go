package services

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"

	"sellin-server/internal/core/domain"
	"sellin-server/internal/core/ports"
)

// Session คือสิ่งที่ handler ได้กลับไปหลังล็อกอินหรือหมุน token สำเร็จ
type Session struct {
	User          domain.User
	AccessToken   string
	AccessExpires time.Time
	// RefreshToken เป็นค่าดิบที่จะถูกส่งกลับเป็น HttpOnly cookie เท่านั้น
	// ฐานข้อมูลเก็บแค่ค่า hash จึงอ่านย้อนกลับไม่ได้แม้ฐานข้อมูลรั่ว
	RefreshToken   string
	RefreshExpires time.Time
}

type AuthService struct {
	users     ports.UserRepo
	tokens    ports.RefreshTokenRepo
	audit     ports.AuditRepo
	hasher    ports.Hasher
	issuer    ports.TokenIssuer
	clock     ports.Clock
	accessTL  time.Duration
	refreshTL time.Duration
}

func NewAuthService(
	users ports.UserRepo, tokens ports.RefreshTokenRepo, audit ports.AuditRepo,
	hasher ports.Hasher, issuer ports.TokenIssuer, clock ports.Clock,
	accessTTL, refreshTTL time.Duration,
) *AuthService {
	return &AuthService{
		users: users, tokens: tokens, audit: audit, hasher: hasher,
		issuer: issuer, clock: clock, accessTL: accessTTL, refreshTL: refreshTTL,
	}
}

func (s *AuthService) Login(ctx context.Context, email, password, userAgent, ip string) (Session, error) {
	u, err := s.users.ByEmail(ctx, email)
	if err != nil {
		// ตอบข้อความเดียวกันทั้งกรณีไม่มีอีเมลและรหัสผิด เพื่อไม่ให้เดาได้ว่าอีเมลไหนมีในระบบ
		return Session{}, domain.ErrInvalidCredentials
	}
	ok, err := s.hasher.Verify(password, u.PasswordHash)
	if err != nil || !ok {
		return Session{}, domain.ErrInvalidCredentials
	}
	if !u.IsActive {
		return Session{}, domain.ErrUserInactive
	}

	sess, _, err := s.issueSession(ctx, u, userAgent, ip)
	if err != nil {
		return Session{}, err
	}
	_ = s.audit.Record(ctx, &u.ID, "auth.login", u.Email, nil, ip)
	return sess, nil
}

// Refresh หมุน refresh token ทุกครั้งที่ใช้ ตัวเดิมถูกเพิกถอนทันที
//
// ถ้ามีคนนำ token ที่ถูกหมุนไปแล้วกลับมาใช้ แปลว่า token รั่วออกไป
// ระบบจะเพิกถอน session ทั้งหมดของผู้ใช้คนนั้น แล้วบังคับให้ล็อกอินใหม่
func (s *AuthService) Refresh(ctx context.Context, raw, userAgent, ip string) (Session, error) {
	hash := hashToken(raw)
	stored, err := s.tokens.ByHash(ctx, hash)
	if err != nil {
		return Session{}, domain.ErrInvalidCredentials
	}

	now := s.clock.Now()
	if !stored.IsUsable(now) {
		if stored.RevokedAt != nil {
			_ = s.tokens.RevokeChain(ctx, stored.UserID)
			_ = s.audit.Record(ctx, &stored.UserID, "auth.refresh_reuse_detected", "",
				map[string]any{"token_id": stored.ID.String()}, ip)
			return Session{}, fmt.Errorf("ตรวจพบการใช้ token ซ้ำ ระบบได้ออกจากระบบทุกอุปกรณ์เพื่อความปลอดภัย")
		}
		return Session{}, domain.ErrInvalidCredentials
	}

	u, err := s.users.ByID(ctx, stored.UserID)
	if err != nil {
		return Session{}, domain.ErrInvalidCredentials
	}
	if !u.IsActive {
		_ = s.tokens.RevokeChain(ctx, u.ID)
		return Session{}, domain.ErrUserInactive
	}

	sess, newTokenID, err := s.issueSession(ctx, u, userAgent, ip)
	if err != nil {
		return Session{}, err
	}
	// ชี้ token เดิมไปยังตัวใหม่ก่อนเพิกถอน เพื่อให้ไล่สายย้อนกลับได้เมื่อพบการใช้ซ้ำภายหลัง
	if err := s.tokens.Rotate(ctx, stored.ID, newTokenID); err != nil {
		return Session{}, err
	}
	return sess, nil
}

func (s *AuthService) Logout(ctx context.Context, raw, ip string) error {
	stored, err := s.tokens.ByHash(ctx, hashToken(raw))
	if err != nil {
		return nil // ออกจากระบบต้องสำเร็จเสมอในสายตาผู้ใช้ แม้ token จะไม่มีอยู่แล้ว
	}
	_ = s.audit.Record(ctx, &stored.UserID, "auth.logout", "", nil, ip)
	return s.tokens.Revoke(ctx, stored.ID)
}

func (s *AuthService) Me(ctx context.Context, id uuid.UUID) (domain.User, error) {
	return s.users.ByID(ctx, id)
}

// CreateUser ใช้โดย admin เท่านั้น ระบบนี้ไม่เปิดให้สมัครเอง
func (s *AuthService) CreateUser(ctx context.Context, actor domain.User, u domain.User, password string) (domain.User, error) {
	if !actor.IsAdmin() {
		return domain.User{}, domain.ErrForbidden
	}
	if err := domain.ValidatePassword(password); err != nil {
		return domain.User{}, err
	}
	if u.Role != domain.RoleAdmin && u.Role != domain.RoleViewer {
		return domain.User{}, errors.New("role ต้องเป็น admin หรือ viewer")
	}

	hash, err := s.hasher.Hash(password)
	if err != nil {
		return domain.User{}, err
	}
	u.IsActive = true
	created, err := s.users.Create(ctx, u, hash)
	if err != nil {
		return domain.User{}, err
	}
	_ = s.audit.Record(ctx, &actor.ID, "user.create", created.Email,
		map[string]any{"role": string(created.Role)}, "")
	return created, nil
}

// issueSession คืน id ของ refresh token ตัวใหม่ด้วย เพราะการหมุน token ต้องบันทึกว่าตัวเก่าถูกแทนด้วยตัวไหน
func (s *AuthService) issueSession(ctx context.Context, u domain.User, userAgent, ip string) (Session, uuid.UUID, error) {
	access, accessExp, err := s.issuer.Issue(u, s.accessTL)
	if err != nil {
		return Session{}, uuid.Nil, err
	}

	raw, err := randomToken()
	if err != nil {
		return Session{}, uuid.Nil, err
	}
	expires := s.clock.Now().Add(s.refreshTL)
	id, err := s.tokens.Store(ctx, domain.RefreshToken{
		UserID:    u.ID,
		TokenHash: hashToken(raw),
		ExpiresAt: expires,
		UserAgent: userAgent,
		IP:        ip,
	})
	if err != nil {
		return Session{}, uuid.Nil, err
	}

	return Session{
		User: u, AccessToken: access, AccessExpires: accessExp,
		RefreshToken: raw, RefreshExpires: expires,
	}, id, nil
}

// randomToken สร้าง refresh token แบบสุ่ม 32 ไบต์ ไม่ใช่ JWT
// เพราะ refresh token ต้องเพิกถอนได้ทันที ซึ่ง JWT ที่ตรวจด้วยลายเซ็นอย่างเดียวทำไม่ได้
func randomToken() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(b), nil
}

// hashToken ใช้ SHA-256 ไม่ใช่ argon2 เพราะค่าที่ hash เป็นค่าสุ่ม 256 บิตอยู่แล้ว
// จึงไม่มีอะไรให้เดา และการค้นหาต้องเร็วพอสำหรับทุก request ที่หมุน token
func hashToken(raw string) []byte {
	sum := sha256.Sum256([]byte(raw))
	return sum[:]
}

// ---------- การจัดการผู้ใช้ (เฉพาะ admin) ----------

func (s *AuthService) ListUsers(ctx context.Context, actor domain.User) ([]domain.User, error) {
	if !actor.IsAdmin() {
		return nil, domain.ErrForbidden
	}
	return s.users.List(ctx)
}

// SetActive เปิดหรือปิดการใช้งานบัญชี
//
// การปิดบัญชีเพิกถอน refresh token ทั้งหมดของคนนั้นด้วย
// ไม่งั้นเซสชันที่เปิดค้างไว้จะยังหมุน token ต่อได้อีกเป็นสัปดาห์
func (s *AuthService) SetActive(ctx context.Context, actor domain.User, target uuid.UUID, active bool) error {
	if !actor.IsAdmin() {
		return domain.ErrForbidden
	}
	// กันไม่ให้ admin ปิดบัญชีตัวเองจนล็อกตัวเองออกจากระบบ
	if actor.ID == target && !active {
		return errors.New("ปิดการใช้งานบัญชีของตัวเองไม่ได้")
	}

	if err := s.users.SetActive(ctx, target, active); err != nil {
		return err
	}
	if !active {
		if err := s.tokens.RevokeChain(ctx, target); err != nil {
			return err
		}
	}
	_ = s.audit.Record(ctx, &actor.ID, "user.set_active", target.String(),
		map[string]any{"active": active}, "")
	return nil
}

// ResetPassword ตั้งรหัสผ่านใหม่ให้ผู้ใช้ แล้วบังคับให้ทุกอุปกรณ์ล็อกอินใหม่
func (s *AuthService) ResetPassword(ctx context.Context, actor domain.User, target uuid.UUID, password string) error {
	if !actor.IsAdmin() {
		return domain.ErrForbidden
	}
	if err := domain.ValidatePassword(password); err != nil {
		return err
	}

	hash, err := s.hasher.Hash(password)
	if err != nil {
		return err
	}
	if err := s.users.UpdatePassword(ctx, target, hash); err != nil {
		return err
	}
	if err := s.tokens.RevokeChain(ctx, target); err != nil {
		return err
	}
	_ = s.audit.Record(ctx, &actor.ID, "user.reset_password", target.String(), nil, "")
	return nil
}

// ChangeOwnPassword ให้ผู้ใช้เปลี่ยนรหัสผ่านตัวเอง โดยต้องยืนยันรหัสเดิมก่อน
func (s *AuthService) ChangeOwnPassword(ctx context.Context, actor domain.User, current, next string) error {
	ok, err := s.hasher.Verify(current, actor.PasswordHash)
	if err != nil || !ok {
		return domain.ErrInvalidCredentials
	}
	if err := domain.ValidatePassword(next); err != nil {
		return err
	}

	hash, err := s.hasher.Hash(next)
	if err != nil {
		return err
	}
	if err := s.users.UpdatePassword(ctx, actor.ID, hash); err != nil {
		return err
	}
	_ = s.audit.Record(ctx, &actor.ID, "user.change_password", actor.Email, nil, "")
	return nil
}
