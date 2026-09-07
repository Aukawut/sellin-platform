# Sell-In Performance Platform

เว็บแอปสำหรับนำเข้าไฟล์ Excel `Data_Template_Sale_Performance_T-T_Final.xlsx`
แล้วสร้าง dashboard 4 หน้า (Sell-In, Sell-Out, Stock Inventory, Sale Analysis Planning)
โดยเก็บประวัติทุกไฟล์ที่เคยอัปโหลด และเลือกดูย้อนหลังได้ทุกชุด

แทนที่ `dashboard_sellin_Final.html` เดิมที่เป็นไฟล์เดียวขนาด 2.1 MB
ซึ่งฝังข้อมูลไว้ในโค้ดและข้อมูลหายทุกครั้งที่รีเฟรช

## สถานะปัจจุบัน

| ส่วน | สถานะ |
|---|---|
| โครงสร้างโปรเจกต์ + migrations + Docker | เสร็จ |
| ตัวอ่าน Excel + validation + golden test | เสร็จ |
| ล็อกอิน / JWT / refresh rotation | เสร็จ |
| คลังไฟล์ + soft delete + กู้คืน | เสร็จ |
| Analytics API 4 หน้า | เสร็จ |
| หน้าเว็บ: ล็อกอิน + คลังไฟล์ + โครงแอป | เสร็จ |
| หน้าเว็บ: dashboard 4 หน้า | เสร็จ |
| Export Excel + หน้าตั้งค่าระบบ | เสร็จ |
| UAT + คู่มือการใช้งาน | ยังไม่เริ่ม |

## Stack

- **Backend** Go 1.24 · Fiber v2 · Hexagonal architecture · pgx/v5
- **Database** PostgreSQL 16
- **Frontend** React + TypeScript · Tailwind CSS · Recharts (+ Chart.js เฉพาะกราฟรายวัน) · Lucide icons

## เริ่มใช้งานในเครื่อง

```bash
# 1. ตั้งค่าความลับ
cp .env.example .env
# แก้ POSTGRES_PASSWORD และ JWT_SECRET — สร้างค่าใหม่ด้วย: openssl rand -base64 48

# 2. เปิดฐานข้อมูล
docker compose up -d db

# 3. สร้างตารางและบัญชีผู้ดูแลระบบคนแรก
cd server
cp .env.example .env    # แก้ DATABASE_URL ให้ตรงกับพอร์ต 5433 ที่ compose เปิดไว้
go run ./cmd/migrate up
go run ./cmd/migrate -email you@prospira.com -name "ชื่อของคุณ" -password "รหัสผ่านอย่างน้อย12ตัว" create-admin

# 4. รันเซิร์ฟเวอร์
go run ./cmd/api
```

ตรวจว่าระบบพร้อมใช้งาน:

```bash
curl http://localhost:8080/healthz   # โปรเซสยังอยู่
curl http://localhost:8080/readyz    # ต่อฐานข้อมูลได้จริง
```

รันทั้งระบบด้วย Docker:

```bash
docker compose up -d --build
docker compose exec api /app/migrate up
```

## รันหน้าเว็บ

ต้องเปิด API ไว้ก่อน แล้วเปิด dev server ในอีกหน้าต่างหนึ่ง:

```bash
cd web
npm install     # ครั้งแรกเท่านั้น
npm run dev     # เปิดที่ http://localhost:5173
```

dev server ยิง `/api` ผ่าน proxy ไปยัง `http://localhost:8080` โดยตั้งใจ
เพื่อให้เบราว์เซอร์มองว่าเป็น origin เดียวกัน — cookie ของ refresh token จึงเดินทางได้ตามปกติ
ถ้า API รันอยู่พอร์ตอื่น ตั้ง `API_URL` ก่อนสั่ง `npm run dev`

โครงหน้าเว็บ:

