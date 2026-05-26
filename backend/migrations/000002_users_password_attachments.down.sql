DROP TABLE IF EXISTS attachments;
ALTER TABLE requests DROP COLUMN IF EXISTS justification;
ALTER TABLE users DROP COLUMN IF EXISTS password_hash;
