package httpserver

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"strings"
	"time"

	"github.com/itshop/api/internal/ai"
	"github.com/itshop/api/internal/pgxutil"
	"github.com/itshop/api/internal/store"
	"github.com/itshop/api/internal/workflow"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
)

type createRequestBody struct {
	Items []struct {
		AccessRoleID  string `json:"access_role_id"`
		Justification string `json:"justification"`
	} `json:"items"`
	// Общее обоснование (опционально): подставляется, если у позиции не указано своё.
	Justification string `json:"justification"`
	// Нужен до (включительно), YYYY-MM-DD; пусто — без ограничения по сроку в заявке.
	NeededUntil string `json:"needed_until"`
}

type createItemPlan struct {
	arID pgtype.UUID
	text string
}

func (s *Server) createRequest(w http.ResponseWriter, r *http.Request, uid uuid.UUID) {
	if r.Method != http.MethodPost {
		writeAPIError(w, http.StatusMethodNotAllowed, "METHOD_NOT_ALLOWED", "ожидался POST")
		return
	}
	var body createRequestBody
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeAPIError(w, http.StatusBadRequest, "VALIDATION_ERROR", "некорректный JSON")
		return
	}
	if len(body.Items) == 0 {
		writeAPIError(w, http.StatusBadRequest, "VALIDATION_ERROR", "нужна хотя бы одна позиция items")
		return
	}
	ctx := r.Context()
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		writeAPIError(w, http.StatusInternalServerError, "INTERNAL", "не удалось начать транзакцию")
		return
	}
	defer tx.Rollback(ctx)

	qx := s.q.WithTx(tx)

	var neededUntil pgtype.Date
	if nu := strings.TrimSpace(body.NeededUntil); nu != "" {
		d, err := parsePgDateYYYYMMDD(nu)
		if err != nil {
			writeAPIError(w, http.StatusBadRequest, "VALIDATION_ERROR", "некорректный needed_until (ожидается YYYY-MM-DD)")
			return
		}
		today := time.Now().UTC()
		todayStart := time.Date(today.Year(), today.Month(), today.Day(), 0, 0, 0, 0, time.UTC)
		if d.Time.Before(todayStart) {
			writeAPIError(w, http.StatusBadRequest, "VALIDATION_ERROR", "дата «нужен до» не может быть в прошлом")
			return
		}
		neededUntil = d
	}

	globalJ := strings.TrimSpace(body.Justification)
	plans := make([]createItemPlan, 0, len(body.Items))
	for _, it := range body.Items {
		arID, err := pgxutil.ParseUUID(it.AccessRoleID)
		if err != nil {
			writeAPIError(w, http.StatusBadRequest, "VALIDATION_ERROR", "некорректный access_role_id")
			return
		}
		if _, err := qx.GetAccessRoleByID(ctx, arID); err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				writeAPIError(w, http.StatusBadRequest, "VALIDATION_ERROR", "роль доступа не найдена")
				return
			}
			writeAPIError(w, http.StatusInternalServerError, "INTERNAL", "ошибка БД")
			return
		}
		j := strings.TrimSpace(it.Justification)
		if j == "" {
			j = globalJ
		}
		if j == "" {
			writeAPIError(w, http.StatusBadRequest, "VALIDATION_ERROR", "укажите обоснование для каждой выбранной роли")
			return
		}
		plans = append(plans, createItemPlan{arID: arID, text: j})
	}

	combinedJ := ""
	for i, p := range plans {
		if i > 0 {
			combinedJ += "\n\n---\n\n"
		}
		combinedJ += p.text
	}

	reqRow, err := qx.InsertRequest(ctx, store.InsertRequestParams{
		InitiatorID:   pgxutil.UUID(uid),
		Status:        "pending",
		Justification: pgxutil.TextOptional(combinedJ),
		RiskScore:     pgtype.Float8{},
		NeededUntil:   neededUntil,
	})
	if err != nil {
		writeAPIError(w, http.StatusInternalServerError, "INTERNAL", "не удалось создать заявку")
		return
	}
	seenRole := make(map[uuid.UUID]struct{})
	roleOrder := make([]pgtype.UUID, 0, len(plans))
	for _, p := range plans {
		if gid, ok := pgxutil.ToGoogle(p.arID); ok {
			if _, dup := seenRole[gid]; !dup {
				seenRole[gid] = struct{}{}
				roleOrder = append(roleOrder, p.arID)
			}
		}
	}
	for _, p := range plans {
		itemID := uuid.New()
		if err := qx.InsertRequestItem(ctx, store.InsertRequestItemParams{
			ID:            pgxutil.UUID(itemID),
			RequestID:     reqRow.ID,
			AccessRoleID:  p.arID,
			Justification: pgxutil.TextOptional(p.text),
		}); err != nil {
			writeAPIError(w, http.StatusInternalServerError, "INTERNAL", "не удалось добавить позицию")
			return
		}
	}
	if s.cfg.AIMock {
		if _, errAI := ai.AnalyzeRequest(ctx, qx, reqRow.ID); errAI != nil {
			writeAPIError(w, http.StatusInternalServerError, "INTERNAL", "ошибка AI-оценки")
			return
		}
	}
	// при AIM_MOCK=false оставляем risk_score пустым — внешний AI можно подключить позже
	chain, err := workflow.ApproverChainForAccessRoles(ctx, qx, roleOrder)
	if err != nil {
		writeAPIError(w, http.StatusInternalServerError, "INTERNAL", "ошибка маршрута согласования")
		return
	}
	for i, appr := range chain {
		if err := qx.InsertApproval(ctx, store.InsertApprovalParams{
			ID:         pgxutil.UUID(uuid.New()),
			RequestID:  reqRow.ID,
			ApproverID: appr,
			StepNumber: int32(i + 1),
			Status:     "pending",
		}); err != nil {
			writeAPIError(w, http.StatusInternalServerError, "INTERNAL", "не удалось создать согласование")
			return
		}
	}

	allLow, errAL := ai.AllRequestItemsAreLowRisk(ctx, qx, reqRow.ID)
	if errAL != nil {
		writeAPIError(w, http.StatusInternalServerError, "INTERNAL", "ошибка оценки уровня риска позиций")
		return
	}
	autoApproved := false
	if len(chain) > 0 && allLow {
		apprs, errLA := qx.ListApprovalsByRequest(ctx, reqRow.ID)
		if errLA != nil {
			writeAPIError(w, http.StatusInternalServerError, "INTERNAL", "ошибка БД")
			return
		}
		autoComment := "Автосогласование: все роли доступа с низким уровнем риска"
		for _, a := range apprs {
			if a.Status != "pending" {
				continue
			}
			if err := qx.UpdateApprovalByID(ctx, store.UpdateApprovalByIDParams{
				ID:      a.ID,
				Status:  "approved",
				Comment: pgxutil.Text(autoComment),
			}); err != nil {
				writeAPIError(w, http.StatusInternalServerError, "INTERNAL", "ошибка автосогласования")
				return
			}
		}
		if err := qx.UpdateRequestStatus(ctx, store.UpdateRequestStatusParams{
			ID:     reqRow.ID,
			Status: "approved",
		}); err != nil {
			writeAPIError(w, http.StatusInternalServerError, "INTERNAL", "не удалось обновить статус заявки")
			return
		}
		autoApproved = true
	}

	if err := tx.Commit(ctx); err != nil {
		writeAPIError(w, http.StatusInternalServerError, "INTERNAL", "не удалось зафиксировать заявку")
		return
	}
	rid, _ := pgxutil.ToGoogle(reqRow.ID)
	auditDet := map[string]any{"request_id": rid.String()}
	if neededUntil.Valid {
		auditDet["needed_until"] = neededUntil.Time.Format("2006-01-02")
	}
	if autoApproved {
		auditDet["auto_approved_low_risk"] = true
	}
	s.audit(ctx, uid, &rid, "request.create", auditDet)
	statusOut := reqRow.Status
	if autoApproved {
		statusOut = "approved"
	}
	writeJSON(w, http.StatusCreated, map[string]any{
		"id":     formatUUID(reqRow.ID),
		"status": statusOut,
	})
}