```
src/
├── app/          โครงแอป — แถบบน แท็บ 4 หน้า และ router
├── features/
│   ├── auth/       AuthContext, หน้าล็อกอิน
│   ├── datasets/   คลังไฟล์ อัปโหลด ถังขยะ กู้คืน
│   └── dashboard/  ScopeBar + 4 หน้ารายงาน + hooks
├── components/
│   ├── ui/         Button, Field, Pill, Icon, States, ThemeToggle
│   ├── data/       KpiTile, RankList, DataTable
│   └── charts/     Panel, SeriesChart, Donut, ThailandMap, palette
├── lib/            api client (หมุน token อัตโนมัติ), formatters ภาษาไทย
└── types/api.ts    ชนิดข้อมูลที่ตรงกับ struct ฝั่ง Go
```

กราฟทั้งหมดใช้ **Recharts** อย่างเดียว ไม่ได้ใช้ Chart.js ตามที่วางแผนไว้ตอนแรก —
กราฟรายวันที่ต้องมีเส้นค่าเฉลี่ยใช้ `ReferenceLine` ของ Recharts ได้ตรง ๆ
การเพิ่มไลบรารีกราฟตัวที่สองเพื่อกราฟเดียวไม่คุ้มกับขนาดที่ต้องโหลดเพิ่ม

### การส่งออกไฟล์ Excel

ไฟล์ที่ได้มีแผ่น `Summary` รวมตัวชี้วัดทั้ง 4 หน้า ตามด้วยข้อมูลดิบตามขอบเขตที่เลือก
โดยใช้ **ชื่อ sheet และหัวคอลัมน์ชุดเดียวกับไฟล์ Template** ไฟล์ที่ส่งออกจึงนำกลับเข้าระบบได้ทันที
ทดสอบแล้วว่าอัปโหลดกลับเข้าไปได้ตัวชี้วัดทั้ง 16 ตัวตรงกับชุดเดิมทุกตัว

ตัวเลขในแผ่นสรุปเขียนเป็นตัวเลขจริง ไม่ใช่ข้อความที่จัดรูปแบบแล้ว ผู้รับไฟล์จึงคำนวณต่อใน Excel ได้ทันที
ต่างจาก dashboard เดิมที่คัดลอกข้อความจากหน้าจอลงไฟล์

**หมายเหตุทางเทคนิค** — excelize ไม่เขียนแท็ก `<dimension>` ให้เอง
Excel อ่านไฟล์ได้โดยไม่มีแท็กนี้ แต่เครื่องมืออื่นหลายตัว เช่น openpyxl ในโหมดอ่านอย่างเดียว
เชื่อค่านี้แล้วหยุดอ่านแค่ช่วงที่ระบุ ทำให้ไฟล์ดูเหมือนมีข้อมูลแค่เซลล์เดียว
`writer.go` จึงเขียนแท็กนี้ให้ชัดเจนทุกแผ่น

### ภาคของศูนย์กระจายสินค้า

ไฟล์ Excel ไม่มีคอลัมน์ภาค dashboard เดิมจึง hardcode ไว้ใน JavaScript
ระบบใหม่เก็บไว้ในตาราง `dc_regions` ซึ่งใช้ข้ามทุกชุดข้อมูล admin แก้ได้จากหน้าตั้งค่า
และการแก้จะไม่หายไปเมื่ออัปโหลดไฟล์ใหม่ เพราะตอนนำเข้าระบบจะเอาค่าจากตารางนี้ทับค่าตั้งต้นเสมอ

การแก้ภาคยังอัปเดตชุดข้อมูลที่นำเข้าไปแล้วย้อนหลังด้วย ไม่งั้นผู้ใช้จะแก้แล้วเห็นแผนที่เหมือนเดิม
จนกว่าจะอัปโหลดไฟล์ใหม่ ซึ่งอ่านได้เหมือนว่าการแก้ไม่มีผล

### แผนที่ประเทศไทย

ใช้ขอบเขตจริงของทั้ง 77 จังหวัดจากแพ็กเกจ `@svg-maps/thailand` (สัญญาอนุญาต **CC BY 4.0**
จึงต้องคงข้อความให้เครดิตใต้แผนที่ไว้) แทนเส้นขอบแบบลดทอนที่ dashboard เดิมวาดไว้

ข้อมูลถูกแปลงเป็นไฟล์ TypeScript ตั้งแต่ตอน build ไม่ใช่ตอน runtime:

```bash
cd web
node scripts/generate_thailand_map.mjs   # เขียนทับ src/components/charts/thailand-provinces.generated.ts
```

สคริปต์นี้เติมชื่อไทยของทุกจังหวัด คำนวณจุดกึ่งกลางไว้ล่วงหน้าสำหรับวางหมุด
และลดทศนิยมของพิกัดเหลือหนึ่งตำแหน่ง (ที่ viewBox 560×1025 ตาไม่เห็นความต่าง แต่ประหยัดขนาดไฟล์ราว 15%)
รันใหม่เมื่ออัปเกรดแพ็กเกจเท่านั้น

**ตำแหน่งหมุดมาจากข้อมูล ไม่ได้ฝังไว้ในโค้ด** — คำนวณจากค่าเฉลี่ยจุดกึ่งกลางของจังหวัดที่ศูนย์นั้นดูแล
ซึ่งอ่านมาจากคอลัมน์ `Region / Province` ในไฟล์ Excel ต่างจาก dashboard เดิมที่ฝังพิกัดของศูนย์ทั้ง 18 แห่งไว้ตายตัว
ศูนย์ที่เพิ่มเข้ามาใหม่จึงขึ้นแผนที่ได้เองโดยไม่ต้องแก้โค้ด

ไฟล์ต้นทางสะกดชื่อจังหวัดต่างจากชื่อทางการอยู่ 5 จังหวัด (`ฉะเฉิงเทรา`, `พะเยาว์`, `สุราษฎร์`,
`อยุธยา`, `แม่ฮองสอน`) ตารางค้นหาจึงรองรับการสะกดเหล่านี้ไว้ด้วย แทนที่จะไปแก้ไฟล์ต้นทาง
เพราะไฟล์ที่อัปโหลดครั้งหน้าก็จะสะกดแบบเดิมอีก

หลักที่ยึดไว้:

- **access token อยู่ในหน่วยความจำเท่านั้น** ไม่เก็บลง `localStorage`
  เพราะถ้าหน้าเว็บมีช่องโหว่ XSS สคริปต์จะอ่าน token จาก storage ได้ทันที
  ส่วน refresh token เป็น HttpOnly cookie ซึ่ง JavaScript อ่านไม่ได้อยู่แล้ว
- **หมุน token ให้เงียบ ๆ** เมื่อ API ตอบ 401 ตัว client จะขอ token ใหม่แล้วยิงซ้ำเอง
  ผู้ใช้ไม่ถูกเด้งออกกลางคันทุก 15 นาที
- **ไอคอนใช้ Lucide ทั้งหมด** ผ่าน wrapper `<Icon />` ตัวเดียวที่คุมขนาดกับความหนาเส้น
  สีรับจาก `currentColor` เสมอ จึงเปลี่ยนตามธีมเอง
- **ธีมรองรับสามสถานะ** สว่าง มืด และตามระบบ โดยเขียนค่าเป็น CSS variable ชั้นเดียว
  แล้ว Tailwind อ้างผ่าน `@theme inline` — เปลี่ยนสีทั้งแอปได้จากที่เดียว

## เชื่อมต่อฐานข้อมูลจากเครื่อง Host

compose เปิดพอร์ตของ Postgres ออกมาที่ **5433** โดยตั้งใจ ไม่ใช่ 5432
เพื่อไม่ให้ชนกับ PostgreSQL ที่อาจติดตั้งไว้ในเครื่องอยู่แล้ว

ตั้งค่าใน DBeaver (New Connection → PostgreSQL):

| ช่อง | ค่า |
|---|---|
| Host | `localhost` |
| Port | `5433` |
| Database | `sellin` |
| Username | `sellin` |
| Password | ค่าของ `POSTGRES_PASSWORD` ในไฟล์ `.env` |
| SSL | ปิด |

ครั้งแรกที่เชื่อมต่อ DBeaver จะถามว่าจะดาวน์โหลด driver ของ PostgreSQL ไหม — ตอบ Download

ถ้าเชื่อมต่อไม่ได้ ให้ไล่ตรวจตามลำดับนี้:

```bash
docker compose ps db                  # ต้องขึ้น Up (healthy)
nc -z localhost 5433 && echo ok       # พอร์ตเปิดจากเครื่อง Host จริงไหม
docker compose logs db --tail 30      # ดูข้อความผิดพลาดของฐานข้อมูล
```

**หมายเหตุเรื่องความปลอดภัย** — พอร์ต 5433 ถูกเปิดที่ `0.0.0.0` ซึ่งหมายถึงเครื่องอื่นในเครือข่ายเดียวกันก็ต่อเข้ามาได้
สะดวกตอนพัฒนา แต่ก่อนขึ้นเซิร์ฟเวอร์จริงควรแก้ `docker-compose.yml` เป็น `"127.0.0.1:5433:5432"`
เพื่อให้ต่อได้เฉพาะจากตัวเครื่องเอง หรือถอดบล็อก `ports` ของ `db` ออกทั้งหมด
เพราะ `api` คุยกับ `db` ผ่านเครือข่ายภายในของ compose อยู่แล้ว ไม่ต้องพึ่งพอร์ตที่เปิดออกมา

### ตารางที่น่าสนใจ

| ตาราง | เก็บอะไร |
|---|---|
| `datasets` | ไฟล์ที่อัปโหลด — `deleted_at` เป็น NULL คือยังไม่ถูกลบ |
| `v_active_datasets` | view ที่กรอง soft delete ออกแล้ว เป็นทางที่ dashboard อ่านจริง |
| `fact_sellin` · `fact_sellout` · `fact_stock` · `fact_target` | ข้อมูลจากแต่ละ sheet ผูกกับ `dataset_id` |
| `dim_customers` · `dim_products` | ศูนย์กระจายสินค้าและสินค้า พร้อมคอลัมน์ `region` และ `status` |
| `import_issues` | ปัญหาที่พบตอนนำเข้า แยกตาม severity |
| `audit_log` | การเข้าสู่ระบบ อัปโหลด ลบ กู้คืน |

ทุกตาราง fact มี `dataset_id` เป็นคอลัมน์แรกของ index เสมอ
เวลา query เองอย่าลืมใส่เงื่อนไขนี้ ไม่งั้นจะได้ข้อมูลของหลายไฟล์ปนกัน:

```sql
-- ยอด Sell-In รายเดือนของชุดข้อมูลล่าสุด
SELECT f.year, f.month,
       sum(f.revenue)  AS revenue,
       sum(f.cartons)  AS cartons
FROM fact_sellin f
WHERE f.dataset_id = (SELECT id FROM v_active_datasets ORDER BY uploaded_at DESC LIMIT 1)
GROUP BY f.year, f.month
ORDER BY f.year, f.month;
```

```sql
-- สินค้าที่ต้องเฝ้าระวังในเดือนล่าสุด: ของหมดหรือสต๊อกเกินสามเดือน
WITH latest AS (SELECT id FROM v_active_datasets ORDER BY uploaded_at DESC LIMIT 1),
     snap AS (
       SELECT max(year * 12 + month) AS idx
       FROM fact_stock WHERE dataset_id = (SELECT id FROM latest)
     )
SELECT s.product_desc, s.group_name,
       sum(s.beginning) AS beginning,
       sum(s.sell_out)  AS sell_out,
       sum(s.ending)    AS ending,
       CASE WHEN sum(s.sell_out) > 0
            THEN round(sum(s.ending) / sum(s.sell_out), 1) END AS cover_months
FROM fact_stock s, snap
WHERE s.dataset_id = (SELECT id FROM latest)
  AND (s.year * 12 + s.month) = snap.idx
GROUP BY s.product_code, s.product_desc, s.group_name
HAVING sum(s.ending) <= 0
    OR (sum(s.sell_out) > 0 AND sum(s.ending) / sum(s.sell_out) > 3)
ORDER BY ending DESC;
```

## ทดสอบ

```bash
cd server
go test ./...
```

ชุดทดสอบแบ่งเป็นสามชั้น:

