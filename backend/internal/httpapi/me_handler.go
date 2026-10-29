package httpapi

import (
	"log/slog"
	"net/http"
	"strconv"
	"time"

	"github.com/tafakorpardazan/huginn/internal/auth"
	"github.com/tafakorpardazan/huginn/internal/contacts"
	"github.com/tafakorpardazan/huginn/internal/interactions"
	"github.com/tafakorpardazan/huginn/internal/leads"
	"github.com/tafakorpardazan/huginn/internal/rbac"
	"github.com/tafakorpardazan/huginn/internal/reports"
	"github.com/tafakorpardazan/huginn/internal/users"
)

type meHandler struct {
	users     *users.Service
	leads     *leads.Service
	inter     *interactions.Repository
	contacts  *contacts.Repository
	reports   *reports.Repository
	logger    *slog.Logger
}

func newMeHandler(users *users.Service, leads *leads.Service, inter *interactions.Repository, contacts *contacts.Repository, reports *reports.Repository, logger *slog.Logger) *meHandler {
	return &meHandler{users: users, leads: leads, inter: inter, contacts: contacts, reports: reports, logger: logger}
}

func (h *meHandler) GetMe(w http.ResponseWriter, r *http.Request) {
	p, ok := auth.FromContext(r.Context())
	if !ok {
		RespondError(w, http.StatusUnauthorized, &APIError{Code: CodeUnauthorized, Message: "authentication required"})
		return
	}
	u, err := h.users.Get(r.Context(), p.UserID)
	if err != nil {
		RespondError(w, http.StatusNotFound, &APIError{Code: CodeNotFound, Message: "user not found"})
		return
	}
	RespondOK(w, u)
}

// PendingReports returns the mandatory-report state for the user.
func (h *meHandler) PendingReports(w http.ResponseWriter, r *http.Request) {
	p, _ := auth.FromContext(r.Context())
	inter, err := h.inter.PendingReportForUser(r.Context(), p.UserID)
	if err != nil {
		internalErr(w, err)
		return
	}
	if inter == nil {
		RespondOK(w, map[string]any{"requires_report": false})
		return
	}
	resp := map[string]any{
		"requires_report": true,
		"interaction_id":  inter.ID,
		"lead_id":         inter.LeadID,
		"channel":         inter.Channel,
		"created_at":      inter.CreatedAt,
	}
	// Canonical persistent facts + company info so the app can pre-fill stable
	// fields (decision maker, correct person) on subsequent reports instead of
	// asking the salesperson to re-enter them.
	if lead, err := h.leads.Get(r.Context(), inter.LeadID); err == nil {
		resp["company_name"] = lead.CompanyName
		resp["correct_person_reached"] = lead.CorrectPersonReached
		resp["decision_maker_identified"] = lead.DecisionMakerIdentified
		resp["decision_maker_name"] = lead.DecisionMakerName
		resp["decision_maker_role"] = lead.DecisionMakerRole
	}
	if inter.ContactID != nil {
		if c, err := h.contacts.Get(r.Context(), *inter.ContactID); err == nil {
			resp["contact_id"] = c.ID
			resp["contact_name"] = c.FullName
			resp["contact_job_title"] = c.JobTitle
			resp["contact_is_decision_maker"] = c.IsDecisionMaker
		}
	}
	RespondOK(w, resp)
}

// MyReports lists submitted contact reports visible to the current user for the
// global Reports screen (GET /api/v1/me/reports). It returns persisted
// contact_reports — never pending-report placeholders. Ownership/RBAC: a sales
// agent sees reports on interactions they performed or reports they submitted;
// managers/admins see all reports. Pagination via limit/offset.
func (h *meHandler) MyReports(w http.ResponseWriter, r *http.Request) {
	p, ok := auth.FromContext(r.Context())
	if !ok {
		RespondError(w, http.StatusUnauthorized, &APIError{Code: CodeUnauthorized, Message: "authentication required"})
		return
	}
	start := time.Now()
	if h.logger != nil {
		h.logger.Info("reports_list_started", "user_id", p.UserID, "role", string(p.Role))
	}

	q := r.URL.Query()
	limit, _ := strconv.Atoi(q.Get("limit"))
	if limit <= 0 || limit > 200 {
		limit = 50
	}
	offset, _ := strconv.Atoi(q.Get("offset"))

	isManager := p.Role == rbac.RoleSuperAdmin || p.Role == rbac.RoleSalesDevManager
	items, total, err := h.reports.ListForUser(r.Context(), p.UserID, isManager, limit, offset)
	if err != nil {
		if h.logger != nil {
			h.logger.Error("reports_list_failed",
				"user_id", p.UserID, "role", string(p.Role),
				"error", err.Error(), "latency_ms", time.Since(start).Milliseconds())
		}
		internalErr(w, err)
		return
	}
	if items == nil {
		items = []*reports.ReportListItem{}
	}
	if h.logger != nil {
		h.logger.Info("reports_list_completed",
			"user_id", p.UserID, "role", string(p.Role),
			"count", len(items), "total", total, "latency_ms", time.Since(start).Milliseconds())
	}
	RespondOK(w, map[string]any{"items": items, "total": total, "count": len(items)})
}
