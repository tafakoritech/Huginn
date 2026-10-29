package httpapi

import (
	"net/http"
	"strconv"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/tafakorpardazan/huginn/internal/auth"
	"github.com/tafakorpardazan/huginn/internal/database"
	"github.com/tafakorpardazan/huginn/internal/duplicates"
	"github.com/tafakorpardazan/huginn/internal/followups"
	"github.com/tafakorpardazan/huginn/internal/leads"
	"github.com/tafakorpardazan/huginn/internal/leadversions"
	"github.com/tafakorpardazan/huginn/internal/rbac"
)

type dataHandler struct {
	s *Services
}

func newDataHandler(s *Services) *dataHandler { return &dataHandler{s: s} }

// ---- Documents ----

// DocumentsList returns all source documents (with batch + lead counts).
func (h *dataHandler) DocumentsList(w http.ResponseWriter, r *http.Request) {
	limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
	if limit <= 0 || limit > 200 {
		limit = 50
	}
	offset, _ := strconv.Atoi(r.URL.Query().Get("offset"))
	rows, err := h.s.DB.Pool().Query(r.Context(), `
		SELECT d.id, d.batch_id, d.object_key, d.mime_type, d.original_filename, d.size_bytes,
		       d.status, d.extracted_text, d.extraction_error, d.created_at, d.updated_at,
		       COALESCE(ib.name,''), (SELECT count(*) FROM leads l WHERE l.source_document_id = d.id)
		FROM source_documents d LEFT JOIN import_batches ib ON ib.id = d.batch_id
		ORDER BY d.created_at DESC LIMIT $1 OFFSET $2`, limit, offset)
	if err != nil {
		internalErr(w, err)
		return
	}
	defer rows.Close()
	docs := []map[string]any{}
	for rows.Next() {
		var id, batchID, objectKey, mime, filename, status string
		var size *int64
		var extractedText, extractionError *string
		var createdAt, updatedAt time.Time
		var batchName string
		var leadCount int
		if err := rows.Scan(&id, &batchID, &objectKey, &mime, &filename, &size, &status,
			&extractedText, &extractionError, &createdAt, &updatedAt, &batchName, &leadCount); err != nil {
			continue
		}
		docs = append(docs, map[string]any{
			"id": id, "batch_id": batchID, "object_key": objectKey, "mime_type": mime,
			"original_filename": filename, "size_bytes": size, "status": status,
			"extracted_text": extractedText, "extraction_error": extractionError,
			"created_at": createdAt, "updated_at": updatedAt, "batch_name": batchName, "lead_count": leadCount,
		})
	}
	RespondOK(w, map[string]any{"documents": docs, "count": len(docs)})
}

// DocumentCandidates returns leads created from a source document.
func (h *dataHandler) DocumentCandidates(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "docID")
	rows, err := h.s.DB.Pool().Query(r.Context(), `
		SELECT l.id, l.company_name, l.score, l.status,
		       COALESCE((SELECT c.full_name FROM contacts c WHERE c.lead_id = l.id AND c.is_primary_contact LIMIT 1), ''),
		       COALESCE((SELECT c.mobile_normalized FROM contacts c WHERE c.lead_id = l.id AND c.is_primary_contact LIMIT 1), '')
		FROM leads l WHERE l.source_document_id=$1 ORDER BY l.score DESC`, id)
	if err != nil {
		internalErr(w, err)
		return
	}
	defer rows.Close()
	out := []map[string]any{}
	for rows.Next() {
		var leadID, company, status, contactName, phone string
		var score int
		if err := rows.Scan(&leadID, &company, &score, &status, &contactName, &phone); err != nil {
			continue
		}
		out = append(out, map[string]any{
			"id": leadID, "company_name": company, "score": score, "status": status,
			"contact_name": contactName, "phone": phone,
		})
	}
	RespondOK(w, out)
}

// ---- Duplicates ----

