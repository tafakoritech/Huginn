package httpapi

import (
	"context"
	"errors"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/tafakorpardazan/huginn/internal/auth"
	"github.com/tafakorpardazan/huginn/internal/leads"
	"github.com/tafakorpardazan/huginn/internal/projects"
)

type projectHandler struct {
	s *Services
}

func newProjectHandler(s *Services) *projectHandler {
	return &projectHandler{s: s}
}

// Project handlers.

// List returns projects with lead counts and a paginated, filtered list.
func (h *projectHandler) List(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()

	params := projects.ListParams{
		Page: intParam(q.Get("page"), 1),
		Size: intParam(q.Get("page_size"), 50),
	}
	if v := q.Get("status"); v != "" {
		s := projects.Status(v)
		params.Status = &s
	}
	if v := q.Get("campaign_id"); v != "" {
		params.CampaignID = &v
	}

	res, err := h.s.Projects.List(r.Context(), params)
	if err != nil {
		internalErr(w, err)
		return
	}
	RespondOK(w, res)
}

// Create creates a new project.
func (h *projectHandler) Create(w http.ResponseWriter, r *http.Request) {
	p, _ := auth.FromContext(r.Context())
	var req struct {
		Title                string  `json:"title"`
		Description          string  `json:"description"`
		ProductName          *string `json:"product_name"`
		ProductDescription   *string `json:"product_description"`
		TargetMarket         *string `json:"target_market"`
		IdealCustomerProfile *string `json:"ideal_customer_profile"`
		CustomerPainPoints   *string `json:"customer_pain_points"`
		ValueProposition     *string `json:"value_proposition"`
		SalesObjective       *string `json:"sales_objective"`
		CommonObjections     *string `json:"common_objections"`
		CampaignID           *string `json:"campaign_id"`
	}
	if apiErr := DecodeJSON(r, &req); apiErr != nil {
		RespondError(w, http.StatusBadRequest, apiErr)
		return
	}
	campaignID, apiErr := h.resolveCampaign(r.Context(), req.CampaignID)
	if apiErr != nil {
		status, resp := ErrorFrom(apiErr)
		RespondError(w, status, resp)
		return
	}
	createdBy := ""
	if p != nil {
		createdBy = p.UserID
	}
	proj, err := h.s.Projects.Create(r.Context(), projects.CreateParams{
		Title:                req.Title,
		Description:          req.Description,
		ProductName:          req.ProductName,
		ProductDescription:   req.ProductDescription,
		TargetMarket:         req.TargetMarket,
		IdealCustomerProfile: req.IdealCustomerProfile,
		CustomerPainPoints:   req.CustomerPainPoints,
		ValueProposition:     req.ValueProposition,
		SalesObjective:       req.SalesObjective,
		CommonObjections:     req.CommonObjections,
		CreatedBy:            strptr(createdBy),
		CampaignID:           campaignID,
	})
	if err != nil {
		RespondError(w, http.StatusBadRequest, &APIError{Code: CodeValidationError, Message: err.Error()})
		return
	}
	h.s.Audit.Record(r.Context(), actorEntry(r, "project.created", "project", proj.ID, nil, nil))
	RespondJSON(w, http.StatusCreated, h.projectDetail(r, proj))
}

// Get returns a single project with related leads and sales activities.
func (h *projectHandler) Get(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "projectID")
	proj, err := h.s.Projects.Get(r.Context(), id)
	if err != nil {
		RespondError(w, http.StatusNotFound, &APIError{Code: CodeNotFound, Message: "project not found"})
		return
	}
	RespondOK(w, h.projectDetail(r, proj))
}

