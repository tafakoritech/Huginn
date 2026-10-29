package httpapi

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/tafakorpardazan/huginn/internal/auth"
	"github.com/tafakorpardazan/huginn/internal/callscenarios"
	"github.com/tafakorpardazan/huginn/internal/followups"
	"github.com/tafakorpardazan/huginn/internal/interactions"
	"github.com/tafakorpardazan/huginn/internal/leads"
	"github.com/tafakorpardazan/huginn/internal/leadversions"
	"github.com/tafakorpardazan/huginn/internal/messaging"
	"github.com/tafakorpardazan/huginn/internal/muninn"
	"github.com/tafakorpardazan/huginn/internal/notes"
	"github.com/tafakorpardazan/huginn/internal/qualification"
	"github.com/tafakorpardazan/huginn/internal/reports"
)

type leadHandler struct {
	s *Services
}

func newLeadHandler(s *Services) *leadHandler {
	return &leadHandler{s: s}
}

// ---- Listing & creation ----

func (h *leadHandler) List(w http.ResponseWriter, r *http.Request) {
	p, _ := auth.FromContext(r.Context())
	q := r.URL.Query()

	params := leads.ListParams{
		Scope:    q.Get("scope"),
		Page:     intParam(q.Get("page"), 1),
		PageSize: intParam(q.Get("page_size"), 50),
		Sort:     q.Get("sort"),
		Order:    q.Get("order"),
	}
	if v := q.Get("status"); v != "" {
		s := leads.Status(v)
		params.Status = &s
	}
	if v := q.Get("owner"); v != "" {
		params.Owner = &v
	}
	if v := q.Get("city"); v != "" {
		params.City = &v
	}
	if v := q.Get("province"); v != "" {
		params.Province = &v
	}
	if v := q.Get("industry"); v != "" {
		params.Industry = &v
	}
	if v := q.Get("business_type"); v != "" {
		params.BusinessType = &v
	}
	if v := q.Get("source"); v != "" {
		params.Source = &v
	}
	if v := q.Get("batch_id"); v != "" {
		params.BatchID = &v
	}
	if v := q.Get("document_id"); v != "" {
		params.DocID = &v
	}
	if v := q.Get("project_id"); v != "" {
		params.ProjectID = &v
	}
	if v := q.Get("tag"); v != "" {
		params.Tag = &v
	}
	if v := q.Get("search"); v != "" {
		params.Search = &v
	}
	if v := q.Get("min_score"); v != "" {
		n := atoi(v)
		params.MinScore = &n
	}
	if v := q.Get("max_score"); v != "" {
		n := atoi(v)
		params.MaxScore = &n
	}
	if v := q.Get("qualified"); v != "" {
		b := v == "true"
		params.Qualified = &b
	}

	res, err := h.s.Leads.List(r.Context(), params, string(p.Role), p.UserID)
	if err != nil {
		internalErr(w, err)
		return
	}
	h.populateOwnerNames(r.Context(), res.Leads)
	RespondOK(w, res)
}

func (h *leadHandler) Create(w http.ResponseWriter, r *http.Request) {
	var req struct {
		CompanyName      string  `json:"company_name"`
		DisplayName      *string `json:"display_name"`
		BusinessType     *string `json:"business_type"`
		Industry         *string `json:"industry"`
		Description      *string `json:"description"`
		Province         *string `json:"province"`
		City             *string `json:"city"`
		Address          *string `json:"address"`
		Website          *string `json:"website"`
		Instagram        *string `json:"instagram"`
		Source           *string `json:"source"`
		SourceBatchID    *string `json:"source_batch_id"`
		SourceDocumentID *string `json:"source_document_id"`
		ProjectID        *string `json:"project_id"`
	}
	if apiErr := DecodeJSON(r, &req); apiErr != nil {
		RespondError(w, http.StatusBadRequest, apiErr)
		return
	}
	// project_id is optional at creation (leads may be imported without one),
	// but when provided it must reference an existing project.
	if req.ProjectID != nil && *req.ProjectID != "" && h.s.Projects != nil {
		if _, err := h.s.Projects.Get(r.Context(), *req.ProjectID); err != nil {
			RespondError(w, http.StatusBadRequest, &APIError{Code: CodeValidationError, Message: "project_id does not exist"})
			return
		}
	}
	lead, err := h.s.Leads.Create(r.Context(), leads.CreateParams{
		CompanyName: req.CompanyName, DisplayName: req.DisplayName, BusinessType: req.BusinessType,
		Industry: req.Industry, Description: req.Description, Province: req.Province, City: req.City,
		Address: req.Address, Website: req.Website, Instagram: req.Instagram, Source: req.Source,
		SourceBatchID: req.SourceBatchID, SourceDocumentID: req.SourceDocumentID, ProjectID: req.ProjectID,
	})
	if err != nil {
		RespondError(w, http.StatusBadRequest, &APIError{Code: CodeValidationError, Message: err.Error()})
		return
	}
	h.s.Timeline.Add(r.Context(), lead.ID, "lead.created", "Lead created", nil, nil, nil)
	RespondJSON(w, http.StatusCreated, h.leadDetail(r, lead))
}

// ---- Detail & update ----

func (h *leadHandler) Get(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "leadID")
	lead, err := h.s.Leads.Get(r.Context(), id)
	if err != nil {
		RespondError(w, http.StatusNotFound, &APIError{Code: CodeNotFound, Message: "lead not found"})
		return
	}
	RespondOK(w, h.leadDetail(r, lead))
}

