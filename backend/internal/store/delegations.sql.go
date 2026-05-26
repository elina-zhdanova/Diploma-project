// Code generated manually — keep in sync with queries/delegations.sql

package store

import (
	"context"

	"github.com/jackc/pgx/v5/pgtype"
)

const insertDelegation = `-- name: InsertDelegation :exec
INSERT INTO delegations (id, from_user_id, to_user_id, start_date, end_date, permanent)
VALUES ($1, $2, $3, $4, $5, $6)
`

type InsertDelegationParams struct {
	ID         pgtype.UUID `json:"id"`
	FromUserID pgtype.UUID `json:"from_user_id"`
	ToUserID   pgtype.UUID `json:"to_user_id"`
	StartDate  pgtype.Date `json:"start_date"`
	EndDate    pgtype.Date `json:"end_date"`
	Permanent  bool        `json:"permanent"`
}

func (q *Queries) InsertDelegation(ctx context.Context, arg InsertDelegationParams) error {
	_, err := q.db.Exec(ctx, insertDelegation,
		arg.ID,
		arg.FromUserID,
		arg.ToUserID,
		arg.StartDate,
		arg.EndDate,
		arg.Permanent,
	)
	return err
}

const insertDelegationScope = `-- name: InsertDelegationScope :exec
INSERT INTO delegation_scopes (delegation_id, access_role_id)
VALUES ($1, $2)
`

type InsertDelegationScopeParams struct {
	DelegationID pgtype.UUID `json:"delegation_id"`
	AccessRoleID pgtype.UUID `json:"access_role_id"`
}

func (q *Queries) InsertDelegationScope(ctx context.Context, arg InsertDelegationScopeParams) error {
	_, err := q.db.Exec(ctx, insertDelegationScope, arg.DelegationID, arg.AccessRoleID)
	return err
}

const listDelegationsForUser = `-- name: ListDelegationsForUser :many
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
ORDER BY d.permanent DESC, d.start_date DESC NULLS LAST
`

func (q *Queries) ListDelegationsForUser(ctx context.Context, userID pgtype.UUID) ([]DelegationListRow, error) {
	rows, err := q.db.Query(ctx, listDelegationsForUser, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items := []DelegationListRow{}
	for rows.Next() {
		var i DelegationListRow
		if err := rows.Scan(
			&i.ID,
			&i.FromUserID,
			&i.ToUserID,
			&i.StartDate,
			&i.EndDate,
			&i.Permanent,
			&i.FromLogin,
			&i.FromFullName,
			&i.ToLogin,
			&i.ToFullName,
			&i.AccessRolesJSON,
		); err != nil {
			return nil, err
		}
		items = append(items, i)
	}
	return items, rows.Err()
}

const approverHasAccessRole = `-- name: ApproverHasAccessRole :one
SELECT EXISTS (
  SELECT 1 FROM access_role_approvers
  WHERE approver_id = $1 AND access_role_id = $2
)::bool
`

func (q *Queries) ApproverHasAccessRole(ctx context.Context, approverID pgtype.UUID, accessRoleID pgtype.UUID) (bool, error) {
	row := q.db.QueryRow(ctx, approverHasAccessRole, approverID, accessRoleID)
	var ok bool
	err := row.Scan(&ok)
	return ok, err
}

const listAccessRolesForApprover = `-- name: ListAccessRolesForApprover :many
SELECT DISTINCT ar.id, ar.name
FROM access_roles ar
INNER JOIN access_role_approvers ara ON ara.access_role_id = ar.id
WHERE ara.approver_id = $1
ORDER BY ar.name
`

func (q *Queries) ListAccessRolesForApprover(ctx context.Context, approverID pgtype.UUID) ([]ApproverAccessRoleRow, error) {
	rows, err := q.db.Query(ctx, listAccessRolesForApprover, approverID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items := []ApproverAccessRoleRow{}
	for rows.Next() {
		var i ApproverAccessRoleRow
		if err := rows.Scan(&i.ID, &i.Name); err != nil {
			return nil, err
		}
		items = append(items, i)
	}
	return items, rows.Err()
}

const deleteDelegation = `-- name: DeleteDelegation :exec
DELETE FROM delegations
WHERE id = $1 AND from_user_id = $2
`

type DeleteDelegationParams struct {
	ID         pgtype.UUID `json:"id"`
	FromUserID pgtype.UUID `json:"from_user_id"`
}

func (q *Queries) DeleteDelegation(ctx context.Context, arg DeleteDelegationParams) error {
	_, err := q.db.Exec(ctx, deleteDelegation, arg.ID, arg.FromUserID)
	return err
}
