-- name: InsertAttachment :one
INSERT INTO attachments (user_id, request_id, original_name, content_type, s3_key, size_bytes)
VALUES ($1, $2, $3, $4, $5, $6)
RETURNING id, user_id, request_id, original_name, content_type, s3_key, size_bytes, created_at;

-- name: GetAttachmentByID :one
SELECT id, user_id, request_id, original_name, content_type, s3_key, size_bytes, created_at
FROM attachments
WHERE id = $1
LIMIT 1;

-- name: ListAttachmentsByRequest :many
SELECT id, user_id, request_id, original_name, content_type, s3_key, size_bytes, created_at
FROM attachments
WHERE request_id = $1
ORDER BY created_at ASC;
