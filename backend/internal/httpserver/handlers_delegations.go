package httpserver

import (
	"encoding/json"
	"errors"
	"net/http"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/itshop/api/internal/pgxutil"
	"github.com/itshop/api/internal/store"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
)

type createDelegationBody struct {
	ToUserID      string   `json:"to_user_id"`
	Permanent     bool     `json:"permanent"`
	StartDate     string   `json:"start_date"`
	EndDate       string   `json:"end_date"`
	AccessRoleIDs []string `json:"access_role_ids"`
}

func parsePgDateYYYYMMDD(s string) (pgtype.Date, error) {
	t, err := time.Parse("2006-01-02", s)
	if err != nil {
		return pgtype.Date{}, err
	}
	return pgtype.Date{Time: t, Valid: true}, nil
}

func parseAccessRolesJSONPayload(s string) []map[string]any {
	if s == "" || s == "[]" {
		return []map[string]any{}
	}
	var raw []struct {
		ID   string `json:"id"`
		Name string `json:"name"`
	}
	if err := json.Unmarshal([]byte(s), &raw); err != nil {
		return []map[string]any{}
	}
	out := make([]map[string]any, len(raw))
	for i, x := range raw {
		out[i] = map[string]any{"id": x.ID, "name": x.Name}
	}
	return out
}

func (s *Server) requireAccessRoleApprover(w http.ResponseWriter, r *http.Request, uid uuid.UUID) bool {
	ok, err := s.q.UserIsAccessRoleApprover(r.Context(), pgxutil.UUID(uid))
	if err != nil {
		writeAPIError(w, http.StatusInternalServerError, "INTERNAL", "ошибка БД")
		return false
	}
	if !ok {
		writeAPIError(w, http.StatusForbidden, "FORBIDDEN", "делегирование доступно только согласующим, назначенным в цепочках ролей доступа")
		return false
	}
	return true
}

// listDelegations — переданные и полученные полномочия (ТЗ 5.5).
func (s *Server) listDelegations(w http.ResponseWriter, r *http.Request, uid uuid.UUID) {
	if r.Method != http.MethodGet {
		writeAPIError(w, http.StatusMethodNotAllowed, "METHOD_NOT_ALLOWED", "ожидался GET")
		return
	}
	if !s.requireAccessRoleApprover(w, r, uid) {
		return
	}
	ctx := r.Context()
	rows, err := s.q.ListDelegationsForUser(ctx, pgxutil.UUID(uid))
	if err != nil {
		writeAPIError(w, http.StatusInternalServerError, "INTERNAL", "ошибка БД")
		return
	}
	now := time.Now()
	today := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.Local)
	out := make([]map[string]any, 0, len(rows))
	for _, d := range rows {
		var active bool
		if d.Permanent {
			active = true
		} else if d.StartDate.Valid && d.EndDate.Valid {
			sd := time.Date(d.StartDate.Time.Year(), d.StartDate.Time.Month(), d.StartDate.Time.Day(), 0, 0, 0, 0, time.Local)
			ed := time.Date(d.EndDate.Time.Year(), d.EndDate.Time.Month(), d.EndDate.Time.Day(), 0, 0, 0, 0, time.Local)
			active = !today.Before(sd) && !today.After(ed)
		}
		row := map[string]any{
			"id":             formatUUID(d.ID),
			"from_user_id":   formatUUID(d.FromUserID),
			"to_user_id":     formatUUID(d.ToUserID),
			"from_login":     d.FromLogin,
			"from_full_name": d.FromFullName,
			"to_login":       d.ToLogin,
			"to_full_name":   d.ToFullName,
			"permanent":      d.Permanent,
			"is_active":      active,
			"access_roles":   parseAccessRolesJSONPayload(d.AccessRolesJSON),
		}
		if d.StartDate.Valid {
			row["start_date"] = d.StartDate.Time.Format("2006-01-02")
		} else {
			row["start_date"] = nil
		}
		if d.EndDate.Valid {
			row["end_date"] = d.EndDate.Time.Format("2006-01-02")
		} else {
			row["end_date"] = nil
		}
		out = append(out, row)
	}
	writeJSON(w, http.StatusOK, map[string]any{"items": out})
}