func (h *leadHandler) Update(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "leadID")
	p, _ := auth.FromContext(r.Context())

	var raw map[string]json.RawMessage
	if apiErr := DecodeJSON(r, &raw); apiErr != nil {
		RespondError(w, http.StatusBadRequest, apiErr)
		return
	}
	patches := map[string]*string{}
	for k, v := range raw {
		if k == "note" || k == "reason" {
			continue
		}
		var sv *string
		_ = json.Unmarshal(v, &sv) // explicit null clears the field
		patches[k] = sv
	}
	note := bodyNote(raw)

	lead, err := h.s.Leads.Update(r.Context(), id, patches, p.UserID, note)
	if err != nil {
		if errors.Is(err, leads.ErrNotFound) {
			RespondError(w, http.StatusNotFound, &APIError{Code: CodeNotFound, Message: "lead not found"})
			return
		}
		if errors.Is(err, leads.ErrInvalidState) {
			RespondError(w, http.StatusConflict, &APIError{Code: CodeInvalidTransition, Message: err.Error()})
			return
		}
		RespondError(w, http.StatusBadRequest, &APIError{Code: CodeValidationError, Message: err.Error()})
		return
	}
	h.s.Timeline.Add(r.Context(), id, "lead.updated", "Lead details updated", strptr(note), &p.UserID, nil)
	h.s.Audit.Record(r.Context(), actorEntry(r, "lead.updated", "lead", id, nil, map[string]any{"fields": len(patches), "note": note}))
	RespondOK(w, h.leadDetail(r, lead))
}

// ---- Versioned change history ----

// Versions returns the immutable edit history (versions) of a lead, newest
// first. Requires the leads.history permission (admins/managers only).
func (h *leadHandler) Versions(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "leadID")
	limit := intParam(r.URL.Query().Get("limit"), 100)
	offset := intParam(r.URL.Query().Get("offset"), 0)
	versions, err := h.s.Leads.ListVersions(r.Context(), id, limit, offset)
	if err != nil {
		internalErr(w, err)
		return
	}
	if versions == nil {
		versions = []*leadversions.Version{}
	}
	count, _ := h.s.Leads.CountVersions(r.Context(), id)
	RespondOK(w, map[string]any{"versions": versions, "total": count, "page_size": limit, "offset": offset})
}

// VersionDetail returns a single immutable snapshot of a lead.
func (h *leadHandler) VersionDetail(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "leadID")
	versionID := chi.URLParam(r, "versionID")
	v, err := h.s.Leads.GetVersion(r.Context(), id, versionID)
	if err != nil {
		RespondError(w, http.StatusNotFound, &APIError{Code: CodeNotFound, Message: "version not found"})
		return
	}
	RespondOK(w, map[string]any{"version": v})
}

// leadDetail returns the enriched lead response with permissions + workflow hints.
func (h *leadHandler) leadDetail(r *http.Request, lead *leads.Lead) map[string]any {
	p, _ := auth.FromContext(r.Context())
	if lead.OwnerUserID != nil {
		if u, err := h.s.Users.Get(r.Context(), *lead.OwnerUserID); err == nil && u.Name != "" {
			lead.OwnerName = &u.Name
		}
	}
	interRepo := h.s.Interactions
	pending, _ := interRepo.PendingReportForUser(r.Context(), p.UserID)

	permissions := map[string]bool{
		"can_call":             rbacHas(p.Role, "leads.contact"),
		"can_message":          rbacHas(p.Role, "leads.contact"),
		"can_edit":             rbacHas(p.Role, "leads.edit"),
		"can_assign":           rbacHas(p.Role, "leads.assign"),
		"can_qualify":          rbacHas(p.Role, "leads.qualify"),
		"can_export_to_muninn": rbacHas(p.Role, "leads.export"),
	}

	workflow := map[string]any{
		"mandatory_report_pending": pending != nil,
		"pending_report_id":        pendingID(pending),
		"next_action":              nextActionFor(lead),
		"next_follow_up_at":        nil,
	}

	contactsList, _ := h.s.ContactRepo.ListForLead(r.Context(), lead.ID)
	timelineEvents, _ := h.s.Timeline.List(r.Context(), lead.ID, 10)

	var projectInfo map[string]any
	if lead.ProjectID != nil && *lead.ProjectID != "" && h.s.Projects != nil {
		if proj, err := h.s.Projects.Get(r.Context(), *lead.ProjectID); err == nil && proj != nil {
			projectInfo = map[string]any{
				"id":          proj.ID,
				"title":       proj.Title,
				"description": proj.Description,
			}
		}
	}

	return map[string]any{
		"lead":          lead,
		"contacts":      contactsList,
		"project":       projectInfo,
		"permissions":   permissions,
		"workflow":      workflow,
		"recent_events": timelineEvents,
	}
}

// ---- Timeline / duplicates / notes / contacts ----

func (h *leadHandler) Timeline(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "leadID")
	events, err := h.s.Timeline.List(r.Context(), id, intParam(r.URL.Query().Get("limit"), 100))
	if err != nil {
		internalErr(w, err)
		return
	}
	RespondOK(w, map[string]any{"timeline": events})
}

func (h *leadHandler) Duplicates(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "leadID")
	pairs, err := h.s.DupsRepo.ListForLead(r.Context(), id)
	if err != nil {
		internalErr(w, err)
		return
	}
	RespondOK(w, map[string]any{"possible_duplicates": pairs})
}

func (h *leadHandler) Notes(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "leadID")
	list, err := h.s.Notes.ListForLead(r.Context(), id)
	if err != nil {
		internalErr(w, err)
		return
	}
	RespondOK(w, map[string]any{"notes": list})
}