// DuplicatesList returns possible duplicate pairs enriched with both leads.
func (h *dataHandler) DuplicatesList(w http.ResponseWriter, r *http.Request) {
	status := r.URL.Query().Get("status")
	limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
	if limit <= 0 || limit > 200 {
		limit = 50
	}
	offset, _ := strconv.Atoi(r.URL.Query().Get("offset"))
	q := `SELECT pd.id, pd.lead_a_id, pd.lead_b_id, pd.reason, pd.similarity, pd.status, pd.created_at,
		la.company_name, la.business_type, la.city, la.score, la.status,
		lb.company_name, lb.business_type, lb.city, lb.score, lb.status
		FROM possible_duplicates pd
		JOIN leads la ON la.id = pd.lead_a_id
		JOIN leads lb ON lb.id = pd.lead_b_id`
	args := []any{}
	argc := 1
	if status != "" {
		q += ` WHERE pd.status=$` + strconv.Itoa(argc)
		args = append(args, status)
		argc++
	}
	q += ` ORDER BY pd.similarity DESC LIMIT $` + strconv.Itoa(argc) + ` OFFSET $` + strconv.Itoa(argc+1)
	args = append(args, limit, offset)

	rows, err := h.s.DB.Pool().Query(r.Context(), q, args...)
	if err != nil {
		internalErr(w, err)
		return
	}
	defer rows.Close()
	out := []map[string]any{}
	for rows.Next() {
		var pair duplicates.Pair
		var laCompany, laBusiness, laCity string
		var laScore int
		var laStatus leads.Status
		var lbCompany, lbBusiness, lbCity string
		var lbScore int
		var lbStatus leads.Status
		if err := rows.Scan(&pair.ID, &pair.LeadAID, &pair.LeadBID, &pair.Reason, &pair.Similarity, &pair.Status, &pair.CreatedAt,
			&laCompany, &laBusiness, &laCity, &laScore, &laStatus,
			&lbCompany, &lbBusiness, &lbCity, &lbScore, &lbStatus); err != nil {
			continue
		}
		out = append(out, map[string]any{
			"id": pair.ID, "lead_a_id": pair.LeadAID, "lead_b_id": pair.LeadBID,
			"reason": pair.Reason, "similarity": pair.Similarity, "status": pair.Status,
			"created_at": pair.CreatedAt,
			"lead_a":     map[string]any{"id": pair.LeadAID, "company_name": laCompany, "business_type": laBusiness, "city": laCity, "score": laScore, "status": laStatus},
			"lead_b":     map[string]any{"id": pair.LeadBID, "company_name": lbCompany, "business_type": lbBusiness, "city": lbCity, "score": lbScore, "status": lbStatus},
		})
	}
	RespondOK(w, map[string]any{"duplicates": out, "count": len(out)})
}

// DuplicateResolve resolves a pair as distinct/later or merges into a target.
func (h *dataHandler) DuplicateResolve(w http.ResponseWriter, r *http.Request) {
	p, _ := auth.FromContext(r.Context())
	pairID := chi.URLParam(r, "pairID")
	var req struct {
		Action       string `json:"action"`
		TargetLeadID string `json:"target_lead_id"`
	}
	if apiErr := DecodeJSON(r, &req); apiErr != nil {
		RespondError(w, http.StatusBadRequest, apiErr)
		return
	}
	switch req.Action {
	case "distinct":
		if err := h.s.DupsRepo.MarkStatus(r.Context(), pairID, duplicates.StatusDistinct, p.UserID); err != nil {
			RespondError(w, http.StatusBadRequest, &APIError{Code: CodeBadRequest, Message: err.Error()})
			return
		}
		RespondOK(w, map[string]any{"resolved": true, "action": "distinct"})
	case "later":
		if err := h.s.DupsRepo.MarkStatus(r.Context(), pairID, duplicates.StatusPossible, p.UserID); err != nil {
			RespondError(w, http.StatusBadRequest, &APIError{Code: CodeBadRequest, Message: err.Error()})
			return
		}
		RespondOK(w, map[string]any{"resolved": true, "action": "later"})
	case "merge":
		if req.TargetLeadID == "" {
			RespondError(w, http.StatusBadRequest, &APIError{Code: CodeValidationError, Message: "target_lead_id required for merge"})
			return
		}
		var leadA, leadB string
		err := h.s.DB.Pool().QueryRow(r.Context(),
			`SELECT lead_a_id, lead_b_id FROM possible_duplicates WHERE id=$1`, pairID).Scan(&leadA, &leadB)
		if err != nil {
			RespondError(w, http.StatusNotFound, &APIError{Code: CodeNotFound, Message: "duplicate pair not found"})
			return
		}
		source := leadA
		if req.TargetLeadID == leadA {
			source = leadB
		} else if req.TargetLeadID != leadB {
			RespondError(w, http.StatusBadRequest, &APIError{Code: CodeValidationError, Message: "target_lead_id must be one of the pair's leads"})
			return
		}
		if err := h.s.Duplicates.Merge(r.Context(), req.TargetLeadID, source, p.UserID); err != nil {
			RespondError(w, http.StatusBadRequest, &APIError{Code: CodeBadRequest, Message: err.Error()})
			return
		}
		_ = h.s.Timeline.Add(r.Context(), req.TargetLeadID, "lead.merged", "Leads merged", nil, &p.UserID, nil)
		RespondOK(w, map[string]any{"resolved": true, "action": "merge", "target_lead_id": req.TargetLeadID})
	default:
		RespondError(w, http.StatusBadRequest, &APIError{Code: CodeValidationError, Message: "action must be merge|distinct|later"})
	}
}

