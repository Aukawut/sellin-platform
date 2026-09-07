// Command api คือ HTTP server ของระบบ Sell-In Performance
//
// ไฟล์นี้ทำหน้าที่เดียวคือประกอบชิ้นส่วนเข้าด้วยกัน (composition root)
// ตรรกะทางธุรกิจทั้งหมดอยู่ใน internal/core ซึ่งไม่รู้จัก Fiber หรือ Postgres เลย
package main

import (
	"context"
	"errors"
	"log/slog"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/cors"
	"github.com/gofiber/fiber/v2/middleware/limiter"
	"github.com/gofiber/fiber/v2/middleware/recover"
	"github.com/gofiber/fiber/v2/middleware/requestid"

	"sellin-server/internal/core/ports"
	"sellin-server/internal/core/services"
	"sellin-server/internal/handlers"
	"sellin-server/internal/middleware"
	"sellin-server/internal/repositories/excel"
	"sellin-server/internal/repositories/filestore"
	"sellin-server/internal/repositories/postgres"
	"sellin-server/pkg/config"
	"sellin-server/pkg/token"
)

const version = "0.1.0"

func main() {
	log := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo}))
	slog.SetDefault(log)

	cfg, err := config.Load()
	if err != nil {
		log.Error("โหลดการตั้งค่าไม่สำเร็จ", "err", err)
		os.Exit(1)
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	pool, err := postgres.Connect(ctx, cfg.DatabaseURL)
	if err != nil {
		log.Error("เชื่อมต่อฐานข้อมูลไม่สำเร็จ", "err", err)
		os.Exit(1)
	}
	defer pool.Close()

	files, err := filestore.NewLocal(cfg.StorageDir)
	if err != nil {
		log.Error("เตรียมที่เก็บไฟล์ไม่สำเร็จ", "err", err)
		os.Exit(1)
	}

	// --- adapters ---
	users := postgres.NewUserRepo(pool)
	refresh := postgres.NewRefreshTokenRepo(pool)
	datasets := postgres.NewDatasetRepo(pool)
	audit := postgres.NewAuditRepo(pool)
	master := postgres.NewMasterRepo(pool)
	sellInRepo := postgres.NewSellInRepo(pool)
	sellOutRepo := postgres.NewSellOutRepo(pool)
	targetRepo := postgres.NewTargetRepo(pool)
	stockRepo := postgres.NewStockRepo(pool)
	regionRepo := postgres.NewRegionRepo(pool)
	exportRepo := postgres.NewExportRepo(pool)
	hasher := token.NewArgon2Hasher()
	issuer := token.NewJWTIssuer(cfg.JWTSecret, "sellin-platform")
	reader := excel.NewReader()

	// --- services ---
	authSvc := services.NewAuthService(users, refresh, audit, hasher, issuer,
		token.SystemClock{}, cfg.AccessTokenTTL, cfg.RefreshTokenTTL)
	importSvc := services.NewImportService(datasets, files, reader, audit, regionRepo, log, cfg.MaxUploadMB)
	analyticsSvc := services.NewAnalyticsService(master, sellInRepo, sellOutRepo, targetRepo, stockRepo, datasets)
	exportSvc := services.NewExportService(exportRepo, analyticsSvc, func() ports.SheetWriter {
		return excel.NewWorkbook()
	})

	// --- HTTP ---
	app := fiber.New(fiber.Config{
		AppName:               "Sell-In Performance API",
		ErrorHandler:          middleware.ErrorHandler(log),
		BodyLimit:             int(cfg.MaxUploadMB*1024*1024) + 1024*1024,
		ReadTimeout:           30 * time.Second,
		WriteTimeout:          60 * time.Second,
		DisableStartupMessage: true,
	})

	app.Use(recover.New())
	app.Use(requestid.New())
	app.Use(cors.New(cors.Config{
		AllowOrigins:     joinOrigins(cfg.CORSOrigins),
		AllowMethods:     "GET,POST,PUT,DELETE,OPTIONS",
		AllowHeaders:     "Origin,Content-Type,Accept,Authorization",
		AllowCredentials: true, // จำเป็นเพราะ refresh token เดินทางเป็น cookie
	}))

	handlers.NewHealthHandler(pool, version).Register(app)

	api := app.Group("/api/v1")

	// ยอดขายรายศูนย์เป็นข้อมูลธุรกิจ ห้ามให้ proxy หรือเบราว์เซอร์เก็บแคชไว้
	api.Use(func(c *fiber.Ctx) error {
		c.Set(fiber.HeaderCacheControl, "no-store")
		return c.Next()
	})

	authHandler := handlers.NewAuthHandler(authSvc, cfg.IsProduction(), cfg.RefreshTokenTTL)

	// จำกัดอัตราการล็อกอินเพื่อกันการเดารหัสผ่านแบบไล่ลอง
	loginLimiter := limiter.New(limiter.Config{
		Max:        5,
		Expiration: time.Minute,
		KeyGenerator: func(c *fiber.Ctx) string {
			return c.IP() + "|" + c.Path()
		},
		LimitReached: func(c *fiber.Ctx) error {
			return fiber.NewError(fiber.StatusTooManyRequests,
				"พยายามเข้าสู่ระบบบ่อยเกินไป กรุณารอสักครู่แล้วลองใหม่")
		},
	})
	api.Post("/auth/login", loginLimiter)
	authHandler.Register(api)

	protected := api.Group("", middleware.RequireAuth(issuer, users))
	protected.Get("/me", authHandler.Me)
	handlers.NewDatasetHandler(importSvc, files, datasets).Register(protected)
	handlers.NewDashboardHandler(analyticsSvc).Register(protected)
	handlers.NewExportHandler(exportSvc).Register(protected)
	protected.Post("/me/password", authHandler.ChangeOwnPassword)

	admin := protected.Group("/admin", middleware.RequireAdmin())
	admin.Post("/users", authHandler.CreateUser)
	handlers.NewAdminHandler(authSvc, regionRepo).Register(admin)

	// --- start & graceful shutdown ---
	go func() {
		log.Info("เริ่มให้บริการ", "port", cfg.Port, "env", cfg.AppEnv, "version", version)
		if err := app.Listen(":" + cfg.Port); err != nil && !errors.Is(err, context.Canceled) {
			log.Error("เซิร์ฟเวอร์หยุดทำงาน", "err", err)
			stop()
		}
	}()

	<-ctx.Done()
	log.Info("กำลังปิดระบบ รอให้ request ที่ค้างอยู่ทำงานจนจบ")

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()
	if err := app.ShutdownWithContext(shutdownCtx); err != nil {
		log.Error("ปิดระบบไม่เรียบร้อย", "err", err)
	}
	log.Info("ปิดระบบเรียบร้อย")
}

func joinOrigins(list []string) string {
	out := ""
	for i, o := range list {
		if i > 0 {
			out += ","
		}
		out += o
	}
	return out
}