func (h *leadHandler) AddNote(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "leadID")
	var req struct {
		Text string `json:"text"`
	}
	if apiErr := DecodeJSON(r, &req); apiErr != nil {
		RespondError(w, http.StatusBadRequest, apiErr)
		return
	}
	p, _ := auth.FromContext(r.Context())
	n, err := h.s.Notes.Create(r.Context(), &notes.Note{LeadID: id, AuthorID: p.UserID, Text: req.Text})
	if err != nil {
		internalErr(w, err)
		return
	}
	h.s.Timeline.Add(r.Context(), id, "note.created", "Note added", &req.Text, &p.UserID, nil)
	RespondJSON(w, http.StatusCreated, n)
}

func (h *leadHandler) Contacts(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "leadID")
	list, err := h.s.ContactRepo.ListForLead(r.Context(), id)
	if err != nil {
		internalErr(w, err)
		return
	}
	RespondOK(w, map[string]any{"contacts": list})
}

func (h *leadHandler) VCard(w http.ResponseWriter, r *http.Request) {
	leadID := chi.URLParam(r, "leadID")
	contactID := chi.URLParam(r, "contactID")
	c, err := h.s.ContactRepo.Get(r.Context(), contactID)
	if err != nil || c.LeadID != leadID {
		RespondError(w, http.StatusNotFound, &APIError{Code: CodeNotFound, Message: "contact not found"})
		return
	}
	lead, _ := h.s.Leads.Get(r.Context(), leadID)
	vcf := messaging.BuildVCard(c, lead.CompanyName)
	filename := "contact.vcf"
	if c.FullName != nil && *c.FullName != "" {
		filename = *c.FullName + ".vcf"
	}
	w.Header().Set("Content-Type", "text/vcard; charset=utf-8")
	w.Header().Set("Content-Disposition", `attachment; filename="`+filename+`"`)
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte(vcf))
}

// ---- Assignments ----

func (h *leadHandler) Assign(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "leadID")
	var req struct {
		AssigneeID string `json:"assignee_id"`
		Method     string `json:"method"`
	}
	if apiErr := DecodeJSON(r, &req); apiErr != nil {
		RespondError(w, http.StatusBadRequest, apiErr)
		return
	}
	p, _ := auth.FromContext(r.Context())
	lead, err := h.s.Leads.Assign(r.Context(), id, req.AssigneeID, p.UserID, req.Method)
	if err != nil {
		RespondError(w, http.StatusBadRequest, &APIError{Code: CodeBadRequest, Message: err.Error()})
		return
	}
	h.s.Timeline.Add(r.Context(), id, "lead.assigned", "Lead assigned", nil, &p.UserID, map[string]any{"assignee_id": req.AssigneeID})
	h.s.Audit.Record(r.Context(), actorEntry(r, "lead.assigned", "lead", id, nil, map[string]any{"assignee_id": req.AssigneeID}))
	RespondOK(w, lead)
}

func (h *leadHandler) AssignBulk(w http.ResponseWriter, r *http.Request) {
	var req struct {
		LeadIDs  []string `json:"lead_ids"`
		AgentIDs []string `json:"agent_ids"`
	}
	if apiErr := DecodeJSON(r, &req); apiErr != nil {
		RespondError(w, http.StatusBadRequest, apiErr)
		return
	}
	p, _ := auth.FromContext(r.Context())
	n, err := h.s.Leads.AssignRoundRobin(r.Context(), req.LeadIDs, req.AgentIDs, p.UserID)
	if err != nil {
		RespondError(w, http.StatusBadRequest, &APIError{Code: CodeBadRequest, Message: err.Error()})
		return
	}
	RespondOK(w, map[string]any{"assigned": n})
}

func (h *leadHandler) Merge(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "leadID")
	var req struct {
		SourceLeadID string `json:"source_lead_id"`
	}
	if apiErr := DecodeJSON(r, &req); apiErr != nil {
		RespondError(w, http.StatusBadRequest, apiErr)
		return
	}
	p, _ := auth.FromContext(r.Context())
	if err := h.s.Duplicates.Merge(r.Context(), id, req.SourceLeadID, p.UserID); err != nil {
		RespondError(w, http.StatusBadRequest, &APIError{Code: CodeBadRequest, Message: err.Error()})
		return
	}
	h.s.Timeline.Add(r.Context(), id, "lead.merged", "Leads merged", nil, &p.UserID, map[string]any{"source_lead_id": req.SourceLeadID})
	h.s.Audit.Record(r.Context(), actorEntry(r, "lead.merged", "lead", id, nil, map[string]any{"source_lead_id": req.SourceLeadID}))
	RespondOK(w, map[string]any{"merged": true, "target_lead_id": id, "source_lead_id": req.SourceLeadID})
}

// ---- Contact actions ----

