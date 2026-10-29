package httpapi

import (
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/tafakorpardazan/huginn/internal/auth"
	"github.com/tafakorpardazan/huginn/internal/followups"
)

// FollowUpGet returns a single follow-up.
func (h *leadHandler) FollowUpGet(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "followupID")
	f, err := h.s.FollowUps.Get(r.Context(), id)
	if err != nil {
		if err == followups.ErrNotFound {
			RespondError(w, http.StatusNotFound, &APIError{Code: CodeNotFound, Message: "follow-up not found"})
			return
		}
		internalErr(w, err)
		return
	}
	RespondOK(w, f)
}

// FollowUpCreateGlobal creates a follow-up for a lead (global entry point).
func (h *leadHandler) FollowUpCreateGlobal(w http.ResponseWriter, r *http.Request) {
	var req struct {
		LeadID        string             `json:"lead_id"`
		ContactID     *string            `json:"contact_id"`
		AssignedTo    *string            `json:"assigned_to"`
		Title         *string            `json:"title"`
		Description   *string            `json:"description"`
		Channel       *string            `json:"channel"`
		Type          string             `json:"type"`
		Priority      followups.Priority `json:"priority"`
		DueAt         time.Time          `json:"due_at"`
		InteractionID *string            `json:"interaction_id"`
	}
	if apiErr := DecodeJSON(r, &req); apiErr != nil {
		RespondError(w, http.StatusBadRequest, apiErr)
		return
	}
	p, _ := auth.FromContext(r.Context())
	f, err := h.s.FollowUps.Create(r.Context(), followups.CreateParams{
		LeadID:        req.LeadID,
		ContactID:     req.ContactID,
		AssignedTo:    req.AssignedTo,
		CreatedBy:     &p.UserID,
		Title:         req.Title,
		Description:   req.Description,
		Channel:       req.Channel,
		Type:          req.Type,
		Priority:      req.Priority,
		DueAt:         req.DueAt,
		InteractionID: req.InteractionID,
	})
	if err != nil {
		RespondError(w, http.StatusBadRequest, &APIError{Code: CodeValidationError, Message: err.Error()})
		return
	}
	RespondJSON(w, http.StatusCreated, f)
}

// FollowUpUpdate updates editable fields (incl. due_at = reschedule).
func (h *leadHandler) FollowUpUpdate(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "followupID")
	var req struct {
		ContactID   *string             `json:"contact_id"`
		AssignedTo  *string             `json:"assigned_to"`
		Title       *string             `json:"title"`
		Description *string             `json:"description"`
		Channel     *string             `json:"channel"`
		Type        *string             `json:"type"`
		Priority    *followups.Priority `json:"priority"`
		DueAt       *time.Time          `json:"due_at"`
	}
	if apiErr := DecodeJSON(r, &req); apiErr != nil {
		RespondError(w, http.StatusBadRequest, apiErr)
		return
	}
	f, err := h.s.FollowUps.Update(r.Context(), id, followups.UpdateParams{
		ContactID:   req.ContactID,
		AssignedTo:  req.AssignedTo,
		Title:       req.Title,
		Description: req.Description,
		Channel:     req.Channel,
		Type:        req.Type,
		Priority:    req.Priority,
		DueAt:       req.DueAt,
	})
	if err != nil {
		if err == followups.ErrNotFound {
			RespondError(w, http.StatusNotFound, &APIError{Code: CodeNotFound, Message: "follow-up not found"})
			return
		}
		RespondError(w, http.StatusBadRequest, &APIError{Code: CodeValidationError, Message: err.Error()})
		return
	}
	RespondOK(w, f)
}

// FollowUpReopen moves a completed follow-up back to pending.
func (h *leadHandler) FollowUpReopen(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "followupID")
	var req struct {
		DueAt *time.Time `json:"due_at"`
	}
	if r.ContentLength > 0 {
		_ = DecodeJSON(r, &req)
	}
	f, err := h.s.FollowUps.Reopen(r.Context(), id, req.DueAt)
	if err != nil {
		if err == followups.ErrNotFound {
			RespondError(w, http.StatusNotFound, &APIError{Code: CodeNotFound, Message: "follow-up not found"})
			return
		}
		RespondError(w, http.StatusBadRequest, &APIError{Code: CodeBadRequest, Message: err.Error()})
		return
	}
	RespondOK(w, f)
}

// FollowUpReschedule moves a follow-up to a new due time (back to pending).
func (h *leadHandler) FollowUpReschedule(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "followupID")
	var req struct {
		DueAt time.Time `json:"due_at"`
	}
	if apiErr := DecodeJSON(r, &req); apiErr != nil {
		RespondError(w, http.StatusBadRequest, apiErr)
		return
	}
	f, err := h.s.FollowUps.Reschedule(r.Context(), id, req.DueAt)
	if err != nil {
		if err == followups.ErrNotFound {
			RespondError(w, http.StatusNotFound, &APIError{Code: CodeNotFound, Message: "follow-up not found"})
			return
		}
		RespondError(w, http.StatusBadRequest, &APIError{Code: CodeValidationError, Message: err.Error()})
		return
	}
	RespondOK(w, f)
}

// FollowUpCancel soft-cancels a follow-up.
func (h *leadHandler) FollowUpCancel(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "followupID")
	f, err := h.s.FollowUps.Cancel(r.Context(), id)
	if err != nil {
		if err == followups.ErrNotFound {
			RespondError(w, http.StatusNotFound, &APIError{Code: CodeNotFound, Message: "follow-up not found"})
			return
		}
		RespondError(w, http.StatusBadRequest, &APIError{Code: CodeBadRequest, Message: err.Error()})
		return
	}
	RespondOK(w, f)
}

// FollowUpDelete hard-deletes a follow-up.
func (h *leadHandler) FollowUpDelete(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "followupID")
	if err := h.s.FollowUps.Delete(r.Context(), id); err != nil {
		RespondError(w, http.StatusBadRequest, &APIError{Code: CodeBadRequest, Message: err.Error()})
		return
	}
	RespondNoContent(w)
}
