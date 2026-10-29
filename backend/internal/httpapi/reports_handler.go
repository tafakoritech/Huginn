package httpapi

import (
	"log/slog"
	"net/http"
	"strconv"
	"time"

	"github.com/tafakorpardazan/huginn/internal/analytics"
	"github.com/tafakorpardazan/huginn/internal/auth"
	"github.com/tafakorpardazan/huginn/internal/rbac"
)

// reportsHandler serves the canonical reporting API.
type reportsHandler struct {
	a      *analytics.Service
	logger *slog.Logger
}

func newReportsHandler(a *analytics.Service, logger *slog.Logger) *reportsHandler {
	return &reportsHandler{a: a, logger: logger}
}

// parseFilters reads the shared report filters from query params. Empty values
// are treated as absent so no empty-string/empty-UUID filters are ever bound.
func parseFilters(r *http.Request) analytics.Filters {
	q := r.URL.Query()
	f := analytics.Filters{}
	if v := q.Get("from"); v != "" {
		if t, err := time.Parse(time.RFC3339, v); err == nil {
			f.From = &t
		}
	}
	if v := q.Get("to"); v != "" {
		if t, err := time.Parse(time.RFC3339, v); err == nil {
			f.To = &t
		}
	}
	f.ProjectID = optStr(q.Get("project_id"))
	f.CampaignID = optStr(q.Get("campaign_id"))
	f.AssignedUserID = optStr(q.Get("assigned_user_id"))
	f.LeadStatus = optStr(q.Get("lead_status"))
	f.QualityStatus = optStr(q.Get("quality_status"))
	f.Channel = optStr(q.Get("channel"))
	f.Outcome = optStr(q.Get("outcome"))
	f.Interest = optStr(q.Get("interest"))
	return f
}

// optStr returns nil for empty strings so an absent filter is never bound.
func optStr(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}

// scope returns filters plus the RBAC user scope: a sales agent only ever sees
// their own data unless they are a manager/admin.
func (h *reportsHandler) scope(r *http.Request, f analytics.Filters) (analytics.Filters, *auth.Principal, bool) {
	p, ok := auth.FromContext(r.Context())
	if !ok {
		return f, nil, false
	}
	if p.Role == rbac.RoleSalesAgent {
		if f.AssignedUserID == nil {
			f.AssignedUserID = &p.UserID
		}
	}
	return f, p, true
}

// Overview KPI cards.
func (h *reportsHandler) Overview(w http.ResponseWriter, r *http.Request) {
	f, p, ok := h.scope(r, parseFilters(r))
	if !ok {
		unauthorized(w)
		return
	}
	out, err := h.a.Overview(r.Context(), f)
	if err != nil {
		h.fail(w, "overview", err)
		return
	}
	h.logReports(r, p, "overview", f)
	RespondOK(w, out)
}

// Funnel conversion stages.
func (h *reportsHandler) Funnel(w http.ResponseWriter, r *http.Request) {
	f, p, ok := h.scope(r, parseFilters(r))
	if !ok {
		unauthorized(w)
		return
	}
	out, err := h.a.Funnel(r.Context(), f)
	if err != nil {
		h.fail(w, "funnel", err)
		return
	}
	h.logReports(r, p, "funnel", f)
	RespondOK(w, out)
}

// Activity time series.
func (h *reportsHandler) Activity(w http.ResponseWriter, r *http.Request) {
	f, p, ok := h.scope(r, parseFilters(r))
	if !ok {
		unauthorized(w)
		return
	}
	bucket := r.URL.Query().Get("bucket")
	if bucket != "week" {
		bucket = "day"
	}
	out, err := h.a.Activity(r.Context(), f, bucket)
	if err != nil {
		h.fail(w, "activity", err)
		return
	}
	h.logReports(r, p, "activity", f)
	RespondOK(w, out)
}

// ScoreDistribution histogram.
func (h *reportsHandler) ScoreDistribution(w http.ResponseWriter, r *http.Request) {
	f, p, ok := h.scope(r, parseFilters(r))
	if !ok {
		unauthorized(w)
		return
	}
	buckets, avg, median, high, err := h.a.ScoreDistribution(r.Context(), f)
	if err != nil {
		h.fail(w, "score_distribution", err)
		return
	}
	h.logReports(r, p, "score_distribution", f)
	RespondOK(w, map[string]any{
		"buckets":      buckets,
		"average":      avg,
		"median":       median,
		"high_quality": high,
	})
}

