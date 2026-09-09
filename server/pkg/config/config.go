package config

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/joho/godotenv"
)

// Config รวมค่าที่ระบบต้องใช้ตอนบูต โหลดจาก environment ทั้งหมด
// ไม่มีค่า default สำหรับความลับ ตั้งใจให้ระบบไม่ยอมสตาร์ทถ้าลืมตั้ง
type Config struct {
	AppEnv   string
	Port     string
	LogLevel string

	DatabaseURL string

	JWTSecret       string
	AccessTokenTTL  time.Duration
	RefreshTokenTTL time.Duration

	// CookieSecure ตั้ง flag Secure ให้ cookie ของ refresh token
	// ค่าเริ่มต้นเปิดเมื่อ APP_ENV=production แต่แยกออกมาเป็นค่าของตัวเองได้
	// เพราะบางที่ deploy หลัง HTTP ธรรมดา (เช่นเปิดด้วย IP ในเครือข่ายภายใน)
	// ถ้าปล่อยให้ Secure ติดอยู่ เบราว์เซอร์จะไม่ส่ง cookie กลับมา แล้วล็อกอินจะค้างเป็นวงจร
	CookieSecure bool

	StorageDir  string
	MaxUploadMB int64

	CORSOrigins []string

	// ลบ fact rows ของ dataset ที่ถูก soft delete นานเกินจำนวนวันนี้
	PurgeAfterDays int
}

func Load() (*Config, error) {
	// .env มีไว้ให้สะดวกตอน dev เท่านั้น ถ้าไม่มีก็ใช้ env ของ container ตามปกติ
	_ = godotenv.Load()

	c := &Config{
		AppEnv:          env("APP_ENV", "development"),
		Port:            env("PORT", "8080"),
		LogLevel:        env("LOG_LEVEL", "info"),
		DatabaseURL:     os.Getenv("DATABASE_URL"),
		JWTSecret:       os.Getenv("JWT_SECRET"),
		AccessTokenTTL:  envDuration("ACCESS_TOKEN_TTL", 15*time.Minute),
		RefreshTokenTTL: envDuration("REFRESH_TOKEN_TTL", 7*24*time.Hour),
		StorageDir:      env("STORAGE_DIR", "./storage"),
		MaxUploadMB:     int64(envInt("MAX_UPLOAD_MB", 25)),
		CORSOrigins:     splitAndTrim(env("CORS_ORIGINS", "http://localhost:5173")),
		PurgeAfterDays:  envInt("PURGE_AFTER_DAYS", 90),
	}
	c.CookieSecure = envBool("COOKIE_SECURE", c.IsProduction())

	var missing []string
	if c.DatabaseURL == "" {
		missing = append(missing, "DATABASE_URL")
	}
	if c.JWTSecret == "" {
		missing = append(missing, "JWT_SECRET")
	}
	if len(missing) > 0 {
		return nil, fmt.Errorf("ขาด environment variable ที่จำเป็น: %s", strings.Join(missing, ", "))
	}
	if len(c.JWTSecret) < 32 {
		return nil, fmt.Errorf("JWT_SECRET ต้องยาวอย่างน้อย 32 ตัวอักษร (ได้ %d)", len(c.JWTSecret))
	}

	return c, nil
}

func (c *Config) IsProduction() bool { return c.AppEnv == "production" }

func env(key, def string) string {
	if v := strings.TrimSpace(os.Getenv(key)); v != "" {
		return v
	}
	return def
}

func envInt(key string, def int) int {
	if v, err := strconv.Atoi(env(key, "")); err == nil {
		return v
	}
	return def
}

func envBool(key string, def bool) bool {
	if v, err := strconv.ParseBool(env(key, "")); err == nil {
		return v
	}
	return def
}

func envDuration(key string, def time.Duration) time.Duration {
	if v, err := time.ParseDuration(env(key, "")); err == nil {
		return v
	}
	return def
}

func splitAndTrim(s string) []string {
	parts := strings.Split(s, ",")
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		if p = strings.TrimSpace(p); p != "" {
			out = append(out, p)
		}
	}
	return out
}