// ---- Change history (admin) ----

// ChangeHistory returns the latest immutable lead versions across all leads,
// newest first, for the admin change-history dashboard.
func (h *dataHandler) ChangeHistory(w http.ResponseWriter, r *http.Request) {
	limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
	if limit <= 0 || limit > 200 {
		limit = 50
	}
	offset, _ := strconv.Atoi(r.URL.Query().Get("offset"))
	items, err := h.s.Leads.RecentChanges(r.Context(), limit, offset)
	if err != nil {
		internalErr(w, err)
		return
	}
	if items == nil {
		items = []*leadversions.RecentItem{}
	}
	RespondOK(w, map[string]any{"items": items, "count": len(items)})
}

// ---- Reports ----

// ReportsList returns contact reports with interaction + lead + agent.
func (h *dataHandler) ReportsList(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	limit, _ := strconv.Atoi(q.Get("limit"))
	if limit <= 0 || limit > 200 {
		limit = 50
	}
	offset, _ := strconv.Atoi(q.Get("offset"))
	sqlQ := `SELECT cr.id, cr.interaction_id, cr.outcome, cr.interest_level, cr.next_action,
		cr.manager_override, cr.created_at,
		i.lead_id, i.channel, i.started_at,
		COALESCE(u.name,''), COALESCE(lu.name,''), COALESCE(l.company_name,'')
		FROM contact_reports cr
		JOIN interactions i ON i.id = cr.interaction_id
		LEFT JOIN users u ON u.id = cr.submitted_by
		LEFT JOIN users lu ON lu.id = i.user_id
		LEFT JOIN leads l ON l.id = i.lead_id`
	args := []any{}
	argc := 1
	conds := []string{}
	if v := q.Get("user_id"); v != "" {
		conds = append(conds, "i.user_id=$"+strconv.Itoa(argc))
		args = append(args, v)
		argc++
	}
	if v := q.Get("status"); v != "" {
		switch v {
		case "OVERRIDE":
			conds = append(conds, "cr.manager_override = true")
		case "PENDING":
			conds = append(conds, "i.report_status='PENDING'")
		default:
			conds = append(conds, "cr.manager_override = false")
		}
	}
	for i, c := range conds {
		if i == 0 {
			sqlQ += " WHERE " + c
		} else {
			sqlQ += " AND " + c
		}
	}
	sqlQ += ` ORDER BY cr.created_at DESC LIMIT $` + strconv.Itoa(argc) + ` OFFSET $` + strconv.Itoa(argc+1)
	args = append(args, limit, offset)

	rows, err := h.s.DB.Pool().Query(r.Context(), sqlQ, args...)
	if err != nil {
		internalErr(w, err)
		return
	}
	defer rows.Close()
	out := []map[string]any{}
	for rows.Next() {
		var reportID, interactionID, outcome, channel, leadID string
		var interest, nextAction *string
		var managerOverride bool
		var createdAt, startedAt time.Time
		var submitter, agent, company string
		if err := rows.Scan(&reportID, &interactionID, &outcome, &interest, &nextAction, &managerOverride, &createdAt,
			&leadID, &channel, &startedAt, &submitter, &agent, &company); err != nil {
			continue
		}
		interaction := map[string]any{"id": interactionID, "channel": channel, "started_at": startedAt, "lead_id": leadID}
		if company != "" {
			interaction["lead"] = map[string]any{"id": leadID, "company_name": company}
		}
		if agent != "" {
			interaction["user"] = map[string]any{"name": agent}
		}
		out = append(out, map[string]any{
			"id": reportID, "interaction_id": interactionID, "outcome": outcome,
			"interest_level": interest, "next_action": nextAction, "manager_override": managerOverride,
			"created_at": createdAt, "interaction": interaction,
			"submitter": map[string]any{"name": submitter},
		})
	}
	RespondOK(w, map[string]any{"reports": out, "count": len(out)})
}

