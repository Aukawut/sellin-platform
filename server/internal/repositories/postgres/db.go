// Package postgres คือ adapter ที่ทำให้ interface ใน core/ports เป็นจริงด้วย PostgreSQL
package postgres

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

// Connect เปิด pool และตรวจว่าต่อฐานข้อมูลได้จริงก่อนคืนค่า
// ตั้งใจให้ระบบไม่สตาร์ทถ้าต่อ DB ไม่ได้ แทนที่จะขึ้นมาแล้วพังตอนมี request แรก
func Connect(ctx context.Context, url string) (*pgxpool.Pool, error) {
	cfg, err := pgxpool.ParseConfig(url)
	if err != nil {
		return nil, fmt.Errorf("DATABASE_URL ไม่ถูกต้อง: %w", err)
	}
	cfg.MaxConns = 10
	cfg.MinConns = 2
	cfg.MaxConnLifetime = time.Hour
	cfg.MaxConnIdleTime = 15 * time.Minute
	cfg.HealthCheckPeriod = time.Minute

	pool, err := pgxpool.NewWithConfig(ctx, cfg)
	if err != nil {
		return nil, err
	}

	pingCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()
	if err := pool.Ping(pingCtx); err != nil {
		pool.Close()
		return nil, fmt.Errorf("เชื่อมต่อฐานข้อมูลไม่สำเร็จ: %w", err)
	}
	return pool, nil
}
