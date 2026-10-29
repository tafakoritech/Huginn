package httpapi

import (
	"encoding/json"
	"errors"
	"net/http"
	"strconv"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/tafakorpardazan/huginn/internal/auth"
	"github.com/tafakorpardazan/huginn/internal/contacts"
	"github.com/tafakorpardazan/huginn/internal/followups"
	"github.com/tafakorpardazan/huginn/internal/interactions"
	"github.com/tafakorpardazan/huginn/internal/leads"
	"github.com/tafakorpardazan/huginn/internal/muninn"
	"github.com/tafakorpardazan/huginn/internal/scoring"
)

// ---- Evidence ----

// LeadEvidence returns extracted evidence rows enriched with the current
// lead field value so the UI can show Field/Value/Confidence/Source/Excerpt.
func (h *leadHandler) Evidence(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "leadID")
	lead, err := h.s.Leads.Get(r.Context(), id)
	if err != nil {
		RespondError(w, http.StatusNotFound, &APIError{Code: CodeNotFound, Message: "lead not found"})
		return
	}
	rows, err := h.s.DB.Pool().Query(r.Context(), `
		SELECT e.id, e.field_name, e.confidence, e.source_page, e.source_row, e.source_excerpt,
		       COALESCE(d.original_filename, '')
		FROM lead_evidence e
		LEFT JOIN source_documents d ON d.id = e.source_document_id
		WHERE e.lead_id=$1 ORDER BY e.confidence DESC`, id)
	if err != nil {
		internalErr(w, err)
		return
	}
	defer rows.Close()
	items := []map[string]any{}
	for rows.Next() {
		var fieldName string
		var confidence float64
		var page, row *int
		var excerpt, docName *string
		if err := rows.Scan(&fieldName, &confidence, &page, &row, &excerpt, &docName); err != nil {
			continue
		}
		ex := ""
		if excerpt != nil {
			ex = *excerpt
		}
		src := "manual"
		if docName != nil && *docName != "" {
			src = *docName
		}
		items = append(items, map[string]any{
			"field_name": fieldName,
			"value":      evidenceValue(lead, fieldName),
			"confidence": int(confidence * 100),
			"source":     src,
			"excerpt":    ex,
			"page":       page,
			"row":        row,
		})
	}
	RespondOK(w, map[string]any{"items": items})
}

func evidenceValue(l *leads.Lead, field string) any {
	switch field {
	case "company_name":
		return l.CompanyName
	case "display_name":
		return l.DisplayName
	case "business_type":
		return l.BusinessType
	case "industry":
		return l.Industry
	case "description":
		return l.Description
	case "province":
		return l.Province
	case "city":
		return l.City
	case "address":
		return l.Address
	case "website":
		return l.Website
	case "instagram":
		return l.Instagram
	}
	return nil
}

// ---- Score breakdown ----

// LeadScore returns the lead's total score and a component breakdown.
func (h *leadHandler) Score(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "leadID")
	lead, err := h.s.Leads.Get(r.Context(), id)
	if err != nil {
		RespondError(w, http.StatusNotFound, &APIError{Code: CodeNotFound, Message: "lead not found"})
		return
	}
	weights := scoring.Weights()
	components := []map[string]any{}
	order := []struct{ key, name string }{
		{"business_fit", "business_fit"},
		{"contactability", "contactability"},
		{"business_scale", "scale"},
		{"engagement", "engagement"},
		{"authority", "authority"},
		{"need", "need"},
		{"timing", "timing"},
	}
	for _, o := range order {
		weight, ok := weights[o.key]
		if !ok {
			continue
		}
		contribution := weight
		if lead.Score > 0 {
			contribution = int(float64(weight) * float64(lead.Score) / 100)
		}
		components = append(components, map[string]any{
			"key":          o.name,
			"name":         o.name,
			"weight":       weight,
			"value":        contribution,
			"contribution": contribution,
			"max":          weight,
		})
	}
	RespondOK(w, map[string]any{
		"total":      lead.Score,
		"max":        100,
		"components": components,
	})
}

// ---- Interactions / reports / follow-ups for a lead ----