// Update updates an existing project. Changes affect future AI generations
// because scenarios are regenerated from the latest project context.
func (h *projectHandler) Update(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "projectID")
	var req struct {
		Title                string  `json:"title"`
		Description          string  `json:"description"`
		ProductName          *string `json:"product_name"`
		ProductDescription   *string `json:"product_description"`
		TargetMarket         *string `json:"target_market"`
		IdealCustomerProfile *string `json:"ideal_customer_profile"`
		CustomerPainPoints   *string `json:"customer_pain_points"`
		ValueProposition     *string `json:"value_proposition"`
		SalesObjective       *string `json:"sales_objective"`
		CommonObjections     *string `json:"common_objections"`
		Status               *string `json:"status"`
		CampaignID           *string `json:"campaign_id"`
	}
	if apiErr := DecodeJSON(r, &req); apiErr != nil {
		RespondError(w, http.StatusBadRequest, apiErr)
		return
	}
	campaignID, apiErr := h.resolveCampaign(r.Context(), req.CampaignID)
	if apiErr != nil {
		status, resp := ErrorFrom(apiErr)
		RespondError(w, status, resp)
		return
	}
	var status *projects.Status
	if req.Status != nil {
		s := projects.Status(*req.Status)
		status = &s
	}
	proj, err := h.s.Projects.Update(r.Context(), id, projects.UpdateParams{
		Title:                req.Title,
		Description:          req.Description,
		ProductName:          req.ProductName,
		ProductDescription:   req.ProductDescription,
		TargetMarket:         req.TargetMarket,
		IdealCustomerProfile: req.IdealCustomerProfile,
		CustomerPainPoints:   req.CustomerPainPoints,
		ValueProposition:     req.ValueProposition,
		SalesObjective:       req.SalesObjective,
		CommonObjections:     req.CommonObjections,
		Status:               status,
		CampaignID:           campaignID,
	})
	if err != nil {
		if err == projects.ErrNotFound {
			RespondError(w, http.StatusNotFound, &APIError{Code: CodeNotFound, Message: "project not found"})
			return
		}
		RespondError(w, http.StatusBadRequest, &APIError{Code: CodeValidationError, Message: err.Error()})
		return
	}
	h.s.Timeline.Add(r.Context(), "", "project.updated", "Project updated", nil, nil, map[string]any{"project_id": id})
	h.s.Audit.Record(r.Context(), actorEntry(r, "project.updated", "project", id, nil, nil))
	RespondOK(w, h.projectDetail(r, proj))
}

// Delete deletes a project. Associated leads keep their row (project_id set
// NULL) so historical data is preserved, but new leads cannot reference it.
func (h *projectHandler) Delete(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "projectID")
	if err := h.s.Projects.Delete(r.Context(), id); err != nil {
		if err == projects.ErrNotFound {
			RespondError(w, http.StatusNotFound, &APIError{Code: CodeNotFound, Message: "project not found"})
			return
		}
		internalErr(w, err)
		return
	}
	h.s.Audit.Record(r.Context(), actorEntry(r, "project.deleted", "project", id, nil, nil))
	RespondNoContent(w)
}

// AttachLead links an existing lead to this project (lead.project_id = project.id).
// The lead itself is not modified beyond the association; the change is versioned
// in the lead's change history and updated_at is refreshed.
func (h *projectHandler) AttachLead(w http.ResponseWriter, r *http.Request) {
	projectID := chi.URLParam(r, "projectID")
	leadID := chi.URLParam(r, "leadID")
	p, _ := auth.FromContext(r.Context())
	actorID := ""
	if p != nil {
		actorID = p.UserID
	}

	if _, err := h.s.Projects.Get(r.Context(), projectID); err != nil {
		RespondError(w, http.StatusNotFound, &APIError{Code: CodeNotFound, Message: "project not found"})
		return
	}
	if _, err := h.s.Leads.Get(r.Context(), leadID); err != nil {
		RespondError(w, http.StatusNotFound, &APIError{Code: CodeNotFound, Message: "lead not found"})
		return
	}

	_, err := h.s.Leads.Update(r.Context(), leadID, map[string]*string{"project_id": &projectID}, actorID, "attached to project")
	if err != nil {
		if errors.Is(err, leads.ErrNotFound) {
			RespondError(w, http.StatusNotFound, &APIError{Code: CodeNotFound, Message: "lead not found"})
			return
		}
		RespondError(w, http.StatusBadRequest, &APIError{Code: CodeValidationError, Message: err.Error()})
		return
	}
	h.s.Timeline.Add(r.Context(), leadID, "lead.project.attached", "Lead attached to project", nil, strptr(actorID), map[string]any{"project_id": projectID})
	h.s.Audit.Record(r.Context(), actorEntry(r, "lead.project.attached", "lead", leadID, nil, map[string]any{"project_id": projectID}))
	RespondJSON(w, http.StatusOK, map[string]any{"success": true, "lead_id": leadID, "project_id": projectID})
}

