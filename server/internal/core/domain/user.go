package domain

import (
	"errors"
	"strings"
	"time"

	"github.com/google/uuid"
)

type Role string

const (
	RoleAdmin  Role = "admin"
	RoleViewer Role = "viewer"
)

var (
	ErrInvalidCredentials = errors.New("อีเมลหรือรหัสผ่านไม่ถูกต้อง")
	ErrUserInactive       = errors.New("บัญชีนี้ถูกปิดการใช้งาน")
	ErrPasswordTooShort   = errors.New("รหัสผ่านต้องยาวอย่างน้อย 12 ตัวอักษร")
	ErrForbidden          = errors.New("ไม่มีสิทธิ์ดำเนินการนี้")
)

const MinPasswordLength = 12

type User struct {
	ID           uuid.UUID `json:"id"`
	Email        string    `json:"email"`
	DisplayName  string    `json:"display_name"`
	Role         Role      `json:"role"`
	IsActive     bool      `json:"is_active"`
	PasswordHash string    `json:"-"`
	CreatedAt    time.Time `json:"created_at"`
}

func (u User) IsAdmin() bool { return u.Role == RoleAdmin }

// NormalizeEmail ทำให้อีเมลเทียบกันได้แบบไม่สนตัวพิมพ์ ตรงกับ unique index ที่ใช้ lower(email)
func NormalizeEmail(e string) string { return strings.ToLower(strings.TrimSpace(e)) }

func ValidatePassword(p string) error {
	if len([]rune(p)) < MinPasswordLength {
		return ErrPasswordTooShort
	}
	return nil
}

// RefreshToken เก็บเฉพาะค่า hash ของ token จริง ตัว token ที่ส่งให้ผู้ใช้ไม่เคยถูกบันทึกไว้
type RefreshToken struct {
	ID         uuid.UUID
	UserID     uuid.UUID
	TokenHash  []byte
	IssuedAt   time.Time
	ExpiresAt  time.Time
	RevokedAt  *time.Time
	ReplacedBy *uuid.UUID
	UserAgent  string
	IP         string
}

func (t RefreshToken) IsUsable(now time.Time) bool {
	return t.RevokedAt == nil && now.Before(t.ExpiresAt)
}
