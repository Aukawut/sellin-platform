-- +goose Up
CREATE EXTENSION IF NOT EXISTS pgcrypto;

CREATE TABLE users (
    id            uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    email         text        NOT NULL,
    password_hash text        NOT NULL,
    display_name  text        NOT NULL,
    role          text        NOT NULL DEFAULT 'viewer',
    is_active     boolean     NOT NULL DEFAULT true,
    created_at    timestamptz NOT NULL DEFAULT now(),
    updated_at    timestamptz NOT NULL DEFAULT now(),
    CONSTRAINT users_role_chk CHECK (role IN ('admin', 'viewer'))
);

-- ค้นหาและบังคับความไม่ซ้ำแบบไม่สนตัวพิมพ์ โดยไม่ต้องพึ่ง citext
CREATE UNIQUE INDEX ux_users_email ON users (lower(email));

CREATE TABLE refresh_tokens (
    id          uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id     uuid        NOT NULL REFERENCES users (id) ON DELETE CASCADE,
    token_hash  bytea       NOT NULL,
    issued_at   timestamptz NOT NULL DEFAULT now(),
    expires_at  timestamptz NOT NULL,
    revoked_at  timestamptz,
    -- ชี้ไปยัง token ที่มาแทนตอนหมุน ใช้ไล่สายเมื่อพบการใช้ token ซ้ำ
    replaced_by uuid REFERENCES refresh_tokens (id) ON DELETE SET NULL,
    user_agent  text,
    ip          inet
);

CREATE UNIQUE INDEX ux_refresh_hash ON refresh_tokens (token_hash);
CREATE INDEX ix_refresh_user ON refresh_tokens (user_id) WHERE revoked_at IS NULL;

CREATE TABLE audit_log (
    id         bigserial PRIMARY KEY,
    at         timestamptz NOT NULL DEFAULT now(),
    user_id    uuid REFERENCES users (id) ON DELETE SET NULL,
    action     text NOT NULL,
    target     text,
    detail     jsonb,
    ip         inet
);

CREATE INDEX ix_audit_at ON audit_log (at DESC);

-- +goose Down
DROP TABLE IF EXISTS audit_log;
DROP TABLE IF EXISTS refresh_tokens;
DROP TABLE IF EXISTS users;
