// Package analytics implements the canonical sales-reporting queries. Every
// metric here is computed server-side from the source-of-truth tables:
//
//	leads           -> lead counts, scores, qualification state, projects
//	interactions    -> activity (calls/messages), contact attempts
//	contact_reports -> submitted call reports, outcomes, interest, objections
//	follow_ups      -> pending/overdue/completed follow-up state
//	leads + contacts -> canonical decision-maker facts
//
// Clients only request filters and render results; they never derive metrics.
package analytics

import (
	"context"
	"fmt"
	"time"

	"github.com/tafakorpardazan/huginn/internal/database"
)

// Filters scope every report. All fields are optional; a nil date means
// unbounded on that edge.
type Filters struct {
	From, To       *time.Time
	ProjectID      *string
	CampaignID     *string
	AssignedUserID *string
	LeadStatus     *string
	QualityStatus  *string
	Channel        *string
	Outcome        *string
	Interest       *string
}

// FromStr / ToStr return RFC3339 strings for logging (nil-safe).
func (f Filters) FromStr() *string {
	if f.From == nil {
		return nil
	}
	s := f.From.UTC().Format(time.RFC3339)
	return &s
}

func (f Filters) ToStr() *string {
	if f.To == nil {
		return nil
	}
	s := f.To.UTC().Format(time.RFC3339)
	return &s
}

// condition is a SQL fragment + args. `add` binds a value; `addRaw` appends a
// literal fragment (never user input).
type condition struct {
	sql  string
	args []any
}

func (c *condition) add(cond string, val any) {
	c.sql += " AND " + cond
	c.args = append(c.args, val)
}

func (c *condition) addRaw(cond string) { c.sql += " AND " + cond }

func nextArg(c *condition) int { return len(c.args) + 1 }

// leadConditions filters the leads table. campaign_id resolves through the
// owning project (campaign -> projects -> leads).
func (f Filters) leadConditions(alias string) condition {
	c := condition{sql: "1=1"}
	if f.ProjectID != nil {
		c.add(fmt.Sprintf("%s.project_id=$%d", alias, nextArg(&c)), *f.ProjectID)
	}
	if f.CampaignID != nil {
		c.add(fmt.Sprintf("%s.project_id IN (SELECT id FROM projects WHERE campaign_id=$%d)", alias, nextArg(&c)), *f.CampaignID)
	}
	if f.AssignedUserID != nil {
		c.add(fmt.Sprintf("%s.owner_user_id=$%d", alias, nextArg(&c)), *f.AssignedUserID)
	}
	if f.LeadStatus != nil {
		c.add(fmt.Sprintf("%s.status=$%d", alias, nextArg(&c)), *f.LeadStatus)
	}
	if f.QualityStatus != nil {
		c.add(fmt.Sprintf("%s.quality_status=$%d", alias, nextArg(&c)), *f.QualityStatus)
	}
	if f.From != nil {
		c.add(fmt.Sprintf("%s.created_at >= $%d", alias, nextArg(&c)), *f.From)
	}
	if f.To != nil {
		c.add(fmt.Sprintf("%s.created_at <= $%d", alias, nextArg(&c)), *f.To)
	}
	return c
}