func (h *leadHandler) ContactAction(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "leadID")
	var req struct {
		ContactID *string `json:"contact_id"`
		Channel   string  `json:"channel"`
		Objective string  `json:"objective"`
		ProjectID *string `json:"project_id"`
	}
	if apiErr := DecodeJSON(r, &req); apiErr != nil {
		RespondError(w, http.StatusBadRequest, apiErr)
		return
	}
	p, _ := auth.FromContext(r.Context())

	result, err := h.s.Messaging.Prepare(r.Context(), messaging.Params{
		LeadID: id, ContactID: req.ContactID, UserID: p.UserID,
		Channel: interactions.Channel(req.Channel), Objective: req.Objective,
		ProjectID: req.ProjectID,
	})
	if err != nil {
		if errors.Is(err, messaging.ErrMandatoryReportPending) {
			RespondError(w, http.StatusForbidden, &APIError{Code: CodeMandatoryReport, Message: "a previous interaction report must be completed"})
			return
		}
		if errors.Is(err, messaging.ErrNoProject) {
			RespondError(w, http.StatusBadRequest, &APIError{Code: CodeValidationError, Message: err.Error()})
			return
		}
		RespondError(w, http.StatusBadRequest, &APIError{Code: CodeBadRequest, Message: err.Error()})
		return
	}
	h.s.Timeline.Add(r.Context(), id, "interaction.created", "Interaction prepared: "+string(result.Channel), nil, &p.UserID, nil)
	RespondJSON(w, http.StatusCreated, result)
}

// CallScenario generates a suggested call scenario for a lead without creating
// an interaction. The Android client asks for it before starting the call.
func (h *leadHandler) CallScenario(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "leadID")
	p, _ := auth.FromContext(r.Context())
	var req struct {
		ContactID  *string `json:"contact_id"`
		ProjectID  *string `json:"project_id"`
		Objective  string  `json:"objective"`
		SalesAngle string  `json:"sales_angle"`
	}
	if apiErr := DecodeJSON(r, &req); apiErr != nil {
		RespondError(w, http.StatusBadRequest, apiErr)
		return
	}
	var userID *string
	if p != nil && p.UserID != "" {
		userID = &p.UserID
	}
	result, err := h.s.Messaging.CallScenario(r.Context(), id, req.ContactID, userID, req.ProjectID, req.SalesAngle)
	if err != nil {
		if errors.Is(err, leads.ErrNotFound) || strings.Contains(err.Error(), "not found") {
			RespondError(w, http.StatusNotFound, &APIError{Code: CodeNotFound, Message: "lead not found"})
			return
		}
		if errors.Is(err, messaging.ErrNoProject) || strings.Contains(err.Error(), "no project") {
			RespondError(w, http.StatusBadRequest, &APIError{Code: CodeValidationError, Message: err.Error()})
			return
		}
		if strings.Contains(err.Error(), "does not belong to the provided project") {
			RespondError(w, http.StatusBadRequest, &APIError{Code: CodeValidationError, Message: err.Error()})
			return
		}
		internalErr(w, err)
		return
	}
	h.s.Timeline.Add(r.Context(), id, "call_scenario.generated", "Suggested call scenario generated", nil, userID, nil)
	scenarioID := ""
	if result.Scenario != nil {
		scenarioID = result.Scenario.ID
	}
	h.s.Audit.Record(r.Context(), actorEntry(r, "call.scenario", "lead", id, nil, map[string]any{"scenario_id": scenarioID, "project_id": result.ProjectID}))
	RespondJSON(w, http.StatusOK, result)
}

func (h *leadHandler) ListCallScenarios(w http.ResponseWriter, r *http.Request) {
	leadID := chi.URLParam(r, "leadID")
	if h.s.CallScenarios == nil {
		RespondJSON(w, http.StatusOK, []*callscenarios.CallScenario{})
		return
	}
	list, err := h.s.CallScenarios.ListForLead(r.Context(), leadID)
	if err != nil {
		internalErr(w, err)
		return
	}
	RespondJSON(w, http.StatusOK, list)
}

func (h *leadHandler) GetCallScenario(w http.ResponseWriter, r *http.Request) {
	leadID := chi.URLParam(r, "leadID")
	scenarioID := chi.URLParam(r, "scenarioID")
	if h.s.CallScenarios == nil {
		RespondError(w, http.StatusNotFound, &APIError{Code: CodeNotFound, Message: "call scenario not found"})
		return
	}
	sc, err := h.s.CallScenarios.Get(r.Context(), leadID, scenarioID)
	if err != nil {
		if errors.Is(err, callscenarios.ErrNotFound) {
			RespondError(w, http.StatusNotFound, &APIError{Code: CodeNotFound, Message: "call scenario not found"})
			return
		}
		internalErr(w, err)
		return
	}
	RespondJSON(w, http.StatusOK, sc)
}

func (h *leadHandler) CreateCallScenario(w http.ResponseWriter, r *http.Request) {
	leadID := chi.URLParam(r, "leadID")
	p, _ := auth.FromContext(r.Context())
	var req struct {
		ContactID           *string  `json:"contact_id"`
		OpeningSentence     string   `json:"opening_sentence"`
		Introduction        string   `json:"introduction"`
		ReasonForCall       string   `json:"reason_for_call"`
		ValueProposition    string   `json:"value_proposition"`
		DiscoveryQuestions  []string `json:"discovery_questions"`
		LikelyObjections    []string `json:"likely_objections"`
		ObjectionResponses  []string `json:"objection_responses"`
		RecommendedNextStep string   `json:"recommended_next_step"`
		ClosingLine         string   `json:"closing_line"`
	}
	if apiErr := DecodeJSON(r, &req); apiErr != nil {
		RespondError(w, http.StatusBadRequest, apiErr)
		return
	}
	if h.s.CallScenarios == nil {
		internalErr(w, errors.New("call scenarios repository not configured"))
		return
	}
	var userID *string
	if p != nil && p.UserID != "" {
		userID = &p.UserID
	}
	sc, err := h.s.CallScenarios.Create(r.Context(), &callscenarios.CallScenario{
		LeadID:              leadID,
		ContactID:           req.ContactID,
		UserID:              userID,
		OpeningSentence:     req.OpeningSentence,
		Introduction:        req.Introduction,
		ReasonForCall:       req.ReasonForCall,
		ValueProposition:    req.ValueProposition,
		DiscoveryQuestions:  req.DiscoveryQuestions,
		LikelyObjections:    req.LikelyObjections,
		ObjectionResponses:  req.ObjectionResponses,
		RecommendedNextStep: req.RecommendedNextStep,
		ClosingLine:         req.ClosingLine,
		IsGenerated:         false,
	})
	if err != nil {
		internalErr(w, err)
		return
	}
	h.s.Timeline.Add(r.Context(), leadID, "call_scenario.created", "Call scenario created", nil, userID, nil)
	h.s.Audit.Record(r.Context(), actorEntry(r, "call_scenario.create", "call_scenario", sc.ID, nil, map[string]any{"lead_id": leadID, "is_generated": sc.IsGenerated}))
	RespondJSON(w, http.StatusCreated, sc)
}

