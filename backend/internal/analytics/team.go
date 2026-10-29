package analytics

import (
	"context"
	"regexp"
	"strconv"
	"strings"
)

// renumber shifts every $N placeholder by base, so several independent filter
// fragments can share one argument list.
func renumber(sql string, base int) string {
	return placeholderRe.ReplaceAllStringFunc(sql, func(m string) string {
		n, _ := strconv.Atoi(strings.TrimPrefix(m, "$"))
		return "$" + strconv.Itoa(n+base)
	})
}

var placeholderRe = regexp.MustCompile(`\$(\d+)`)

// ---------------------------------------------------------------------------
// Agent performance

// AgentPerformanceRow is the per-agent performance summary. Activity metrics
// (interactions, reports, successful contacts) are distinct from outcome
// metrics (qualified, conversion). Rates are computed server-side.
type AgentPerformanceRow struct {
	UserID                   string  `json:"user_id"`
	Name                     string  `json:"name"`
	AssignedLeads            int     `json:"assigned_leads"`
	Interactions             int     `json:"interactions"`
	ReportsSubmitted         int     `json:"reports_submitted"`
	SuccessfulContacts       int     `json:"successful_contacts"`
	DecisionMakersIdentified int     `json:"decision_makers_identified"`
	QualifiedLeads           int     `json:"qualified_leads"`
	FollowUpsCompleted       int     `json:"followups_completed"`
	OverdueFollowUps         int     `json:"overdue_followups"`
	ContactRate              float64 `json:"contact_rate"`
	QualificationRate        float64 `json:"qualification_rate"`
}

