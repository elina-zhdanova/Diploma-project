-- name: InsertAuditLog :exec
INSERT INTO audit_logs (id, user_id, request_id, action, details)
VALUES ($1, $2, $3, $4, $5);

-- name: ListAuditLogs :many
-- search: пустая строка = без фильтра по тексту (ищем в action и в details::text)
-- filter_user_id / filter_request_id: нулевой uuid = без фильтра (передавайте '00000000-0000-0000-0000-000000000000')
SELECT a.id, a.user_id, a.request_id, a.action, a.details, a.created_at,
  COALESCE(u.login, '') AS user_login
FROM audit_logs a
LEFT JOIN users u ON u.id = a.user_id
WHERE
  ($3 = '' OR a.action ILIKE '%' || $3 || '%' OR COALESCE(a.details::text, '') ILIKE '%' || $3 || '%')
  AND ($4 = '00000000-0000-0000-0000-000000000000'::uuid OR a.user_id = $4)
  AND ($5 = '00000000-0000-0000-0000-000000000000'::uuid OR a.request_id = $5)
ORDER BY a.created_at DESC
LIMIT $1 OFFSET $2;
