package httpapi

import (
	"encoding/json"
	"net/http"
	"strconv"
	"time"

	"github.com/go-chi/chi/v5"
)

// AgentsList returns per-agent performance rows.
func (h *dataHandler) AgentsList(w http.ResponseWriter, r *http.Request) {
	rows, err := h.s.Dashboard.AgentPerformance(r.Context())
	if err != nil {
		internalErr(w, err)
		return
	}
	type agentRow struct {
		UserID             string  `json:"user_id"`
		Name               string  `json:"name"`
		AssignedLeads      int     `json:"assigned_leads"`
		CallsToday         int     `json:"calls_today"`
		SuccessfulContacts int     `json:"successful_contacts"`
		PendingReports     int     `json:"pending_reports"`
		OverdueFollowups   int     `json:"overdue_followups"`
		QualifiedLeads     int     `json:"qualified_leads"`
		QualificationRate  float64 `json:"qualification_rate"`
	}
	out := make([]agentRow, 0, len(rows))
	for _, row := range rows {
		out = append(out, agentRow{
			UserID: row.UserID, Name: row.Name, AssignedLeads: row.AssignedLeads,
			CallsToday: row.CallsToday, SuccessfulContacts: row.SuccessfulContacts,
			PendingReports: row.PendingReports, OverdueFollowups: row.OverdueFollowups,
			QualifiedLeads: row.QualifiedLeads, QualificationRate: row.QualificationRate,
		})
	}
	RespondOK(w, out)
}

// AgentGet returns one agent's performance, current queue and recent activity.
func (h *dataHandler) AgentGet(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "userID")
	db := h.s.DB

	var name string
	if err := db.Pool().QueryRow(r.Context(), `SELECT name FROM users WHERE id=$1 AND is_active AND deleted_at IS NULL`, id).Scan(&name); err != nil {
		RespondError(w, http.StatusNotFound, &APIError{Code: CodeNotFound, Message: "agent not found"})
		return
	}

	var assigned, callsToday, success, pending, overdue, qualified, disqualified int
	_ = db.Pool().QueryRow(r.Context(), `SELECT count(*) FROM leads WHERE owner_user_id=$1 AND status NOT IN ('ARCHIVED')`, id).Scan(&assigned)
	_ = db.Pool().QueryRow(r.Context(), `SELECT count(*) FROM interactions WHERE user_id=$1 AND channel='CALL' AND started_at >= date_trunc('day', now())`, id).Scan(&callsToday)
	_ = db.Pool().QueryRow(r.Context(), `SELECT count(*) FROM contact_reports cr JOIN interactions i ON i.id=cr.interaction_id WHERE i.user_id=$1 AND cr.outcome IN ('INTERESTED','HIGHLY_INTERESTED','ANSWERED') AND i.started_at >= date_trunc('day', now())`, id).Scan(&success)
	_ = db.Pool().QueryRow(r.Context(), `SELECT count(*) FROM interactions WHERE user_id=$1 AND report_status='PENDING'`, id).Scan(&pending)
	_ = db.Pool().QueryRow(r.Context(), `SELECT count(*) FROM follow_ups WHERE assigned_to=$1 AND status='OVERDUE'`, id).Scan(&overdue)
	_ = db.Pool().QueryRow(r.Context(), `SELECT count(*) FROM leads WHERE owner_user_id=$1 AND status='QUALIFIED'`, id).Scan(&qualified)
	_ = db.Pool().QueryRow(r.Context(), `SELECT count(*) FROM leads WHERE owner_user_id=$1 AND status='DISQUALIFIED'`, id).Scan(&disqualified)

	qualRate := 0.0
	if qualified+disqualified > 0 {
		qualRate = float64(qualified) / float64(qualified+disqualified) * 100
	}

	// Current queue.
	qrows, err := db.Pool().Query(r.Context(), `
		SELECT id, company_name, city, score, status FROM leads
		WHERE owner_user_id=$1 AND status NOT IN ('ARCHIVED','EXPORTED_TO_MUNINN')
		ORDER BY score DESC LIMIT 50`, id)
	if err != nil {
		internalErr(w, err)
		return
	}
	defer qrows.Close()
	queue := []map[string]any{}
	for qrows.Next() {
		var leadID, company, city, status string
		var score int
		if err := qrows.Scan(&leadID, &company, &city, &score, &status); err != nil {
			continue
		}
		queue = append(queue, map[string]any{"id": leadID, "company_name": company, "city": city, "score": score, "status": status})
	}

	// Recent activity.
	arows, err := db.Pool().Query(r.Context(), `
		SELECT t.id, t.lead_id, t.event_type, t.title, t.description, t.created_at,
		       COALESCE(l.company_name,'')
		FROM timeline_events t LEFT JOIN leads l ON l.id = t.lead_id
		WHERE t.actor_user_id=$1 ORDER BY t.created_at DESC LIMIT 15`, id)
	if err != nil {
		internalErr(w, err)
		return
	}
	defer arows.Close()
	activity := []map[string]any{}
	for arows.Next() {
		var evID, leadID, eventType, title, company string
		var description *string
		var createdAt time.Time
		if err := arows.Scan(&evID, &leadID, &eventType, &title, &description, &createdAt, &company); err != nil {
			continue
		}
		activity = append(activity, map[string]any{
			"id": evID, "lead_id": leadID, "event_type": eventType, "title": title,
			"description": description, "created_at": createdAt, "company_name": company,
		})
	}

	RespondOK(w, map[string]any{
		"user":                map[string]any{"id": id, "name": name},
		"assigned_leads":      assigned,
		"calls_today":         callsToday,
		"successful_contacts": success,
		"pending_reports":     pending,
		"overdue_followups":   overdue,
		"qualified_leads":     qualified,
		"qualification_rate":  qualRate,
		"queue":               queue,
		"recent_activity":     activity,
	})
}

