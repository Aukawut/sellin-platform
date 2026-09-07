-- +goose Up

-- หมายเหตุเรื่องความละเอียดของตัวเลข:
--   คอลัมน์ปริมาณใช้ numeric(18,6) เพราะ Sum Cartoon ในไฟล์ต้นทางมีทศนิยมเกิน 2 ตำแหน่ง
--   ถ้าเก็บเป็น numeric(14,2) ยอดรวมของทั้งไฟล์จะคลาดจาก dashboard เดิมราว 0.17 ลัง
--   คอลัมน์จำนวนเงินใช้ numeric(18,4) ซึ่งละเอียดพอสำหรับยอดขายระดับร้อยล้านบาท

-- ทุกตารางในไฟล์นี้ผูกกับ dataset_id เสมอ: หนึ่งไฟล์ที่อัปโหลด = หนึ่งชุดข้อมูลที่แยกขาดจากกัน
-- group_name / category ถูกเก็บซ้ำในตาราง fact โดยตั้งใจ เพราะไฟล์ต้นทางมีค่าเหล่านี้ติดมากับทุกแถว
-- และบางแถวไม่ตรงกับ Data_Product ถ้า join เอาอย่างเดียวยอดจะไม่ตรงกับ dashboard เดิม

CREATE TABLE dim_customers (
    dataset_id uuid NOT NULL REFERENCES datasets (id) ON DELETE CASCADE,
    code       text NOT NULL,
    name       text NOT NULL,
    status     text NOT NULL DEFAULT 'Active',
    provinces  text[] NOT NULL DEFAULT '{}',
    region     text,
    PRIMARY KEY (dataset_id, code)
);

CREATE TABLE dim_products (
    dataset_id   uuid NOT NULL REFERENCES datasets (id) ON DELETE CASCADE,
    code         text NOT NULL,
    description  text,
    group_name   text,
    category     text,
    innerbox     numeric(18, 6),
    price_unit   numeric(14, 4),
    price_carton numeric(14, 4),
    status       text,
    PRIMARY KEY (dataset_id, code)
);

CREATE TABLE fact_sellin (
    id            bigserial PRIMARY KEY,
    dataset_id    uuid NOT NULL REFERENCES datasets (id) ON DELETE CASCADE,
    sale_date     date,
    year          int  NOT NULL,
    month         int  NOT NULL,
    day           int,
    doc_no        text,
    customer_code text NOT NULL,
    product_code  text,
    product_desc  text,
    group_name    text,
    category      text,
    qty           numeric(18, 6),
    unit          text,
    innerbox      numeric(18, 6),
    cartons       numeric(18, 6) NOT NULL DEFAULT 0,
    price_unit    numeric(14, 4),
    revenue       numeric(18, 4) NOT NULL DEFAULT 0,
    status_sale   text
);

CREATE TABLE fact_sellout (
    id            bigserial PRIMARY KEY,
    dataset_id    uuid NOT NULL REFERENCES datasets (id) ON DELETE CASCADE,
    year          int  NOT NULL,
    month         int  NOT NULL,
    customer_code text NOT NULL,
    product_code  text,
    product_desc  text,
    group_name    text,
    category      text,
    cartons       numeric(18, 6) NOT NULL DEFAULT 0,
    unit          text,
    price_unit    numeric(14, 4),
    revenue       numeric(18, 4) NOT NULL DEFAULT 0,
    status        text
);

CREATE TABLE fact_stock (
    id            bigserial PRIMARY KEY,
    dataset_id    uuid NOT NULL REFERENCES datasets (id) ON DELETE CASCADE,
    year          int  NOT NULL,
    month         int  NOT NULL,
    customer_code text NOT NULL,
    product_code  text,
    product_desc  text,
    group_name    text,
    category      text,
    beginning     numeric(18, 6) NOT NULL DEFAULT 0,
    sell_in       numeric(18, 6) NOT NULL DEFAULT 0,
    sell_out      numeric(18, 6) NOT NULL DEFAULT 0,
    ending        numeric(18, 6) NOT NULL DEFAULT 0
);

CREATE TABLE fact_target (
    id            bigserial PRIMARY KEY,
    dataset_id    uuid NOT NULL REFERENCES datasets (id) ON DELETE CASCADE,
    year          int  NOT NULL,
    month         int  NOT NULL,
    customer_code text NOT NULL,
    customer_name text,
    category      text,
    target_amount numeric(18, 4) NOT NULL DEFAULT 0
);

CREATE TABLE fact_tracking_nsfs (
    id            bigserial PRIMARY KEY,
    dataset_id    uuid NOT NULL REFERENCES datasets (id) ON DELETE CASCADE,
    year          int  NOT NULL,
    month         int  NOT NULL,
    customer_code text NOT NULL,
    group_name    text,
    target_qty    numeric(18, 6) NOT NULL DEFAULT 0,
    cartons       numeric(18, 6) NOT NULL DEFAULT 0
);

-- dataset_id นำหน้าเสมอ เพราะทุก query ของ dashboard เริ่มจากการเลือกไฟล์
CREATE INDEX ix_sellin_scope ON fact_sellin (dataset_id, year, month, customer_code)
    INCLUDE (revenue, cartons);
CREATE INDEX ix_sellin_group ON fact_sellin (dataset_id, group_name);
CREATE INDEX ix_sellout_scope ON fact_sellout (dataset_id, year, month, customer_code)
    INCLUDE (revenue, cartons);
CREATE INDEX ix_sellout_group ON fact_sellout (dataset_id, group_name);
CREATE INDEX ix_stock_snap ON fact_stock (dataset_id, year, month, customer_code);
CREATE INDEX ix_target_scope ON fact_target (dataset_id, year, month, customer_code);
CREATE INDEX ix_tracking_scope ON fact_tracking_nsfs (dataset_id, year, month, customer_code);

-- +goose Down
DROP TABLE IF EXISTS fact_tracking_nsfs;
DROP TABLE IF EXISTS fact_target;
DROP TABLE IF EXISTS fact_stock;
DROP TABLE IF EXISTS fact_sellout;
DROP TABLE IF EXISTS fact_sellin;
DROP TABLE IF EXISTS dim_products;
DROP TABLE IF EXISTS dim_customers;
