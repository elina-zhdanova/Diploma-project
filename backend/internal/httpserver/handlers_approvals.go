package httpserver

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"strings"

	"github.com/itshop/api/internal/pgxutil"
	"github.com/itshop/api/internal/store"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
)

// errApprovalPendingNotFound — у пользователя нет ожидающего шага по этой заявке.
var errApprovalPendingNotFound = errors.New("no pending approval for approver")

type commentBody struct {
	Comment string `json:"comment"`
}

type delegateBody struct {
	ToUserID string `json:"to_user_id"`
	Comment  string `json:"comment"`
}

func (s *Server) inbox(w http.ResponseWriter, r *http.Request, uid uuid.UUID) {
	rows, err := s.q.InboxForApprover(r.Context(), pgxutil.UUID(uid))
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
			"current_step":       row.StepNumber,
			"access_role_names":  strings.TrimSpace(row.AccessRoleNames),
		})
	}
	writeJSON(w, http.StatusOK, map[string]any{"items": out})
}

// approveSingle выполняет одно согласование текущего пользователя по заявке (без аудита).
func (s *Server) approveSingle(ctx context.Context, uid uuid.UUID, reqPG pgtype.UUID, comment string) error {
	appr, err := s.q.GetPendingApprovalForRequestByApprover(ctx, store.GetPendingApprovalForRequestByApproverParams{
		RequestID:  reqPG,
		ApproverID: pgxutil.UUID(uid),
	})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return errApprovalPendingNotFound
		}
		return err
	}
	if err := s.q.UpdateApprovalByID(ctx, store.UpdateApprovalByIDParams{
		ID:      appr.ID,
		Status:  "approved",
		Comment: pgxutil.TextOptional(comment),
	}); err != nil {
		return err
	}
	n, err := s.q.CountPendingApprovalsForRequest(ctx, reqPG)
	if err != nil {
		return err
	}
	if n == 0 {
		if err := s.q.UpdateRequestStatus(ctx, store.UpdateRequestStatusParams{
			ID:     reqPG,
			Status: "approved",
		}); err != nil {
			return err
		}
	}
	return nil
}

// rejectSingle — отказ по текущему шагу для пользователя (без аудита).
func (s *Server) rejectSingle(ctx context.Context, uid uuid.UUID, reqPG pgtype.UUID, comment string) error {
	appr, err := s.q.GetPendingApprovalForRequestByApprover(ctx, store.GetPendingApprovalForRequestByApproverParams{
		RequestID:  reqPG,
		ApproverID: pgxutil.UUID(uid),
	})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return errApprovalPendingNotFound
		}
		return err
	}
	if err := s.q.UpdateApprovalByID(ctx, store.UpdateApprovalByIDParams{
		ID:      appr.ID,
		Status:  "rejected",
		Comment: pgxutil.TextOptional(comment),
	}); err != nil {
		return err
	}
	if err := s.q.CancelAllPendingApprovalsForRequest(ctx, reqPG); err != nil {
		return err
	}
	if err := s.q.UpdateRequestStatus(ctx, store.UpdateRequestStatusParams{
		ID:     reqPG,
		Status: "rejected",
	}); err != nil {
		return err
	}
	return nil
}

func (s *Server) approve(w http.ResponseWriter, r *http.Request, uid uuid.UUID) {
	reqIDStr := r.PathValue("requestId")
	reqPG, err := pgxutil.ParseUUID(reqIDStr)
	if err != nil {
		writeAPIError(w, http.StatusBadRequest, "VALIDATION_ERROR", "некорректный requestId")
		return
	}
	var body commentBody
	_ = json.NewDecoder(r.Body).Decode(&body)
	ctx := r.Context()
	if err := s.approveSingle(ctx, uid, reqPG, body.Comment); err != nil {
		if errors.Is(err, errApprovalPendingNotFound) {
			writeAPIError(w, http.StatusNotFound, "NOT_FOUND", "нет ожидающего шага для вас")
			return
		}
		writeAPIError(w, http.StatusInternalServerError, "INTERNAL", "ошибка БД")
		return
	}
	rid, _ := pgxutil.ToGoogle(reqPG)
	s.audit(ctx, uid, &rid, "approval.approve", map[string]any{"request_id": reqIDStr})
	writeJSON(w, http.StatusOK, map[string]any{"status": "ok"})
}

