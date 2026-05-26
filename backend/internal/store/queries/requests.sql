-- name: InsertRequest :one
INSERT INTO requests (initiator_id, status, justification, risk_score, needed_until)
VALUES ($1, $2, $3, $4, $5)
RETURNING id, initiator_id, status, risk_score, created_at, updated_at, justification, needed_until;

-- name: UpdateRequestStatus :exec
UPDATE requests
SET status = $2,
    updated_at = NOW()
WHERE id = $1;

-- name: UpdateRequestRisk :exec
UPDATE requests
SET risk_score = $2,
    updated_at = NOW()
WHERE id = $1;

-- name: ListRequestsByInitiator :many
SELECT
  r.id,
  r.initiator_id,
  r.status,
  r.risk_score,
  r.created_at,
  r.updated_at,
  r.justification,
  r.needed_until,
  COALESCE((
    SELECT string_agg(subq.role_name, ', ' ORDER BY subq.role_name)
    FROM (
      SELECT DISTINCT ar2.name AS role_name
      FROM request_items ri2
      INNER JOIN access_roles ar2 ON ar2.id = ri2.access_role_id
      WHERE ri2.request_id = r.id
    ) AS subq
  ), '')::text AS access_role_names,
  COALESCE((
    SELECT COUNT(*)::int FROM approvals ap WHERE ap.request_id = r.id
  ), 0) AS approvals_total,
  COALESCE((
    SELECT COUNT(*)::int FROM approvals ap WHERE ap.request_id = r.id AND ap.status = 'approved'
  ), 0) AS approvals_done,
  COALESCE(iu.full_name, '')::text AS initiator_full_name,
  COALESCE(iu.login, '')::text AS initiator_login
FROM requests r
INNER JOIN users iu ON iu.id = r.initiator_id
WHERE r.initiator_id = $1
ORDER BY r.created_at DESC;

-- name: GetRequestByID :one
SELECT id, initiator_id, status, risk_score, created_at, updated_at, justification, needed_until
FROM requests
WHERE id = $1
LIMIT 1;

-- name: InsertRequestItem :exec
INSERT INTO request_items (id, request_id, access_role_id, justification)
VALUES ($1, $2, $3, $4);

-- name: ListRequestItems :many
SELECT ri.id, ri.request_id, ri.access_role_id, ar.name AS access_role_name, ar.risk_level, r.name AS resource_name, ri.justification
FROM request_items ri
INNER JOIN access_roles ar ON ar.id = ri.access_role_id
INNER JOIN resources r ON r.id = ar.resource_id
WHERE ri.request_id = $1;