// Interactions returns the lead's interaction history.
func (h *leadHandler) Interactions(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "leadID")
	rows, err := h.s.DB.Pool().Query(r.Context(), `
		SELECT i.id, i.lead_id, i.user_id, i.channel, i.direction, i.objective, i.started_at,
		       i.ended_at, i.duration_seconds, i.action_status, i.report_status,
		       COALESCE(u.name,'')
		FROM interactions i LEFT JOIN users u ON u.id = i.user_id
		WHERE i.lead_id=$1 ORDER BY i.started_at DESC`, id)
	if err != nil {
		internalErr(w, err)
		return
	}
	defer rows.Close()
	out := []map[string]any{}
	for rows.Next() {
		var it interactions.Interaction
		var userName string
		if err := rows.Scan(&it.ID, &it.LeadID, &it.UserID, &it.Channel, &it.Direction, &it.Objective,
			&it.StartedAt, &it.EndedAt, &it.DurationSeconds, &it.ActionStatus, &it.ReportStatus, &userName); err != nil {
			continue
		}
		m := map[string]any{
			"id": it.ID, "lead_id": it.LeadID, "user_id": it.UserID,
			"channel": it.Channel, "direction": it.Direction, "objective": it.Objective,
			"started_at": it.StartedAt, "ended_at": it.EndedAt, "duration_seconds": it.DurationSeconds,
			"action_status": it.ActionStatus, "report_status": it.ReportStatus,
		}
		if userName != "" {
			m["user"] = map[string]any{"id": it.UserID, "name": userName}
		}
		out = append(out, m)
	}
	RespondOK(w, out)
}

// Reports returns the lead's contact reports.
func (h *leadHandler) Reports(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "leadID")
	rows, err := h.s.DB.Pool().Query(r.Context(), `
		SELECT cr.id, cr.interaction_id, cr.outcome, cr.interest_level, cr.next_action,
		       cr.summary, cr.manager_override, cr.manager_override_reason, cr.created_at
		FROM contact_reports cr
		JOIN interactions i ON i.id = cr.interaction_id
		WHERE i.lead_id=$1 ORDER BY cr.created_at DESC`, id)
	if err != nil {
		internalErr(w, err)
		return
	}
	defer rows.Close()
	out := []map[string]any{}
	for rows.Next() {
		var reportID, interactionID, outcome string
		var interest, nextAction, summary, overrideReason *string
		var managerOverride bool
		var createdAt time.Time
		if err := rows.Scan(&reportID, &interactionID, &outcome, &interest, &nextAction, &summary, &managerOverride, &overrideReason, &createdAt); err != nil {
			continue
		}
		out = append(out, map[string]any{
			"id": reportID, "interaction_id": interactionID, "outcome": outcome,
			"interest_level": interest, "next_action": nextAction, "summary": summary,
			"manager_override": managerOverride, "manager_override_reason": overrideReason,
			"created_at": createdAt,
		})
	}
	RespondOK(w, out)
}

// LeadFollowUps returns the lead's follow-ups.
func (h *leadHandler) LeadFollowUps(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "leadID")
	items, err := h.s.FollowUps.ListForLead(r.Context(), id)
	if err != nil {
		internalErr(w, err)
		return
	}
	if items == nil {
		items = []*followups.FollowUp{}
	}
	RespondOK(w, items)
}

// ---- Contact management (versioned) ----

// AddContact adds a contact to a lead and records a new immutable version.
func (h *leadHandler) AddContact(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "leadID")
	p, _ := auth.FromContext(r.Context())

	var raw map[string]json.RawMessage
	if apiErr := DecodeJSON(r, &raw); apiErr != nil {
		RespondError(w, http.StatusBadRequest, apiErr)
		return
	}
	note := bodyNote(raw)
	delete(raw, "note")
	delete(raw, "reason")
	raw["lead_id"] = json.RawMessage(`"` + id + `"`)
	body, _ := json.Marshal(raw)
	var req contacts.Contact
	if err := json.Unmarshal(body, &req); err != nil {
		RespondError(w, http.StatusBadRequest, &APIError{Code: CodeBadRequest, Message: "invalid JSON body: " + err.Error()})
		return
	}
	c, err := h.s.Leads.CreateContact(r.Context(), id, &req, p.UserID, note)
	if err != nil {
		if errors.Is(err, leads.ErrNotFound) {
			RespondError(w, http.StatusNotFound, &APIError{Code: CodeNotFound, Message: "lead not found"})
			return
		}
		internalErr(w, err)
		return
	}
	h.s.Timeline.Add(r.Context(), id, "contact.created", "Contact added", contactFullName(c), &p.UserID, nil)
	RespondJSON(w, http.StatusCreated, c)
}

type updateContactRequest struct {
	FirstName        *string `json:"first_name"`
	LastName         *string `json:"last_name"`
	FullName         *string `json:"full_name"`
	JobTitle         *string `json:"job_title"`
	Department       *string `json:"department"`
	Mobile           *string `json:"mobile"`
	Phone            *string `json:"phone"`
	Email            *string `json:"email"`
	Whatsapp         *string `json:"whatsapp"`
	Telegram         *string `json:"telegram"`
	Bale             *string `json:"bale"`
	Eitaa            *string `json:"eitaa"`
	Rubika           *string `json:"rubika"`
	IsDecisionMaker  *bool   `json:"is_decision_maker"`
	IsPrimaryContact *bool   `json:"is_primary_contact"`
}

