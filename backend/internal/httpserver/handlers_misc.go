package httpserver

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/itshop/api/internal/ai"
	"github.com/itshop/api/internal/objectstore"
	"github.com/itshop/api/internal/pgxutil"
	"github.com/itshop/api/internal/store"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/minio/minio-go/v7"
)

func (s *Server) listAudit(w http.ResponseWriter, r *http.Request, uid uuid.UUID) {
	ctx := r.Context()
	if !s.hasRbac(ctx, uid, "admin") {
		writeAPIError(w, http.StatusForbidden, "FORBIDDEN", "требуется роль admin")
		return
	}
	limit := int32(100)
	off := int32(0)
	if v := r.URL.Query().Get("limit"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 && n <= 500 {
			limit = int32(n)
		}
	}
	if v := r.URL.Query().Get("offset"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n >= 0 {
			off = int32(n)
		}
	}
	search := strings.TrimSpace(r.URL.Query().Get("q"))
	if search == "" {
		search = strings.TrimSpace(r.URL.Query().Get("search"))
	}

	filterUserID := pgxutil.UUID(uuid.Nil)
	if v := strings.TrimSpace(r.URL.Query().Get("user_id")); v != "" {
		uu, err := pgxutil.ParseUUID(v)
		if err != nil {
			writeAPIError(w, http.StatusBadRequest, "VALIDATION_ERROR", "некорректный user_id")
			return
		}
		filterUserID = uu
	}
	filterReqID := pgxutil.UUID(uuid.Nil)
	if v := strings.TrimSpace(r.URL.Query().Get("request_id")); v != "" {
		ru, err := pgxutil.ParseUUID(v)
		if err != nil {
			writeAPIError(w, http.StatusBadRequest, "VALIDATION_ERROR", "некорректный request_id")
			return
		}
		filterReqID = ru
	}

	rows, err := s.q.ListAuditLogs(r.Context(), store.ListAuditLogsParams{
		Limit:           limit,
		Offset:          off,
		Search:          search,
		FilterUserID:    filterUserID,
		FilterRequestID: filterReqID,
	})
	if err != nil {
		writeAPIError(w, http.StatusInternalServerError, "INTERNAL", "ошибка БД")
		return
	}
	out := make([]map[string]any, 0, len(rows))
	for _, row := range rows {
		var details any
		if row.Details != nil {
			_ = json.Unmarshal(row.Details, &details)
		}
		login := strings.TrimSpace(row.UserLogin)
		if login == "" {
			login = "—"
		}
		out = append(out, map[string]any{
			"id":         formatUUID(row.ID),
			"user_login": login,
			"request_id": formatUUID(row.RequestID),
			"action":     row.Action,
			"details":    details,
			"created_at": formatTimePtr(row.CreatedAt),
		})
	}
	writeJSON(w, http.StatusOK, map[string]any{"items": out})
}