func (h *leadHandler) UpdateCallScenario(w http.ResponseWriter, r *http.Request) {
	leadID := chi.URLParam(r, "leadID")
	scenarioID := chi.URLParam(r, "scenarioID")
	p, _ := auth.FromContext(r.Context())
	var req struct {
		ContactID           *string  `json:"contact_id"`
		OpeningSentence     string   `json:"opening_sentence"`
		Introduction        string   `json:"introduction"`
		ReasonForCall       string   `json:"reason_for_call"`
		ValueProposition    string   `json:"value_proposition"`
		DiscoveryQuestions  []string `json:"discovery_questions"`
		LikelyObjections    []string `json:"likely_objections"`
		ObjectionResponses  []string `json:"objection_responses"`
		RecommendedNextStep string   `json:"recommended_next_step"`
		ClosingLine         string   `json:"closing_line"`
	}
	if apiErr := DecodeJSON(r, &req); apiErr != nil {
		RespondError(w, http.StatusBadRequest, apiErr)
		return
	}
	if h.s.CallScenarios == nil {
		internalErr(w, errors.New("call scenarios repository not configured"))
		return
	}
	_, err := h.s.CallScenarios.Get(r.Context(), leadID, scenarioID)
	if err != nil {
		if errors.Is(err, callscenarios.ErrNotFound) {
			RespondError(w, http.StatusNotFound, &APIError{Code: CodeNotFound, Message: "call scenario not found"})
			return
		}
		internalErr(w, err)
		return
	}
	sc, err := h.s.CallScenarios.Update(r.Context(), &callscenarios.CallScenario{
		ID:                  scenarioID,
		LeadID:              leadID,
		ContactID:           req.ContactID,
		OpeningSentence:     req.OpeningSentence,
		Introduction:        req.Introduction,
		ReasonForCall:       req.ReasonForCall,
		ValueProposition:    req.ValueProposition,
		DiscoveryQuestions:  req.DiscoveryQuestions,
		LikelyObjections:    req.LikelyObjections,
		ObjectionResponses:  req.ObjectionResponses,
		RecommendedNextStep: req.RecommendedNextStep,
		ClosingLine:         req.ClosingLine,
	})
	if err != nil {
		if errors.Is(err, callscenarios.ErrNotFound) {
			RespondError(w, http.StatusNotFound, &APIError{Code: CodeNotFound, Message: "call scenario not found"})
			return
		}
		internalErr(w, err)
		return
	}
	var userID *string
	if p != nil && p.UserID != "" {
		userID = &p.UserID
	}
	h.s.Timeline.Add(r.Context(), leadID, "call_scenario.updated", "Call scenario updated", nil, userID, nil)
	h.s.Audit.Record(r.Context(), actorEntry(r, "call_scenario.update", "call_scenario", sc.ID, nil, map[string]any{"lead_id": leadID}))
	RespondJSON(w, http.StatusOK, sc)
}

func (h *leadHandler) DeleteCallScenario(w http.ResponseWriter, r *http.Request) {
	leadID := chi.URLParam(r, "leadID")
	scenarioID := chi.URLParam(r, "scenarioID")
	p, _ := auth.FromContext(r.Context())
	if h.s.CallScenarios == nil {
		internalErr(w, errors.New("call scenarios repository not configured"))
		return
	}
	err := h.s.CallScenarios.Delete(r.Context(), leadID, scenarioID)
	if err != nil {
		if errors.Is(err, callscenarios.ErrNotFound) {
			RespondError(w, http.StatusNotFound, &APIError{Code: CodeNotFound, Message: "call scenario not found"})
			return
		}
		internalErr(w, err)
		return
	}
	var userID *string
	if p != nil && p.UserID != "" {
		userID = &p.UserID
	}
	h.s.Timeline.Add(r.Context(), leadID, "call_scenario.deleted", "Call scenario deleted", nil, userID, nil)
	h.s.Audit.Record(r.Context(), actorEntry(r, "call_scenario.delete", "call_scenario", scenarioID, nil, map[string]any{"lead_id": leadID}))
	RespondNoContent(w)
}

func (h *leadHandler) MarkOpened(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "interactionID")
	if err := h.s.Interactions.UpdateActionStatus(r.Context(), id, interactions.ActionOpened, nil, nil); err != nil {
		internalErr(w, err)
		return
	}
	RespondNoContent(w)
}

