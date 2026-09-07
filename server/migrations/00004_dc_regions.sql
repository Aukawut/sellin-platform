-- +goose Up

-- ภาคของศูนย์กระจายสินค้าไม่มีอยู่ในไฟล์ Excel ต้นทาง
-- dashboard เดิม hardcode ไว้ใน JavaScript ทำให้เพิ่มศูนย์ใหม่แล้วต้องแก้โค้ด
--
-- ตารางนี้เก็บไว้ที่เดียวและใช้ข้ามทุกชุดข้อมูล การแก้ของ admin จึงไม่หายไปเมื่ออัปโหลดไฟล์ใหม่
CREATE TABLE dc_regions (
    code       text PRIMARY KEY,
    region     text NOT NULL,
    updated_at timestamptz NOT NULL DEFAULT now(),
    updated_by uuid REFERENCES users (id) ON DELETE SET NULL
);

-- ค่าตั้งต้นยกมาจาก DC_REGION ของ dashboard เดิมทั้ง 18 รายการ
INSERT INTO dc_regions (code, region) VALUES
    ('BP018', 'อีสาน'), ('BP022', 'เหนือ'), ('BP015', 'อีสาน'), ('BP010', 'เหนือ'),
    ('BP010-1', 'อีสาน'), ('BP024', 'อีสาน'), ('TT091', 'อีสาน'), ('BP069', 'กลาง'),
    ('BP023', 'ตะวันออก'), ('TT065', 'กลาง'), ('BP007', 'ตะวันตก'), ('BP009', 'อีสาน'),
    ('BP070', 'กลาง'), ('BP002', 'กลาง'), ('BP011', 'ใต้'), ('BP012', 'อีสาน'),
    ('BP071', 'เหนือ'), ('BP004', 'กลาง');

-- +goose Down
DROP TABLE IF EXISTS dc_regions;
