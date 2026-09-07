// Package migrations ฝังไฟล์ .sql ไว้ในไบนารี เพื่อให้ deploy ได้ด้วยไฟล์เดียว
// ไม่ต้องคอยจำว่าต้องคัดลอกโฟลเดอร์ migrations ตามไปด้วย
package migrations

import "embed"

//go:embed *.sql
var FS embed.FS