type bulkApproveBody struct {
	RequestIDs []string `json:"request_ids"`
	Comment    string   `json:"comment"`
}

func (s *Server) bulkApprove(w http.ResponseWriter, r *http.Request, uid uuid.UUID) {
	if r.Method != http.MethodPost {
		writeAPIError(w, http.StatusMethodNotAllowed, "METHOD_NOT_ALLOWED", "ожидался POST")
		return
	}
	var body bulkApproveBody
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeAPIError(w, http.StatusBadRequest, "VALIDATION_ERROR", "некорректный JSON")
		return
	}
	ids := body.RequestIDs
	if len(ids) == 0 {
		writeAPIError(w, http.StatusBadRequest, "VALIDATION_ERROR", "нужен непустой request_ids")
		return
	}
	const maxBulk = 100
	if len(ids) > maxBulk {
		writeAPIError(w, http.StatusBadRequest, "VALIDATION_ERROR", "не более 100 заявок за раз")
		return
	}
	seen := make(map[string]struct{}, len(ids))
	uniq := make([]string, 0, len(ids))
	for _, id := range ids {
		t := strings.TrimSpace(id)
		if t == "" {
			continue
		}
		if _, ok := seen[t]; ok {
			continue
		}
		seen[t] = struct{}{}
		uniq = append(uniq, t)
	}
	if len(uniq) == 0 {
		writeAPIError(w, http.StatusBadRequest, "VALIDATION_ERROR", "нужен непустой request_ids")
		return
	}

	ctx := r.Context()
	approved := make([]string, 0, len(uniq))
	failed := make([]map[string]string, 0)

	for _, reqIDStr := range uniq {
		reqPG, err := pgxutil.ParseUUID(reqIDStr)
		if err != nil {
			failed = append(failed, map[string]string{"id": reqIDStr, "error": "некорректный id"})
			continue
		}
		err = s.approveSingle(ctx, uid, reqPG, body.Comment)
		if err != nil {
			if errors.Is(err, errApprovalPendingNotFound) {
				failed = append(failed, map[string]string{"id": reqIDStr, "error": "нет ожидающего шага для вас"})
				continue
			}
			failed = append(failed, map[string]string{"id": reqIDStr, "error": "ошибка сервера"})
			continue
		}
		approved = append(approved, reqIDStr)
		rid, _ := pgxutil.ToGoogle(reqPG)
		s.audit(ctx, uid, &rid, "approval.approve", map[string]any{"request_id": reqIDStr, "bulk": true})
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"approved": approved,
		"failed":   failed,
	})
}

func (s *Server) reject(w http.ResponseWriter, r *http.Request, uid uuid.UUID) {
	reqIDStr := r.PathValue("requestId")
	reqPG, err := pgxutil.ParseUUID(reqIDStr)
	if err != nil {
		writeAPIError(w, http.StatusBadRequest, "VALIDATION_ERROR", "некорректный requestId")
		return
	}
	var body commentBody
	_ = json.NewDecoder(r.Body).Decode(&body)
	ctx := r.Context()
	if err := s.rejectSingle(ctx, uid, reqPG, body.Comment); err != nil {
		if errors.Is(err, errApprovalPendingNotFound) {
			writeAPIError(w, http.StatusNotFound, "NOT_FOUND", "нет ожидающего шага для вас")
			return
		}
		writeAPIError(w, http.StatusInternalServerError, "INTERNAL", "ошибка БД")
		return
	}
	rid, _ := pgxutil.ToGoogle(reqPG)
	s.audit(ctx, uid, &rid, "approval.reject", map[string]any{"request_id": reqIDStr})
	writeJSON(w, http.StatusOK, map[string]any{"status": "ok"})
}

type bulkRejectBody struct {
	RequestIDs []string `json:"request_ids"`
	Comment    string   `json:"comment"`
}

