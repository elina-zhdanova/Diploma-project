-- name: ListSystems :many
SELECT id, name, description
FROM systems
ORDER BY name;

-- name: ListResourcesBySystem :many
SELECT id, system_id, name, description, owner_id, sensitive
FROM resources
WHERE system_id = $1
ORDER BY name;

-- name: ListAccessRolesByResource :many
SELECT id, resource_id, name, description, risk_level
FROM access_roles
WHERE resource_id = $1
ORDER BY name;

-- name: GetAccessRoleByID :one
SELECT id, resource_id, name, description, risk_level
FROM access_roles
WHERE id = $1
LIMIT 1;

-- name: GetAccessRoleDetail :one
SELECT
    ar.id AS access_role_id,
    ar.resource_id,
    ar.name AS role_name,
    ar.description AS role_description,
    ar.risk_level,
    r.name AS resource_name,
    r.sensitive AS resource_sensitive,
    s.id AS system_id,
    s.name AS system_name
FROM access_roles ar
INNER JOIN resources r ON r.id = ar.resource_id
INNER JOIN systems s ON s.id = r.system_id
WHERE ar.id = $1
LIMIT 1;

-- name: SearchAccessRoles :many
SELECT
    ar.id AS access_role_id,
    ar.resource_id,
    ar.name AS role_name,
    ar.description AS role_description,
    ar.risk_level,
    r.name AS resource_name,
    r.sensitive AS resource_sensitive,
    s.id AS system_id,
    s.name AS system_name
FROM access_roles ar
INNER JOIN resources r ON r.id = ar.resource_id
INNER JOIN systems s ON s.id = r.system_id
WHERE
    (sqlc.arg(search_q)::text = '' OR ar.name ILIKE '%' || sqlc.arg(search_q) || '%' OR COALESCE(ar.description, '') ILIKE '%' || sqlc.arg(search_q) || '%')
    AND (sqlc.narg(filter_system_id)::uuid IS NULL OR s.id = sqlc.narg(filter_system_id)::uuid)
    AND (sqlc.narg(filter_resource_id)::uuid IS NULL OR r.id = sqlc.narg(filter_resource_id)::uuid)
ORDER BY LOWER(ar.name)
LIMIT 5000;

-- name: ListCatalogFlat :many
SELECT
    s.id AS system_id,
    s.name AS system_name,
    s.description AS system_description,
    r.id AS resource_id,
    r.name AS resource_name,
    r.sensitive AS resource_sensitive,
    ar.id AS access_role_id,
    ar.name AS role_name,
    ar.description AS role_description,
    ar.risk_level
FROM systems s
INNER JOIN resources r ON r.system_id = s.id
INNER JOIN access_roles ar ON ar.resource_id = r.id
ORDER BY LOWER(s.name), LOWER(r.name), LOWER(ar.name);
