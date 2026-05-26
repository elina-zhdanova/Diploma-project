package httpserver

import (
	"net/http"
	"time"

	"github.com/itshop/api/internal/auth"
	"github.com/itshop/api/internal/config"
	"github.com/itshop/api/internal/store"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/minio/minio-go/v7"
)

type Server struct {
	cfg   config.Config
	pool  *pgxpool.Pool
	q     *store.Queries
	minio *minio.Client
}

func New(cfg config.Config, pool *pgxpool.Pool, minioClient *minio.Client) http.Handler {
	s := &Server{
		cfg:   cfg,
		pool:  pool,
		q:     store.New(pool),
		minio: minioClient,
	}
	mux := http.NewServeMux()

	mux.HandleFunc("GET /health", s.health)
	mux.HandleFunc("GET /api/v1/meta", s.meta)
	mux.HandleFunc("GET /api/meta", s.meta)

	mux.HandleFunc("POST /api/auth/login", s.login)
	mux.HandleFunc("GET /api/auth/me", s.auth(s.getMe))

	mux.HandleFunc("GET /api/systems", s.auth(s.listSystems))
	mux.HandleFunc("GET /api/resources", s.auth(s.listResources))
	mux.HandleFunc("GET /api/catalog/tree", s.auth(s.catalogTree))
	mux.HandleFunc("GET /api/access-roles", s.auth(s.listAccessRoles))
	mux.HandleFunc("GET /api/access-roles/search", s.auth(s.searchAccessRoles))
	mux.HandleFunc("GET /api/access-roles/{id}", s.auth(s.getAccessRoleDetail))

	mux.HandleFunc("POST /api/requests", s.auth(s.createRequest))
	mux.HandleFunc("GET /api/requests/my", s.auth(s.myRequests))
	mux.HandleFunc("GET /api/requests/{id}", s.auth(s.getRequest))
	mux.HandleFunc("POST /api/requests/{id}/cancel", s.auth(s.cancelRequest))

	mux.HandleFunc("GET /api/approvals/inbox", s.auth(s.inbox))
	mux.HandleFunc("POST /api/approvals/bulk-approve", s.auth(s.bulkApprove))
	mux.HandleFunc("POST /api/approvals/bulk-reject", s.auth(s.bulkReject))
	mux.HandleFunc("POST /api/approvals/{requestId}/approve", s.auth(s.approve))
	mux.HandleFunc("POST /api/approvals/{requestId}/reject", s.auth(s.reject))
	mux.HandleFunc("POST /api/approvals/{requestId}/delegate", s.auth(s.delegate))

	mux.HandleFunc("GET /api/audit", s.auth(s.listAudit))
	mux.HandleFunc("GET /api/admin/overview", s.auth(s.getAdminOverview))
	mux.HandleFunc("GET /api/admin/access-role/form", s.auth(s.getAdminAccessRoleForm))
	mux.HandleFunc("GET /api/admin/users/search", s.auth(s.adminSearchUsers))
	mux.HandleFunc("POST /api/admin/access-roles", s.auth(s.createAdminAccessRole))

	mux.HandleFunc("GET /api/delegations", s.auth(s.listDelegations))
	mux.HandleFunc("POST /api/delegations", s.auth(s.createDelegation))
	mux.HandleFunc("DELETE /api/delegations/{id}", s.auth(s.deleteDelegation))
	mux.HandleFunc("GET /api/delegation/candidates", s.auth(s.delegationCandidates))
	mux.HandleFunc("GET /api/delegation/users/search", s.auth(s.delegationSearchUsers))
	mux.HandleFunc("GET /api/delegation/my-access-roles", s.auth(s.delegationMyAccessRoles))

	mux.HandleFunc("POST /api/attachments", s.auth(s.uploadAttachment))
	mux.HandleFunc("GET /api/attachments/{id}", s.auth(s.getAttachment))

	mux.HandleFunc("POST /internal/ai/analyze", s.internalAIAnalyze)
	mux.HandleFunc("GET /internal/iam/user", s.internalIAMUser)

	return mux
}

func (s *Server) health(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]any{
		"status":  "ok",
		"service": "itshop-api",
		"time":    time.Now().UTC().Format(time.RFC3339),
	})
}

func (s *Server) meta(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]any{
		"name":    "IT Shop API",
		"version": "0.2.0",
		"docs":    "docs/api.md",
	})
}

type authed = func(http.ResponseWriter, *http.Request, uuid.UUID)

func (s *Server) auth(h authed) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		tok := bearerToken(r)
		if tok == "" {
			writeAPIError(w, http.StatusUnauthorized, "UNAUTHORIZED", "требуется Bearer-токен")
			return
		}
		cl, err := auth.Parse(s.cfg.JWTSecret, tok)
		if err != nil {
			writeAPIError(w, http.StatusUnauthorized, "UNAUTHORIZED", "недействительный токен")
			return
		}
		uid, err := uuid.Parse(cl.Subject)
		if err != nil {
			writeAPIError(w, http.StatusUnauthorized, "UNAUTHORIZED", "некорректный subject")
			return
		}
		ctx := withUserID(r.Context(), uid)
		h(w, r.WithContext(ctx), uid)
	}
}