func (h *leadHandler) SubmitReport(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "interactionID")
	var req struct {
		Outcome                     string     `json:"outcome"`
		ContactReached              *bool      `json:"contact_reached"`
		CorrectPersonReached        *bool      `json:"correct_person_reached"`
		DecisionMakerIdentified     *bool      `json:"decision_maker_identified"`
		DecisionMakerName           *string    `json:"decision_maker_name"`
		DecisionMakerRole           *string    `json:"decision_maker_role"`
		InterestLevel               *string    `json:"interest_level"`
		NeedLevel                   *string    `json:"need_level"`
		EstimatedPurchaseCapacity   *string    `json:"estimated_purchase_capacity"`
		PurchaseTiming              *string    `json:"purchase_timing"`
		MainObjection               *string    `json:"main_objection"`
		Summary                     *string    `json:"summary"`
		NextAction                  *string    `json:"next_action"`
		FollowUpAt                  *time.Time `json:"follow_up_at"`
		QualificationRecommendation *string    `json:"qualification_recommendation"`
	}
	if apiErr := DecodeJSON(r, &req); apiErr != nil {
		RespondError(w, http.StatusBadRequest, apiErr)
		return
	}
	p, _ := auth.FromContext(r.Context())

	interest := reports.Interest(stringOrEmpty(req.InterestLevel))
	nextAct := reports.NextAction(stringOrEmpty(req.NextAction))

	rep, err := h.s.Reports.Submit(r.Context(), reports.SubmitParams{
		InteractionID:               id,
		Outcome:                     reports.Outcome(req.Outcome),
		ContactReached:              req.ContactReached,
		CorrectPersonReached:        req.CorrectPersonReached,
		DecisionMakerIdentified:     req.DecisionMakerIdentified,
		DecisionMakerName:           req.DecisionMakerName,
		DecisionMakerRole:           req.DecisionMakerRole,
		InterestLevel:               interestPtr(interest),
		NeedLevel:                   interestPtr(reports.Interest(stringOrEmpty(req.NeedLevel))),
		EstimatedPurchaseCapacity:   req.EstimatedPurchaseCapacity,
		PurchaseTiming:              req.PurchaseTiming,
		MainObjection:               req.MainObjection,
		Summary:                     req.Summary,
		NextAction:                  nextActionPtr(nextAct),
		FollowUpAt:                  req.FollowUpAt,
		QualificationRecommendation: req.QualificationRecommendation,
		SubmittedBy:                 p.UserID,
	})
	if err != nil {
		if errors.Is(err, reports.ErrValidation) {
			RespondError(w, http.StatusBadRequest, &APIError{Code: CodeValidationError, Message: err.Error()})
			return
		}
		if errors.Is(err, reports.ErrReportExists) {
			RespondError(w, http.StatusConflict, &APIError{Code: CodeConflict, Message: "report already submitted"})
			return
		}
		internalErr(w, err)
		return
	}

	// update lead status based on report
	inter, _ := h.s.Interactions.Get(r.Context(), id)
	if inter != nil {
		updateStatusFromReport(r, h, inter.LeadID, rep)
		if rep.FollowUpAt != nil {
			// Idempotent: at most one follow-up per interaction, so resubmitting
			// or retrying a report reschedules rather than duplicating.
			_, _ = h.s.FollowUps.UpsertForInteraction(r.Context(), followups.CreateParams{
				LeadID:        inter.LeadID,
				ContactID:     inter.ContactID,
				AssignedTo:    &p.UserID,
				InteractionID: &id,
				DueAt:         *rep.FollowUpAt,
				Type:          "FOLLOW_UP",
				Description:   rep.Summary,
				CreatedBy:     &p.UserID,
			})
		}
		// Promote stable facts (decision maker, correct person) onto the lead/
		// contact and re-evaluate the accumulated score from ALL reports so every
		// report moves the score, never only the first one.
		if h.s.LeadScorer != nil {
			actor := p.UserID
			if err := h.s.LeadScorer.OnReportSubmitted(r.Context(), inter.LeadID, inter.ContactID, rep, &actor); err != nil {
				h.s.Logger.Error("report_fact_promotion_and_rescore_failed",
					"lead_id", inter.LeadID, "interaction_id", id, "error", err.Error())
			}
		}
		h.s.Timeline.Add(r.Context(), inter.LeadID, "report.submitted", "Contact report submitted: "+string(rep.Outcome), req.Summary, &p.UserID, nil)
	}
	RespondJSON(w, http.StatusCreated, rep)
}

func (h *leadHandler) VoidReport(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "interactionID")
	var req struct {
		Reason string `json:"reason"`
	}
	if apiErr := DecodeJSON(r, &req); apiErr != nil {
		RespondError(w, http.StatusBadRequest, apiErr)
		return
	}
	p, _ := auth.FromContext(r.Context())
	if err := h.s.Reports.Void(r.Context(), id, p.UserID, req.Reason); err != nil {
		RespondError(w, http.StatusBadRequest, &APIError{Code: CodeBadRequest, Message: err.Error()})
		return
	}
	inter, _ := h.s.Interactions.Get(r.Context(), id)
	if inter != nil {
		h.s.Timeline.Add(r.Context(), inter.LeadID, "report.voided", "Report voided by manager", &req.Reason, &p.UserID, nil)
	}
	h.s.Audit.Record(r.Context(), actorEntry(r, "report.override", "interaction", id, nil, map[string]any{"reason": req.Reason}))
	RespondNoContent(w)
}

// ---- Follow-ups ----

