package httpserver

import (
	"context"
	"net/http"

	"github.com/google/uuid"
	"github.com/itshop/api/internal/workflow"
)

// getAdminOverview — сводка для панели администрирования (ТЗ 5.9): только rbac admin.
func (s *Server) getAdminOverview(w http.ResponseWriter, r *http.Request, uid uuid.UUID) {
	ctx := r.Context()
	if !s.hasRbac(ctx, uid, "admin") {
		writeAPIError(w, http.StatusForbidden, "FORBIDDEN", "требуется роль admin")
		return
	}

	var nSys, nRes, nAR, nUsers, nRequests, nAudit int64
	_ = s.pool.QueryRow(ctx, `SELECT COUNT(*)::bigint FROM systems`).Scan(&nSys)
	_ = s.pool.QueryRow(ctx, `SELECT COUNT(*)::bigint FROM resources`).Scan(&nRes)
	_ = s.pool.QueryRow(ctx, `SELECT COUNT(*)::bigint FROM access_roles`).Scan(&nAR)
	_ = s.pool.QueryRow(ctx, `SELECT COUNT(*)::bigint FROM users`).Scan(&nUsers)
	_ = s.pool.QueryRow(ctx, `SELECT COUNT(*)::bigint FROM requests`).Scan(&nRequests)
	_ = s.pool.QueryRow(ctx, `SELECT COUNT(*)::bigint FROM audit_logs`).Scan(&nAudit)

	chain := workflow.DefaultApproverChain()
	steps := make([]map[string]any, 0, len(chain))
	for i, appr := range chain {
		row := map[string]any{
			"step":    i + 1,
			"user_id": formatUUID(appr),
		}
		u, err := s.q.GetUserByID(ctx, appr)
		if err == nil {
			row["login"] = u.Login
			row["full_name"] = u.FullName
		} else {
			row["login"] = nil
			row["full_name"] = nil
		}
		steps = append(steps, row)
	}

	users, err := s.adminListUsersWithRoles(ctx)
	if err != nil {
		writeAPIError(w, http.StatusInternalServerError, "INTERNAL", "ошибка БД")
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"catalog": map[string]any{
			"systems_count":       nSys,
			"resources_count":   nRes,
			"access_roles_count":  nAR,
		},
		"metrics": map[string]any{
			"users_count":        nUsers,
			"requests_count":     nRequests,
			"audit_entries_count": nAudit,
		},
		"workflow": map[string]any{
			"description": "Маршрут заявки строится из цепочек ролей доступа (таблица access_role_approvers); если у роли нет записей — используется workflow.DefaultApproverChain. Ниже — демо-цепочка по умолчанию.",
			"steps":       steps,
		},
		"users": users,
	})
}

func (s *Server) adminListUsersWithRoles(ctx context.Context) ([]map[string]any, error) {
	rows, err := s.pool.Query(ctx, `
		SELECT u.id::text, u.login, u.full_name, u.email,
			COALESCE(string_agg(r.name, ', ' ORDER BY r.name), '') AS roles
		FROM users u
		LEFT JOIN user_roles ur ON ur.user_id = u.id
		LEFT JOIN rbac_roles r ON r.id = ur.rbac_role_id
		GROUP BY u.id, u.login, u.full_name, u.email
		ORDER BY u.login`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := make([]map[string]any, 0, 16)
	for rows.Next() {
		var id, login, fn, email, roles string
		if err := rows.Scan(&id, &login, &fn, &email, &roles); err != nil {
			return nil, err
		}
		out = append(out, map[string]any{
			"id":        id,
			"login":     login,
			"full_name": fn,
			"email":     email,
			"roles":     roles,
		})
	}
	return out, rows.Err()
}