func (s *Server) uploadAttachment(w http.ResponseWriter, r *http.Request, uid uuid.UUID) {
	if err := r.ParseMultipartForm(32 << 20); err != nil {
		writeAPIError(w, http.StatusBadRequest, "VALIDATION_ERROR", "ожидался multipart")
		return
	}
	ctx := r.Context()
	var reqPG pgtype.UUID
	if rs := r.FormValue("request_id"); rs != "" {
		parsed, err := pgxutil.ParseUUID(rs)
		if err != nil {
			writeAPIError(w, http.StatusBadRequest, "VALIDATION_ERROR", "некорректный request_id")
			return
		}
		reqRow, err := s.q.GetRequestByID(ctx, parsed)
		if err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				writeAPIError(w, http.StatusNotFound, "NOT_FOUND", "заявка не найдена")
				return
			}
			writeAPIError(w, http.StatusInternalServerError, "INTERNAL", "ошибка БД")
			return
		}
		iniu, ok := pgxutil.ToGoogle(reqRow.InitiatorID)
		if !ok || iniu != uid {
			writeAPIError(w, http.StatusForbidden, "FORBIDDEN", "вложение к заявке может добавить только инициатор")
			return
		}
		if reqRow.Status != "pending" && reqRow.Status != "draft" {
			writeAPIError(w, http.StatusConflict, "CONFLICT", "вложения к завершённой заявке недоступны")
			return
		}
		reqPG = parsed
	}
	file, header, err := r.FormFile("file")
	if err != nil {
		writeAPIError(w, http.StatusBadRequest, "VALIDATION_ERROR", "поле file обязательно")
		return
	}
	defer file.Close()
	body, err := io.ReadAll(io.LimitReader(file, 32<<20))
	if err != nil {
		writeAPIError(w, http.StatusBadRequest, "VALIDATION_ERROR", "не удалось прочитать файл")
		return
	}
	ct := header.Header.Get("Content-Type")
	if ct == "" {
		ct = "application/octet-stream"
	}
	name := filepath.Base(header.Filename)
	if name == "." || name == "" {
		name = "upload.bin"
	}
	key := "attachments/" + uuid.New().String() + "/" + name
	_, err = s.minio.PutObject(ctx, s.cfg.S3Bucket, key, bytes.NewReader(body), int64(len(body)), minio.PutObjectOptions{
		ContentType: ct,
	})
	if err != nil {
		writeAPIError(w, http.StatusInternalServerError, "INTERNAL", "ошибка загрузки в хранилище")
		return
	}
	att, err := s.q.InsertAttachment(ctx, store.InsertAttachmentParams{
		UserID:       pgxutil.UUID(uid),
		RequestID:    reqPG,
		OriginalName: name,
		ContentType:  ct,
		S3Key:        key,
		SizeBytes:    int64(len(body)),
	})
	if err != nil {
		writeAPIError(w, http.StatusInternalServerError, "INTERNAL", "ошибка БД")
		return
	}
	auditDet := map[string]any{"attachment_id": formatUUID(att.ID), "s3_key": key}
	if reqPG.Valid {
		rid, _ := pgxutil.ToGoogle(reqPG)
		s.audit(ctx, uid, &rid, "attachment.upload", auditDet)
	} else {
		s.audit(ctx, uid, nil, "attachment.upload", auditDet)
	}
	writeJSON(w, http.StatusCreated, map[string]any{
		"id":            formatUUID(att.ID),
		"original_name": att.OriginalName,
		"content_type":  att.ContentType,
		"size_bytes":    att.SizeBytes,
		"created_at":    formatTimePtr(att.CreatedAt),
	})
}

func (s *Server) getAttachment(w http.ResponseWriter, r *http.Request, uid uuid.UUID) {
	idStr := r.PathValue("id")
	aid, err := pgxutil.ParseUUID(idStr)
	if err != nil {
		writeAPIError(w, http.StatusBadRequest, "VALIDATION_ERROR", "некорректный id")
		return
	}
	ctx := r.Context()
	att, err := s.q.GetAttachmentByID(ctx, aid)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			writeAPIError(w, http.StatusNotFound, "NOT_FOUND", "вложение не найдено")
			return
		}
		writeAPIError(w, http.StatusInternalServerError, "INTERNAL", "ошибка БД")
		return
	}
	owner, ok := pgxutil.ToGoogle(att.UserID)
	if !ok {
		writeAPIError(w, http.StatusForbidden, "FORBIDDEN", "нет доступа")
		return
	}
	allowed := owner == uid || s.hasRbac(ctx, uid, "admin")
	if !allowed && att.RequestID.Valid {
		reqRow, err := s.q.GetRequestByID(ctx, att.RequestID)
		if err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				writeAPIError(w, http.StatusNotFound, "NOT_FOUND", "заявка не найдена")
				return
			}
			writeAPIError(w, http.StatusInternalServerError, "INTERNAL", "ошибка БД")
			return
		}
		var errCV error
		allowed, errCV = s.canViewRequest(ctx, uid, att.RequestID, reqRow.InitiatorID)
		if errCV != nil {
			writeAPIError(w, http.StatusInternalServerError, "INTERNAL", "ошибка БД")
			return
		}
	}
	if !allowed {
		writeAPIError(w, http.StatusForbidden, "FORBIDDEN", "нет доступа")
		return
	}
	url, err := objectstore.PresignedGET(ctx, s.minio, s.cfg.S3Bucket, att.S3Key, 15*time.Minute)
	if err != nil {
		writeAPIError(w, http.StatusInternalServerError, "INTERNAL", "не удалось выдать ссылку")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"id":            formatUUID(att.ID),
		"original_name": att.OriginalName,
		"content_type":  att.ContentType,
		"url":           url,
		"expires_in":    900,
	})
}

