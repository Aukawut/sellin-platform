// Command migrate รัน database migration และสร้างบัญชี admin เริ่มต้น
//
//	migrate up              — อัปเดตฐานข้อมูลให้เป็นเวอร์ชันล่าสุด
//	migrate down            — ถอยกลับหนึ่งขั้น
//	migrate status          — ดูว่าเวอร์ชันไหนถูกรันไปแล้วบ้าง
//	migrate create-admin    — สร้างผู้ใช้ระดับ admin คนแรก
package main

import (
	"context"
	"database/sql"
	"flag"
	"fmt"
	"log"
	"os"
	"time"

	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/pressly/goose/v3"

	"sellin-server/internal/core/domain"
	"sellin-server/internal/repositories/postgres"
	"sellin-server/migrations"
	"sellin-server/pkg/config"
	"sellin-server/pkg/token"
)

func main() {
	email := flag.String("email", "", "อีเมลของ admin (ใช้กับคำสั่ง create-admin)")
	name := flag.String("name", "", "ชื่อที่แสดงของ admin")
	password := flag.String("password", "", "รหัสผ่านเริ่มต้น อย่างน้อย 12 ตัวอักษร")
	flag.Parse()

	cmd := flag.Arg(0)
	if cmd == "" {
		cmd = "up"
	}

	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("โหลดการตั้งค่าไม่สำเร็จ: %v", err)
	}

	if cmd == "create-admin" {
		if err := createAdmin(cfg.DatabaseURL, *email, *name, *password); err != nil {
			log.Fatalf("สร้างบัญชี admin ไม่สำเร็จ: %v", err)
		}
		return
	}

	db, err := sql.Open("pgx", cfg.DatabaseURL)
	if err != nil {
		log.Fatalf("เชื่อมต่อฐานข้อมูลไม่สำเร็จ: %v", err)
	}
	defer db.Close()

	goose.SetBaseFS(migrations.FS)
	if err := goose.SetDialect("postgres"); err != nil {
		log.Fatal(err)
	}

	if err := goose.RunContext(context.Background(), cmd, db, "."); err != nil {
		log.Fatalf("รันคำสั่ง %q ไม่สำเร็จ: %v", cmd, err)
	}
}

func createAdmin(dbURL, email, name, password string) error {
	if email == "" || name == "" || password == "" {
		return fmt.Errorf("ต้องระบุ -email, -name และ -password ครบทั้งสามค่า")
	}
	if err := domain.ValidatePassword(password); err != nil {
		return err
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	pool, err := postgres.Connect(ctx, dbURL)
	if err != nil {
		return err
	}
	defer pool.Close()

	hash, err := token.NewArgon2Hasher().Hash(password)
	if err != nil {
		return err
	}

	users := postgres.NewUserRepo(pool)
	created, err := users.Create(ctx, domain.User{
		Email:       email,
		DisplayName: name,
		Role:        domain.RoleAdmin,
		IsActive:    true,
	}, hash)
	if err != nil {
		return err
	}

	fmt.Fprintf(os.Stdout, "สร้างบัญชี admin แล้ว: %s (%s)\n", created.DisplayName, created.Email)
	return nil
}