func (s *Server) createDelegation(w http.ResponseWriter, r *http.Request, uid uuid.UUID) {
	if r.Method != http.MethodPost {
		writeAPIError(w, http.StatusMethodNotAllowed, "METHOD_NOT_ALLOWED", "ожидался POST")
		return
	}
	var body createDelegationBody
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil || body.ToUserID == "" {
		writeAPIError(w, http.StatusBadRequest, "VALIDATION_ERROR", "нужен to_user_id")
		return
	}
	if !body.Permanent && (body.StartDate == "" || body.EndDate == "") {
		writeAPIError(w, http.StatusBadRequest, "VALIDATION_ERROR", "для делегирования на период укажите start_date и end_date (YYYY-MM-DD) или включите permanent")
		return
	}
	toID, err := pgxutil.ParseUUID(body.ToUserID)
	if err != nil {
		writeAPIError(w, http.StatusBadRequest, "VALIDATION_ERROR", "некорректный to_user_id")
		return
	}
	gid, _ := pgxutil.ToGoogle(toID)
	if gid == uid {
		writeAPIError(w, http.StatusBadRequest, "VALIDATION_ERROR", "нельзя делегировать самому себе")
		return
	}
	var sd, ed pgtype.Date
	if !body.Permanent {
		sd, err = parsePgDateYYYYMMDD(body.StartDate)
		if err != nil {
			writeAPIError(w, http.StatusBadRequest, "VALIDATION_ERROR", "некорректный start_date")
			return
		}
		ed, err = parsePgDateYYYYMMDD(body.EndDate)
		if err != nil {
			writeAPIError(w, http.StatusBadRequest, "VALIDATION_ERROR", "некорректный end_date")
			return
		}
		if ed.Time.Before(sd.Time) {
			writeAPIError(w, http.StatusBadRequest, "VALIDATION_ERROR", "end_date раньше start_date")
			return
		}
	}
	if !s.requireAccessRoleApprover(w, r, uid) {
		return
	}
	ctx := r.Context()
	if _, err := s.q.GetUserByID(ctx, toID); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			writeAPIError(w, http.StatusBadRequest, "VALIDATION_ERROR", "получатель не найден")
			return
		}
		writeAPIError(w, http.StatusInternalServerError, "INTERNAL", "ошибка БД")
		return
	}

	seen := make(map[string]struct{})
	var roleUUIDs []pgtype.UUID
	for _, rid := range body.AccessRoleIDs {
		rid = strings.TrimSpace(rid)
		if rid == "" {
			continue
		}
		if _, dup := seen[rid]; dup {
			continue
		}
		seen[rid] = struct{}{}
		arid, err := pgxutil.ParseUUID(rid)
		if err != nil {
			writeAPIError(w, http.StatusBadRequest, "VALIDATION_ERROR", "некорректный access_role_ids")
			return
		}
		ok, err := s.q.ApproverHasAccessRole(ctx, pgxutil.UUID(uid), arid)
		if err != nil {
			writeAPIError(w, http.StatusInternalServerError, "INTERNAL", "ошибка БД")
			return
		}
		if !ok {
			writeAPIError(w, http.StatusBadRequest, "VALIDATION_ERROR", "можно передавать только роли, где вы указаны согласующим")
			return
		}
		roleUUIDs = append(roleUUIDs, arid)
	}

	did := pgxutil.UUID(uuid.New())
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		writeAPIError(w, http.StatusInternalServerError, "INTERNAL", "ошибка БД")
		return
	}
	defer tx.Rollback(ctx)
	qtx := store.New(tx)
	if err := qtx.InsertDelegation(ctx, store.InsertDelegationParams{
		ID:         did,
		FromUserID: pgxutil.UUID(uid),
		ToUserID:   toID,
		StartDate:  sd,
		EndDate:    ed,
		Permanent:  body.Permanent,
	}); err != nil {
		writeAPIError(w, http.StatusInternalServerError, "INTERNAL", "ошибка БД")
		return
	}
	for _, arid := range roleUUIDs {
		if err := qtx.InsertDelegationScope(ctx, store.InsertDelegationScopeParams{
			DelegationID: did,
			AccessRoleID: arid,
		}); err != nil {
			writeAPIError(w, http.StatusInternalServerError, "INTERNAL", "ошибка БД")
			return
		}
	}
	if err := tx.Commit(ctx); err != nil {
		writeAPIError(w, http.StatusInternalServerError, "INTERNAL", "ошибка БД")
		return
	}

	s.audit(ctx, uid, nil, "delegation.create", map[string]any{
		"to_user_id":       body.ToUserID,
		"permanent":        body.Permanent,
		"start_date":       body.StartDate,
		"end_date":         body.EndDate,
		"access_role_ids":  body.AccessRoleIDs,
	})
	writeJSON(w, http.StatusCreated, map[string]any{"status": "ok"})
}