| ชั้น | ทดสอบอะไร | ต้องมี DB ไหม |
|---|---|---|
| `internal/core/domain` | สูตรคำนวณล้วน — สถานะสต๊อก, Stock Cover, Forecast, การจัดอันดับ | ไม่ต้อง |
| `internal/repositories/excel` | การอ่านไฟล์ Excel เทียบกับ dashboard เดิม | ไม่ต้อง |
| `scripts/verify_dashboard.py` | ผลลัพธ์ของ API ทุก endpoint เทียบกับ dashboard เดิม | ต้องมี |

**ไฟล์ทดสอบไม่ได้อยู่ใน repo** เพราะเป็นข้อมูลยอดขายจริงของบริษัท
เมื่อไม่มีไฟล์ golden test จะข้ามพร้อมบอกวิธีเตรียม ไม่ใช่ล้มเหลวจนดูเหมือนโค้ดพัง
เตรียมได้ด้วยสองคำสั่งนี้ (รันจากโฟลเดอร์ `sellin-platform`):

```bash
cp "<โฟลเดอร์ที่เก็บไฟล์>/Data_Template_Sale_Performance_T-T_Final.xlsx" server/testdata/template.xlsx
python3 scripts/extract_golden.py "<โฟลเดอร์ที่เก็บไฟล์>/dashboard_sellin_Final.html" server/testdata/golden_dashboard.json
```

ชั้นที่สามรันแยกเพราะต้องมีฐานข้อมูลและไฟล์ที่นำเข้าแล้ว:

```bash
python3 scripts/verify_dashboard.py "http://localhost:8080/api/v1" "<access token>" "<dataset id>"
```

ตรวจ 100 รายการ ครอบคลุมทุก KPI ทุกกราฟ ทุกตาราง ของทั้งสี่หน้า

`internal/repositories/excel` มี **golden test** ที่อ่านไฟล์ Template จริง
แล้วเทียบยอดรวมทุกตัวกับ `const DATA` ของ dashboard เดิม
(สกัดไว้ที่ `testdata/golden_dashboard.json` ด้วย `scripts/extract_golden.py`)

ถ้าแก้ตรรกะการอ่านไฟล์แล้วยอดเพี้ยนแม้แต่สตางค์เดียว test จะฟ้องทันที

สร้าง fixture ใหม่อีกครั้งเมื่อไฟล์ต้นฉบับเปลี่ยน โดยใช้คำสั่งเดียวกับด้านบน

## API ของ dashboard

ทุก endpoint รับพารามิเตอร์ชุดเดียวกัน — `dataset_id`, `dc`, `year`, `month`
โดย `ALL` เป็นค่าที่ถูกต้องของสามตัวหลัง ตรงกับพฤติกรรม dropdown ของ dashboard เดิม

```
GET /api/v1/datasets/:id/filters      ตัวเลือก dropdown ที่มีจริงในชุดข้อมูลนี้
GET /api/v1/dashboard/sellin          KPI 4 · trend · daily · tracking · donut ×2 · ranking · แผนที่
GET /api/v1/dashboard/sellout         KPI 4 · trend · เทียบสองปี · donut ×2 พร้อม YoY · ตารางสินค้า · ranking
GET /api/v1/dashboard/stock           KPI 4 · trend · flow · donut ×2 · ตารางสถานะรายสินค้า
GET /api/v1/dashboard/planning        KPI 4 · actual/target/forecast · category bar · group trend · ภาค · ตารางแผน
GET /api/v1/export                    ส่งออกข้อมูลตามขอบเขตเป็นไฟล์ .xlsx
```

เส้นทางสำหรับผู้ดูแลระบบ:

```
GET  /api/v1/admin/users              รายชื่อผู้ใช้ทั้งหมด
POST /api/v1/admin/users              สร้างบัญชีใหม่
POST /api/v1/admin/users/:id/active   เปิด/ปิดการใช้งานบัญชี
POST /api/v1/admin/users/:id/password ตั้งรหัสผ่านใหม่ให้ผู้ใช้
GET  /api/v1/admin/regions            ภาคของศูนย์กระจายสินค้าทั้งหมด
POST /api/v1/admin/regions/:code      แก้ภาคของศูนย์
POST /api/v1/me/password              เปลี่ยนรหัสผ่านของตัวเอง (ทุกระดับสิทธิ์)
```