// Objections top list.
func (h *reportsHandler) Objections(w http.ResponseWriter, r *http.Request) {
	f, p, ok := h.scope(r, parseFilters(r))
	if !ok {
		unauthorized(w)
		return
	}
	limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
	out, total, err := h.a.Objections(r.Context(), f, limit)
	if err != nil {
		h.fail(w, "objections", err)
		return
	}
	h.logReports(r, p, "objections", f)
	RespondOK(w, map[string]any{"objections": out, "total_with_objection": total})
}

// Outcomes distribution.
func (h *reportsHandler) Outcomes(w http.ResponseWriter, r *http.Request) {
	f, p, ok := h.scope(r, parseFilters(r))
	if !ok {
		unauthorized(w)
		return
	}
	out, err := h.a.Outcomes(r.Context(), f)
	if err != nil {
		h.fail(w, "outcomes", err)
		return
	}
	h.logReports(r, p, "outcomes", f)
	RespondOK(w, map[string]any{"outcomes": out})
}

// Interest distribution.
func (h *reportsHandler) Interest(w http.ResponseWriter, r *http.Request) {
	f, p, ok := h.scope(r, parseFilters(r))
	if !ok {
		unauthorized(w)
		return
	}
	out, err := h.a.Interest(r.Context(), f)
	if err != nil {
		h.fail(w, "interest", err)
		return
	}
	h.logReports(r, p, "interest", f)
	RespondOK(w, map[string]any{"interest": out})
}

// Agents performance (RBAC: agents see only themselves).
func (h *reportsHandler) Agents(w http.ResponseWriter, r *http.Request) {
	f, p, ok := h.scope(r, parseFilters(r))
	if !ok {
		unauthorized(w)
		return
	}
	manager := p.Role == rbac.RoleSuperAdmin || p.Role == rbac.RoleSalesDevManager
	out, err := h.a.Agents(r.Context(), f, p.UserID, manager)
	if err != nil {
		h.fail(w, "agents", err)
		return
	}
	h.logReports(r, p, "agents", f)
	RespondOK(w, map[string]any{"agents": out})
}

// Projects performance.
func (h *reportsHandler) Projects(w http.ResponseWriter, r *http.Request) {
	f, p, ok := h.scope(r, parseFilters(r))
	if !ok {
		unauthorized(w)
		return
	}
	out, err := h.a.Projects(r.Context(), f)
	if err != nil {
		h.fail(w, "projects", err)
		return
	}
	h.logReports(r, p, "projects", f)
	RespondOK(w, map[string]any{"projects": out})
}

// FollowUps reporting.
func (h *reportsHandler) FollowUps(w http.ResponseWriter, r *http.Request) {
	f, p, ok := h.scope(r, parseFilters(r))
	if !ok {
		unauthorized(w)
		return
	}
	out, err := h.a.FollowUps(r.Context(), f)
	if err != nil {
		h.fail(w, "followups", err)
		return
	}
	h.logReports(r, p, "followups", f)
	RespondOK(w, out)
}

// DecisionMakers report.
func (h *reportsHandler) DecisionMakers(w http.ResponseWriter, r *http.Request) {
	f, p, ok := h.scope(r, parseFilters(r))
	if !ok {
		unauthorized(w)
		return
	}
	out, err := h.a.DecisionMakers(r.Context(), f)
	if err != nil {
		h.fail(w, "decision_makers", err)
		return
	}
	h.logReports(r, p, "decision_makers", f)
	RespondOK(w, out)
}

func (h *reportsHandler) fail(w http.ResponseWriter, section string, err error) {
	if h.logger != nil {
		h.logger.Error("reports_failed", "section", section, "error", err.Error())
	}
	internalErr(w, err)
}

func (h *reportsHandler) logReports(r *http.Request, p *auth.Principal, section string, f analytics.Filters) {
	if p == nil || h.logger == nil {
		return
	}
	h.logger.Info("reports_requested",
		"user_id", p.UserID,
		"role", string(p.Role),
		"section", section,
		"from", valueOrEmpty(f.FromStr()),
		"to", valueOrEmpty(f.ToStr()),
		"project_id", valueOrEmpty(f.ProjectID),
		"campaign_id", valueOrEmpty(f.CampaignID),
		"assigned_user_id", valueOrEmpty(f.AssignedUserID),
		"lead_status", valueOrEmpty(f.LeadStatus),
	)
}

func unauthorized(w http.ResponseWriter) {
	RespondError(w, http.StatusUnauthorized, &APIError{Code: CodeUnauthorized, Message: "authentication required"})
}
