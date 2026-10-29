package httpapi

import (
	"net/http"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/tafakorpardazan/huginn/internal/audit"
	"github.com/tafakorpardazan/huginn/internal/auth"
	"github.com/tafakorpardazan/huginn/internal/followups"
	"github.com/tafakorpardazan/huginn/internal/nextactions"
)

// nextActionHandler serves the Next Best Action / Work Queue endpoints. The
// backend is the single source of truth for ranking; Android and Dashboard only
// render the ordered items it returns.
type nextActionHandler struct {
	na        *nextactions.Service
	followUps *followups.Service
	audit     *audit.Logger
}

func newNextActionHandler(na *nextactions.Service, fu *followups.Service, a *audit.Logger) *nextActionHandler {
	return &nextActionHandler{na: na, followUps: fu, audit: a}
}

// MyActions returns the authenticated user's ranked work queue. `limit` bounds
// the returned items (default 3, matching the home "کار بعدی" card); counts are
// always computed over the full actionable set.
func (h *nextActionHandler) MyActions(w http.ResponseWriter, r *http.Request) {
	p, _ := auth.FromContext(r.Context())
	limit := intParam(r.URL.Query().Get("limit"), 3)
	resp, err := h.na.Build(r.Context(), nextactions.Params{UserID: p.UserID, Limit: limit})
	if err != nil {
		internalErr(w, err)
		return
	}
	RespondOK(w, resp)
}

// TeamActions returns the manager view across all agents. RBAC is enforced by
// the router (dashboard.view permission).
func (h *nextActionHandler) TeamActions(w http.ResponseWriter, r *http.Request) {
	limit := intParam(r.URL.Query().Get("limit"), 200)
	resp, err := h.na.Build(r.Context(), nextactions.Params{Team: true, Limit: limit})
	if err != nil {
		internalErr(w, err)
		return
	}
	RespondOK(w, resp)
}

// Snooze reschedules a follow-up-backed action to a later time. It updates the
// real follow-up record (never local UI state). Only follow_up items can be
// snoozed; pending reports must be completed.
func (h *nextActionHandler) Snooze(w http.ResponseWriter, r *http.Request) {
	actionID := chi.URLParam(r, "actionID")
	var req struct {
		Until time.Time `json:"until"`
	}
	if apiErr := DecodeJSON(r, &req); apiErr != nil {
		RespondError(w, http.StatusBadRequest, apiErr)
		return
	}
	if req.Until.IsZero() {
		RespondError(w, http.StatusBadRequest, &APIError{Code: CodeValidationError, Message: "until is required"})
		return
	}
	sourceType, sourceID := splitActionID(actionID)
	if sourceType != string(nextactions.SourceFollowUp) {
		RespondError(w, http.StatusBadRequest, &APIError{Code: CodeValidationError, Message: "only follow-up actions can be snoozed"})
		return
	}
	if _, err := h.followUps.Reschedule(r.Context(), sourceID, req.Until.UTC()); err != nil {
		if err == followups.ErrNotFound {
			RespondError(w, http.StatusNotFound, &APIError{Code: CodeNotFound, Message: "follow-up not found"})
			return
		}
		internalErr(w, err)
		return
	}
	if h.audit != nil {
		h.audit.Record(r.Context(), actorEntry(r, "next_action_snoozed", "follow_up", sourceID, nil, map[string]any{"until": req.Until.UTC().Format(time.RFC3339)}))
	}
	RespondOK(w, map[string]any{"snoozed": true, "action_id": actionID, "until": req.Until.UTC()})
}

// Complete maps a follow-up-backed action to a completed follow-up record.
// Pending-report completion must go through report submission, not this route.
func (h *nextActionHandler) Complete(w http.ResponseWriter, r *http.Request) {
	actionID := chi.URLParam(r, "actionID")
	var req struct {
		Note    *string `json:"note"`
		Outcome *string `json:"outcome"`
	}
	if r.ContentLength > 0 {
		_ = DecodeJSON(r, &req)
	}
	p, _ := auth.FromContext(r.Context())
	sourceType, sourceID := splitActionID(actionID)
	if sourceType != string(nextactions.SourceFollowUp) {
		RespondError(w, http.StatusBadRequest, &APIError{Code: CodeValidationError, Message: "this action cannot be completed here"})
		return
	}
	if _, err := h.followUps.Complete(r.Context(), sourceID, req.Note, req.Outcome); err != nil {
		if err == followups.ErrNotFound {
			RespondError(w, http.StatusNotFound, &APIError{Code: CodeNotFound, Message: "follow-up not found"})
			return
		}
		internalErr(w, err)
		return
	}
	if h.audit != nil {
		h.audit.Record(r.Context(), actorEntry(r, "next_action_completed", "follow_up", sourceID, nil, map[string]any{"by": p.UserID}))
	}
	RespondOK(w, map[string]any{"completed": true, "action_id": actionID})
}

// splitActionID splits "<source_type>:<source_id>" back into its parts.
func splitActionID(actionID string) (string, string) {
	if i := strings.Index(actionID, ":"); i >= 0 {
		return actionID[:i], actionID[i+1:]
	}
	return actionID, actionID
}
