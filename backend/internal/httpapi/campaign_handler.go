package httpapi

import (
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/tafakorpardazan/huginn/internal/auth"
	"github.com/tafakorpardazan/huginn/internal/campaigns"
)

type campaignHandler struct {
	s *Services
}

func newCampaignHandler(s *Services) *campaignHandler {
	return &campaignHandler{s: s}
}

// Campaign handlers.

// List returns campaigns with pagination and an optional status filter.
func (h *campaignHandler) List(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()

	params := campaigns.ListParams{
		Page: intParam(q.Get("page"), 1),
		Size: intParam(q.Get("page_size"), 50),
	}
	if v := q.Get("status"); v != "" {
		s := campaigns.Status(v)
		params.Status = &s
	}

	res, err := h.s.Campaigns.List(r.Context(), params)
	if err != nil {
		internalErr(w, err)
		return
	}
	RespondOK(w, res)
}

// Create creates a new campaign.
func (h *campaignHandler) Create(w http.ResponseWriter, r *http.Request) {
	p, _ := auth.FromContext(r.Context())
	var req struct {
		Title       string  `json:"title"`
		Description *string `json:"description"`
		Objective   *string `json:"objective"`
	}
	if apiErr := DecodeJSON(r, &req); apiErr != nil {
		RespondError(w, http.StatusBadRequest, apiErr)
		return
	}
	createdBy := ""
	if p != nil {
		createdBy = p.UserID
	}
	c, err := h.s.Campaigns.Create(r.Context(), campaigns.CreateParams{
		Title:       req.Title,
		Description: req.Description,
		Objective:   req.Objective,
		CreatedBy:   strptr(createdBy),
	})
	if err != nil {
		RespondError(w, http.StatusBadRequest, &APIError{Code: CodeValidationError, Message: err.Error()})
		return
	}
	h.s.Audit.Record(r.Context(), actorEntry(r, "campaign.created", "campaign", c.ID, nil, nil))
	RespondJSON(w, http.StatusCreated, c)
}

// Get returns a single campaign with its attached projects.
func (h *campaignHandler) Get(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "campaignID")
	c, err := h.s.Campaigns.Get(r.Context(), id)
	if err != nil {
		RespondError(w, http.StatusNotFound, &APIError{Code: CodeNotFound, Message: "campaign not found"})
		return
	}
	projects, _ := h.s.Campaigns.ListProjects(r.Context(), id)
	if projects == nil {
		projects = []*campaigns.ProjectRef{}
	}
	RespondOK(w, map[string]any{"campaign": c, "projects": projects})
}

// Update updates an existing campaign.
func (h *campaignHandler) Update(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "campaignID")
	var req struct {
		Title       *string `json:"title"`
		Description *string `json:"description"`
		Objective   *string `json:"objective"`
		Status      *string `json:"status"`
	}
	if apiErr := DecodeJSON(r, &req); apiErr != nil {
		RespondError(w, http.StatusBadRequest, apiErr)
		return
	}
	params := campaigns.UpdateParams{
		Title:       req.Title,
		Description: req.Description,
		Objective:   req.Objective,
	}
	if req.Status != nil {
		s := campaigns.Status(*req.Status)
		params.Status = &s
	}
	c, err := h.s.Campaigns.Update(r.Context(), id, params)
	if err != nil {
		if err == campaigns.ErrNotFound {
			RespondError(w, http.StatusNotFound, &APIError{Code: CodeNotFound, Message: "campaign not found"})
			return
		}
		RespondError(w, http.StatusBadRequest, &APIError{Code: CodeValidationError, Message: err.Error()})
		return
	}
	h.s.Audit.Record(r.Context(), actorEntry(r, "campaign.updated", "campaign", id, nil, nil))
	RespondOK(w, c)
}

// Delete deletes a campaign. Its projects are preserved (campaign_id set NULL).
func (h *campaignHandler) Delete(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "campaignID")
	if err := h.s.Campaigns.Delete(r.Context(), id); err != nil {
		if err == campaigns.ErrNotFound {
			RespondError(w, http.StatusNotFound, &APIError{Code: CodeNotFound, Message: "campaign not found"})
			return
		}
		internalErr(w, err)
		return
	}
	h.s.Audit.Record(r.Context(), actorEntry(r, "campaign.deleted", "campaign", id, nil, nil))
	RespondNoContent(w)
}