// AuditList returns paginated audit entries.
func (h *dataHandler) AuditList(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	page := intQuery(q.Get("page"), 1)
	pageSize := intQuery(q.Get("page_size"), 50)
	if pageSize > 200 {
		pageSize = 200
	}
	if pageSize <= 0 {
		pageSize = 50
	}
	offset := (page - 1) * pageSize

	args := []any{}
	argc := 1
	conds := []string{}
	addCond := func(c string, v any) {
		conds = append(conds, c+"=$"+strconv.Itoa(argc))
		args = append(args, v)
		argc++
	}
	if v := q.Get("user_id"); v != "" {
		addCond("a.actor_user_id", v)
	}
	if v := q.Get("action"); v != "" {
		addCond("a.action ILIKE '%'||$"+strconv.Itoa(argc), "") // placeholder replaced below
		_ = v
		args[len(args)-1] = "%" + v + "%"
		conds[len(conds)-1] = "a.action ILIKE $"+strconv.Itoa(argc-1)
	}
	if v := q.Get("entity_type"); v != "" {
		addCond("a.entity_type", v)
	}

	where := ""
	if len(conds) > 0 {
		where = " WHERE " + joinAnd(conds)
	}

	var total int
	countArgs := append([]any{}, args...)
	_ = h.s.DB.Pool().QueryRow(r.Context(), "SELECT count(*) FROM audit_log a"+where, countArgs...).Scan(&total)

	args = append(args, pageSize, offset)
	sqlQ := `SELECT a.id, a.actor_user_id, a.action, a.entity_type, a.entity_id, a.before_json, a.after_json,
		a.ip, a.user_agent, a.request_id, a.created_at, a.metadata, COALESCE(u.name,'')
		FROM audit_log a LEFT JOIN users u ON u.id = a.actor_user_id` + where +
		` ORDER BY a.created_at DESC LIMIT $` + strconv.Itoa(argc) + ` OFFSET $` + strconv.Itoa(argc+1)
	rows, err := h.s.DB.Pool().Query(r.Context(), sqlQ, args...)
	if err != nil {
		internalErr(w, err)
		return
	}
	defer rows.Close()
	items := []map[string]any{}
	for rows.Next() {
		var e auditEntry
		var actorName string
		if err := rows.Scan(&e.ID, &e.ActorUserID, &e.Action, &e.EntityType, &e.EntityID, &e.BeforeJSON, &e.AfterJSON,
			&e.IP, &e.UserAgent, &e.RequestID, &e.CreatedAt, &e.Metadata, &actorName); err != nil {
			continue
		}
		item := map[string]any{
			"id": e.ID, "actor_user_id": e.ActorUserID, "action": e.Action, "entity_type": e.EntityType,
			"entity_id": e.EntityID, "before_json": e.BeforeJSON, "after_json": e.AfterJSON,
			"ip": e.IP, "user_agent": e.UserAgent, "request_id": e.RequestID, "created_at": e.CreatedAt,
			"metadata": e.Metadata,
		}
		if actorName != "" {
			item["actor"] = map[string]any{"id": e.ActorUserID, "name": actorName}
		}
		items = append(items, item)
	}
	RespondOK(w, map[string]any{"items": items, "total": total, "page": page, "page_size": pageSize})
}

