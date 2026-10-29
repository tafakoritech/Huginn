package reports

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"time"

	"github.com/tafakorpardazan/huginn/internal/database"
	"github.com/tafakorpardazan/huginn/internal/interactions"
)

// Outcome of a contact attempt.
type Outcome string

const (
	OutcomeNoAnswer         Outcome = "NO_ANSWER"
	OutcomeBusy             Outcome = "BUSY"
	OutcomeWrongNumber      Outcome = "WRONG_NUMBER"
	OutcomeAnswered         Outcome = "ANSWERED"
	OutcomeCallbackReq      Outcome = "CALL_BACK_REQUESTED"
	OutcomeNotRelevant      Outcome = "NOT_RELEVANT"
	OutcomeInterested       Outcome = "INTERESTED"
	OutcomeHighlyInterested Outcome = "HIGHLY_INTERESTED"
	OutcomeDisqualified     Outcome = "DISQUALIFIED"
)

// Interest level.
type Interest string

const (
	InterestNone   Interest = "NONE"
	InterestLow    Interest = "LOW"
	InterestMedium Interest = "MEDIUM"
	InterestHigh   Interest = "HIGH"
)

// NextAction recommendation.
type NextAction string

const (
	NextCallAgain  NextAction = "CALL_AGAIN"
	NextSendInfo   NextAction = "SEND_INFORMATION"
	NextSendSample NextAction = "SEND_SAMPLE"
	NextFollowUp   NextAction = "FOLLOW_UP"
	NextQualify    NextAction = "QUALIFY"
	NextDisqualify NextAction = "DISQUALIFY"
	NextTransfer   NextAction = "TRANSFER_TO_MUNINN"
	NextNone       NextAction = "NONE"
)

// Report is the structured contact report.
type Report struct {
	ID                          string      `json:"id"`
	InteractionID               string      `json:"interaction_id"`
	Outcome                     Outcome     `json:"outcome"`
	ContactReached              *bool       `json:"contact_reached"`
	CorrectPersonReached        *bool       `json:"correct_person_reached"`
	DecisionMakerIdentified     *bool       `json:"decision_maker_identified"`
	DecisionMakerName           *string     `json:"decision_maker_name"`
	DecisionMakerRole           *string     `json:"decision_maker_role"`
	InterestLevel               *Interest   `json:"interest_level"`
	NeedLevel                   *Interest   `json:"need_level"`
	EstimatedPurchaseCapacity   *string     `json:"estimated_purchase_capacity"`
	PurchaseTiming              *string     `json:"purchase_timing"`
	MainObjection               *string     `json:"main_objection"`
	Summary                     *string     `json:"summary"`
	NextAction                  *NextAction `json:"next_action"`
	FollowUpAt                  *time.Time  `json:"follow_up_at"`
	QualificationRecommendation *string     `json:"qualification_recommendation"`
	SubmittedBy                 string      `json:"submitted_by"`
	ManagerOverride             bool        `json:"manager_override"`
	ManagerOverrideReason       *string     `json:"manager_override_reason"`
	CreatedAt                   time.Time   `json:"created_at"`
	UpdatedAt                   time.Time   `json:"updated_at"`
}

// Repository persists reports.
type Repository struct {
	db *database.DB
}

func NewRepository(db *database.DB) *Repository { return &Repository{db: db} }

const cols = `id, interaction_id, outcome, contact_reached, correct_person_reached, decision_maker_identified,
	decision_maker_name, decision_maker_role, interest_level, need_level, estimated_purchase_capacity,
	purchase_timing, main_objection, summary, next_action, follow_up_at, qualification_recommendation,
	submitted_by, manager_override, manager_override_reason, created_at, updated_at`

// colsQualified is the same column list explicitly qualified from contact_reports
// (alias cr). It MUST be used for any query that JOINs contact_reports with
// interactions, because `id` (and `created_at`) exist on both tables and an
// unqualified reference raises PostgreSQL "column reference \"id\" is ambiguous".
const colsQualified = `cr.id, cr.interaction_id, cr.outcome, cr.contact_reached, cr.correct_person_reached, cr.decision_maker_identified,
	cr.decision_maker_name, cr.decision_maker_role, cr.interest_level, cr.need_level, cr.estimated_purchase_capacity,
	cr.purchase_timing, cr.main_objection, cr.summary, cr.next_action, cr.follow_up_at, cr.qualification_recommendation,
	cr.submitted_by, cr.manager_override, cr.manager_override_reason, cr.created_at, cr.updated_at`