// interactionConditions filters queries rooted at interactions (activity,
// reports, outcomes). The FROM clause must provide `i` (interactions) and
// `cr` (contact_reports); lead-scoped filters go through a subquery so no join
// is required by the caller.
func (f Filters) interactionConditions() condition {
	c := condition{sql: "1=1"}
	if f.ProjectID != nil {
		c.add("i.lead_id IN (SELECT id FROM leads WHERE project_id=$"+itoa(nextArg(&c))+")", *f.ProjectID)
	}
	if f.CampaignID != nil {
		c.add("i.lead_id IN (SELECT id FROM leads WHERE project_id IN (SELECT id FROM projects WHERE campaign_id=$"+itoa(nextArg(&c))+"))", *f.CampaignID)
	}
	if f.AssignedUserID != nil {
		c.add("i.user_id=$"+itoa(nextArg(&c)), *f.AssignedUserID)
	}
	if f.Channel != nil {
		c.add("i.channel=$"+itoa(nextArg(&c)), *f.Channel)
	}
	if f.Outcome != nil {
		c.add("cr.outcome=$"+itoa(nextArg(&c)), *f.Outcome)
	}
	if f.Interest != nil {
		c.add("cr.interest_level=$"+itoa(nextArg(&c)), *f.Interest)
	}
	if f.From != nil {
		c.add("i.started_at >= $"+itoa(nextArg(&c)), *f.From)
	}
	if f.To != nil {
		c.add("i.started_at <= $"+itoa(nextArg(&c)), *f.To)
	}
	return c
}

func itoa(n int) string { return fmt.Sprintf("%d", n) }

// Service runs the reporting queries against PostgreSQL.
type Service struct {
	db *database.DB
}

func NewService(db *database.DB) *Service { return &Service{db: db} }

// ---------------------------------------------------------------------------
// Overview

// Overview is the executive KPI summary.
type Overview struct {
	TotalLeads              int     `json:"total_leads"`
	NewLeads                int     `json:"new_leads"`
	ContactedLeads          int     `json:"contacted_leads"`
	SuccessfulContacts      int     `json:"successful_contacts"`
	ReportsSubmitted        int     `json:"reports_submitted"`
	QualifiedLeads          int     `json:"qualified_leads"`
	DisqualifiedLeads       int     `json:"disqualified_leads"`
	PendingFollowUps        int     `json:"pending_followups"`
	OverdueFollowUps        int     `json:"overdue_followups"`
	AvgLeadScore            float64 `json:"avg_lead_score"`
	ConversionRate          float64 `json:"conversion_rate"`
	DecisionMakersIdentified int    `json:"decision_makers_identified"`
	AvgInterestLevel        float64 `json:"avg_interest_level"`
}

