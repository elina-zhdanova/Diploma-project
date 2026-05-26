package httpserver

import (
	"encoding/json"
	"errors"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/itshop/api/internal/auth"
	"github.com/itshop/api/internal/pgxutil"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"golang.org/x/crypto/bcrypt"
)

type loginReq struct {
	Login    string `json:"login"`
	Password string `json:"password"`
}

func (s *Server) login(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeAPIError(w, http.StatusMethodNotAllowed, "METHOD_NOT_ALLOWED", "ожидался POST")
		return
	}
	var body loginReq
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeAPIError(w, http.StatusBadRequest, "VALIDATION_ERROR", "некорректный JSON")
		return
	}
	body.Login = strings.TrimSpace(body.Login)
	if body.Login == "" || body.Password == "" {
		writeAPIError(w, http.StatusBadRequest, "VALIDATION_ERROR", "login и password обязательны")
		return
	}
	u, err := s.q.GetUserByLogin(r.Context(), body.Login)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			writeAPIError(w, http.StatusUnauthorized, "INVALID_CREDENTIALS", "неверный логин или пароль")
			return
		}
		log.Printf("auth login GetUserByLogin: %v", err)
		writeAPIError(w, http.StatusInternalServerError, "INTERNAL", "ошибка БД")
		return
	}
	if !u.PasswordHash.Valid {
		writeAPIError(w, http.StatusUnauthorized, "INVALID_CREDENTIALS", "пароль не задан")
		return
	}
	if err := bcrypt.CompareHashAndPassword([]byte(u.PasswordHash.String), []byte(body.Password)); err != nil {
		writeAPIError(w, http.StatusUnauthorized, "INVALID_CREDENTIALS", "неверный логин или пароль")
		return
	}
	gid, _ := pgxutil.ToGoogle(u.ID)
	roles, err := s.q.ListRbacNamesForUser(r.Context(), u.ID)
	if err != nil {
		log.Printf("auth login ListRbacNamesForUser: %v", err)
		writeAPIError(w, http.StatusInternalServerError, "INTERNAL", "не удалось загрузить роли")
		return
	}
	tok, err := auth.Sign(s.cfg.JWTSecret, gid, u.Login, roles, 24*time.Hour)
	if err != nil {
		writeAPIError(w, http.StatusInternalServerError, "INTERNAL", "не удалось выдать токен")
		return
	}
	s.audit(r.Context(), gid, nil, "auth.login", map[string]any{"login": body.Login})
	writeJSON(w, http.StatusOK, map[string]any{
		"access_token": tok,
		"token_type":   "Bearer",
		"expires_in":   86400,
	})
}

func (s *Server) getMe(w http.ResponseWriter, r *http.Request, uid uuid.UUID) {
	pg := pgxutil.UUID(uid)
	u, err := s.q.GetUserByID(r.Context(), pg)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			writeAPIError(w, http.StatusNotFound, "NOT_FOUND", "пользователь не найден")
			return
		}
		log.Printf("auth me GetUserByID: %v", err)
		writeAPIError(w, http.StatusInternalServerError, "INTERNAL", "ошибка БД")
		return
	}
	roles, err := s.q.ListRbacNamesForUser(r.Context(), pg)
	if err != nil {
		log.Printf("auth me ListRbacNamesForUser: %v", err)
		writeAPIError(w, http.StatusInternalServerError, "INTERNAL", "не удалось загрузить роли")
		return
	}
	canDelegate, err := s.q.UserIsAccessRoleApprover(r.Context(), pg)
	if err != nil {
		log.Printf("auth me UserIsAccessRoleApprover: %v", err)
		writeAPIError(w, http.StatusInternalServerError, "INTERNAL", "ошибка БД")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"id":                   formatUUID(u.ID),
		"login":                u.Login,
		"full_name":            u.FullName,
		"email":                u.Email,
		"department_id":        formatUUID(u.DepartmentID),
		"position":             u.Position,
		"is_active":            u.IsActive,
		"created_at":           formatTimePtr(u.CreatedAt),
		"roles":                roles,
		"can_delegate":         canDelegate,
		"iam_catalog_emulated": s.cfg.IAMMock,
		"ai_rules_emulated":    s.cfg.AIMock,
	})
}