`sellout` และ `stock` รับ `group` เพิ่ม ส่วน `stock` รับ `status` ด้วย

ทุกคำตอบมีบล็อก `scope` ที่บอกว่าตัวเลขชุดนี้ครอบคลุมอะไร รวมถึง `effective_month`
ซึ่งคือเดือนที่ระบบเลือกให้เองเมื่อผู้ใช้เลือก "ทุกเดือน" — หน้าเว็บต้องใช้ค่านี้ขึ้นป้ายกำกับให้ถูก

**backend ส่งตัวเลขและความหมาย ไม่ส่งข้อความที่จัดรูปแบบแล้ว** เช่น KPI การเติบโตส่ง
`{pct, direction, current, previous, base_year, prev_year}` ให้หน้าเว็บประกอบประโยคเอง
ต่างจาก dashboard เดิมที่ประกอบข้อความไทยไว้ในโค้ด จนแก้คำพูดทีต้องแก้ตรรกะไปด้วย

## โครงสร้าง

```
server/
├── cmd/api          เซิร์ฟเวอร์ HTTP (composition root)
├── cmd/migrate      รัน migration และสร้างบัญชี admin
├── internal/core/
│   ├── domain       โครงสร้างข้อมูลและกฎธุรกิจล้วน ไม่ import อะไรนอก stdlib
│   ├── ports        interface ที่แกนกลางเรียกใช้
│   └── services     ประสานงานระหว่าง repository กับ domain
├── internal/handlers        adapter ฝั่ง Fiber
├── internal/repositories/   adapter ฝั่ง Postgres, Excel, ที่เก็บไฟล์
└── migrations/              ไฟล์ .sql ฝังอยู่ในไบนารี
```

ทิศทางการพึ่งพาชี้เข้าข้างในเสมอ — `core` ไม่รู้จัก Fiber, Postgres หรือ Excel เลย

## สิ่งที่ต่างจาก dashboard เดิมโดยตั้งใจ

1. **แก้บั๊ก Ending Stock** โค้ดเดิมอ่านคอลัมน์ `Stock on Hand` แต่หัวคอลัมน์จริงคือ `Ending Stock`
   ทำให้สต๊อกคงเหลือเป็น 0 ทุกแถวหลัง import ระบบใหม่รับทั้งสองชื่อและมี test เฝ้าไว้
2. **อ่าน `Data_Product`** เดิมสถานะสินค้า 97 รายการถูก hardcode ไว้ใน JavaScript
3. **ไม่ผูกกับปี 2568/2569** เดิมกราฟเขียน `if(r[0]===2025)` ตรง ๆ ระบบใหม่คำนวณปีฐานจากข้อมูลจริง
4. **หนึ่งไฟล์ = หนึ่งชุดข้อมูล** เดิมใช้ `Object.assign` ทำให้ sheet ที่ขาดหายไปยังใช้ข้อมูลไฟล์ก่อนหน้าค้างอยู่
5. **ภาคของศูนย์อยู่ในฐานข้อมูล** เดิม hardcode `DC_REGION` ไว้ 18 รายการใน JavaScript
   ตอนนี้อยู่ในคอลัมน์ `region` ของ `dim_customers` ซึ่งแก้ไขได้
6. **พิกัดแผนที่เป็นของหน้าเว็บ** backend ส่งเฉพาะรหัสศูนย์กับยอด ไม่ส่งพิกัด x/y
   เพราะพิกัดเป็นคุณสมบัติของภาพ SVG ไม่ใช่ของข้อมูล

## ข้อควรระวังตอน deploy

ไฟล์ Excel ต้นฉบับอยู่บน Docker volume ส่วน metadata อยู่ใน Postgres
การสำรองข้อมูลต้อง **`pg_dump` ก่อน แล้วค่อยคัดลอก volume** เสมอ
ลำดับนี้ทำให้กรณีแย่ที่สุดคือมีไฟล์กำพร้าที่ไม่มี record ชี้ถึง ซึ่งไม่ทำให้ระบบเสียหาย
ถ้าทำสลับลำดับจะได้ record ที่ชี้ไปยังไฟล์ที่ยังไม่มีอยู่แทน
