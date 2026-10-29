// Package leads implements the lead domain model and state machine.
package leads

import (
	"fmt"
	"time"

	"github.com/tafakorpardazan/huginn/internal/rbac"
)

// Status is a lead lifecycle state.
type Status string

const (
	StatusNew            Status = "NEW"
	StatusReady          Status = "READY"
	StatusAssigned       Status = "ASSIGNED"
	StatusContacting     Status = "CONTACTING"
	StatusFollowUp       Status = "FOLLOW_UP"
	StatusQualified      Status = "QUALIFIED"
	StatusDisqualified   Status = "DISQUALIFIED"
	StatusExportedMuninn Status = "EXPORTED_TO_MUNINN"
	StatusArchived       Status = "ARCHIVED"
)

// AllStatuses lists every valid status.
var AllStatuses = []Status{
	StatusNew, StatusReady, StatusAssigned, StatusContacting, StatusFollowUp,
	StatusQualified, StatusDisqualified, StatusExportedMuninn, StatusArchived,
}

// IsValidStatus reports whether s is a valid lead status.
func IsValidStatus(s Status) bool {
	for _, v := range AllStatuses {
		if v == s {
			return true
		}
	}
	return false
}

// transitions defines allowed moves. Empty slice means terminal (no moves allowed).
var transitions = map[Status][]Status{
	StatusNew:            {StatusReady, StatusArchived, StatusDisqualified},
	StatusReady:          {StatusAssigned, StatusArchived, StatusDisqualified},
	StatusAssigned:       {StatusContacting, StatusFollowUp, StatusQualified, StatusDisqualified, StatusArchived, StatusReady},
	StatusContacting:     {StatusFollowUp, StatusQualified, StatusDisqualified, StatusAssigned, StatusArchived},
	StatusFollowUp:       {StatusContacting, StatusQualified, StatusDisqualified, StatusAssigned, StatusArchived},
	StatusQualified:      {StatusExportedMuninn, StatusDisqualified, StatusArchived},
	StatusDisqualified:   {StatusArchived, StatusAssigned, StatusReady},
	StatusExportedMuninn: {StatusArchived},
	StatusArchived:       {},
}

// ErrInvalidTransition indicates a disallowed state change.
type ErrInvalidTransition struct {
	From, To Status
}

func (e *ErrInvalidTransition) Error() string {
	return fmt.Sprintf("invalid lead status transition %s -> %s", e.From, e.To)
}

// CanTransition reports whether moving from -> to is allowed.
func CanTransition(from, to Status) bool {
	if from == to {
		return true
	}
	allowed, ok := transitions[from]
	if !ok {
		return false
	}
	for _, a := range allowed {
		if a == to {
			return true
		}
	}
	return false
}

// QualityStatus marks lead data quality.
type QualityStatus string

const (
	QualityRaw        QualityStatus = "RAW"
	QualityNormalized QualityStatus = "NORMALIZED"
	QualityVerified   QualityStatus = "VERIFIED"
	QualityUnverified QualityStatus = "UNVERIFIED"
)

// Lead is a potential sales lead.
type Lead struct {
	ID               string        `json:"id"`
	CompanyName      string        `json:"company_name"`
	DisplayName      *string       `json:"display_name"`
	BusinessType     *string       `json:"business_type"`
	Industry         *string       `json:"industry"`
	Description      *string       `json:"description"`
	Province         *string       `json:"province"`
	City             *string       `json:"city"`
	Address          *string       `json:"address"`
	Website          *string       `json:"website"`
	Instagram        *string       `json:"instagram"`
	Source           *string       `json:"source"`
	SourceBatchID    *string       `json:"source_batch_id"`
	SourceDocumentID *string       `json:"source_document_id"`
	Status           Status        `json:"status"`
	QualityStatus    QualityStatus `json:"quality_status"`
	Score            int           `json:"score"`
	// CorrectPersonReached / DecisionMaker* are canonical facts promoted from
	// call reports: they hold the current best-known truth about the lead and
	// pre-fill the next report so stable information is never re-asked.
	CorrectPersonReached    *bool   `json:"correct_person_reached,omitempty"`
	DecisionMakerIdentified bool    `json:"decision_maker_identified"`
	DecisionMakerName       *string `json:"decision_maker_name,omitempty"`
	DecisionMakerRole       *string `json:"decision_maker_role,omitempty"`
	OwnerUserID             *string `json:"owner_user_id"`
	// ProjectID references the sales Project this lead belongs to (1..* per project).
	ProjectID *string `json:"project_id"`
	// OwnerName is populated by the HTTP layer (never scanned from the leads
	// table) so clients can display the assignee's human-readable name instead
	// of a raw database ID.
	OwnerName  *string    `json:"owner_name,omitempty"`
	CreatedAt  time.Time  `json:"created_at"`
	UpdatedAt  time.Time  `json:"updated_at"`
	ArchivedAt *time.Time `json:"archived_at"`
	Tags       []string   `json:"tags,omitempty"`
}

// WorkQueueItem is a prioritized task returned by the daily work queue.
type WorkQueueItem struct {
	LeadID      string     `json:"lead_id"`
	CompanyName string     `json:"company_name"`
	Priority    int        `json:"priority"`
	Reason      string     `json:"reason"`
	Status      Status     `json:"status"`
	Score       int        `json:"score"`
	DueAt       *time.Time `json:"due_at"`
	// NextAction is the actionable next step for the lead (e.g. CALL, REPORT,
	// FOLLOW_UP, or a report-recommended action such as CALL_AGAIN).
	NextAction string `json:"next_action"`
}

// Role alias for convenience.
type Role = rbac.Role

// ScoreSnapshot is a historical score record.
type ScoreSnapshot struct {
	Score      int            `json:"score"`
	Components map[string]int `json:"components"`
	Reason     *string        `json:"reason"`
	CreatedAt  time.Time      `json:"created_at"`
}
