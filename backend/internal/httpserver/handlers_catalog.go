package httpserver

import (
	"errors"
	"net/http"
	"strings"

	"github.com/google/uuid"
	"github.com/itshop/api/internal/pgxutil"
	"github.com/itshop/api/internal/store"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
)

func (s *Server) listSystems(w http.ResponseWriter, r *http.Request, _ uuid.UUID) {
	rows, err := s.q.ListSystems(r.Context())
	if err != nil {
		writeAPIError(w, http.StatusInternalServerError, "INTERNAL", "ошибка БД")
		return
	}
	out := make([]map[string]any, 0, len(rows))
	for _, sys := range rows {
		out = append(out, map[string]any{
			"id":          formatUUID(sys.ID),
			"name":        sys.Name,
			"description": textString(sys.Description),
		})
	}
	writeJSON(w, http.StatusOK, map[string]any{"items": out})
}

func (s *Server) listResources(w http.ResponseWriter, r *http.Request, _ uuid.UUID) {
	sid := r.URL.Query().Get("system_id")
	if sid == "" {
		writeAPIError(w, http.StatusBadRequest, "VALIDATION_ERROR", "query system_id обязателен")
		return
	}
	pg, err := pgxutil.ParseUUID(sid)
	if err != nil {
		writeAPIError(w, http.StatusBadRequest, "VALIDATION_ERROR", "некорректный system_id")
		return
	}
	rows, err := s.q.ListResourcesBySystem(r.Context(), pg)
	if err != nil {
		writeAPIError(w, http.StatusInternalServerError, "INTERNAL", "ошибка БД")
		return
	}
	out := make([]map[string]any, 0, len(rows))
	for _, res := range rows {
		out = append(out, map[string]any{
			"id":          formatUUID(res.ID),
			"system_id":   formatUUID(res.SystemID),
			"name":        res.Name,
			"description": textString(res.Description),
			"owner_id":    formatUUID(res.OwnerID),
			"sensitive":   res.Sensitive,
		})
	}
	writeJSON(w, http.StatusOK, map[string]any{"items": out})
}

func (s *Server) listAccessRoles(w http.ResponseWriter, r *http.Request, _ uuid.UUID) {
	rid := r.URL.Query().Get("resource_id")
	if rid == "" {
		writeAPIError(w, http.StatusBadRequest, "VALIDATION_ERROR", "query resource_id обязателен")
		return
	}
	pg, err := pgxutil.ParseUUID(rid)
	if err != nil {
		writeAPIError(w, http.StatusBadRequest, "VALIDATION_ERROR", "некорректный resource_id")
		return
	}
	rows, err := s.q.ListAccessRolesByResource(r.Context(), pg)
	if err != nil {
		writeAPIError(w, http.StatusInternalServerError, "INTERNAL", "ошибка БД")
		return
	}
	out := make([]map[string]any, 0, len(rows))
	for _, ar := range rows {
		out = append(out, map[string]any{
			"id":          formatUUID(ar.ID),
			"resource_id": formatUUID(ar.ResourceID),
			"name":        ar.Name,
			"description": textString(ar.Description),
			"risk_level":  ar.RiskLevel,
		})
	}
	writeJSON(w, http.StatusOK, map[string]any{"items": out})
}