// Overview computes the KPI cards. Definitions:
//
//	total_leads           = all non-archived leads matching filters
//	new_leads             = leads created within the selected date range
//	contacted_leads       = leads with >=1 interaction (or report)
//	successful_contacts   = contact_reports.contact_reached = true
//	reports_submitted     = rows in contact_reports
//	qualified_leads       = leads with status QUALIFIED or EXPORTED_TO_MUNINN
//	disqualified_leads    = leads with status DISQUALIFIED
//	pending_followups     = follow_ups.status = PENDING
//	overdue_followups     = follow_ups.status = PENDING AND due_at < now()
//	avg_lead_score        = avg(leads.score) across filtered leads
//	conversion_rate       = qualified / total (percentage)
//	decision_makers_identified = leads.decision_maker_identified = true
//	avg_interest_level    = numeric mean of report interest (HIGH=3..NONE=0)
func (s *Service) Overview(ctx context.Context, f Filters) (*Overview, error) {
	out := &Overview{}
	lc := f.leadConditions("l")
	leadWhere, largs := lc.sql, lc.args

	// total / qualified / disqualified / avg score / decision makers.
	err := s.db.Pool().QueryRow(ctx, `
		SELECT count(*) AS total,
		       count(*) FILTER (WHERE l.status IN ('QUALIFIED','EXPORTED_TO_MUNINN')) AS qualified,
		       count(*) FILTER (WHERE l.status = 'DISQUALIFIED') AS disqualified,
		       COALESCE(avg(l.score),0) AS avg_score,
		       count(*) FILTER (WHERE l.decision_maker_identified) AS dms
		FROM leads l WHERE `+leadWhere+` AND l.status <> 'ARCHIVED'`, largs...).
		Scan(&out.TotalLeads, &out.QualifiedLeads, &out.DisqualifiedLeads, &out.AvgLeadScore, &out.DecisionMakersIdentified)
	if err != nil {
		return nil, err
	}
	if out.TotalLeads > 0 {
		out.ConversionRate = float64(out.QualifiedLeads) / float64(out.TotalLeads) * 100
	}

	// new leads within the date range (unbounded when no range given).
	newC := f.leadConditions("l")
	_ = newC
	newCond := condition{sql: "1=1"}
	if f.From != nil {
		newCond.add("l.created_at >= $"+itoa(nextArg(&newCond)), *f.From)
	}
	if f.To != nil {
		newCond.add("l.created_at <= $"+itoa(nextArg(&newCond)), *f.To)
	}
	err = s.db.Pool().QueryRow(ctx, `
		SELECT count(*) FROM leads l WHERE `+newCond.sql+` AND l.status <> 'ARCHIVED'`, newCond.args...).Scan(&out.NewLeads)
	if err != nil {
		return nil, err
	}

	// contacted leads + reports + successful contacts.
	ic := f.interactionConditions()
	ic2 := f.interactionConditions()
	ic2.sql += " AND cr.contact_reached = true"
	err = s.db.Pool().QueryRow(ctx, `
		SELECT count(DISTINCT i.lead_id), count(cr.id)
		FROM interactions i LEFT JOIN contact_reports cr ON cr.interaction_id = i.id
		WHERE `+ic.sql, ic.args...).
		Scan(&out.ContactedLeads, &out.ReportsSubmitted)
	if err != nil {
		return nil, err
	}
	err = s.db.Pool().QueryRow(ctx, `
		SELECT count(cr.id) FROM contact_reports cr JOIN interactions i ON i.id = cr.interaction_id
		WHERE `+ic2.sql, ic2.args...).Scan(&out.SuccessfulContacts)
	if err != nil {
		return nil, err
	}

	// follow-ups.
	fc := f.followUpConditions()
	err = s.db.Pool().QueryRow(ctx, `
		SELECT count(*) FILTER (WHERE status='PENDING' AND due_at >= now()),
		       count(*) FILTER (WHERE status='PENDING' AND due_at < now())
		FROM follow_ups f WHERE `+fc.sql, fc.args...).
		Scan(&out.PendingFollowUps, &out.OverdueFollowUps)
	if err != nil {
		return nil, err
	}

	// average interest (numeric: HIGH=3, MEDIUM=2, LOW=1, NONE=0).
	interestC := f.interactionConditions()
	err = s.db.Pool().QueryRow(ctx, `
		SELECT COALESCE(avg(CASE cr.interest_level
			WHEN 'HIGH' THEN 3 WHEN 'MEDIUM' THEN 2 WHEN 'LOW' THEN 1 ELSE 0 END),0)
		FROM contact_reports cr JOIN interactions i ON i.id = cr.interaction_id
		WHERE `+interestC.sql, interestC.args...).Scan(&out.AvgInterestLevel)
	if err != nil {
		return nil, err
	}

	return out, nil
}

func (f Filters) followUpConditions() condition {
	c := condition{sql: "1=1"}
	if f.ProjectID != nil {
		c.add("f.lead_id IN (SELECT id FROM leads WHERE project_id=$"+itoa(nextArg(&c))+")", *f.ProjectID)
	}
	if f.CampaignID != nil {
		c.add("f.lead_id IN (SELECT id FROM leads WHERE project_id IN (SELECT id FROM projects WHERE campaign_id=$"+itoa(nextArg(&c))+"))", *f.CampaignID)
	}
	if f.AssignedUserID != nil {
		c.add("f.assigned_to=$"+itoa(nextArg(&c)), *f.AssignedUserID)
	}
	if f.From != nil {
		c.add("f.due_at >= $"+itoa(nextArg(&c)), *f.From)
	}
	if f.To != nil {
		c.add("f.due_at <= $"+itoa(nextArg(&c)), *f.To)
	}
	return c
}