// ReportGet returns a single report with full detail.
func (h *dataHandler) ReportGet(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "reportID")
	row := h.s.DB.Pool().QueryRow(r.Context(), `
		SELECT cr.id, cr.interaction_id, cr.outcome, cr.contact_reached, cr.correct_person_reached,
			cr.decision_maker_identified, cr.decision_maker_name, cr.decision_maker_role,
			cr.interest_level, cr.need_level, cr.estimated_purchase_capacity, cr.purchase_timing,
			cr.main_objection, cr.summary, cr.next_action, cr.follow_up_at,
			cr.qualification_recommendation, cr.manager_override, cr.manager_override_reason,
			cr.created_at, cr.updated_at,
			COALESCE(i.lead_id::text,''), COALESCE(i.channel::text,''), COALESCE(i.started_at, cr.created_at),
			COALESCE(u.name,''), COALESCE(lu.name,''), COALESCE(l.company_name,''),
			COALESCE(cr.overridden_by::text,'')
		FROM contact_reports cr
		JOIN interactions i ON i.id = cr.interaction_id
		LEFT JOIN users u ON u.id = cr.submitted_by
		LEFT JOIN users lu ON lu.id = i.user_id
		LEFT JOIN leads l ON l.id = i.lead_id
		WHERE cr.id=$1`, id)
	var reportID, interactionID, outcome, leadID, channel, submitter, agent, company, overriddenBy string
	var contactReached, correctPerson, dmIdentified *bool
	var dmName, dmRole, interest, need, capacity, timing, objection, summary, nextAction, qualRec, overrideReason *string
	var followUpAt *time.Time
	var managerOverride bool
	var createdAt, updatedAt, startedAt time.Time
	err := row.Scan(&reportID, &interactionID, &outcome, &contactReached, &correctPerson, &dmIdentified,
		&dmName, &dmRole, &interest, &need, &capacity, &timing, &objection, &summary, &nextAction, &followUpAt,
		&qualRec, &managerOverride, &overrideReason, &createdAt, &updatedAt,
		&leadID, &channel, &startedAt, &submitter, &agent, &company, &overriddenBy)
	if err != nil {
		if database.IsNoRows(err) {
			RespondError(w, http.StatusNotFound, &APIError{Code: CodeNotFound, Message: "report not found"})
			return
		}
		internalErr(w, err)
		return
	}
	interaction := map[string]any{"id": interactionID, "channel": channel, "started_at": startedAt, "lead_id": leadID}
	if company != "" {
		interaction["lead"] = map[string]any{"id": leadID, "company_name": company}
	}
	if agent != "" {
		interaction["user"] = map[string]any{"name": agent}
	}
	RespondOK(w, map[string]any{
		"id": reportID, "interaction_id": interactionID, "outcome": outcome,
		"contact_reached": contactReached, "correct_person_reached": correctPerson,
		"decision_maker_identified": dmIdentified, "decision_maker_name": dmName, "decision_maker_role": dmRole,
		"interest_level": interest, "need_level": need, "estimated_purchase_capacity": capacity,
		"purchase_timing": timing, "main_objection": objection, "summary": summary, "next_action": nextAction,
		"follow_up_at": followUpAt, "qualification_recommendation": qualRec,
		"manager_override": managerOverride, "manager_override_reason": overrideReason,
		"created_at": createdAt, "updated_at": updatedAt, "interaction": interaction,
		"submitter":     map[string]any{"name": submitter},
		"overridden_by": overriddenBy,
	})
}

// ReportOverride marks a report as manager-overridden.
func (h *dataHandler) ReportOverride(w http.ResponseWriter, r *http.Request) {
	p, _ := auth.FromContext(r.Context())
	id := chi.URLParam(r, "reportID")
	var req struct {
		Reason string `json:"reason"`
	}
	if apiErr := DecodeJSON(r, &req); apiErr != nil {
		RespondError(w, http.StatusBadRequest, apiErr)
		return
	}
	if req.Reason == "" {
		RespondError(w, http.StatusBadRequest, &APIError{Code: CodeValidationError, Message: "reason is required"})
		return
	}
	res, err := h.s.DB.Pool().Exec(r.Context(), `
		UPDATE contact_reports SET manager_override=true, manager_override_reason=$2, overridden_by=$3, updated_at=now()
		WHERE id=$1`, id, req.Reason, p.UserID)
	if err != nil {
		internalErr(w, err)
		return
	}
	if res.RowsAffected() == 0 {
		RespondError(w, http.StatusNotFound, &APIError{Code: CodeNotFound, Message: "report not found"})
		return
	}
	RespondOK(w, map[string]any{"overridden": true})
}

// ---- Follow-ups ----

