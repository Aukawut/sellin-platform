package token

import (
	"testing"
	"time"

	"github.com/google/uuid"

	"sellin-server/internal/core/domain"
)

func TestArgon2RoundTrip(t *testing.T) {
	h := NewArgon2Hasher()
	encoded, err := h.Hash("รหัสผ่านที่ยาวพอ-12ตัว")
	if err != nil {
		t.Fatalf("hash ล้มเหลว: %v", err)
	}

	ok, err := h.Verify("รหัสผ่านที่ยาวพอ-12ตัว", encoded)
	if err != nil || !ok {
		t.Errorf("รหัสผ่านที่ถูกต้องควรผ่าน (ok=%v err=%v)", ok, err)
	}
	ok, err = h.Verify("รหัสผ่านผิด", encoded)
	if err != nil {
		t.Errorf("รหัสผ่านผิดไม่ควรทำให้เกิด error: %v", err)
	}
	if ok {
		t.Error("รหัสผ่านผิดไม่ควรผ่าน")
	}
}

func TestArgon2SaltsDiffer(t *testing.T) {
	h := NewArgon2Hasher()
	a, _ := h.Hash("รหัสผ่านเดียวกันทุกประการ")
	b, _ := h.Hash("รหัสผ่านเดียวกันทุกประการ")
	if a == b {
		t.Error("รหัสผ่านเดียวกันต้องได้ค่า hash ต่างกัน เพราะ salt ต้องสุ่มใหม่ทุกครั้ง")
	}
}

func TestJWTRoundTrip(t *testing.T) {
	j := NewJWTIssuer("ความลับสำหรับทดสอบที่ยาวเกินสามสิบสองตัวอักษรแน่นอน", "sellin-test")
	u := domain.User{ID: uuid.New(), Email: "somchai@example.com", Role: domain.RoleAdmin}

	raw, exp, err := j.Issue(u, 15*time.Minute)
	if err != nil {
		t.Fatalf("ออก token ล้มเหลว: %v", err)
	}
	if time.Until(exp) > 16*time.Minute {
		t.Errorf("อายุ token ยาวเกินที่กำหนด: %v", time.Until(exp))
	}

	claims, err := j.Parse(raw)
	if err != nil {
		t.Fatalf("ตรวจ token ล้มเหลว: %v", err)
	}
	if claims.UserID != u.ID || claims.Email != u.Email || claims.Role != domain.RoleAdmin {
		t.Errorf("ข้อมูลใน token ไม่ตรง: %+v", claims)
	}
}

func TestJWTRejectsWrongSecret(t *testing.T) {
	a := NewJWTIssuer("ความลับชุดที่หนึ่งซึ่งยาวเกินสามสิบสองตัวอักษรแน่นอน", "sellin-test")
	b := NewJWTIssuer("ความลับชุดที่สองซึ่งยาวเกินสามสิบสองตัวอักษรแน่นอนเช่นกัน", "sellin-test")

	raw, _, _ := a.Issue(domain.User{ID: uuid.New(), Role: domain.RoleViewer}, time.Minute)
	if _, err := b.Parse(raw); err == nil {
		t.Error("token ที่ลงลายเซ็นด้วยความลับอื่นต้องไม่ผ่าน")
	}
}

func TestJWTRejectsExpired(t *testing.T) {
	j := NewJWTIssuer("ความลับสำหรับทดสอบที่ยาวเกินสามสิบสองตัวอักษรแน่นอน", "sellin-test")
	raw, _, _ := j.Issue(domain.User{ID: uuid.New(), Role: domain.RoleViewer}, -time.Minute)
	if _, err := j.Parse(raw); err == nil {
		t.Error("token ที่หมดอายุแล้วต้องไม่ผ่าน")
	}
}