func (s *Server) hasRbac(ctx context.Context, uid uuid.UUID, role string) bool {
	roles, err := s.q.ListRbacNamesForUser(ctx, pgxutil.UUID(uid))
	if err != nil {
		return false
	}
	for _, rr := range roles {
		if rr == role {
			return true
		}
	}
	return false
}

type aiAnalyzeReq struct {
	RequestID string `json:"request_id"`
}

func (s *Server) internalAIAnalyze(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeAPIError(w, http.StatusMethodNotAllowed, "METHOD_NOT_ALLOWED", "ожидался POST")
		return
	}
	var body aiAnalyzeReq
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil || body.RequestID == "" {
		writeAPIError(w, http.StatusBadRequest, "VALIDATION_ERROR", "нужен request_id")
		return
	}
	rid, err := pgxutil.ParseUUID(body.RequestID)
	if err != nil {
		writeAPIError(w, http.StatusBadRequest, "VALIDATION_ERROR", "некорректный request_id")
		return
	}
	ctx := r.Context()
	ad, err := ai.AnalyzeRequest(ctx, s.q, rid)
	if err != nil {
		writeAPIError(w, http.StatusInternalServerError, "INTERNAL", err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"id":             formatUUID(ad.ID),
		"recommendation": ad.Recommendation,
		"risk_score":     ad.RiskScore,
		"confidence":     ad.Confidence,
		"reason":         textString(ad.Reason),
		"created_at":     formatTimePtr(ad.CreatedAt),
	})
}

func (s *Server) internalIAMUser(w http.ResponseWriter, r *http.Request) {
	if !s.cfg.IAMMock {
		writeAPIError(w, http.StatusServiceUnavailable, "SERVICE_UNAVAILABLE", "IAM mock отключён")
		return
	}
	login := r.URL.Query().Get("login")
	idStr := r.URL.Query().Get("id")
	ctx := r.Context()
	if login != "" {
		u, err := s.q.GetUserByLogin(ctx, login)
		if err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				writeAPIError(w, http.StatusNotFound, "NOT_FOUND", "пользователь не найден")
				return
			}
			writeAPIError(w, http.StatusInternalServerError, "INTERNAL", "ошибка БД")
			return
		}
		row, err := s.q.GetUserByIDForIAM(ctx, u.ID)
		if err != nil {
			writeAPIError(w, http.StatusInternalServerError, "INTERNAL", "ошибка БД")
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{
			"source":          "mock",
			"id":              formatUUID(u.ID),
			"login":           row.Login,
			"full_name":       row.FullName,
			"email":           row.Email,
			"position":        row.Position,
			"is_active":       row.IsActive,
			"department_name": textString(row.DepartmentName),
		})
		return
	}
	if idStr != "" {
		pg, err := pgxutil.ParseUUID(idStr)
		if err != nil {
			writeAPIError(w, http.StatusBadRequest, "VALIDATION_ERROR", "некорректный id")
			return
		}
		row, err := s.q.GetUserByIDForIAM(ctx, pg)
		if err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				writeAPIError(w, http.StatusNotFound, "NOT_FOUND", "пользователь не найден")
				return
			}
			writeAPIError(w, http.StatusInternalServerError, "INTERNAL", "ошибка БД")
			return
		}
		gid, _ := pgxutil.ToGoogle(row.ID)
		writeJSON(w, http.StatusOK, map[string]any{
			"source":          "mock",
			"id":              gid.String(),
			"login":           row.Login,
			"full_name":       row.FullName,
			"email":           row.Email,
			"position":        row.Position,
			"is_active":       row.IsActive,
			"department_name": textString(row.DepartmentName),
		})
		return
	}
	writeAPIError(w, http.StatusBadRequest, "VALIDATION_ERROR", "укажите query login или id")
}
