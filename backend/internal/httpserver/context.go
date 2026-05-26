package httpserver

import (
	"context"
	"net/http"

	"github.com/google/uuid"
)

type ctxKey int

const keyUserID ctxKey = 1

func withUserID(ctx context.Context, id uuid.UUID) context.Context {
	return context.WithValue(ctx, keyUserID, id)
}

func userIDFromCtx(ctx context.Context) (uuid.UUID, bool) {
	v := ctx.Value(keyUserID)
	if v == nil {
		return uuid.Nil, false
	}
	u, ok := v.(uuid.UUID)
	return u, ok
}

func bearerToken(r *http.Request) string {
	h := r.Header.Get("Authorization")
	if len(h) < 8 || h[:7] != "Bearer " {
		return ""
	}
	return h[7:]
}