func (s *Server) myRequests(w http.ResponseWriter, r *http.Request, uid uuid.UUID) {
	rows, err := s.q.ListRequestsByInitiator(r.Context(), pgxutil.UUID(uid))
	if err != nil {
		writeAPIError(w, http.StatusInternalServerError, "INTERNAL", "ошибка БД")
		return
	}
	out := make([]map[string]any, 0, len(rows))
	for _, row := range rows {
		out = append(out, map[string]any{
			"id":                 formatUUID(row.ID),
			"initiator_id":       formatUUID(row.InitiatorID),
			"initiator_name":     strings.TrimSpace(row.InitiatorFullName),
			"initiator_login":    strings.TrimSpace(row.InitiatorLogin),
			"status":             row.Status,
			"risk_score":         formatFloat8(row.RiskScore),
			"created_at":         formatTimePtr(row.CreatedAt),
			"updated_at":         formatTimePtr(row.UpdatedAt),
			"justification":      textString(row.Justification),
			"needed_until":       formatDateISO(row.NeededUntil),
			"access_role_names":  strings.TrimSpace(row.AccessRoleNames),
			"approvals_total":    row.ApprovalsTotal,
			"approvals_done":     row.ApprovalsDone,
		})
	}
	writeJSON(w, http.StatusOK, map[string]any{"items": out})
}

