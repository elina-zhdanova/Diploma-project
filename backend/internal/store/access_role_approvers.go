// Ручные запросы: цепочка согласующих по роли доступа (см. migrations/000008).

package store

import (
	"context"

	"github.com/jackc/pgx/v5/pgtype"
)

const listApproversOrderedForAccessRole = `
SELECT approver_id FROM access_role_approvers WHERE access_role_id = $1 ORDER BY step_number ASC
`

func (q *Queries) ListApproversOrderedForAccessRole(ctx context.Context, accessRoleID pgtype.UUID) ([]pgtype.UUID, error) {
	rows, err := q.db.Query(ctx, listApproversOrderedForAccessRole, accessRoleID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []pgtype.UUID
	for rows.Next() {
		var id pgtype.UUID
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		out = append(out, id)
	}
	return out, rows.Err()
}

const insertAccessRoleRow = `
INSERT INTO access_roles (id, resource_id, name, description, risk_level)
VALUES ($1, $2, $3, $4, $5)
`

const insertAccessRoleApproverRow = `
INSERT INTO access_role_approvers (access_role_id, step_number, approver_id)
VALUES ($1, $2, $3)
`

type InsertAccessRoleRowParams struct {
	ID          pgtype.UUID
	ResourceID  pgtype.UUID
	Name        string
	Description pgtype.Text
	RiskLevel   string
}

func (q *Queries) InsertAccessRoleRow(ctx context.Context, arg InsertAccessRoleRowParams) error {
	_, err := q.db.Exec(ctx, insertAccessRoleRow, arg.ID, arg.ResourceID, arg.Name, arg.Description, arg.RiskLevel)
	return err
}

func (q *Queries) InsertAccessRoleApproverRow(ctx context.Context, accessRoleID pgtype.UUID, stepNumber int32, approverID pgtype.UUID) error {
	_, err := q.db.Exec(ctx, insertAccessRoleApproverRow, accessRoleID, stepNumber, approverID)
	return err
}

// UserIsAccessRoleApprover — пользователь указан как согласующий хотя бы в одной цепочке роли доступа (роль → ресурс → ИС).
const userIsAccessRoleApprover = `
SELECT EXISTS (
	SELECT 1 FROM access_role_approvers WHERE approver_id = $1
)`

func (q *Queries) UserIsAccessRoleApprover(ctx context.Context, approverID pgtype.UUID) (bool, error) {
	var ok bool
	err := q.db.QueryRow(ctx, userIsAccessRoleApprover, approverID).Scan(&ok)
	return ok, err
}
