package dashboard

import (
	"context"

	"github.com/tafakorpardazan/huginn/internal/followups"
	"github.com/tafakorpardazan/huginn/internal/interactions"
	"github.com/tafakorpardazan/huginn/internal/leads"
	"github.com/tafakorpardazan/huginn/internal/muninn"
	"github.com/tafakorpardazan/huginn/internal/qualification"
	"github.com/tafakorpardazan/huginn/internal/reports"
)

// ManagerMetrics are administrative KPIs.
type ManagerMetrics struct {
	TotalLeads      int `json:"total_leads"`
	NewLeads        int `json:"new_leads"`
	AssignedLeads   int `json:"assigned_leads"`
	UnassignedLeads int `json:"unassigned_leads"`
	ReviewRequired  int `json:"review_required"`

	CallsToday      int `json:"calls_today"`
	SuccessfulContacts int `json:"successful_contacts"`
	NoAnswers       int `json:"no_answers"`
	PendingReports  int `json:"pending_reports"`

	FollowUpsToday  int `json:"follow_ups_today"`
	OverdueFollowUps int `json:"overdue_follow_ups"`

	QualifiedLeads   int `json:"qualified_leads"`
	DisqualifiedLeads int `json:"disqualified_leads"`

	QualificationRate float64 `json:"qualification_rate"`
	ContactRate       float64 `json:"contact_rate"`

	ExportedToMuninn int `json:"exported_to_muninn"`

	ByEmployee   map[string]int         `json:"by_employee"`
	BySource     map[string]int         `json:"by_source"`
	ByIndustry   map[string]int         `json:"by_industry"`
	ByCity       map[string]int         `json:"by_city"`
	ByBusinessType map[string]int       `json:"by_business_type"`
	ByRejectReason map[string]int       `json:"by_reject_reason"`
}

// AgentMetrics are per-employee KPIs.
type AgentMetrics struct {
	AssignedLeads int `json:"assigned_leads"`
	CallsToday    int `json:"calls_today"`
	RemainingWorkQueue int `json:"remaining_work_queue"`
	PendingReports int `json:"pending_reports"`
	FollowUps     int `json:"follow_ups"`
	QualifiedLeads int `json:"qualified_leads"`
	ContactRate   float64 `json:"contact_rate"`
}

// Service aggregates dashboard metrics.
type Service struct {
	leadRepo       *leads.Repository
	interRepo      *interactions.Repository
	followupRepo   *followups.Repository
	reportRepo     *reports.Repository
	qualRepo       *qualification.Repository
	exportRepo     *muninn.Repository
}

func NewService(leadRepo *leads.Repository, interRepo *interactions.Repository,
	followupRepo *followups.Repository, reportRepo *reports.Repository,
	qualRepo *qualification.Repository, exportRepo *muninn.Repository) *Service {
	return &Service{leadRepo: leadRepo, interRepo: interRepo, followupRepo: followupRepo,
		reportRepo: reportRepo, qualRepo: qualRepo, exportRepo: exportRepo}
}

// Manager aggregates the manager dashboard.
func (s *Service) Manager(ctx context.Context) (*ManagerMetrics, error) {
	m := &ManagerMetrics{}

	total, err := s.leadRepo.CountAll(ctx)
	if err != nil {
		return nil, err
	}
	m.TotalLeads = total

	for _, st := range []struct {
		status leads.Status
		dest   *int
	}{
		{leads.StatusNew, &m.NewLeads},
		{leads.StatusAssigned, &m.AssignedLeads},
		{leads.StatusQualified, &m.QualifiedLeads},
		{leads.StatusDisqualified, &m.DisqualifiedLeads},
		{leads.StatusExportedMuninn, &m.ExportedToMuninn},
	} {
		n, err := s.leadRepo.CountLeads(ctx, st.status)
		if err != nil {
			return nil, err
		}
		*st.dest = n
	}

	m.UnassignedLeads, _ = s.countUnassigned(ctx)

	for _, st := range []struct {
		status leads.Status
		dest   *int
	}{
		{leads.StatusNew, &m.NewLeads},
		{leads.StatusReady, &m.ReviewRequired},
	} {
		n, err := s.leadRepo.CountLeads(ctx, st.status)
		if err != nil {
			return nil, err
		}
		*st.dest = n
	}
	m.ReviewRequired += m.NewLeads

	m.CallsToday, _ = s.interRepo.CountCallsToday(ctx, "")
	m.PendingReports, _ = s.interRepo.CountPendingReportsAll(ctx)
	m.SuccessfulContacts, _ = s.reportRepo.CountOutcomeToday(ctx, reports.OutcomeInterested)
	nhi, _ := s.reportRepo.CountOutcomeToday(ctx, reports.OutcomeHighlyInterested)
	m.SuccessfulContacts += nhi
	m.NoAnswers, _ = s.reportRepo.CountOutcomeToday(ctx, reports.OutcomeNoAnswer)

	m.FollowUpsToday, _ = s.followupRepo.CountOpenToday(ctx, "")
	m.OverdueFollowUps, _ = s.followupRepo.CountOverdue(ctx, "")

	totalDisq := m.DisqualifiedLeads
	if totalDisq > 0 {
		m.QualificationRate = float64(m.QualifiedLeads) / float64(m.QualifiedLeads+totalDisq) * 100
	}
	if m.CallsToday > 0 {
		m.ContactRate = float64(m.SuccessfulContacts) / float64(m.CallsToday) * 100
	}

	m.ByEmployee, _ = s.byEmployee(ctx)
	m.BySource, _ = s.bySource(ctx)
	m.ByIndustry, _ = s.byIndustry(ctx)
	m.ByCity, _ = s.byCity(ctx)
	m.ByBusinessType, _ = s.byBusinessType(ctx)
	m.ByRejectReason, _ = s.byRejectReason(ctx)

	return m, nil
}

