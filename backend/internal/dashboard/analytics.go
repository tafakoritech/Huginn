package dashboard

import (
	"context"
	"fmt"

	"github.com/tafakorpardazan/huginn/internal/leads"
)

// FunnelStep is one stage of the lead conversion funnel.
type FunnelStep struct {
	Stage string `json:"stage"`
	Count int    `json:"count"`
}

// QualificationBySource aggregates leads by source with qualified counts.
type QualificationBySource struct {
	Source    string `json:"source"`
	Total     int    `json:"total"`
	Qualified int    `json:"qualified"`
}

// AgentPerformanceRow is the performance summary for a single agent.
type AgentPerformanceRow struct {
	UserID              string  `json:"user_id"`
	Name                string  `json:"name"`
	AssignedLeads       int     `json:"assigned_leads"`
	CallsToday          int     `json:"calls_today"`
	SuccessfulContacts  int     `json:"successful_contacts"`
	PendingReports      int     `json:"pending_reports"`
	OverdueFollowups    int     `json:"overdue_followups"`
	QualifiedLeads      int     `json:"qualified_leads"`
	QualificationRate   float64 `json:"qualification_rate"`
}

// TrendPoint is a value at a point in time.
type TrendPoint struct {
	Label string `json:"label"`
	Value int    `json:"value"`
}

// RejectionReason aggregates disqualification reasons.
type RejectionReason struct {
	Reason string `json:"reason"`
	Count  int    `json:"count"`
}

// CityCount aggregates leads by city.
type CityCount struct {
	Label string `json:"label"`
	Value int    `json:"value"`
}

// Funnel returns the lead conversion funnel counts.
func (s *Service) Funnel(ctx context.Context) ([]FunnelStep, error) {
	steps := []FunnelStep{}
	statuses := []struct {
		key  string
		st   leads.Status
	}{
		{"raw", ""},
		{"assigned", leads.StatusAssigned},
		{"contacted", leads.StatusContacting},
		{"interested", leads.StatusFollowUp},
		{"qualified", leads.StatusQualified},
		{"muninn", leads.StatusExportedMuninn},
	}
	for _, st := range statuses {
		var n int
		if st.key == "raw" {
			n, _ = s.leadRepo.CountAll(ctx)
		} else {
			n, _ = s.leadRepo.CountLeads(ctx, st.st)
		}
		steps = append(steps, FunnelStep{Stage: st.key, Count: n})
	}
	return steps, nil
}

