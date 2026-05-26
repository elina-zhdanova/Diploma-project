package httpserver

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
)

func writeJSON(w http.ResponseWriter, code int, v any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(code)
	_ = json.NewEncoder(w).Encode(v)
}

type errBody struct {
	Error struct {
		Code    string         `json:"code"`
		Message string         `json:"message"`
		Details map[string]any `json:"details,omitempty"`
	} `json:"error"`
}

func writeAPIError(w http.ResponseWriter, code int, apiCode, msg string) {
	var e errBody
	e.Error.Code = apiCode
	e.Error.Message = msg
	writeJSON(w, code, e)
}

func formatUUID(u pgtype.UUID) string {
	if !u.Valid {
		return ""
	}
	return uuid.UUID(u.Bytes).String()
}

func formatTimePtr(t pgtype.Timestamptz) *string {
	if !t.Valid {
		return nil
	}
	s := t.Time.UTC().Format(time.RFC3339)
	return &s
}

func formatFloat8(f pgtype.Float8) any {
	if !f.Valid {
		return nil
	}
	return f.Float64
}

func textString(t pgtype.Text) any {
	if !t.Valid {
		return nil
	}
	return t.String
}

func formatDateISO(d pgtype.Date) any {
	if !d.Valid {
		return nil
	}
	return d.Time.Format("2006-01-02")
}