func (s *Server) getRequest(w http.ResponseWriter, r *http.Request, uid uuid.UUID) {
	idStr := r.PathValue("id")
	rid, err := pgxutil.ParseUUID(idStr)
	if err != nil {
		writeAPIError(w, http.StatusBadRequest, "VALIDATION_ERROR", "некорректный id")
		return
	}
	ctx := r.Context()
	reqRow, err := s.q.GetRequestByID(ctx, rid)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			writeAPIError(w, http.StatusNotFound, "NOT_FOUND", "заявка не найдена")
			return
		}
		writeAPIError(w, http.StatusInternalServerError, "INTERNAL", "ошибка БД")
		return
	}
	ok, err := s.canViewRequest(ctx, uid, rid, reqRow.InitiatorID)
	if err != nil || !ok {
		writeAPIError(w, http.StatusForbidden, "FORBIDDEN", "нет доступа к заявке")
		return
	}
	items, err := s.q.ListRequestItems(ctx, rid)
	if err != nil {
		writeAPIError(w, http.StatusInternalServerError, "INTERNAL", "ошибка БД")
		return
	}
	appr, err := s.q.ListApprovalsByRequestWithUsers(ctx, rid)
	if err != nil {
		writeAPIError(w, http.StatusInternalServerError, "INTERNAL", "ошибка БД")
		return
	}
	initiator, err := s.q.GetUserByID(ctx, reqRow.InitiatorID)
	initiatorName := ""
	initiatorLogin := ""
	if err == nil {
		initiatorName = initiator.FullName
		initiatorLogin = initiator.Login
	}
	itemOut := make([]map[string]any, 0, len(items))
	for _, it := range items {
		itemOut = append(itemOut, map[string]any{
			"id":               formatUUID(it.ID),
			"access_role_id":   formatUUID(it.AccessRoleID),
			"access_role_name": it.AccessRoleName,
			"risk_level":       it.RiskLevel,
			"resource_name":    it.ResourceName,
			"justification":    textString(it.Justification),
		})
	}
	apprOut := make([]map[string]any, 0, len(appr))
	for _, a := range appr {
		apprOut = append(apprOut, map[string]any{
			"id":               formatUUID(a.ID),
			"approver_id":      formatUUID(a.ApproverID),
			"approver_name":    a.ApproverFullName,
			"approver_login":   textString(a.ApproverLogin),
			"step_number":      a.StepNumber,
			"status":           a.Status,
			"decision_at":      formatTimePtr(a.DecisionAt),
			"comment":          textString(a.Comment),
		})
	}

	viewCanAct := false
	viewAsDelegate := false
	pa, errPA := s.q.GetPendingApprovalForRequestByApprover(ctx, store.GetPendingApprovalForRequestByApproverParams{
		RequestID:  rid,
		ApproverID: pgxutil.UUID(uid),
	})
	if errPA == nil {
		viewCanAct = true
		if pu, ok := pgxutil.ToGoogle(pa.ApproverID); ok {
			viewAsDelegate = pu != uid
		}
	} else if !errors.Is(errPA, pgx.ErrNoRows) {
		writeAPIError(w, http.StatusInternalServerError, "INTERNAL", "ошибка БД")
		return
	}

	attOut := make([]map[string]any, 0)
	if atts, err := s.q.ListAttachmentsByRequest(ctx, rid); err == nil {
		for _, att := range atts {
			attOut = append(attOut, map[string]any{
				"id":             formatUUID(att.ID),
				"original_name":  att.OriginalName,
				"content_type":   att.ContentType,
				"size_bytes":     att.SizeBytes,
				"created_at":     formatTimePtr(att.CreatedAt),
				"uploaded_by_id": formatUUID(att.UserID),
			})
		}
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"id":               formatUUID(reqRow.ID),
		"initiator_id":     formatUUID(reqRow.InitiatorID),
		"initiator_name":   initiatorName,
		"initiator_login":  initiatorLogin,
		"status":           reqRow.Status,
		"risk_score":       formatFloat8(reqRow.RiskScore),
		"created_at":       formatTimePtr(reqRow.CreatedAt),
		"updated_at":       formatTimePtr(reqRow.UpdatedAt),
		"justification":    textString(reqRow.Justification),
		"needed_until":     formatDateISO(reqRow.NeededUntil),
		"items":            itemOut,
		"approvals":        apprOut,
		"attachments":      attOut,
		"viewer_can_act_on_current_step": viewCanAct,
		"viewer_acts_as_delegate":        viewAsDelegate,
	})
}

