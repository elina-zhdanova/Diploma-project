-- name: InsertApproval :exec
INSERT INTO approvals (id, request_id, approver_id, step_number, status)
VALUES ($1, $2, $3, $4, $5);

-- name: ListApprovalsByRequest :many
SELECT id, request_id, approver_id, step_number, status, decision_at, comment
FROM approvals
WHERE request_id = $1
ORDER BY step_number;

-- name: ListApprovalsByRequestWithUsers :many
SELECT
    a.id,
    a.request_id,
    a.approver_id,
    a.step_number,
    a.status,
    a.decision_at,
    a.comment,
    COALESCE(u.full_name, '')::text AS approver_full_name,
    u.login AS approver_login
FROM approvals a
LEFT JOIN users u ON u.id = a.approver_id
WHERE a.request_id = $1
ORDER BY a.step_number;

-- name: GetPendingApprovalForRequestByApprover :one
SELECT a.id, a.request_id, a.approver_id, a.step_number, a.status, a.decision_at, a.comment
FROM approvals a
WHERE a.request_id = $1
  AND a.status = 'pending'
  AND NOT EXISTS (
    SELECT 1
    FROM approvals p
    WHERE p.request_id = a.request_id
      AND p.status = 'pending'
      AND p.step_number < a.step_number
  )
  AND (
    a.approver_id = $2
    OR EXISTS (
      SELECT 1
      FROM delegations d
      WHERE d.from_user_id = a.approver_id
        AND d.to_user_id = $2
        AND (
          d.permanent
          OR (
            d.start_date IS NOT NULL
            AND d.end_date IS NOT NULL
            AND CURRENT_DATE BETWEEN d.start_date AND d.end_date
          )
        )
        AND (
          NOT EXISTS (SELECT 1 FROM delegation_scopes ds0 WHERE ds0.delegation_id = d.id)
          OR EXISTS (
            SELECT 1
            FROM request_items ri
            INNER JOIN delegation_scopes ds ON ds.delegation_id = d.id AND ds.access_role_id = ri.access_role_id
            WHERE ri.request_id = a.request_id
          )
        )
    )
  )
ORDER BY a.step_number ASC, a.id ASC
LIMIT 1;

-- name: UpdateApprovalByID :exec
UPDATE approvals
SET status = $2,
    decision_at = NOW(),
    comment = $3
WHERE id = $1
  AND status = 'pending';

-- name: CancelAllPendingApprovalsExcept :exec
UPDATE approvals
SET status = 'cancelled',
    decision_at = COALESCE(decision_at, NOW())
WHERE request_id = $1
  AND status = 'pending'
  AND id <> $2;

-- name: CancelAllPendingApprovalsForRequest :exec
UPDATE approvals
SET status = 'cancelled',
    decision_at = NOW()
WHERE request_id = $1
  AND status = 'pending';

-- name: DelegateApproval :exec
UPDATE approvals AS ap
SET approver_id = $3,
    comment = COALESCE($4, ap.comment)
WHERE ap.id = $1
  AND ap.status = 'pending'
  AND (
    ap.approver_id = $2
    OR EXISTS (
      SELECT 1
      FROM delegations d
      WHERE d.from_user_id = ap.approver_id
        AND d.to_user_id = $2
        AND (
          d.permanent
          OR (
            d.start_date IS NOT NULL
            AND d.end_date IS NOT NULL
            AND CURRENT_DATE BETWEEN d.start_date AND d.end_date
          )
        )
        AND (
          NOT EXISTS (SELECT 1 FROM delegation_scopes ds0 WHERE ds0.delegation_id = d.id)
          OR EXISTS (
            SELECT 1
            FROM request_items ri
            INNER JOIN delegation_scopes ds ON ds.delegation_id = d.id AND ds.access_role_id = ri.access_role_id
            WHERE ri.request_id = ap.request_id
          )
        )
    )
  );

-- name: CountPendingApprovalsForRequest :one
SELECT COUNT(*)::bigint
FROM approvals
WHERE request_id = $1
  AND status = 'pending';

-- name: InboxForApprover :many
SELECT sub.id, sub.initiator_id, sub.status, sub.risk_score, sub.created_at, sub.updated_at, sub.justification,
       sub.needed_until, sub.step_number, sub.access_role_names,
       sub.initiator_full_name, sub.initiator_login
FROM (
  SELECT DISTINCT ON (r.id)
    r.id, r.initiator_id, r.status, r.risk_score, r.created_at, r.updated_at, r.justification,
    r.needed_until, a.step_number,
    COALESCE((
      SELECT string_agg(subq.role_name, ', ' ORDER BY subq.role_name)
      FROM (
        SELECT DISTINCT ar2.name AS role_name
        FROM request_items ri2
        INNER JOIN access_roles ar2 ON ar2.id = ri2.access_role_id
        WHERE ri2.request_id = r.id
      ) AS subq
    ), '')::text AS access_role_names,
    COALESCE(iu.full_name, '')::text AS initiator_full_name,
    COALESCE(iu.login, '')::text AS initiator_login
  FROM requests r
  INNER JOIN users iu ON iu.id = r.initiator_id
  INNER JOIN approvals a ON a.request_id = r.id
  WHERE a.status = 'pending'
    AND r.status = 'pending'
    AND NOT EXISTS (
      SELECT 1
      FROM approvals p
      WHERE p.request_id = r.id
        AND p.status = 'pending'
        AND p.step_number < a.step_number
    )
    AND (
      a.approver_id = $1
      OR EXISTS (
        SELECT 1
        FROM delegations d
        WHERE d.from_user_id = a.approver_id
          AND d.to_user_id = $1
          AND (
            d.permanent
            OR (
              d.start_date IS NOT NULL
              AND d.end_date IS NOT NULL
              AND CURRENT_DATE BETWEEN d.start_date AND d.end_date
            )
          )
          AND (
            NOT EXISTS (SELECT 1 FROM delegation_scopes ds0 WHERE ds0.delegation_id = d.id)
            OR EXISTS (
              SELECT 1
              FROM request_items ri
              INNER JOIN delegation_scopes ds ON ds.delegation_id = d.id AND ds.access_role_id = ri.access_role_id
              WHERE ri.request_id = r.id
            )
          )
      )
    )
  ORDER BY r.id, a.step_number, a.id
) sub
ORDER BY sub.created_at DESC;