func (s *Server) bulkReject(w http.ResponseWriter, r *http.Request, uid uuid.UUID) {
	if r.Method != http.MethodPost {
		writeAPIError(w, http.StatusMethodNotAllowed, "METHOD_NOT_ALLOWED", "ожидался POST")
		return
	}
	var body bulkRejectBody
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeAPIError(w, http.StatusBadRequest, "VALIDATION_ERROR", "некорректный JSON")
		return
	}
	ids := body.RequestIDs
	if len(ids) == 0 {
		writeAPIError(w, http.StatusBadRequest, "VALIDATION_ERROR", "нужен непустой request_ids")
		return
	}
	const maxBulk = 100
	if len(ids) > maxBulk {
		writeAPIError(w, http.StatusBadRequest, "VALIDATION_ERROR", "не более 100 заявок за раз")
		return
	}
	seen := make(map[string]struct{}, len(ids))
	uniq := make([]string, 0, len(ids))
	for _, id := range ids {
		t := strings.TrimSpace(id)
		if t == "" {
			continue
		}
		if _, ok := seen[t]; ok {
			continue
		}
		seen[t] = struct{}{}
		uniq = append(uniq, t)
	}
	if len(uniq) == 0 {
		writeAPIError(w, http.StatusBadRequest, "VALIDATION_ERROR", "нужен непустой request_ids")
		return
	}

	ctx := r.Context()
	rejected := make([]string, 0, len(uniq))
	failed := make([]map[string]string, 0)

	for _, reqIDStr := range uniq {
		reqPG, err := pgxutil.ParseUUID(reqIDStr)
		if err != nil {
			failed = append(failed, map[string]string{"id": reqIDStr, "error": "некорректный id"})
			continue
		}
		err = s.rejectSingle(ctx, uid, reqPG, body.Comment)
		if err != nil {
			if errors.Is(err, errApprovalPendingNotFound) {
				failed = append(failed, map[string]string{"id": reqIDStr, "error": "нет ожидающего шага для вас"})
				continue
			}
			failed = append(failed, map[string]string{"id": reqIDStr, "error": "ошибка сервера"})
			continue
		}
		rejected = append(rejected, reqIDStr)
		rid, _ := pgxutil.ToGoogle(reqPG)
		s.audit(ctx, uid, &rid, "approval.reject", map[string]any{"request_id": reqIDStr, "bulk": true})
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"rejected": rejected,
		"failed":   failed,
	})
}

func (s *Server) delegate(w http.ResponseWriter, r *http.Request, uid uuid.UUID) {
	reqIDStr := r.PathValue("requestId")
	reqPG, err := pgxutil.ParseUUID(reqIDStr)
	if err != nil {
		writeAPIError(w, http.StatusBadRequest, "VALIDATION_ERROR", "некорректный requestId")
		return
	}
	var body delegateBody
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil || body.ToUserID == "" {
		writeAPIError(w, http.StatusBadRequest, "VALIDATION_ERROR", "нужен to_user_id")
		return
	}
	target, err := pgxutil.ParseUUID(body.ToUserID)
	if err != nil {
		writeAPIError(w, http.StatusBadRequest, "VALIDATION_ERROR", "некорректный to_user_id")
		return
	}
	ctx := r.Context()
	appr, err := s.q.GetPendingApprovalForRequestByApprover(ctx, store.GetPendingApprovalForRequestByApproverParams{
		RequestID:  reqPG,
		ApproverID: pgxutil.UUID(uid),
	})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			writeAPIError(w, http.StatusNotFound, "NOT_FOUND", "нет ожидающего шага для вас")
			return
		}
		writeAPIError(w, http.StatusInternalServerError, "INTERNAL", "ошибка БД")
		return
	}
	if err := s.q.DelegateApproval(ctx, store.DelegateApprovalParams{
		ID:           appr.ID,
		ApproverID:   pgxutil.UUID(uid),
		ApproverID_2: target,
		Comment:      pgxutil.TextOptional(body.Comment),
	}); err != nil {
		writeAPIError(w, http.StatusInternalServerError, "INTERNAL", "ошибка БД")
		return
	}
	rid, _ := pgxutil.ToGoogle(reqPG)
	s.audit(ctx, uid, &rid, "approval.delegate", map[string]any{
		"request_id": reqIDStr,
		"to_user_id": body.ToUserID,
	})
	writeJSON(w, http.StatusOK, map[string]any{"status": "ok"})
}