func (s *Server) deleteDelegation(w http.ResponseWriter, r *http.Request, uid uuid.UUID) {
	if r.Method != http.MethodDelete {
		writeAPIError(w, http.StatusMethodNotAllowed, "METHOD_NOT_ALLOWED", "ожидался DELETE")
		return
	}
	idStr := r.PathValue("id")
	did, err := pgxutil.ParseUUID(idStr)
	if err != nil {
		writeAPIError(w, http.StatusBadRequest, "VALIDATION_ERROR", "некорректный id")
		return
	}
	if !s.requireAccessRoleApprover(w, r, uid) {
		return
	}
	ctx := r.Context()
	if err := s.q.DeleteDelegation(ctx, store.DeleteDelegationParams{
		ID:         did,
		FromUserID: pgxutil.UUID(uid),
	}); err != nil {
		writeAPIError(w, http.StatusInternalServerError, "INTERNAL", "ошибка БД")
		return
	}
	s.audit(ctx, uid, nil, "delegation.delete", map[string]any{"delegation_id": idStr})
	writeJSON(w, http.StatusOK, map[string]any{"status": "ok"})
}

// delegationCandidates — пользователи, которым можно передать полномочия (роли согласования).
func (s *Server) delegationCandidates(w http.ResponseWriter, r *http.Request, uid uuid.UUID) {
	if r.Method != http.MethodGet {
		writeAPIError(w, http.StatusMethodNotAllowed, "METHOD_NOT_ALLOWED", "ожидался GET")
		return
	}
	if !s.requireAccessRoleApprover(w, r, uid) {
		return
	}
	ctx := r.Context()
	rows, err := s.pool.Query(ctx, `
		SELECT u.id, u.login, u.full_name
		FROM users u
		WHERE u.id <> $1
		  AND EXISTS (
		    SELECT 1 FROM access_role_approvers a WHERE a.approver_id = u.id
		  )
		ORDER BY u.login`, uid)
	if err != nil {
		writeAPIError(w, http.StatusInternalServerError, "INTERNAL", "ошибка БД")
		return
	}
	defer rows.Close()
	out := make([]map[string]any, 0, 8)
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

// delegationSearchUsers — поиск любого пользователя для поля «Кому передать» (логин или ФИО, кроме себя).
func (s *Server) delegationSearchUsers(w http.ResponseWriter, r *http.Request, uid uuid.UUID) {
	if r.Method != http.MethodGet {
		writeAPIError(w, http.StatusMethodNotAllowed, "METHOD_NOT_ALLOWED", "ожидался GET")
		return
	}
	if !s.requireAccessRoleApprover(w, r, uid) {
		return
	}
	ctx := r.Context()
	q := strings.TrimSpace(r.URL.Query().Get("q"))
	if len([]rune(q)) < 2 {
		writeJSON(w, http.StatusOK, map[string]any{"items": []map[string]any{}})
		return
	}
	pattern := "%" + q + "%"
	rows, err := s.pool.Query(ctx, `
		SELECT u.id::text, u.login, u.full_name
		FROM users u
		WHERE u.id <> $1
		  AND (u.login ILIKE $2 OR COALESCE(TRIM(u.full_name), '') ILIKE $2)
		ORDER BY u.login
		LIMIT 40`, uid, pattern)
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

// delegationMyAccessRoles — роли доступа из каталога, где текущий пользователь — согласующий (для выбора области делегирования).
func (s *Server) delegationMyAccessRoles(w http.ResponseWriter, r *http.Request, uid uuid.UUID) {
	if r.Method != http.MethodGet {
		writeAPIError(w, http.StatusMethodNotAllowed, "METHOD_NOT_ALLOWED", "ожидался GET")
		return
	}
	if !s.requireAccessRoleApprover(w, r, uid) {
		return
	}
	ctx := r.Context()
	rows, err := s.q.ListAccessRolesForApprover(ctx, pgxutil.UUID(uid))
	if err != nil {
		writeAPIError(w, http.StatusInternalServerError, "INTERNAL", "ошибка БД")
		return
	}
	out := make([]map[string]any, 0, len(rows))
	for _, ar := range rows {
		out = append(out, map[string]any{
			"id":   formatUUID(ar.ID),
			"name": ar.Name,
		})
	}
	writeJSON(w, http.StatusOK, map[string]any{"items": out})
}
