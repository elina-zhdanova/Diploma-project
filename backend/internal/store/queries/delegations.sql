-- name: InsertDelegation :exec
INSERT INTO delegations (id, from_user_id, to_user_id, start_date, end_date, permanent)
VALUES ($1, $2, $3, $4, $5, $6);

-- name: InsertDelegationScope :exec
INSERT INTO delegation_scopes (delegation_id, access_role_id)
VALUES ($1, $2);

-- name: ListDelegationsForUser :many
SELECT d.id, d.from_user_id, d.to_user_id, d.start_date, d.end_date, d.permanent,
  uf.login, uf.full_name, ut.login, ut.full_name,
  COALESCE(
    (
      SELECT json_agg(json_build_object('id', ar.id::text, 'name', ar.name) ORDER BY ar.name)
      FROM delegation_scopes ds
      INNER JOIN access_roles ar ON ar.id = ds.access_role_id
      WHERE ds.delegation_id = d.id
    ),
    '[]'::json
  )::text AS access_roles_json
FROM delegations d
INNER JOIN users uf ON uf.id = d.from_user_id
INNER JOIN users ut ON ut.id = d.to_user_id
WHERE d.from_user_id = $1 OR d.to_user_id = $1
ORDER BY d.permanent DESC, d.start_date DESC NULLS LAST;

-- name: ApproverHasAccessRole :one
SELECT EXISTS (
  SELECT 1 FROM access_role_approvers
  WHERE approver_id = $1 AND access_role_id = $2
)::bool;

-- name: ListAccessRolesForApprover :many
SELECT DISTINCT ar.id, ar.name
FROM access_roles ar
INNER JOIN access_role_approvers ara ON ara.access_role_id = ar.id
WHERE ara.approver_id = $1
ORDER BY ar.name;

-- name: DeleteDelegation :exec
DELETE FROM delegations
WHERE id = $1 AND from_user_id = $2;