func (s *Server) catalogTree(w http.ResponseWriter, r *http.Request, _ uuid.UUID) {
	rows, err := s.q.ListCatalogFlat(r.Context())
	if err != nil {
		writeAPIError(w, http.StatusInternalServerError, "INTERNAL", "ошибка БД")
		return
	}

	type roleOut struct {
		ID          string `json:"id"`
		Name        string `json:"name"`
		Description any    `json:"description"`
		RiskLevel   string `json:"risk_level"`
	}
	type resOut struct {
		ID        string    `json:"id"`
		Name      string    `json:"name"`
		Sensitive bool      `json:"sensitive"`
		Roles     []roleOut `json:"roles"`
	}
	type sysOut struct {
		ID          string    `json:"id"`
		Name        string    `json:"name"`
		Description any       `json:"description"`
		Resources   []resOut  `json:"resources"`
	}

	systems := map[string]*sysOut{}
	systemResources := map[string]map[string]*resOut{}
	resourceOrder := map[string][]string{}
	var sysOrder []string
	seenSys := map[string]bool{}

	for _, row := range rows {
		sid := formatUUID(row.SystemID)
		rid := formatUUID(row.ResourceID)
		if !seenSys[sid] {
			seenSys[sid] = true
			sysOrder = append(sysOrder, sid)
			systems[sid] = &sysOut{
				ID:          sid,
				Name:        row.SystemName,
				Description: textString(row.SystemDescription),
			}
			systemResources[sid] = map[string]*resOut{}
			resourceOrder[sid] = nil
		}
		if systemResources[sid][rid] == nil {
			systemResources[sid][rid] = &resOut{
				ID:        rid,
				Name:      row.ResourceName,
				Sensitive: row.ResourceSensitive,
				Roles:     []roleOut{},
			}
			resourceOrder[sid] = append(resourceOrder[sid], rid)
		}
		res := systemResources[sid][rid]
		res.Roles = append(res.Roles, roleOut{
			ID:          formatUUID(row.AccessRoleID),
			Name:        row.RoleName,
			Description: textString(row.RoleDescription),
			RiskLevel:   row.RiskLevel,
		})
	}

	out := make([]sysOut, 0, len(sysOrder))
	for _, sid := range sysOrder {
		sy := systems[sid]
		rids := resourceOrder[sid]
		resources := make([]resOut, 0, len(rids))
		for _, rid := range rids {
			resources = append(resources, *systemResources[sid][rid])
		}
		sy.Resources = resources
		out = append(out, *sy)
	}

	writeJSON(w, http.StatusOK, map[string]any{"items": out})
}

func (s *Server) searchAccessRoles(w http.ResponseWriter, r *http.Request, _ uuid.UUID) {
	q := strings.TrimSpace(r.URL.Query().Get("q"))
	params := store.SearchAccessRolesParams{SearchQ: q}

	if sid := strings.TrimSpace(r.URL.Query().Get("system_id")); sid != "" {
		p, err := pgxutil.ParseUUID(sid)
		if err != nil {
			writeAPIError(w, http.StatusBadRequest, "VALIDATION_ERROR", "некорректный system_id")
			return
		}
		params.FilterSystemID = p
	} else {
		params.FilterSystemID = pgtype.UUID{Valid: false}
	}

	if rid := strings.TrimSpace(r.URL.Query().Get("resource_id")); rid != "" {
		p, err := pgxutil.ParseUUID(rid)
		if err != nil {
			writeAPIError(w, http.StatusBadRequest, "VALIDATION_ERROR", "некорректный resource_id")
			return
		}
		params.FilterResourceID = p
	} else {
		params.FilterResourceID = pgtype.UUID{Valid: false}
	}

	rows, err := s.q.SearchAccessRoles(r.Context(), params)
	if err != nil {
		writeAPIError(w, http.StatusInternalServerError, "INTERNAL", "ошибка БД")
		return
	}
	out := make([]map[string]any, 0, len(rows))
	for _, row := range rows {
		out = append(out, map[string]any{
			"id":            formatUUID(row.AccessRoleID),
			"resource_id":   formatUUID(row.ResourceID),
			"name":          row.RoleName,
			"description":   textString(row.RoleDescription),
			"risk_level":    row.RiskLevel,
			"resource_name": row.ResourceName,
			"sensitive":     row.ResourceSensitive,
			"system_id":     formatUUID(row.SystemID),
			"system_name":   row.SystemName,
		})
	}
	writeJSON(w, http.StatusOK, map[string]any{"items": out})
}

func (s *Server) getAccessRoleDetail(w http.ResponseWriter, r *http.Request, _ uuid.UUID) {
	idStr := r.PathValue("id")
	pg, err := pgxutil.ParseUUID(idStr)
	if err != nil {
		writeAPIError(w, http.StatusBadRequest, "VALIDATION_ERROR", "некорректный id")
		return
	}
	row, err := s.q.GetAccessRoleDetail(r.Context(), pg)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			writeAPIError(w, http.StatusNotFound, "NOT_FOUND", "роль доступа не найдена")
			return
		}
		writeAPIError(w, http.StatusInternalServerError, "INTERNAL", "ошибка БД")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"id":            formatUUID(row.AccessRoleID),
		"resource_id":   formatUUID(row.ResourceID),
		"name":          row.RoleName,
		"description":   textString(row.RoleDescription),
		"risk_level":    row.RiskLevel,
		"resource_name": row.ResourceName,
		"sensitive":     row.ResourceSensitive,
		"system_id":     formatUUID(row.SystemID),
		"system_name":   row.SystemName,
		"is_active":     true,
		"code":          row.RoleName,
	})
}