func (h *leadHandler) CreateFollowUp(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "leadID")
	var req struct {
		AssignedTo    *string            `json:"assigned_to"`
		InteractionID *string            `json:"interaction_id"`
		DueAt         time.Time          `json:"due_at"`
		Type          string             `json:"type"`
		Title         *string            `json:"title"`
		Description   *string            `json:"description"`
		Channel       *string            `json:"channel"`
		Priority      followups.Priority `json:"priority"`
	}
	if apiErr := DecodeJSON(r, &req); apiErr != nil {
		RespondError(w, http.StatusBadRequest, apiErr)
		return
	}
	p, _ := auth.FromContext(r.Context())
	params := followups.CreateParams{
		LeadID: id, AssignedTo: req.AssignedTo, InteractionID: req.InteractionID,
		DueAt: req.DueAt, Type: req.Type, Title: req.Title, Description: req.Description,
		Channel: req.Channel, Priority: req.Priority, CreatedBy: &p.UserID,
	}
	var f *followups.FollowUp
	var err error
	if req.InteractionID != nil {
		f, err = h.s.FollowUps.UpsertForInteraction(r.Context(), params)
	} else {
		f, err = h.s.FollowUps.Create(r.Context(), params)
	}
	if err != nil {
		RespondError(w, http.StatusBadRequest, &APIError{Code: CodeValidationError, Message: err.Error()})
		return
	}
	h.s.Timeline.Add(r.Context(), id, "followup.created", "Follow-up scheduled", strptr(req.DueAt.Format(time.RFC3339)), &p.UserID, nil)
	RespondJSON(w, http.StatusCreated, f)
}

// ---- Qualification ----

func (h *leadHandler) Qualify(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "leadID")
	var req struct {
		BusinessRelevant          bool   `json:"business_relevant"`
		ContactInfoValid          bool   `json:"contact_info_valid"`
		MeaningfulContactOccurred bool   `json:"meaningful_contact_occurred"`
		NeedOrInterestExists      bool   `json:"need_or_interest_exists"`
		DecisionMakerIdentified   bool   `json:"decision_maker_identified"`
		NextSalesActionExists     bool   `json:"next_sales_action_exists"`
		Reason                    string `json:"reason"`
	}
	if apiErr := DecodeJSON(r, &req); apiErr != nil {
		RespondError(w, http.StatusBadRequest, apiErr)
		return
	}
	p, _ := auth.FromContext(r.Context())
	lead, err := h.s.Leads.Get(r.Context(), id)
	if err != nil {
		RespondError(w, http.StatusNotFound, &APIError{Code: CodeNotFound, Message: "lead not found"})
		return
	}
	_, err = h.s.Qualify.Qualify(r.Context(), id, qualification.QualificationInputs{
		BusinessRelevant: req.BusinessRelevant, ContactInfoValid: req.ContactInfoValid,
		MeaningfulContactOccurred: req.MeaningfulContactOccurred, NeedOrInterestExists: req.NeedOrInterestExists,
		DecisionMakerIdentified: req.DecisionMakerIdentified, NextSalesActionExists: req.NextSalesActionExists,
		Score: lead.Score,
	}, req.Reason, p.UserID)
	if err != nil {
		var nq *qualification.NotQualifiableError
		if errors.As(err, &nq) {
			RespondError(w, http.StatusConflict, &APIError{Code: CodeConflict, Message: "lead does not meet qualification rules", Details: map[string]any{"missing": nq.Missing}})
			return
		}
		RespondError(w, http.StatusBadRequest, &APIError{Code: CodeBadRequest, Message: err.Error()})
		return
	}
	h.s.Timeline.Add(r.Context(), id, "lead.qualified", "Lead qualified", &req.Reason, &p.UserID, nil)
	h.s.Audit.Record(r.Context(), actorEntry(r, "lead.qualified", "lead", id, nil, map[string]any{"reason": req.Reason}))
	RespondOK(w, map[string]any{"qualified": true, "lead_id": id})
}

func (h *leadHandler) Disqualify(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "leadID")
	var req struct {
		Reason      qualification.DisqualifyReason `json:"reason"`
		Explanation *string                        `json:"explanation"`
	}
	if apiErr := DecodeJSON(r, &req); apiErr != nil {
		RespondError(w, http.StatusBadRequest, apiErr)
		return
	}
	p, _ := auth.FromContext(r.Context())
	if err := h.s.Qualify.Disqualify(r.Context(), id, req.Reason, req.Explanation, p.UserID); err != nil {
		RespondError(w, http.StatusBadRequest, &APIError{Code: CodeBadRequest, Message: err.Error()})
		return
	}
	h.s.Timeline.Add(r.Context(), id, "lead.disqualified", "Lead disqualified: "+string(req.Reason), req.Explanation, &p.UserID, nil)
	h.s.Audit.Record(r.Context(), actorEntry(r, "lead.disqualified", "lead", id, nil, map[string]any{"reason": req.Reason}))
	RespondOK(w, map[string]any{"disqualified": true, "lead_id": id})
}

func (h *leadHandler) ExportToMuninn(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "leadID")
	p, _ := auth.FromContext(r.Context())
	exp, err := h.s.Muninn.ExportQualifiedLead(r.Context(), id, p.UserID)
	if err != nil {
		if errors.Is(err, muninn.ErrAlreadyExported) {
			RespondError(w, http.StatusConflict, &APIError{Code: CodeDuplicateExport, Message: "lead already exported to Muninn"})
			return
		}
		if errors.Is(err, muninn.ErrNotQualified) {
			RespondError(w, http.StatusConflict, &APIError{Code: CodeConflict, Message: "only qualified leads can be exported"})
			return
		}
		internalErr(w, err)
		return
	}
	h.s.Timeline.Add(r.Context(), id, "lead.exported", "Queued for Muninn export", nil, &p.UserID, nil)
	h.s.Audit.Record(r.Context(), actorEntry(r, "lead.exported", "lead", id, nil, nil))
	RespondJSON(w, http.StatusAccepted, exp)
}

