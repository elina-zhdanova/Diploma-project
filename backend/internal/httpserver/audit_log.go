package httpserver

import (
	"context"
	"encoding/json"

	"github.com/itshop/api/internal/pgxutil"
	"github.com/itshop/api/internal/store"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
)

func (s *Server) audit(ctx context.Context, userID uuid.UUID, requestID *uuid.UUID, action string, details map[string]any) {
	id := uuid.New()
	b := []byte("{}")
	if details != nil {
		if x, err := json.Marshal(details); err == nil {
			b = x
		}
	}
	var rid pgtype.UUID
	if requestID != nil {
		rid = pgxutil.UUID(*requestID)
	}
	_ = s.q.InsertAuditLog(ctx, store.InsertAuditLogParams{
		ID:        pgxutil.UUID(id),
		UserID:    pgxutil.UUID(userID),
		RequestID: rid,
		Action:    action,
		Details:   b,
	})
}