func scan(row database.RowScanner) (*Report, error) {
	var r Report
	err := row.Scan(&r.ID, &r.InteractionID, &r.Outcome, &r.ContactReached, &r.CorrectPersonReached,
		&r.DecisionMakerIdentified, &r.DecisionMakerName, &r.DecisionMakerRole, &r.InterestLevel,
		&r.NeedLevel, &r.EstimatedPurchaseCapacity, &r.PurchaseTiming, &r.MainObjection, &r.Summary,
		&r.NextAction, &r.FollowUpAt, &r.QualificationRecommendation, &r.SubmittedBy,
		&r.ManagerOverride, &r.ManagerOverrideReason, &r.CreatedAt, &r.UpdatedAt)
	if err != nil {
		return nil, err
	}
	return &r, nil
}

func (r *Repository) Create(ctx context.Context, rep *Report) (*Report, error) {
	row := r.db.Pool().QueryRow(ctx, `
		INSERT INTO contact_reports (interaction_id, outcome, contact_reached, correct_person_reached,
			decision_maker_identified, decision_maker_name, decision_maker_role, interest_level, need_level,
			estimated_purchase_capacity, purchase_timing, main_objection, summary, next_action, follow_up_at,
			qualification_recommendation, submitted_by, manager_override, manager_override_reason)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15,$16,$17,$18,$19)
		RETURNING `+cols,
		rep.InteractionID, rep.Outcome, rep.ContactReached, rep.CorrectPersonReached,
		rep.DecisionMakerIdentified, rep.DecisionMakerName, rep.DecisionMakerRole, rep.InterestLevel, rep.NeedLevel,
		rep.EstimatedPurchaseCapacity, rep.PurchaseTiming, rep.MainObjection, rep.Summary, rep.NextAction, rep.FollowUpAt,
		rep.QualificationRecommendation, rep.SubmittedBy, rep.ManagerOverride, rep.ManagerOverrideReason)
	return scan(row)
}

func (r *Repository) GetByInteraction(ctx context.Context, interactionID string) (*Report, error) {
	row := r.db.Pool().QueryRow(ctx, `SELECT `+cols+` FROM contact_reports WHERE interaction_id=$1`, interactionID)
	return scan(row)
}

