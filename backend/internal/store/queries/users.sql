-- name: GetUserByLogin :one
SELECT id, login, full_name, email, department_id, position, is_active, created_at, password_hash
FROM users
WHERE login = $1
LIMIT 1;

-- name: GetUserByID :one
SELECT id, login, full_name, email, department_id, position, is_active, created_at, password_hash
FROM users
WHERE id = $1
LIMIT 1;

-- name: ListRbacNamesForUser :many
SELECT r.name
FROM rbac_roles r
INNER JOIN user_roles ur ON ur.rbac_role_id = r.id
WHERE ur.user_id = $1
ORDER BY r.name;

-- name: GetUserByIDForIAM :one
SELECT u.id, u.login, u.full_name, u.email, u.position, u.is_active, d.name AS department_name
FROM users u
LEFT JOIN departments d ON d.id = u.department_id
WHERE u.id = $1
LIMIT 1;