// Agent aggregates the per-employee dashboard.
func (s *Service) Agent(ctx context.Context, userID string) (*AgentMetrics, error) {
	a := &AgentMetrics{}

	q := s.leadRepo.DB()
	if err := q.Pool().QueryRow(ctx, `SELECT count(*) FROM leads WHERE owner_user_id=$1 AND status NOT IN ('QUALIFIED','DISQUALIFIED','EXPORTED_TO_MUNINN','ARCHIVED')`, userID).Scan(&a.AssignedLeads); err != nil {
		return nil, err
	}
	a.CallsToday, _ = s.interRepo.CountCallsToday(ctx, userID)
	a.PendingReports, _ = s.interRepo.CountPendingReportsForUser(ctx, userID)
	a.FollowUps, _ = s.followupRepo.CountOpenToday(ctx, userID)

	wq, err := s.leadRepo.WorkQueue(ctx, userID)
	if err == nil {
		a.RemainingWorkQueue = len(wq)
	}
	if err := q.Pool().QueryRow(ctx, `SELECT count(*) FROM leads WHERE owner_user_id=$1 AND status='QUALIFIED'`, userID).Scan(&a.QualifiedLeads); err != nil {
		return nil, err
	}
	if a.CallsToday > 0 {
		success, _ := s.countAgentSuccess(ctx, userID)
		a.ContactRate = float64(success) / float64(a.CallsToday) * 100
	}
	return a, nil
}

func (s *Service) countUnassigned(ctx context.Context) (int, error) {
	var n int
	err := s.leadRepo.DB().Pool().QueryRow(ctx, `SELECT count(*) FROM leads WHERE owner_user_id IS NULL`).Scan(&n)
	return n, err
}

func (s *Service) countAgentSuccess(ctx context.Context, userID string) (int, error) {
	var n int
	err := s.leadRepo.DB().Pool().QueryRow(ctx, `
		SELECT count(*) FROM contact_reports cr
		JOIN interactions i ON i.id = cr.interaction_id
		WHERE i.user_id=$1 AND cr.outcome IN ('INTERESTED','HIGHLY_INTERESTED','ANSWERED') AND i.started_at >= date_trunc('day', now())`, userID).Scan(&n)
	return n, err
}

func (s *Service) byEmployee(ctx context.Context) (map[string]int, error) {
	rows, err := s.leadRepo.DB().Pool().Query(ctx, `
		SELECT COALESCE(u.name, 'unassigned'), count(l.id) FROM leads l
		LEFT JOIN users u ON u.id = l.owner_user_id
		GROUP BY 1 ORDER BY 2 DESC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := map[string]int{}
	for rows.Next() {
		var name string
		var n int
		if err := rows.Scan(&name, &n); err != nil {
			return nil, err
		}
		out[name] = n
	}
	return out, rows.Err()
}

func (s *Service) bySource(ctx context.Context) (map[string]int, error) {
	rows, err := s.leadRepo.DB().Pool().Query(ctx, `
		SELECT COALESCE(ib.source_type::text, 'manual'), count(l.id) FROM leads l
		LEFT JOIN import_batches ib ON ib.id = l.source_batch_id
		GROUP BY 1 ORDER BY 2 DESC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := map[string]int{}
	for rows.Next() {
		var name string
		var n int
		if err := rows.Scan(&name, &n); err != nil {
			return nil, err
		}
		out[name] = n
	}
	return out, rows.Err()
}

func (s *Service) byIndustry(ctx context.Context) (map[string]int, error) {
	return s.groupCount(ctx, "industry")
}

func (s *Service) byCity(ctx context.Context) (map[string]int, error) {
	return s.groupCount(ctx, "city")
}

func (s *Service) byBusinessType(ctx context.Context) (map[string]int, error) {
	return s.groupCount(ctx, "business_type")
}

func (s *Service) groupCount(ctx context.Context, col string) (map[string]int, error) {
	rows, err := s.leadRepo.DB().Pool().Query(ctx, `
		SELECT COALESCE(NULLIF(`+col+`,''), 'unknown'), count(*) FROM leads GROUP BY 1 ORDER BY 2 DESC LIMIT 20`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := map[string]int{}
	for rows.Next() {
		var name string
		var n int
		if err := rows.Scan(&name, &n); err != nil {
			return nil, err
		}
		out[name] = n
	}
	return out, rows.Err()
}

func (s *Service) byRejectReason(ctx context.Context) (map[string]int, error) {
	rows, err := s.leadRepo.DB().Pool().Query(ctx, `
		SELECT reason, count(*) FROM disqualifications GROUP BY 1 ORDER BY 2 DESC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := map[string]int{}
	for rows.Next() {
		var name string
		var n int
		if err := rows.Scan(&name, &n); err != nil {
			return nil, err
		}
		out[name] = n
	}
	return out, rows.Err()
}
