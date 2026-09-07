// Package ports ประกาศสัญญาระหว่างแกนกลางของระบบกับโลกภายนอก
//
// ทุก interface ในไฟล์นี้ถูก "เรียกใช้" โดย services และถูก "ทำให้เป็นจริง" โดย adapters
// ใน internal/repositories — แกนกลางจึงไม่รู้จัก Postgres, Fiber หรือ Excel เลย
package ports

import (
	"context"
	"io"
	"time"

	"github.com/google/uuid"

	"sellin-server/internal/core/domain"
)

// ---------- โครงสร้างพื้นฐาน ----------

// FileStore เก็บไฟล์ Excel ต้นฉบับ ปัจจุบันเขียนลง volume ในเครื่อง
// ถ้าย้ายขึ้น object storage ภายหลัง เปลี่ยนเฉพาะ adapter ที่ทำ interface นี้
type FileStore interface {
	Put(ctx context.Context, key string, r io.Reader) (size int64, err error)
	Open(ctx context.Context, key string) (io.ReadSeekCloser, error)
	Delete(ctx context.Context, key string) error
}

// WorkbookReader แปลงไฟล์ Excel เป็นข้อมูลของ domain
type WorkbookReader interface {
	Read(src io.Reader) (*domain.Workbook, error)
}

// Hasher จัดการรหัสผ่าน แยกเป็น interface เพื่อให้ test ใช้ตัวที่เร็วกว่า argon2 ได้
type Hasher interface {
	Hash(plain string) (string, error)
	Verify(plain, encoded string) (bool, error)
}

// TokenIssuer ออกและตรวจ access token
type TokenIssuer interface {
	Issue(u domain.User, ttl time.Duration) (token string, expiresAt time.Time, err error)
	Parse(token string) (Claims, error)
}

type Claims struct {
	UserID uuid.UUID
	Email  string
	Role   domain.Role
}

// Clock แยกเวลาออกจากตรรกะ เพื่อให้ test กำหนดเวลาเองได้
type Clock interface{ Now() time.Time }

// ---------- Repository ----------

type UserRepo interface {
	Create(ctx context.Context, u domain.User, passwordHash string) (domain.User, error)
	ByID(ctx context.Context, id uuid.UUID) (domain.User, error)
	ByEmail(ctx context.Context, email string) (domain.User, error)
	List(ctx context.Context) ([]domain.User, error)
	SetActive(ctx context.Context, id uuid.UUID, active bool) error
	UpdatePassword(ctx context.Context, id uuid.UUID, passwordHash string) error
}

// RefreshTokenRepo เก็บ refresh token แบบ hash เท่านั้น ไม่เคยเก็บค่าจริง
type RefreshTokenRepo interface {
	Store(ctx context.Context, t domain.RefreshToken) (uuid.UUID, error)
	ByHash(ctx context.Context, hash []byte) (domain.RefreshToken, error)
	Rotate(ctx context.Context, oldID, newID uuid.UUID) error
	Revoke(ctx context.Context, id uuid.UUID) error
	// RevokeChain เพิกถอนทั้งสายเมื่อพบว่ามีการนำ token ที่ใช้ไปแล้วกลับมาใช้ซ้ำ
	RevokeChain(ctx context.Context, userID uuid.UUID) error
}

type DatasetFilter struct {
	OwnerID *uuid.UUID
	Deleted bool // true = ดูถังขยะแทนรายการปกติ
	Limit   int
	Offset  int
}

type DatasetRepo interface {
	Create(ctx context.Context, d domain.Dataset, fileKey string) (domain.Dataset, error)
	ByID(ctx context.Context, id uuid.UUID) (domain.Dataset, error)
	FileKey(ctx context.Context, id uuid.UUID) (string, error)
	List(ctx context.Context, f DatasetFilter) ([]domain.Dataset, error)
	FindBySHA(ctx context.Context, sha []byte) (domain.Dataset, error)

	MarkReady(ctx context.Context, id uuid.UUID, counts map[string]int, yearMin, yearMax *int) error
	MarkFailed(ctx context.Context, id uuid.UUID, reason string) error

	SoftDelete(ctx context.Context, id, by uuid.UUID) error
	Restore(ctx context.Context, id uuid.UUID) error

	SaveIssues(ctx context.Context, id uuid.UUID, issues []domain.ImportIssue) error
	Issues(ctx context.Context, id uuid.UUID, limit, offset int) ([]domain.ImportIssue, int, error)

	// ReplaceFacts เขียนข้อมูลทั้งชุดของ dataset นี้ใน transaction เดียว
	// ถ้าล้มเหลวกลางคันต้องไม่มีข้อมูลค้างเหลืออยู่เลย
	ReplaceFacts(ctx context.Context, id uuid.UUID, wb *domain.Workbook) error

	// PurgeExpired ลบ fact rows ของ dataset ที่ถูก soft delete นานเกินกำหนด
	PurgeExpired(ctx context.Context, olderThan time.Time) (int, error)
}

type AuditRepo interface {
	Record(ctx context.Context, userID *uuid.UUID, action, target string, detail map[string]any, ip string) error
}

// RegionRepo เก็บภาคของศูนย์กระจายสินค้าไว้ที่เดียวและใช้ข้ามทุกชุดข้อมูล
// เพราะไฟล์ Excel ต้นทางไม่มีคอลัมน์ภาค การแก้ของ admin จึงต้องไม่หายไปเมื่ออัปโหลดไฟล์ใหม่
type RegionRepo interface {
	All(ctx context.Context) (map[string]string, error)
	Set(ctx context.Context, code, region string, by uuid.UUID) error
}
