package httpserver

import (
	"encoding/json"
	"errors"
	"net/http"
	"strings"

	"github.com/google/uuid"
	"github.com/itshop/api/internal/pgxutil"
	"github.com/itshop/api/internal/store"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
)

// getAdminAccessRoleForm — ресурсы каталога и справочники для создания роли (согласующих подбирают через поиск по логину).
func (s *Server) getAdminAccessRoleForm(w http.ResponseWriter, r *http.Request, uid uuid.UUID) {
	if r.Method != http.MethodGet {
		writeAPIError(w, http.StatusMethodNotAllowed, "METHOD_NOT_ALLOWED", "ожидался GET")
		return
	}
	ctx := r.Context()
	if !s.hasRbac(ctx, uid, "admin") {
		writeAPIError(w, http.StatusForbidden, "FORBIDDEN", "требуется роль admin")
		return
	}
	rows, err := s.pool.Query(ctx, `
		SELECT r.id::text, s.name, r.name
		FROM resources r
		INNER JOIN systems s ON s.id = r.system_id
		ORDER BY s.name, r.name`)
	if err != nil {
		writeAPIError(w, http.StatusInternalServerError, "INTERNAL", "ошибка БД")
		return
	}
	defer rows.Close()
	resOut := make([]map[string]any, 0, 64)
	for rows.Next() {
		var id, sys, rn string
		if err := rows.Scan(&id, &sys, &rn); err != nil {
			writeAPIError(w, http.StatusInternalServerError, "INTERNAL", "ошибка БД")
			return
		}
		resOut = append(resOut, map[string]any{
			"id":           id,
			"system_name":  sys,
			"resource_name": rn,
			"label":        sys + " / " + rn,
		})
	}
	if err := rows.Err(); err != nil {
		writeAPIError(w, http.StatusInternalServerError, "INTERNAL", "ошибка БД")
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"resources":   resOut,
		"approvers":   []map[string]any{},
		"risk_levels": []string{"low", "medium", "high"},
	})
}

// adminSearchUsers — поиск пользователей по логину (для цепочки согласования при создании роли доступа).
func (s *Server) adminSearchUsers(w http.ResponseWriter, r *http.Request, uid uuid.UUID) {
	if r.Method != http.MethodGet {
		writeAPIError(w, http.StatusMethodNotAllowed, "METHOD_NOT_ALLOWED", "ожидался GET")
		return
	}
	ctx := r.Context()
	if !s.hasRbac(ctx, uid, "admin") {
		writeAPIError(w, http.StatusForbidden, "FORBIDDEN", "требуется роль admin")
		return
	}
	q := strings.TrimSpace(r.URL.Query().Get("q"))
	if len([]rune(q)) < 2 {
		writeJSON(w, http.StatusOK, map[string]any{"items": []map[string]any{}})
		return
	}
	pattern := "%" + q + "%"
	rows, err := s.pool.Query(ctx, `
		SELECT u.id::text, u.login, u.full_name
		FROM users u
		WHERE u.login ILIKE $1
		ORDER BY u.login
		LIMIT 40`, pattern)
	if err != nil {
		writeAPIError(w, http.StatusInternalServerError, "INTERNAL", "ошибка БД")
		return
	}
	defer rows.Close()
	out := make([]map[string]any, 0, 16)
	for rows.Next() {
		var id, login, fn string
		if err := rows.Scan(&id, &login, &fn); err != nil {
			writeAPIError(w, http.StatusInternalServerError, "INTERNAL", "ошибка БД")
			return
		}
		out = append(out, map[string]any{"id": id, "login": login, "full_name": fn})
	}
	if err := rows.Err(); err != nil {
		writeAPIError(w, http.StatusInternalServerError, "INTERNAL", "ошибка БД")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"items": out})
}

type createAdminAccessRoleBody struct {
	ResourceID      string   `json:"resource_id"`
	Name            string   `json:"name"`
	Description     string   `json:"description"`
	RiskLevel       string   `json:"risk_level"`
	ApproverUserIDs []string `json:"approver_user_ids"`
}

