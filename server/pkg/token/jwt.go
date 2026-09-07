package token

import (
	"errors"
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"

	"sellin-server/internal/core/domain"
	"sellin-server/internal/core/ports"
)

var ErrInvalidToken = errors.New("token ไม่ถูกต้องหรือหมดอายุแล้ว")

// JWTIssuer ออก access token อายุสั้น ส่วน refresh token เป็นค่าสุ่มที่เก็บ hash ไว้ในฐานข้อมูล
// จึงไม่ใช้ JWT กับ refresh เพราะต้องเพิกถอนได้ทันทีเมื่อพบการใช้ซ้ำ
type JWTIssuer struct {
	secret []byte
	issuer string
}

func NewJWTIssuer(secret, issuer string) *JWTIssuer {
	return &JWTIssuer{secret: []byte(secret), issuer: issuer}
}

func (j *JWTIssuer) Issue(u domain.User, ttl time.Duration) (string, time.Time, error) {
	now := time.Now()
	exp := now.Add(ttl)

	claims := jwt.MapClaims{
		"sub":   u.ID.String(),
		"email": u.Email,
		"role":  string(u.Role),
		"iss":   j.issuer,
		"iat":   now.Unix(),
		"exp":   exp.Unix(),
		"nbf":   now.Add(-30 * time.Second).Unix(), // เผื่อนาฬิกาเครื่องต่างกันเล็กน้อย
	}
	signed, err := jwt.NewWithClaims(jwt.SigningMethodHS256, claims).SignedString(j.secret)
	if err != nil {
		return "", time.Time{}, err
	}
	return signed, exp, nil
}

func (j *JWTIssuer) Parse(raw string) (ports.Claims, error) {
	tok, err := jwt.Parse(raw, func(t *jwt.Token) (any, error) {
		// ปฏิเสธ token ที่อ้างว่าใช้อัลกอริทึมอื่น กันการปลอมด้วย alg=none
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("อัลกอริทึมลงลายเซ็นไม่ถูกต้อง: %v", t.Header["alg"])
		}
		return j.secret, nil
	}, jwt.WithIssuer(j.issuer), jwt.WithValidMethods([]string{"HS256"}))
	if err != nil || !tok.Valid {
		return ports.Claims{}, ErrInvalidToken
	}

	claims, ok := tok.Claims.(jwt.MapClaims)
	if !ok {
		return ports.Claims{}, ErrInvalidToken
	}
	sub, _ := claims["sub"].(string)
	id, err := uuid.Parse(sub)
	if err != nil {
		return ports.Claims{}, ErrInvalidToken
	}
	email, _ := claims["email"].(string)
	role, _ := claims["role"].(string)

	return ports.Claims{UserID: id, Email: email, Role: domain.Role(role)}, nil
}

// SystemClock คือนาฬิกาจริง ส่วน test ใช้ตัวปลอมที่กำหนดเวลาเองได้
type SystemClock struct{}

func (SystemClock) Now() time.Time { return time.Now() }