func (h *leadHandler) AddTag(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "leadID")
	p, _ := auth.FromContext(r.Context())
	var req struct {
		Name   string `json:"name"`
		Source string `json:"source"`
		Note   string `json:"note"`
	}
	if apiErr := DecodeJSON(r, &req); apiErr != nil {
		RespondError(w, http.StatusBadRequest, apiErr)
		return
	}
	if req.Source == "" {
		req.Source = "manual"
	}
	if err := h.s.Leads.AddTag(r.Context(), id, req.Name, req.Source, p.UserID, req.Note); err != nil {
		if errors.Is(err, leads.ErrNotFound) {
			RespondError(w, http.StatusNotFound, &APIError{Code: CodeNotFound, Message: "lead not found"})
			return
		}
		internalErr(w, err)
		return
	}
	h.s.Timeline.Add(r.Context(), id, "lead.tag_added", "Tag added: "+req.Name, strptr(req.Note), &p.UserID, nil)
	RespondJSON(w, http.StatusCreated, map[string]any{"tag": req.Name, "source": req.Source})
}

func (h *leadHandler) RemoveTag(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "leadID")
	p, _ := auth.FromContext(r.Context())
	var req struct {
		Name string `json:"name"`
		Note string `json:"note"`
	}
	if apiErr := DecodeJSON(r, &req); apiErr != nil {
		RespondError(w, http.StatusBadRequest, apiErr)
		return
	}
	if err := h.s.Leads.RemoveTag(r.Context(), id, req.Name, p.UserID, req.Note); err != nil {
		if errors.Is(err, leads.ErrNotFound) {
			RespondError(w, http.StatusNotFound, &APIError{Code: CodeNotFound, Message: "lead not found"})
			return
		}
		internalErr(w, err)
		return
	}
	h.s.Timeline.Add(r.Context(), id, "lead.tag_removed", "Tag removed: "+req.Name, strptr(req.Note), &p.UserID, nil)
	RespondOK(w, map[string]any{"tag": req.Name, "removed": true})
}

func (h *leadHandler) Archive(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "leadID")
	p, _ := auth.FromContext(r.Context())
	if _, err := h.s.Leads.Archive(r.Context(), id, p.UserID, "archived"); err != nil {
		RespondError(w, http.StatusBadRequest, &APIError{Code: CodeBadRequest, Message: err.Error()})
		return
	}
	RespondNoContent(w)
}

// ---- helpers ----

func updateStatusFromReport(r *http.Request, h *leadHandler, leadID string, rep *reports.Report) {
	p, _ := auth.FromContext(r.Context())
	switch rep.Outcome {
	case reports.OutcomeInterested, reports.OutcomeHighlyInterested, reports.OutcomeCallbackReq:
		_, _ = h.s.Leads.ChangeStatus(r.Context(), leadID, leads.StatusFollowUp, "report: "+string(rep.Outcome), p.UserID)
	case reports.OutcomeDisqualified:
		_ = h.s.Qualify.Disqualify(r.Context(), leadID, qualification.ReasonNoInterest, rep.Summary, p.UserID)
	case reports.OutcomeNotRelevant:
		_ = h.s.Qualify.Disqualify(r.Context(), leadID, qualification.ReasonNotRelevant, rep.Summary, p.UserID)
	default:
		_, _ = h.s.Leads.ChangeStatus(r.Context(), leadID, leads.StatusContacting, "report submitted", p.UserID)
	}
}

func nextActionFor(lead *leads.Lead) string {
	switch lead.Status {
	case leads.StatusNew, leads.StatusReady:
		return "ASSIGN"
	case leads.StatusAssigned:
		return "CALL"
	case leads.StatusContacting:
		return "REPORT"
	case leads.StatusFollowUp:
		return "FOLLOW_UP"
	case leads.StatusQualified:
		return "EXPORT"
	case leads.StatusDisqualified, leads.StatusArchived, leads.StatusExportedMuninn:
		return "NONE"
	default:
		return "NONE"
	}
}

func pendingID(i *interactions.Interaction) any {
	if i == nil {
		return nil
	}
	return i.ID
}

// populateOwnerNames fills Lead.OwnerName for a batch of leads with a single
// users query, so list responses show assignee names instead of raw IDs.
func (h *leadHandler) populateOwnerNames(ctx context.Context, leadsList []*leads.Lead) {
	populateOwnerNames(ctx, h.s.DB, leadsList)
}

func intParam(s string, def int) int {
	if s == "" {
		return def
	}
	if n, err := strconv.Atoi(s); err == nil {
		return n
	}
	return def
}

func atoi(s string) int {
	n, _ := strconv.Atoi(s)
	return n
}

func derefStr(s string) string { return s }

func interestPtr(i reports.Interest) *reports.Interest {
	if i == "" {
		return nil
	}
	return &i
}

func nextActionPtr(n reports.NextAction) *reports.NextAction {
	if n == "" {
		return nil
	}
	return &n
}

func rbacHas(role auth.Role, perm string) bool {
	return auth.HasPermission(role, perm)
}

func stringOrEmpty(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}

func strptr(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}

// bodyNote extracts an optional edit note/reason from a raw JSON body.
func bodyNote(raw map[string]json.RawMessage) string {
	for _, k := range []string{"note", "reason"} {
		if v, ok := raw[k]; ok {
			var s string
			if err := json.Unmarshal(v, &s); err == nil {
				return s
			}
		}
	}
	return ""
}