func (s *Server) cancelRequest(w http.ResponseWriter, r *http.Request, uid uuid.UUID) {
	idStr := r.PathValue("id")
	rid, err := pgxutil.ParseUUID(idStr)
	if err != nil {
		writeAPIError(w, http.StatusBadRequest, "VALIDATION_ERROR", "некорректный id")
		return
	}
	ctx := r.Context()
	reqRow, err := s.q.GetRequestByID(ctx, rid)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			writeAPIError(w, http.StatusNotFound, "NOT_FOUND", "заявка не найдена")
			return
		}
		writeAPIError(w, http.StatusInternalServerError, "INTERNAL", "ошибка БД")
		return
	}
	initiator, _ := pgxutil.ToGoogle(reqRow.InitiatorID)
	if initiator != uid {
		writeAPIError(w, http.StatusForbidden, "FORBIDDEN", "отозвать может только инициатор")
		return
	}
	if reqRow.Status != "pending" && reqRow.Status != "draft" {
		writeAPIError(w, http.StatusConflict, "CONFLICT", "заявку нельзя отозвать в текущем статусе")
		return
	}
	if err := s.q.CancelAllPendingApprovalsForRequest(ctx, rid); err != nil {
		writeAPIError(w, http.StatusInternalServerError, "INTERNAL", "ошибка БД")
		return
	}
	if err := s.q.UpdateRequestStatus(ctx, store.UpdateRequestStatusParams{
		ID:     rid,
		Status: "cancelled",
	}); err != nil {
		writeAPIError(w, http.StatusInternalServerError, "INTERNAL", "ошибка БД")
		return
	}
	ridGo, _ := pgxutil.ToGoogle(rid)
	s.audit(ctx, uid, &ridGo, "request.cancel", map[string]any{"request_id": idStr})
	writeJSON(w, http.StatusOK, map[string]any{"status": "cancelled"})
}

func (s *Server) canViewRequest(ctx context.Context, uid uuid.UUID, reqID pgtype.UUID, initiator pgtype.UUID) (bool, error) {
	iu, ok := pgxutil.ToGoogle(initiator)
	if ok && iu == uid {
		return true, nil
	}
	roles, err := s.q.ListRbacNamesForUser(ctx, pgxutil.UUID(uid))
	if err != nil {
		return false, err
	}
	for _, r := range roles {
		if r == "admin" {
			return true, nil
		}
	}
	appr, err := s.q.ListApprovalsByRequest(ctx, reqID)
	if err != nil {
		return false, err
	}
	for _, a := range appr {
		au, ok := pgxutil.ToGoogle(a.ApproverID)
		if ok && au == uid {
			return true, nil
		}
	}
	// Заместитель: активное делегирование от согласующего на шаге заявки
	var delCount int64
	err = s.pool.QueryRow(ctx, `
		SELECT COUNT(*)::bigint FROM approvals a
		INNER JOIN delegations d ON d.from_user_id = a.approver_id AND d.to_user_id = $1
		WHERE a.request_id = $2
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
		      WHERE ri.request_id = $2
		    )
		  )`,
		pgxutil.UUID(uid), reqID).Scan(&delCount)
	if err != nil {
		return false, err
	}
	if delCount > 0 {
		return true, nil
	}
	return false, nil
}
