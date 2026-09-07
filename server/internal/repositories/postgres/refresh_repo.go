package postgres

import (
	"context"
	"errors"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"sellin-server/internal/core/domain"
)

var ErrTokenNotFound = errors.New("ไม่พบ refresh token นี้")

type RefreshTokenRepo struct{ pool *pgxpool.Pool }

func NewRefreshTokenRepo(pool *pgxpool.Pool) *RefreshTokenRepo {
	return &RefreshTokenRepo{pool: pool}
}

func (r *RefreshTokenRepo) Store(ctx context.Context, t domain.RefreshToken) (uuid.UUID, error) {
	var id uuid.UUID
	err := r.pool.QueryRow(ctx, `
		INSERT INTO refresh_tokens (user_id, token_hash, expires_at, user_agent, ip)
		VALUES ($1, $2, $3, nullif($4, ''), nullif($5, '')::inet)
		RETURNING id`,
		t.UserID, t.TokenHash, t.ExpiresAt, t.UserAgent, t.IP).Scan(&id)
	return id, err
}

func (r *RefreshTokenRepo) ByHash(ctx context.Context, hash []byte) (domain.RefreshToken, error) {
	var t domain.RefreshToken
	err := r.pool.QueryRow(ctx, `
		SELECT id, user_id, token_hash, issued_at, expires_at, revoked_at, replaced_by
		FROM refresh_tokens WHERE token_hash = $1`, hash).
		Scan(&t.ID, &t.UserID, &t.TokenHash, &t.IssuedAt, &t.ExpiresAt, &t.RevokedAt, &t.ReplacedBy)
	if errors.Is(err, pgx.ErrNoRows) {
		return domain.RefreshToken{}, ErrTokenNotFound
	}
	return t, err
}

// Rotate ทำเครื่องหมายว่า token เดิมถูกใช้ไปแล้วและถูกแทนด้วยตัวใหม่
// การเชื่อม replaced_by ไว้ทำให้ตรวจจับการนำ token เก่ากลับมาใช้ซ้ำได้
func (r *RefreshTokenRepo) Rotate(ctx context.Context, oldID, newID uuid.UUID) error {
	_, err := r.pool.Exec(ctx, `
		UPDATE refresh_tokens SET revoked_at = now(), replaced_by = $2
		WHERE id = $1 AND revoked_at IS NULL`, oldID, newID)
	return err
}

func (r *RefreshTokenRepo) Revoke(ctx context.Context, id uuid.UUID) error {
	_, err := r.pool.Exec(ctx,
		`UPDATE refresh_tokens SET revoked_at = now() WHERE id = $1 AND revoked_at IS NULL`, id)
	return err
}

// RevokeChain เพิกถอน token ที่ยังใช้ได้ทั้งหมดของผู้ใช้คนนี้
// เรียกเมื่อพบว่ามีคนนำ token ที่ถูกหมุนไปแล้วกลับมาใช้ ซึ่งแปลว่า token รั่วออกไป
func (r *RefreshTokenRepo) RevokeChain(ctx context.Context, userID uuid.UUID) error {
	_, err := r.pool.Exec(ctx,
		`UPDATE refresh_tokens SET revoked_at = now() WHERE user_id = $1 AND revoked_at IS NULL`, userID)
	return err
}