// Agents returns per-agent rows. When manager is false (a sales agent), only
// the requesting user's row is returned (RBAC).
func (s *Service) Agents(ctx context.Context, f Filters, viewerID string, manager bool) ([]AgentPerformanceRow, error) {
	args := []any{}
	userWhere := "u.role = 'sales_agent' AND u.is_active AND u.deleted_at IS NULL"
	if !manager {
		userWhere = "u.id = $1"
		args = append(args, viewerID)
	}

	// Each CTE uses the shared filters; renumber placeholders so the combined
	// statement has one flat argument list.
	lc := f.leadConditions("l")
	leadFragSQL := strings.ReplaceAll(lc.sql, "1=1 AND ", "")
	leadFragSQL = strings.ReplaceAll(leadFragSQL, "1=1", "")
	leadArgs := lc.args

	ic := f.interactionConditions()
	interFragSQL := strings.ReplaceAll(ic.sql, "1=1 AND ", "")
	interFragSQL = strings.ReplaceAll(interFragSQL, "1=1", "")
	interArgs := append([]any{}, ic.args...)

	fc := f.followUpConditions()
	fuFragSQL := strings.ReplaceAll(fc.sql, "1=1 AND ", "")
	fuFragSQL = strings.ReplaceAll(fuFragSQL, "1=1", "")
	fuArgs := append([]any{}, fc.args...)

	// Build the fragments with correctly numbered placeholders.
	leadFrag := renumber(leadFragSQL, len(args))
	args = append(args, leadArgs...)
	interFrag := renumber(interFragSQL, len(args))
	args = append(args, interArgs...)
	fuFrag := renumber(fuFragSQL, len(args))
	args = append(args, fuArgs...)

	query := `
	WITH agent_users AS (SELECT id, name FROM users WHERE ` + userWhere + `),
	lead_agg AS (
		SELECT l.owner_user_id AS uid, count(*) AS assigned,
		       count(*) FILTER (WHERE l.status IN ('QUALIFIED','EXPORTED_TO_MUNINN')) AS qualified
		FROM leads l WHERE ` + leadFrag + `
		GROUP BY 1),
	inter_agg AS (
		SELECT i.user_id AS uid,
		       count(DISTINCT i.id) AS interactions,
		       count(DISTINCT cr.id) AS reports,
		       count(DISTINCT cr.id) FILTER (WHERE cr.contact_reached) AS success,
		       count(DISTINCT cr.id) FILTER (WHERE cr.decision_maker_identified) AS dms
		FROM interactions i LEFT JOIN contact_reports cr ON cr.interaction_id = i.id
		WHERE ` + interFrag + `
		GROUP BY 1),
	fu_agg AS (
		SELECT f.assigned_to AS uid,
		       count(*) FILTER (WHERE f.status='COMPLETED') AS completed,
		       count(*) FILTER (WHERE f.status='PENDING' AND f.due_at < now()) AS overdue
		FROM follow_ups f WHERE ` + fuFrag + `
		GROUP BY 1)
	SELECT u.id, u.name,
	       COALESCE(l.assigned,0), COALESCE(i.interactions,0), COALESCE(i.reports,0),
	       COALESCE(i.success,0), COALESCE(i.dms,0), COALESCE(l.qualified,0),
	       COALESCE(f.completed,0), COALESCE(f.overdue,0), COALESCE(l.assigned,0)
	FROM agent_users u
	LEFT JOIN lead_agg l ON l.uid = u.id
	LEFT JOIN inter_agg i ON i.uid = u.id
	LEFT JOIN fu_agg f ON f.uid = u.id
	ORDER BY u.name`

	rows, err := s.db.Pool().Query(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []AgentPerformanceRow{}
	for rows.Next() {
		var r AgentPerformanceRow
		var decidedBase int
		if err := rows.Scan(&r.UserID, &r.Name, &r.AssignedLeads, &r.Interactions, &r.ReportsSubmitted,
			&r.SuccessfulContacts, &r.DecisionMakersIdentified, &r.QualifiedLeads,
			&r.FollowUpsCompleted, &r.OverdueFollowUps, &decidedBase); err != nil {
			return nil, err
		}
		if r.Interactions > 0 {
			r.ContactRate = float64(r.SuccessfulContacts) / float64(r.Interactions) * 100
		}
		if decidedBase > 0 {
			r.QualificationRate = float64(r.QualifiedLeads) / float64(decidedBase) * 100
		}
		out = append(out, r)
	}
	return out, rows.Err()
}

// ---------------------------------------------------------------------------
// Project / campaign performance

type ProjectPerformance struct {
	ProjectID      string  `json:"project_id"`
	Title          string  `json:"title"`
	Leads          int     `json:"leads"`
	Contacts       int     `json:"contacts"`
	Reports        int     `json:"reports"`
	Qualified      int     `json:"qualified"`
	ConversionRate float64 `json:"conversion_rate"`
	AvgScore       float64 `json:"avg_score"`
}

// Projects groups performance by project (optionally restricted by campaign).
func (s *Service) Projects(ctx context.Context, f Filters) ([]ProjectPerformance, error) {
	c := condition{sql: "1=1"}
	if f.CampaignID != nil {
		c.add("p.campaign_id=$"+itoa(len(c.args)+1), *f.CampaignID)
	}
	if f.AssignedUserID != nil {
		c.add("l.owner_user_id=$"+itoa(len(c.args)+1), *f.AssignedUserID)
	}
	if f.LeadStatus != nil {
		c.add("l.status=$"+itoa(len(c.args)+1), *f.LeadStatus)
	}
	if f.QualityStatus != nil {
		c.add("l.quality_status=$"+itoa(len(c.args)+1), *f.QualityStatus)
	}
	if f.From != nil {
		c.add("l.created_at >= $"+itoa(len(c.args)+1), *f.From)
	}
	if f.To != nil {
		c.add("l.created_at <= $"+itoa(len(c.args)+1), *f.To)
	}

	rows, err := s.db.Pool().Query(ctx, `
		SELECT p.id, p.title,
		       count(DISTINCT l.id) AS leads,
		       count(DISTINCT i.id) AS contacts,
		       count(DISTINCT cr.id) AS reports,
		       count(DISTINCT l.id) FILTER (WHERE l.status IN ('QUALIFIED','EXPORTED_TO_MUNINN')) AS qualified,
		       COALESCE(avg(l.score),0) AS avg_score
		FROM projects p
		LEFT JOIN leads l ON l.project_id = p.id
		LEFT JOIN interactions i ON i.lead_id = l.id
		LEFT JOIN contact_reports cr ON cr.interaction_id = i.id
		WHERE `+c.sql+`
		GROUP BY p.id, p.title
		ORDER BY leads DESC`, c.args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []ProjectPerformance{}
	for rows.Next() {
		var p ProjectPerformance
		if err := rows.Scan(&p.ProjectID, &p.Title, &p.Leads, &p.Contacts, &p.Reports, &p.Qualified, &p.AvgScore); err != nil {
			return nil, err
		}
		if p.Leads > 0 {
			p.ConversionRate = float64(p.Qualified) / float64(p.Leads) * 100
		}
		out = append(out, p)
	}
	return out, rows.Err()
}

// ---------------------------------------------------------------------------
// Follow-up reporting

type FollowUpStatusCount struct {
	Status string `json:"status"`
	Count  int    `json:"count"`
}

type FollowUpReport struct {
	Pending        int                   `json:"pending"`
	DueToday       int                   `json:"due_today"`
	Overdue        int                   `json:"overdue"`
	Completed      int                   `json:"completed"`
	CompletionRate float64               `json:"completion_rate"`
	AvgDelaySeconds float64              `json:"avg_delay_seconds"`
	ByStatus       []FollowUpStatusCount `json:"by_status"`
}

func (s *Service) FollowUps(ctx context.Context, f Filters) (*FollowUpReport, error) {
	fc := f.followUpConditions()
	var out FollowUpReport
	var decidedTotal int
	err := s.db.Pool().QueryRow(ctx, `
		SELECT count(*) FILTER (WHERE status='PENDING'),
		       count(*) FILTER (WHERE status='PENDING' AND due_at >= date_trunc('day', now()) AND due_at < date_trunc('day', now()) + interval '1 day'),
		       count(*) FILTER (WHERE status='PENDING' AND due_at < now()),
		       count(*) FILTER (WHERE status='COMPLETED'),
		       count(*) FILTER (WHERE status='COMPLETED' OR status='PENDING'),
		       COALESCE(avg(EXTRACT(EPOCH FROM (completed_at - due_at))) FILTER (WHERE status='COMPLETED' AND completed_at IS NOT NULL),0)
		FROM follow_ups f WHERE `+fc.sql, fc.args...).
		Scan(&out.Pending, &out.DueToday, &out.Overdue, &out.Completed, &decidedTotal, &out.AvgDelaySeconds)
	if err != nil {
		return nil, err
	}
	if decidedTotal > 0 {
		out.CompletionRate = float64(out.Completed) / float64(decidedTotal) * 100
	}
	// status distribution
	rows, err := s.db.Pool().Query(ctx, `
		SELECT COALESCE(status,'UNKNOWN'), count(*) FROM follow_ups f WHERE `+fc.sql+`
		GROUP BY 1 ORDER BY 2 DESC`, fc.args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	for rows.Next() {
		var sc FollowUpStatusCount
		if err := rows.Scan(&sc.Status, &sc.Count); err != nil {
			return nil, err
		}
		out.ByStatus = append(out.ByStatus, sc)
	}
	return &out, rows.Err()
}

// ---------------------------------------------------------------------------
// Decision-maker report

type DecisionMakerReport struct {
	Identified              int     `json:"identified"`
	NotIdentified           int     `json:"not_identified"`
	CorrectPersonReached    int     `json:"correct_person_reached"`
	IdentificationRate      float64 `json:"identification_rate"`
}

func (s *Service) DecisionMakers(ctx context.Context, f Filters) (*DecisionMakerReport, error) {
	lc := f.leadConditions("l")
	lc.sql += " AND l.status <> 'ARCHIVED'"
	var out DecisionMakerReport
	var base int
	err := s.db.Pool().QueryRow(ctx, `
		SELECT count(*) AS base,
		       count(*) FILTER (WHERE l.decision_maker_identified) AS identified,
		       count(*) FILTER (WHERE NOT l.decision_maker_identified) AS not_identified,
		       count(*) FILTER (WHERE l.correct_person_reached) AS correct
		FROM leads l WHERE `+lc.sql, lc.args...).
		Scan(&base, &out.Identified, &out.NotIdentified, &out.CorrectPersonReached)
	if err != nil {
		return nil, err
	}
	if base > 0 {
		out.IdentificationRate = float64(out.Identified) / float64(base) * 100
	}
	return &out, nil
}