func (s *Server) createAdminAccessRole(w http.ResponseWriter, r *http.Request, uid uuid.UUID) {
	if r.Method != http.MethodPost {
		writeAPIError(w, http.StatusMethodNotAllowed, "METHOD_NOT_ALLOWED", "ожидался POST")
		return
	}
	ctx := r.Context()
	if !s.hasRbac(ctx, uid, "admin") {
		writeAPIError(w, http.StatusForbidden, "FORBIDDEN", "требуется роль admin")
		return
	}
	var body createAdminAccessRoleBody
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeAPIError(w, http.StatusBadRequest, "VALIDATION_ERROR", "некорректный JSON")
		return
	}
	name := strings.TrimSpace(body.Name)
	if name == "" || body.ResourceID == "" {
		writeAPIError(w, http.StatusBadRequest, "VALIDATION_ERROR", "нужны resource_id и name")
		return
	}
	rl := strings.ToLower(strings.TrimSpace(body.RiskLevel))
	switch rl {
	case "low", "medium", "high":
	default:
		writeAPIError(w, http.StatusBadRequest, "VALIDATION_ERROR", "risk_level: low, medium или high")
		return
	}
	if len(body.ApproverUserIDs) == 0 {
		writeAPIError(w, http.StatusBadRequest, "VALIDATION_ERROR", "нужен хотя бы один согласующий (approver_user_ids)")
		return
	}
	resID, err := pgxutil.ParseUUID(body.ResourceID)
	if err != nil {
		writeAPIError(w, http.StatusBadRequest, "VALIDATION_ERROR", "некорректный resource_id")
		return
	}
	var one int
	if err := s.pool.QueryRow(ctx, `SELECT 1 FROM resources WHERE id = $1`, resID).Scan(&one); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			writeAPIError(w, http.StatusBadRequest, "VALIDATION_ERROR", "ресурс не найден")
			return
		}
		writeAPIError(w, http.StatusInternalServerError, "INTERNAL", "ошибка БД")
		return
	}

	seen := make(map[uuid.UUID]struct{})
	approvers := make([]pgtype.UUID, 0, len(body.ApproverUserIDs))
	for _, idStr := range body.ApproverUserIDs {
		idStr = strings.TrimSpace(idStr)
		if idStr == "" {
			continue
		}
		aid, err := pgxutil.ParseUUID(idStr)
		if err != nil {
			writeAPIError(w, http.StatusBadRequest, "VALIDATION_ERROR", "некорректный approver id")
			return
		}
		gid, ok := pgxutil.ToGoogle(aid)
		if !ok {
			continue
		}
		if _, dup := seen[gid]; dup {
			writeAPIError(w, http.StatusBadRequest, "VALIDATION_ERROR", "повтор согласующего в цепочке")
			return
		}
		seen[gid] = struct{}{}
		if _, err := s.q.GetUserByID(ctx, aid); err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				writeAPIError(w, http.StatusBadRequest, "VALIDATION_ERROR", "пользователь не найден")
				return
			}
			writeAPIError(w, http.StatusInternalServerError, "INTERNAL", "ошибка БД")
			return
		}
		approvers = append(approvers, aid)
	}
	if len(approvers) == 0 {
		writeAPIError(w, http.StatusBadRequest, "VALIDATION_ERROR", "нужен хотя бы один корректный согласующий")
		return
	}

	tx, err := s.pool.Begin(ctx)
	if err != nil {
		writeAPIError(w, http.StatusInternalServerError, "INTERNAL", "не удалось начать транзакцию")
		return
	}
	defer tx.Rollback(ctx)
	qx := s.q.WithTx(tx)

	arID := uuid.New()
	desc := pgtype.Text{}
	if strings.TrimSpace(body.Description) != "" {
		desc = pgxutil.TextOptional(body.Description)
	}
	if err := qx.InsertAccessRoleRow(ctx, store.InsertAccessRoleRowParams{
		ID:          pgxutil.UUID(arID),
		ResourceID:  resID,
		Name:        name,
		Description: desc,
		RiskLevel:   rl,
	}); err != nil {
		writeAPIError(w, http.StatusInternalServerError, "INTERNAL", "не удалось создать роль доступа")
		return
	}
	arPg := pgxutil.UUID(arID)
	for i, appr := range approvers {
		if err := qx.InsertAccessRoleApproverRow(ctx, arPg, int32(i+1), appr); err != nil {
			writeAPIError(w, http.StatusInternalServerError, "INTERNAL", "не удалось назначить согласующих")
			return
		}
	}
	if err := tx.Commit(ctx); err != nil {
		writeAPIError(w, http.StatusInternalServerError, "INTERNAL", "не удалось зафиксировать")
		return
	}
	s.audit(ctx, uid, nil, "access_role.create", map[string]any{
		"access_role_id": arID.String(),
		"resource_id":    body.ResourceID,
		"name":           name,
		"steps":          len(approvers),
	})
	writeJSON(w, http.StatusCreated, map[string]any{
		"id":     arID.String(),
		"status": "created",
	})
}
