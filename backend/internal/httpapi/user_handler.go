package httpapi

import (
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/tafakorpardazan/huginn/internal/audit"
	"github.com/tafakorpardazan/huginn/internal/auth"
	"github.com/tafakorpardazan/huginn/internal/rbac"
	"github.com/tafakorpardazan/huginn/internal/users"
)

type userHandler struct {
	users *users.Service
	auth  *auth.Middleware
	audit *audit.Logger
}

func newUserHandler(users *users.Service, auth *auth.Middleware, audit *audit.Logger) *userHandler {
	return &userHandler{users: users, auth: auth, audit: audit}
}

func (h *userHandler) Create(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Name     string  `json:"name"`
		Username string  `json:"username"`
		Email    *string `json:"email"`
		Mobile   *string `json:"mobile"`
		Password string  `json:"password"`
		Role     rbac.Role `json:"role"`
	}
	if apiErr := DecodeJSON(r, &req); apiErr != nil {
		RespondError(w, http.StatusBadRequest, apiErr)
		return
	}
	u, err := h.users.Create(r.Context(), users.CreateParams{
		Name: req.Name, Username: req.Username, Email: req.Email, Mobile: req.Mobile,
		Password: req.Password, Role: req.Role,
	})
	if err != nil {
		RespondError(w, http.StatusBadRequest, &APIError{Code: CodeValidationError, Message: err.Error()})
		return
	}
	h.audit.Record(r.Context(), actorEntry(r, "user.created", "user", u.ID, nil, map[string]any{"role": u.Role}))
	RespondJSON(w, http.StatusCreated, u)
}

func (h *userHandler) List(w http.ResponseWriter, r *http.Request) {
	u, err := h.users.List(r.Context(), nil, nil)
	if err != nil {
		internalErr(w, err)
		return
	}
	RespondOK(w, map[string]any{"users": u, "count": len(u)})
}

func (h *userHandler) Get(w http.ResponseWriter, r *http.Request) {
	u, err := h.users.Get(r.Context(), chi.URLParam(r, "userID"))
	if err != nil {
		RespondError(w, http.StatusNotFound, &APIError{Code: CodeNotFound, Message: "user not found"})
		return
	}
	RespondOK(w, u)
}

func (h *userHandler) Update(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "userID")
	var req struct {
		Name   string    `json:"name"`
		Email  *string   `json:"email"`
		Mobile *string   `json:"mobile"`
		Role   rbac.Role `json:"role"`
	}
	if apiErr := DecodeJSON(r, &req); apiErr != nil {
		RespondError(w, http.StatusBadRequest, apiErr)
		return
	}
	before, _ := h.users.Get(r.Context(), id)
	u, err := h.users.Update(r.Context(), id, users.UpdateParams{Name: req.Name, Email: req.Email, Mobile: req.Mobile, Role: req.Role})
	if err != nil {
		RespondError(w, http.StatusBadRequest, &APIError{Code: CodeValidationError, Message: err.Error()})
		return
	}
	h.audit.Record(r.Context(), actorEntry(r, "user.updated", "user", id, beforeMap(before), map[string]any{"role": u.Role}))
	RespondOK(w, u)
}

func (h *userHandler) Deactivate(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "userID")
	if err := h.users.Deactivate(r.Context(), id); err != nil {
		RespondError(w, http.StatusNotFound, &APIError{Code: CodeNotFound, Message: "user not found"})
		return
	}
	h.audit.Record(r.Context(), actorEntry(r, "user.deactivated", "user", id, nil, nil))
	RespondNoContent(w)
}

func (h *userHandler) Reactivate(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "userID")
	if err := h.users.Reactivate(r.Context(), id); err != nil {
		RespondError(w, http.StatusNotFound, &APIError{Code: CodeNotFound, Message: "user not found"})
		return
	}
	h.audit.Record(r.Context(), actorEntry(r, "user.reactivated", "user", id, nil, nil))
	RespondNoContent(w)
}

func (h *userHandler) ResetPassword(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "userID")
	var req struct {
		Password string `json:"password"`
	}
	if apiErr := DecodeJSON(r, &req); apiErr != nil {
		RespondError(w, http.StatusBadRequest, apiErr)
		return
	}
	if err := h.users.ResetPassword(r.Context(), id, req.Password); err != nil {
		RespondError(w, http.StatusBadRequest, &APIError{Code: CodeValidationError, Message: err.Error()})
		return
	}
	h.audit.Record(r.Context(), actorEntry(r, "user.password_reset", "user", id, nil, nil))
	RespondNoContent(w)
}

// ListSessions returns the active sessions of a user.
func (h *userHandler) ListSessions(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "userID")
	sessions, err := h.users.ListSessions(r.Context(), id)
	if err != nil {
		internalErr(w, err)
		return
	}
	if sessions == nil {
		sessions = []*users.Session{}
	}
	RespondOK(w, map[string]any{"sessions": sessions, "count": len(sessions)})
}

// RevokeSession revokes a single session.
func (h *userHandler) RevokeSession(w http.ResponseWriter, r *http.Request) {
	userID := chi.URLParam(r, "userID")
	sessionID := chi.URLParam(r, "sessionID")
	if err := h.users.RevokeSession(r.Context(), userID, sessionID); err != nil {
		RespondError(w, http.StatusBadRequest, &APIError{Code: CodeBadRequest, Message: err.Error()})
		return
	}
	RespondNoContent(w)
}