// DetachLead removes a lead from this project (lead.project_id = NULL). The
// lead record is preserved — only the association is dropped.
func (h *projectHandler) DetachLead(w http.ResponseWriter, r *http.Request) {
	projectID := chi.URLParam(r, "projectID")
	leadID := chi.URLParam(r, "leadID")
	p, _ := auth.FromContext(r.Context())
	actorID := ""
	if p != nil {
		actorID = p.UserID
	}

	if _, err := h.s.Projects.Get(r.Context(), projectID); err != nil {
		RespondError(w, http.StatusNotFound, &APIError{Code: CodeNotFound, Message: "project not found"})
		return
	}
	if _, err := h.s.Leads.Get(r.Context(), leadID); err != nil {
		RespondError(w, http.StatusNotFound, &APIError{Code: CodeNotFound, Message: "lead not found"})
		return
	}

	_, err := h.s.Leads.Update(r.Context(), leadID, map[string]*string{"project_id": nil}, actorID, "detached from project")
	if err != nil {
		if errors.Is(err, leads.ErrNotFound) {
			RespondError(w, http.StatusNotFound, &APIError{Code: CodeNotFound, Message: "lead not found"})
			return
		}
		RespondError(w, http.StatusBadRequest, &APIError{Code: CodeValidationError, Message: err.Error()})
		return
	}
	h.s.Timeline.Add(r.Context(), leadID, "lead.project.detached", "Lead detached from project", nil, strptr(actorID), map[string]any{"project_id": projectID})
	h.s.Audit.Record(r.Context(), actorEntry(r, "lead.project.detached", "lead", leadID, nil, map[string]any{"project_id": projectID}))
	RespondJSON(w, http.StatusOK, map[string]any{"success": true, "lead_id": leadID, "project_id": nil})
}

// resolveCampaign validates an optional campaign reference for a project. It
// returns the normalized campaign id to persist (nil when absent/empty) or an
// APIError when the id is not a valid UUID (400) or the campaign does not exist
// (404). Authorization for campaign access is enforced by the route-level RBAC
// middleware; campaigns carry no per-record tenant ownership in this codebase.
func (h *projectHandler) resolveCampaign(ctx context.Context, id *string) (*string, *APIError) {
	if id == nil || *id == "" {
		return nil, nil
	}
	if _, err := uuid.Parse(*id); err != nil {
		return nil, &APIError{Code: CodeValidationError, Message: "campaign_id must be a valid UUID"}
	}
	if _, err := h.s.Campaigns.Get(ctx, *id); err != nil {
		return nil, &APIError{Code: CodeNotFound, Message: "campaign not found"}
	}
	return id, nil
}

// projectDetail builds the enriched project response including lead count,
// related leads and generated sales activities (call scenarios).
func (h *projectHandler) projectDetail(r *http.Request, proj *projects.Project) map[string]any {
	count, _ := h.s.Projects.LeadCount(r.Context(), proj.ID)

	relatedLeads, _ := h.s.LeadRepo.List(r.Context(), leads.ListParams{ProjectID: &proj.ID, Page: 1, PageSize: 50}, "sales_development_manager", "")
	populateOwnerNames(r.Context(), h.s.DB, relatedLeads.Leads)

	// Generated sales activities (call scenarios) scoped to this project's leads.
	var scenarios []map[string]any
	if h.s.CallScenarios != nil && h.s.DB != nil {
		rows, err := h.s.DB.Pool().Query(r.Context(), `
			SELECT cs.id, cs.lead_id, cs.contact_id, cs.is_generated, cs.created_at, cs.opening_sentence
			FROM call_scenarios cs
			JOIN leads l ON l.id = cs.lead_id
			WHERE l.project_id = $1
			ORDER BY cs.created_at DESC LIMIT 50`, proj.ID)
		if err == nil {
			defer rows.Close()
			for rows.Next() {
				var scID, leadID, contactID, opening string
				var isGen bool
				var createdAt interface{}
				if err := rows.Scan(&scID, &leadID, &contactID, &isGen, &createdAt, &opening); err == nil {
					scenarios = append(scenarios, map[string]any{
						"id": scID, "lead_id": leadID, "contact_id": contactID,
						"is_generated": isGen, "created_at": createdAt, "opening_sentence": opening,
					})
				}
			}
		}
	}
	if scenarios == nil {
		scenarios = []map[string]any{}
	}

	return map[string]any{
		"project":          proj,
		"lead_count":       count,
		"related_leads":    relatedLeads,
		"sales_activities": scenarios,
	}
}
