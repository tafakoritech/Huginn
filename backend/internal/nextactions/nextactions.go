// Package nextactions implements the Next Best Action / Work Queue engine.
//
// It collects candidate work items from the underlying domain records
// (follow-ups, pending mandatory reports, and lead-level inferred actions),
// normalizes them into a single ActionItem shape, scores them with an explicit,
// deterministic rule-based model, de-duplicates them, and sorts them so the
// backend is the single source of truth for the salesperson's queue. The
// ranking is intentionally free of any LLM dependency: the home screen must
// remain fast and reliable.
package nextactions

import (
	"time"
)

// SourceType identifies where an action originated. It is the canonical
// deduplication/display category, not a UI label.
type SourceType string

const (
	SourceFollowUp     SourceType = "follow_up"
	SourcePendingReport SourceType = "pending_report"
	SourceFirstContact SourceType = "first_contact"
	SourceStaleLead    SourceType = "stale_lead"
)

// ActionType is the concrete executable step shown to the agent.
type ActionType string

const (
	ActionCall          ActionType = "call"
	ActionMessage       ActionType = "message"
	ActionEmail         ActionType = "email"
	ActionReport        ActionType = "report"
	ActionFollowUp      ActionType = "follow_up"
	ActionQualification ActionType = "qualification"
	ActionLeadReview    ActionType = "lead_review"
)

// Tier is the semantic priority band (never exposed as a raw score).
type Tier string

const (
	TierUrgent    Tier = "urgent"
	TierImportant Tier = "important"
	TierNormal    Tier = "normal"
)

// TierThresholds map a numeric score to a semantic priority tier.
const (
	urgentMin    = 80
	importantMin = 50
)

// TierForScore returns the semantic priority tier for a numeric score.
func TierForScore(score int) Tier {
	switch {
	case score >= urgentMin:
		return TierUrgent
	case score >= importantMin:
		return TierImportant
	default:
		return TierNormal
	}
}

// ActionItem is the normalized work item returned to clients. It carries
// exactly the fields needed to render a card and execute the action; internal
// scoring complexity (other than priority_score) is intentionally omitted.
type ActionItem struct {
	ID         string     `json:"id"`
	SourceType SourceType `json:"source_type"`
	SourceID   string     `json:"source_id"`

	LeadID      string  `json:"lead_id"`
	LeadName    string  `json:"lead_name"`
	CompanyName string  `json:"company_name"`
	ContactName *string `json:"contact_name,omitempty"`

	// AssigneeID is internal (used for ownership filtering in the team view).
	AssigneeID   *string `json:"-"`
	AssigneeName string  `json:"assignee_name,omitempty"`

	ActionType ActionType `json:"action_type"`

	Title   string `json:"title"`
	Reason  string `json:"reason"`
	Context string `json:"context,omitempty"`

	Priority      Tier `json:"priority"`
	PriorityScore int  `json:"priority_score"`

	DueAt   *time.Time `json:"due_at,omitempty"`
	Overdue bool       `json:"overdue"`

	SuggestedAction string `json:"suggested_action"`
	CTALabel        string `json:"cta_label"`

	ProjectID    *string `json:"project_id,omitempty"`
	ProjectTitle string  `json:"project_title,omitempty"`

	LastInteractionAt *time.Time `json:"last_interaction_at,omitempty"`

	// Explainability lists the human-readable reasons this action was chosen.
	Explainability []string `json:"explainability"`
}

// Response is the envelope returned by the next-actions endpoints.
type Response struct {
	Items          []*ActionItem `json:"items"`
	GeneratedAt    time.Time     `json:"generated_at"`
	TotalActionable int          `json:"total_actionable"`
	UrgentCount    int           `json:"urgent_count"`
	OverdueCount   int           `json:"overdue_count"`
}

// candidate is an in-progress item before deduplication/sorting.
type candidate struct {
	item  *ActionItem
	// rank is the raw score used for deduplication precedence; equal-score
	// ties are broken by source precedence.
	rank int
}

// sourcePrecedence orders sources when de-duplicating a lead into a single
// card. Lower wins: an explicit pending report outranks an inferred action.
var sourcePrecedence = map[SourceType]int{
	SourcePendingReport: 0,
	SourceFollowUp:      1,
	SourceFirstContact:  2,
	SourceStaleLead:     3,
}

func sourceRank(s SourceType) int {
	if r, ok := sourcePrecedence[s]; ok {
		return r
	}
	return 99
}

// ctaLabelFor maps an action type to its Persian CTA label.
func ctaLabelFor(a ActionType) string {
	switch a {
	case ActionCall:
		return "تماس"
	case ActionMessage:
		return "پیام"
	case ActionEmail:
		return "ایمیل"
	case ActionReport:
		return "ثبت گزارش"
	case ActionFollowUp:
		return "پیگیری"
	case ActionQualification:
		return "ارزیابی"
	default:
		return "مشاهده"
	}
}

// actionTypeForChannel maps a follow-up channel/type to the executable action.
func actionTypeForChannel(channel, fuType string) ActionType {
	switch channel {
	case "CALL":
		return ActionCall
	case "SMS", "WHATSAPP", "TELEGRAM", "BALE", "EITAA", "RUBIKA":
		return ActionMessage
	case "EMAIL":
		return ActionEmail
	}
	switch fuType {
	case "CALL":
		return ActionCall
	case "MESSAGE":
		return ActionMessage
	case "EMAIL":
		return ActionEmail
	}
	return ActionFollowUp
}
