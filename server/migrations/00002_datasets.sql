-- +goose Up
CREATE TABLE datasets (
    id                uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    owner_id          uuid        NOT NULL REFERENCES users (id) ON DELETE RESTRICT,
    original_filename text        NOT NULL,
    -- ชื่อไฟล์บนดิสก์เป็นค่าสุ่ม ไม่เคยใช้ชื่อที่ผู้ใช้ส่งมาเขียนไฟล์จริง
    file_key          text        NOT NULL,
    sha256            bytea       NOT NULL,
    size_bytes        bigint      NOT NULL,
    status            text        NOT NULL DEFAULT 'processing',
    label             text,
    row_counts        jsonb       NOT NULL DEFAULT '{}'::jsonb,
    year_min          int,
    year_max          int,
    error_message     text,
    uploaded_at       timestamptz NOT NULL DEFAULT now(),
    finished_at       timestamptz,
    deleted_at        timestamptz,
    deleted_by        uuid REFERENCES users (id) ON DELETE SET NULL,
    -- เวลาที่ cron ล้าง fact rows ทิ้ง ตัว record และไฟล์ต้นฉบับยังอยู่ ให้ re-import ได้
    purged_at         timestamptz,
    CONSTRAINT datasets_status_chk CHECK (status IN ('processing', 'ready', 'failed'))
);

CREATE INDEX ix_datasets_live ON datasets (uploaded_at DESC) WHERE deleted_at IS NULL;
CREATE INDEX ix_datasets_owner ON datasets (owner_id, uploaded_at DESC);
-- เตือนไฟล์ซ้ำได้เร็วโดยไม่บล็อกการอัปโหลด จึงเป็น index ธรรมดา ไม่ใช่ unique
CREATE INDEX ix_datasets_sha ON datasets (sha256) WHERE deleted_at IS NULL;

-- Repository ของ dashboard อ่านผ่าน view นี้เท่านั้น เพื่อไม่ต้องหวังว่าจะไม่ลืมเติมเงื่อนไข soft delete
CREATE VIEW v_active_datasets AS
SELECT * FROM datasets WHERE deleted_at IS NULL;

CREATE TABLE import_issues (
    id         bigserial PRIMARY KEY,
    dataset_id uuid NOT NULL REFERENCES datasets (id) ON DELETE CASCADE,
    sheet      text NOT NULL,
    row_no     int,
    severity   text NOT NULL,
    field      text,
    message    text NOT NULL,
    CONSTRAINT import_issues_sev_chk CHECK (severity IN ('info', 'warning', 'blocking'))
);

CREATE INDEX ix_issues_dataset ON import_issues (dataset_id, severity);

-- +goose Down
DROP TABLE IF EXISTS import_issues;
DROP VIEW IF EXISTS v_active_datasets;
DROP TABLE IF EXISTS datasets;