// ListForLead returns the call reports for a lead, newest first, so AI
// generation can see what happened on previous calls. All selected columns are
// qualified from `cr` because the join with interactions would otherwise make
// `id` (and `created_at`) ambiguous.
func (r *Repository) ListForLead(ctx context.Context, leadID string) ([]*Report, error) {
	rows, err := r.db.Pool().Query(ctx, `
		SELECT `+colsQualified+` FROM contact_reports cr
		JOIN interactions i ON i.id = cr.interaction_id
		WHERE i.lead_id = $1
		ORDER BY cr.created_at DESC`, leadID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []*Report{}
	for rows.Next() {
		rep, err := scan(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, rep)
	}
	return out, rows.Err()
}

// ReportListItem is a submitted contact report enriched with the lead and
// interaction context required by the global Reports screen. It is a flat,
// wire-friendly shape: clients (Android) must NOT reconstruct the list from
// multiple per-lead calls.
type ReportListItem struct {
	ID             string      `json:"id"`
	InteractionID  string      `json:"interaction_id"`
	LeadID         string      `json:"lead_id"`
	LeadName       string      `json:"lead_name"`
	CompanyName    string      `json:"company_name"`
	Channel        string      `json:"channel"`
	Outcome        Outcome     `json:"outcome"`
	ContactReached *bool       `json:"contact_reached"`
	InterestLevel  *Interest   `json:"interest_level"`
	NeedLevel      *Interest   `json:"need_level"`
	MainObjection  *string     `json:"main_objection"`
	Summary        *string     `json:"summary"`
	NextAction     *NextAction `json:"next_action"`
	FollowUpAt     *time.Time  `json:"follow_up_at"`
	SubmittedBy    string      `json:"submitted_by"`
	SubmitterName  string      `json:"submitter_name"`
	CreatedAt      time.Time   `json:"created_at"`
	UpdatedAt      time.Time   `json:"updated_at"`
}

// ListForUser returns submitted contact reports visible to a user, newest
// first, with lead + interaction context. Visibility follows Huginn ownership
// semantics: a sales agent sees reports on interactions they performed
// (interactions.user_id) or reports they submitted (contact_reports.submitted_by);
// managers/admins see every report. VOIDED manager-override placeholders are not
// contact reports and are excluded.
//
// Every selected column is explicitly qualified from `cr`/`i`/`l`/`u` because
// the JOIN shares column names (`id`, `created_at`) between contact_reports and
// interactions — an unqualified reference raises PostgreSQL
// `column reference "id" is ambiguous`.
func (r *Repository) ListForUser(ctx context.Context, userID string, isManager bool, limit, offset int) ([]*ReportListItem, int, error) {
	where := `cr.outcome <> 'VOIDED'`
	args := []any{}
	if !isManager {
		where += ` AND (i.user_id = $1 OR cr.submitted_by = $1)`
		args = append(args, userID)
	}
	var total int
	if err := r.db.Pool().QueryRow(ctx, `
		SELECT count(*) FROM contact_reports cr
		JOIN interactions i ON i.id = cr.interaction_id
		WHERE `+where, args...).Scan(&total); err != nil {
		return nil, 0, err
	}

	args = append(args, limit, offset)
	limitArg, offsetArg := len(args)-1, len(args)
	rows, err := r.db.Pool().Query(ctx, `
		SELECT cr.id, cr.interaction_id, cr.outcome, cr.contact_reached,
		       cr.interest_level, cr.need_level, cr.main_objection, cr.summary,
		       cr.next_action, cr.follow_up_at, cr.submitted_by, cr.created_at, cr.updated_at,
		       i.lead_id, i.channel,
		       COALESCE(l.company_name,''), COALESCE(u.name,'')
		FROM contact_reports cr
		JOIN interactions i ON i.id = cr.interaction_id
		LEFT JOIN leads l ON l.id = i.lead_id
		LEFT JOIN users u ON u.id = cr.submitted_by
		WHERE `+where+`
		ORDER BY cr.created_at DESC
		LIMIT $`+strconv.Itoa(limitArg)+` OFFSET $`+strconv.Itoa(offsetArg), args...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()
	out := []*ReportListItem{}
	for rows.Next() {
		it := &ReportListItem{}
		if err := rows.Scan(&it.ID, &it.InteractionID, &it.Outcome, &it.ContactReached,
			&it.InterestLevel, &it.NeedLevel, &it.MainObjection, &it.Summary,
			&it.NextAction, &it.FollowUpAt, &it.SubmittedBy, &it.CreatedAt, &it.UpdatedAt,
			&it.LeadID, &it.Channel, &it.CompanyName, &it.SubmitterName); err != nil {
			return nil, 0, err
		}
		it.LeadName = it.CompanyName
		out = append(out, it)
	}
	return out, total, rows.Err()
}

// UpdateInteractionStatus is a helper used by the service after creating a report.
func (r *Repository) UpdateInteractionStatus(ctx context.Context, interactionID string, status interactions.ReportStatus) error {
	_, err := r.db.Pool().Exec(ctx, `UPDATE interactions SET report_status=$2, updated_at=now() WHERE id=$1`, interactionID, status)
	return err
}

// CountOutcomeToday counts reports with the given outcome today.
func (r *Repository) CountOutcomeToday(ctx context.Context, outcome Outcome) (int, error) {
	var n int
	err := r.db.Pool().QueryRow(ctx, `
		SELECT count(*) FROM contact_reports cr
		JOIN interactions i ON i.id = cr.interaction_id
		WHERE cr.outcome=$1 AND i.started_at >= date_trunc('day', now())`, outcome).Scan(&n)
	return n, err
}

// Service handles report submission with outcome-dependent validation.
type Service struct {
	repo      *Repository
	interRepo *interactions.Repository
}

func NewService(repo *Repository, interRepo *interactions.Repository) *Service {
	return &Service{repo: repo, interRepo: interRepo}
}

var (
	ErrInteractionNotFound = errors.New("interaction not found")
	ErrReportExists        = errors.New("report already submitted for this interaction")
	ErrReportNotPending    = errors.New("interaction does not require a report")
	ErrValidation          = errors.New("report validation failed")
)

// SubmitParams is a report submission.
type SubmitParams struct {
	InteractionID               string
	Outcome                     Outcome
	ContactReached              *bool
	CorrectPersonReached        *bool
	DecisionMakerIdentified     *bool
	DecisionMakerName           *string
	DecisionMakerRole           *string
	InterestLevel               *Interest
	NeedLevel                   *Interest
	EstimatedPurchaseCapacity   *string
	PurchaseTiming              *string
	MainObjection               *string
	Summary                     *string
	NextAction                  *NextAction
	FollowUpAt                  *time.Time
	QualificationRecommendation *string
	SubmittedBy                 string
	ManagerOverride             bool
}

// Validate applies outcome-dependent validation rules.
func Validate(p *SubmitParams) error {
	switch p.Outcome {
	case OutcomeNoAnswer, OutcomeBusy, OutcomeWrongNumber:
		// minimal contact outcomes: no purchase capacity required
		if p.Summary == nil || *p.Summary == "" {
			return fmt.Errorf("%w: summary is required for outcome %s", ErrValidation, p.Outcome)
		}
	case OutcomeNotRelevant, OutcomeDisqualified:
		if p.MainObjection == nil || *p.MainObjection == "" {
			// objection optional but summary required
			if p.Summary == nil || *p.Summary == "" {
				return fmt.Errorf("%w: summary is required for outcome %s", ErrValidation, p.Outcome)
			}
		}
	case OutcomeCallbackReq:
		if p.FollowUpAt == nil {
			return fmt.Errorf("%w: follow_up_at is required for CALL_BACK_REQUESTED", ErrValidation)
		}
	case OutcomeInterested, OutcomeHighlyInterested:
		if p.InterestLevel == nil {
			return fmt.Errorf("%w: interest_level is required for outcome %s", ErrValidation, p.Outcome)
		}
		if p.NextAction == nil {
			return fmt.Errorf("%w: next_action is required for outcome %s", ErrValidation, p.Outcome)
		}
	}
	return nil
}

// Submit creates the report and marks the interaction as submitted.
func (s *Service) Submit(ctx context.Context, p SubmitParams) (*Report, error) {
	inter, err := s.interRepo.Get(ctx, p.InteractionID)
	if err != nil {
		if database.IsNoRows(err) {
			return nil, ErrInteractionNotFound
		}
		return nil, err
	}
	if inter.UserID != p.SubmittedBy && !p.ManagerOverride {
		return nil, ErrValidation
	}
	if inter.ReportStatus == interactions.ReportSubmitted {
		return nil, ErrReportExists
	}
	if inter.ReportStatus == interactions.ReportVoidedByManager {
		return nil, ErrReportNotPending
	}

	if err := Validate(&p); err != nil {
		return nil, err
	}

	var result *Report
	err = s.repo.db.WithTx(ctx, func(q database.Querier) error {
		rep, err := s.repo.Create(ctx, &Report{
			InteractionID:               p.InteractionID,
			Outcome:                     p.Outcome,
			ContactReached:              p.ContactReached,
			CorrectPersonReached:        p.CorrectPersonReached,
			DecisionMakerIdentified:     p.DecisionMakerIdentified,
			DecisionMakerName:           p.DecisionMakerName,
			DecisionMakerRole:           p.DecisionMakerRole,
			InterestLevel:               p.InterestLevel,
			NeedLevel:                   p.NeedLevel,
			EstimatedPurchaseCapacity:   p.EstimatedPurchaseCapacity,
			PurchaseTiming:              p.PurchaseTiming,
			MainObjection:               p.MainObjection,
			Summary:                     p.Summary,
			NextAction:                  p.NextAction,
			FollowUpAt:                  p.FollowUpAt,
			QualificationRecommendation: p.QualificationRecommendation,
			SubmittedBy:                 p.SubmittedBy,
		})
		if err != nil {
			return err
		}
		if err := s.repo.UpdateInteractionStatus(ctx, p.InteractionID, interactions.ReportSubmitted); err != nil {
			return err
		}
		result = rep
		return nil
	})
	if err != nil {
		return nil, err
	}
	return result, nil
}

// Void voids a pending report (manager override). Requires a reason.
func (s *Service) Void(ctx context.Context, interactionID, managerID, reason string) error {
	if reason == "" {
		return fmt.Errorf("%w: reason is required for manager override", ErrValidation)
	}
	inter, err := s.interRepo.Get(ctx, interactionID)
	if err != nil {
		return err
	}
	if inter.ReportStatus != interactions.ReportPending {
		return ErrReportNotPending
	}
	if err := s.repo.UpdateInteractionStatus(ctx, interactionID, interactions.ReportVoidedByManager); err != nil {
		return err
	}
	_, err = s.repo.db.Pool().Exec(ctx, `
		INSERT INTO contact_reports (interaction_id, outcome, submitted_by, manager_override, manager_override_reason, overridden_by)
		VALUES ($1,'VOIDED',$2,true,$3,$4)
		ON CONFLICT (interaction_id) DO UPDATE SET manager_override=true, manager_override_reason=EXCLUDED.manager_override_reason, overridden_by=EXCLUDED.overridden_by`,
		interactionID, managerID, reason, managerID)
	return err
}