// UpdateContact patches a contact on a lead and records a new immutable
// version. Only fields present in the request body are changed.
func (h *leadHandler) UpdateContact(w http.ResponseWriter, r *http.Request) {
	leadID := chi.URLParam(r, "leadID")
	contactID := chi.URLParam(r, "contactID")
	p, _ := auth.FromContext(r.Context())

	var raw map[string]json.RawMessage
	if apiErr := DecodeJSON(r, &raw); apiErr != nil {
		RespondError(w, http.StatusBadRequest, apiErr)
		return
	}
	patches := map[string]any{}
	for k, v := range raw {
		switch k {
		case "note", "reason":
			continue
		case "is_decision_maker", "is_primary_contact":
			var b *bool
			if err := json.Unmarshal(v, &b); err == nil && b != nil {
				patches[k] = b
			}
		default:
			var s *string
			if err := json.Unmarshal(v, &s); err == nil {
				patches[k] = s
			}
		}
	}
	if len(patches) == 0 {
		RespondError(w, http.StatusBadRequest, &APIError{Code: CodeValidationError, Message: "no fields to update"})
		return
	}
	note := bodyNote(raw)
	c, err := h.s.Leads.UpdateContact(r.Context(), leadID, contactID, patches, p.UserID, note)
	if err != nil {
		if err.Error() == "contact not found" {
			RespondError(w, http.StatusNotFound, &APIError{Code: CodeNotFound, Message: "contact not found"})
			return
		}
		if errors.Is(err, leads.ErrNotFound) {
			RespondError(w, http.StatusNotFound, &APIError{Code: CodeNotFound, Message: "lead not found"})
			return
		}
		internalErr(w, err)
		return
	}
	h.s.Timeline.Add(r.Context(), leadID, "contact.updated", "Contact updated", strptrPtr(note), &p.UserID, nil)
	h.s.Audit.Record(r.Context(), actorEntry(r, "contact.updated", "contact", contactID, nil, map[string]any{"lead_id": leadID}))
	RespondOK(w, c)
}

// DeleteContact soft-deletes a contact from a lead and records a new immutable
// version. The contact data remains available in historical versions.
func (h *leadHandler) DeleteContact(w http.ResponseWriter, r *http.Request) {
	leadID := chi.URLParam(r, "leadID")
	contactID := chi.URLParam(r, "contactID")
	p, _ := auth.FromContext(r.Context())

	contactsList, _ := h.s.ContactRepo.ListForLead(r.Context(), leadID)
	var removedName *string
	for _, c := range contactsList {
		if c.ID == contactID {
			removedName = contactFullName(c)
			break
		}
	}

	if err := h.s.Leads.DeleteContact(r.Context(), leadID, contactID, p.UserID, "removed contact"); err != nil {
		if err.Error() == "contact not found" {
			RespondError(w, http.StatusNotFound, &APIError{Code: CodeNotFound, Message: "contact not found"})
			return
		}
		if errors.Is(err, leads.ErrNotFound) {
			RespondError(w, http.StatusNotFound, &APIError{Code: CodeNotFound, Message: "lead not found"})
			return
		}
		internalErr(w, err)
		return
	}
	h.s.Timeline.Add(r.Context(), leadID, "contact.deleted", "Contact removed", removedName, &p.UserID, nil)
	h.s.Audit.Record(r.Context(), actorEntry(r, "contact.deleted", "contact", contactID, nil, map[string]any{"lead_id": leadID}))
	RespondNoContent(w)
}

func contactFullName(c *contacts.Contact) *string {
	if c == nil {
		return nil
	}
	name := ""
	if c.FullName != nil {
		name = *c.FullName
	} else if c.FirstName != nil {
		name = *c.FirstName
		if c.LastName != nil {
			name += " " + *c.LastName
		}
	}
	if name == "" {
		return nil
	}
	return &name
}

func strptrPtr(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}

// ExportGet returns the current Muninn export state for a lead (or null).
func (h *leadHandler) ExportGet(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "leadID")
	row := h.s.DB.Pool().QueryRow(r.Context(), `
		SELECT id, huginn_lead_id, muninn_lead_id, status, attempt_count, last_error,
		       exported_at, created_at, updated_at
		FROM crm_exports WHERE huginn_lead_id=$1`, id)
	var e muninn.Export
	err := row.Scan(&e.ID, &e.HuginnLeadID, &e.MuninnLeadID, &e.Status, &e.AttemptCount, &e.LastError,
		&e.ExportedAt, &e.CreatedAt, &e.UpdatedAt)
	if err != nil {
		RespondOK(w, map[string]any{"export": nil})
		return
	}
	RespondOK(w, map[string]any{"export": &e})
}

func joinCols(cols []string) string {
	out := ""
	for i, c := range cols {
		if i > 0 {
			out += ", "
		}
		out += c
	}
	return out
}

func itoa(n int) string {
	return strconv.Itoa(n)
}