type auditEntry struct {
	ID          string          `json:"id"`
	ActorUserID *string         `json:"actor_user_id"`
	Action      string          `json:"action"`
	EntityType  string          `json:"entity_type"`
	EntityID    string          `json:"entity_id"`
	BeforeJSON  json.RawMessage `json:"before_json,omitempty"`
	AfterJSON   json.RawMessage `json:"after_json,omitempty"`
	IP          string          `json:"ip,omitempty"`
	UserAgent   string          `json:"user_agent,omitempty"`
	RequestID   string          `json:"request_id,omitempty"`
	CreatedAt   time.Time       `json:"created_at"`
	Metadata    json.RawMessage `json:"metadata,omitempty"`
}

// GlobalSearch searches leads and users.
func (h *dataHandler) GlobalSearch(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query().Get("q")
	if q == "" {
		RespondOK(w, map[string]any{"leads": []any{}, "users": []any{}})
		return
	}
	pattern := "%" + q + "%"
	leadsRows, err := h.s.DB.Pool().Query(r.Context(), `
		SELECT id, company_name, city, status, score FROM leads
		WHERE company_name ILIKE $1 OR city ILIKE $1 OR id::text ILIKE $1
		ORDER BY score DESC LIMIT 8`, pattern)
	if err != nil {
		internalErr(w, err)
		return
	}
	defer leadsRows.Close()
	leadsOut := []map[string]any{}
	for leadsRows.Next() {
		var id, company, city, status string
		var score int
		if err := leadsRows.Scan(&id, &company, &city, &status, &score); err != nil {
			continue
		}
		leadsOut = append(leadsOut, map[string]any{"id": id, "company_name": company, "city": city, "status": status, "score": score})
	}

	userRows, err := h.s.DB.Pool().Query(r.Context(), `
		SELECT id, name, username FROM users
		WHERE name ILIKE $1 OR username ILIKE $1 OR email ILIKE $1 OR mobile ILIKE $1
		LIMIT 5`, pattern)
	if err != nil {
		internalErr(w, err)
		return
	}
	defer userRows.Close()
	usersOut := []map[string]any{}
	for userRows.Next() {
		var id, name, username string
		if err := userRows.Scan(&id, &name, &username); err != nil {
			continue
		}
		usersOut = append(usersOut, map[string]any{"id": id, "name": name, "username": username})
	}
	RespondOK(w, map[string]any{"leads": leadsOut, "users": usersOut})
}

func intQuery(s string, def int) int {
	if s == "" {
		return def
	}
	n, err := strconv.Atoi(s)
	if err != nil {
		return def
	}
	return n
}

func joinAnd(conds []string) string {
	out := ""
	for i, c := range conds {
		if i > 0 {
			out += " AND "
		}
		out += c
	}
	return out
}