// FollowUpsList lists follow-ups. Sales agents get their own assigned
// follow-ups by default; managers/admins can filter by assigned_user or see
// all. Supports scoped views (my, pending, overdue, today, upcoming,
// completed, all) plus status/lead/date filters and pagination.
func (h *dataHandler) FollowUpsList(w http.ResponseWriter, r *http.Request) {
	p, _ := auth.FromContext(r.Context())
	start := time.Now()

	limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
	offset, _ := strconv.Atoi(r.URL.Query().Get("offset"))
	view := r.URL.Query().Get("view")
	scope := r.URL.Query().Get("scope")
	if scope == "" {
		scope = view
	}
	status := r.URL.Query().Get("status")

	params := followups.ListParams{Limit: limit, Offset: offset, Scope: scope}
	if s := followups.Status(status); status != "" {
		params.Status = &s
	}
	if uid := r.URL.Query().Get("assigned_user"); uid != "" {
		params.AssignedTo = strptr(uid)
	}
	if leadID := r.URL.Query().Get("lead_id"); leadID != "" {
		params.LeadID = strptr(leadID)
	}
	if from := r.URL.Query().Get("from"); from != "" {
		if t, err := time.Parse(time.RFC3339, from); err == nil {
			params.From = &t
		}
	}
	if to := r.URL.Query().Get("to"); to != "" {
		if t, err := time.Parse(time.RFC3339, to); err == nil {
			params.To = &t
		}
	}
	if overdue := r.URL.Query().Get("overdue"); overdue == "1" || overdue == "true" {
		params.OverdueOnly = true
	}

	// My Follow-Ups: sales agents are always scoped to their assigned items
	// unless an explicit assigned_user or lead_id scope is requested. Managers
	// and admins see all users' follow-ups.
	if params.AssignedTo == nil && params.LeadID == nil && p != nil && p.Role == rbac.RoleSalesAgent {
		params.AssignedTo = &p.UserID
	}
	if scope == "my" && p != nil {
		params.AssignedTo = &p.UserID
	}

	items, total, err := h.s.FollowUps.List(r.Context(), params)
	if err != nil {
		internalErr(w, err)
		return
	}
	out := make([]map[string]any, 0, len(items))
	for _, it := range items {
		out = append(out, followUpListItem(it))
	}
	h.s.Logger.Info("followups_listed",
		"user_id", p.UserID,
		"view", view,
		"status", status,
		"lead_id", valueOrEmpty(params.LeadID),
		"assigned_user", valueOrEmpty(params.AssignedTo),
		"result_count", len(out),
		"total", total,
		"latency_ms", time.Since(start).Milliseconds(),
	)
	RespondOK(w, map[string]any{"followups": out, "count": len(out), "total": total})
}

// followUpListItem converts a service list item to the wire shape shared by all
// clients: flat fields for mobile, plus nested lead/assignee for the dashboard.
func followUpListItem(it *followups.ListItem) map[string]any {
	m := map[string]any{
		"id":              it.ID,
		"lead_id":         it.LeadID,
		"contact_id":      it.ContactID,
		"project_id":      it.ProjectID,
		"campaign_id":     it.CampaignID,
		"interaction_id":  it.InteractionID,
		"assigned_to":     it.AssignedTo,
		"created_by":      it.CreatedBy,
		"title":           it.Title,
		"description":     it.Description,
		"channel":         it.Channel,
		"type":            it.Type,
		"priority":        it.Priority,
		"status":          it.Status,
		"due_at":          it.DueAt,
		"completed_at":    it.CompletedAt,
		"snoozed_until":   it.SnoozedUntil,
		"completion_note": it.CompletionNote,
		"outcome":         it.Outcome,
		"source":          it.Source,
		"created_at":      it.CreatedAt,
		"updated_at":      it.UpdatedAt,
		"is_overdue":      it.IsOverdue,
		"company_name":    it.CompanyName,
		"contact_name":    it.ContactName,
		"assignee_name":   it.AssigneeName,
	}
	if it.CompanyName != "" {
		m["lead"] = map[string]any{"id": it.LeadID, "company_name": it.CompanyName}
	}
	if it.AssigneeName != nil {
		m["assignee"] = map[string]any{"id": valueOrEmpty(it.AssignedTo), "name": *it.AssigneeName}
	}
	return m
}

// FollowUpComplete marks a follow-up completed with an optional note/outcome.
func (h *dataHandler) FollowUpComplete(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "followupID")
	var req struct {
		Note    *string `json:"note"`
		Outcome *string `json:"outcome"`
	}
	if r.ContentLength > 0 {
		_ = DecodeJSON(r, &req)
	}
	f, err := h.s.FollowUps.Complete(r.Context(), id, req.Note, req.Outcome)
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

func valueOrEmpty(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}
