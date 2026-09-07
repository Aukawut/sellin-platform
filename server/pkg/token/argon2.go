package token

import (
	"crypto/rand"
	"crypto/subtle"
	"encoding/base64"
	"errors"
	"fmt"
	"strings"

	"golang.org/x/crypto/argon2"
)

// Argon2Hasher เข้ารหัสรหัสผ่านด้วย argon2id ตามพารามิเตอร์ที่ OWASP แนะนำ
// รูปแบบที่เก็บเป็น PHC string มาตรฐาน จึงเปลี่ยนพารามิเตอร์ในอนาคตได้โดยของเก่ายังตรวจได้อยู่
type Argon2Hasher struct {
	time    uint32
	memory  uint32
	threads uint8
	keyLen  uint32
	saltLen uint32
}

func NewArgon2Hasher() *Argon2Hasher {
	return &Argon2Hasher{time: 3, memory: 64 * 1024, threads: 2, keyLen: 32, saltLen: 16}
}

func (h *Argon2Hasher) Hash(plain string) (string, error) {
	salt := make([]byte, h.saltLen)
	if _, err := rand.Read(salt); err != nil {
		return "", err
	}
	key := argon2.IDKey([]byte(plain), salt, h.time, h.memory, h.threads, h.keyLen)
	return fmt.Sprintf("$argon2id$v=%d$m=%d,t=%d,p=%d$%s$%s",
		argon2.Version, h.memory, h.time, h.threads,
		base64.RawStdEncoding.EncodeToString(salt),
		base64.RawStdEncoding.EncodeToString(key),
	), nil
}

var errBadHashFormat = errors.New("รูปแบบรหัสผ่านที่เก็บไว้ไม่ถูกต้อง")

func (h *Argon2Hasher) Verify(plain, encoded string) (bool, error) {
	parts := strings.Split(encoded, "$")
	if len(parts) != 6 || parts[1] != "argon2id" {
		return false, errBadHashFormat
	}
	var version int
	if _, err := fmt.Sscanf(parts[2], "v=%d", &version); err != nil {
		return false, errBadHashFormat
	}
	var memory, timeCost uint32
	var threads uint8
	if _, err := fmt.Sscanf(parts[3], "m=%d,t=%d,p=%d", &memory, &timeCost, &threads); err != nil {
		return false, errBadHashFormat
	}
	salt, err := base64.RawStdEncoding.DecodeString(parts[4])
	if err != nil {
		return false, errBadHashFormat
	}
	want, err := base64.RawStdEncoding.DecodeString(parts[5])
	if err != nil {
		return false, errBadHashFormat
	}

	got := argon2.IDKey([]byte(plain), salt, timeCost, memory, threads, uint32(len(want)))
	// เทียบแบบเวลาคงที่ ไม่ให้เวลาที่ใช้ตรวจบอกใบ้ว่ารหัสถูกกี่ตัว
	return subtle.ConstantTimeCompare(got, want) == 1, nil
}
