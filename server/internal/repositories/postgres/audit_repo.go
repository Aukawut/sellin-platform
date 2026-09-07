package postgres

import (
	"context"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

type AuditRepo struct{ pool *pgxpool.Pool }

func NewAuditRepo(pool *pgxpool.Pool) *AuditRepo { return &AuditRepo{pool: pool} }

// Record บันทึกเหตุการณ์สำคัญ: เข้าสู่ระบบ อัปโหลด ลบ กู้คืน และการแก้ผู้ใช้
//
// ตั้งใจไม่คืน error ให้ผู้เรียกต้องจัดการ เพราะ audit ที่ล้มเหลวไม่ควรทำให้งานหลักล้มตาม
// ผู้เรียกที่สนใจสามารถตรวจ error ได้ แต่ handler ส่วนใหญ่จะ log ทิ้งไว้เฉย ๆ
func (r *AuditRepo) Record(ctx context.Context, userID *uuid.UUID, action, target string, detail map[string]any, ip string) error {
	_, err := r.pool.Exec(ctx, `
		INSERT INTO audit_log (user_id, action, target, detail, ip)
		VALUES ($1, $2, nullif($3, ''), $4, nullif($5, '')::inet)`,
		userID, action, target, detail, ip)
	return err
}