// QualificationBySource returns total and qualified lead counts per source.
func (s *Service) QualificationBySource(ctx context.Context) ([]QualificationBySource, error) {
	rows, err := s.leadRepo.DB().Pool().Query(ctx, `
		SELECT COALESCE(ib.source_type::text, 'manual') AS source,
		       count(l.id) AS total,
		       count(*) FILTER (WHERE l.status IN ('QUALIFIED','EXPORTED_TO_MUNINN')) AS qualified
		FROM leads l
		LEFT JOIN import_batches ib ON ib.id = l.source_batch_id
		GROUP BY 1 ORDER BY total DESC LIMIT 20`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []QualificationBySource{}
	for rows.Next() {
		var q QualificationBySource
		if err := rows.Scan(&q.Source, &q.Total, &q.Qualified); err != nil {
			return nil, err
		}
		out = append(out, q)
	}
	return out, rows.Err()
}

// AgentPerformance returns per-agent performance rows for all sales agents.
func (s *Service) AgentPerformance(ctx context.Context) ([]AgentPerformanceRow, error) {
	db := s.leadRepo.DB()
	rows, err := db.Pool().Query(ctx, `
		SELECT u.id, u.name,
		       (SELECT count(*) FROM leads l WHERE l.owner_user_id = u.id AND l.status NOT IN ('ARCHIVED')) AS assigned_leads,
		       (SELECT count(*) FROM interactions i WHERE i.user_id = u.id AND i.channel = 'CALL' AND i.started_at >= date_trunc('day', now())) AS calls_today,
		       (SELECT count(*) FROM contact_reports cr JOIN interactions i ON i.id = cr.interaction_id
		         WHERE i.user_id = u.id AND cr.outcome IN ('INTERESTED','HIGHLY_INTERESTED','ANSWERED') AND i.started_at >= date_trunc('day', now())) AS successful_contacts,
		       (SELECT count(*) FROM interactions i WHERE i.user_id = u.id AND i.report_status = 'PENDING') AS pending_reports,
		       (SELECT count(*) FROM follow_ups f WHERE f.assigned_to = u.id AND f.status = 'OVERDUE') AS overdue_followups,
		       (SELECT count(*) FROM leads l WHERE l.owner_user_id = u.id AND l.status = 'QUALIFIED') AS qualified_leads,
		       (SELECT count(*) FROM leads l WHERE l.owner_user_id = u.id AND l.status IN ('DISQUALIFIED','QUALIFIED')) AS decided
		FROM users u
		WHERE u.role = 'sales_agent' AND u.is_active AND u.deleted_at IS NULL
		ORDER BY u.name`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []AgentPerformanceRow{}
	for rows.Next() {
		var r AgentPerformanceRow
		var decided int
		if err := rows.Scan(&r.UserID, &r.Name, &r.AssignedLeads, &r.CallsToday, &r.SuccessfulContacts, &r.PendingReports, &r.OverdueFollowups, &r.QualifiedLeads, &decided); err != nil {
			return nil, err
		}
		if decided > 0 {
			r.QualificationRate = float64(r.QualifiedLeads) / float64(decided) * 100
		}
		out = append(out, r)
	}
	return out, rows.Err()
}

// LeadTrend returns lead creation counts bucketed by day/week/month.
func (s *Service) LeadTrend(ctx context.Context, period string) ([]TrendPoint, error) {
	var trunc, bucket, format string
	switch period {
	case "week":
		trunc, bucket, format = "week", "11 weeks", "YYYY-MM-DD"
	case "month":
		trunc, bucket, format = "month", "11 months", "YYYY-MM"
	default:
		trunc, bucket, format = "day", "13 days", "YYYY-MM-DD"
	}
	query := fmt.Sprintf(`
		WITH series AS (
			SELECT generate_series(date_trunc('%s', now()) - interval '%s', date_trunc('%s', now()), interval '1 %s') AS bucket
		)
		SELECT to_char(s.bucket, '%s'), count(l.id)
		FROM series s
		LEFT JOIN leads l ON date_trunc('%s', l.created_at) = s.bucket
		GROUP BY 1 ORDER BY 1`, trunc, bucket, trunc, trunc, format, trunc)

	rows, err := s.leadRepo.DB().Pool().Query(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []TrendPoint{}
	for rows.Next() {
		var p TrendPoint
		if err := rows.Scan(&p.Label, &p.Value); err != nil {
			return nil, err
		}
		out = append(out, p)
	}
	if len(out) == 0 {
		out = make([]TrendPoint, 0)
	}
	return out, rows.Err()
}

// RejectionReasons returns disqualification reason counts.
func (s *Service) RejectionReasons(ctx context.Context) ([]RejectionReason, error) {
	m, err := s.byRejectReason(ctx)
	if err != nil {
		return nil, err
	}
	out := make([]RejectionReason, 0, len(m))
	for reason, count := range m {
		out = append(out, RejectionReason{Reason: reason, Count: count})
	}
	return out, nil
}

// LeadsByCity returns lead counts grouped by city.
func (s *Service) LeadsByCity(ctx context.Context) ([]CityCount, error) {
	m, err := s.byCity(ctx)
	if err != nil {
		return nil, err
	}
	out := make([]CityCount, 0, len(m))
	for city, count := range m {
		out = append(out, CityCount{Label: city, Value: count})
	}
	return out, nil
}
