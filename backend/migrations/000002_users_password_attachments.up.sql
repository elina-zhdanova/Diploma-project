-- Пароли для логина, обоснование заявки, вложения (метаданные + ключ в S3)

ALTER TABLE users ADD COLUMN IF NOT EXISTS password_hash VARCHAR(255);

ALTER TABLE requests ADD COLUMN IF NOT EXISTS justification TEXT;

CREATE TABLE IF NOT EXISTS attachments (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL REFERENCES users(id),
    request_id UUID REFERENCES requests(id) ON DELETE SET NULL,
    original_name VARCHAR(512) NOT NULL,
    content_type VARCHAR(255) NOT NULL DEFAULT 'application/octet-stream',
    s3_key TEXT NOT NULL UNIQUE,
    size_bytes BIGINT NOT NULL DEFAULT 0,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_attachments_user ON attachments(user_id);
CREATE INDEX IF NOT EXISTS idx_attachments_request ON attachments(request_id);
