package qualification

import (
	"context"
	"errors"
	"time"

	"github.com/tafakorpardazan/huginn/internal/database"
	"github.com/tafakorpardazan/huginn/internal/leads"
	"github.com/tafakorpardazan/huginn/internal/scoring"
)

// DisqualifyReason is a structured disqualification reason.
type DisqualifyReason string

const (
	ReasonNotRelevant   DisqualifyReason = "NOT_RELEVANT"
	ReasonInvalidContact DisqualifyReason = "INVALID_CONTACT"
	ReasonTooSmall      DisqualifyReason = "TOO_SMALL"
	ReasonNoInterest    DisqualifyReason = "NO_INTEREST"
	ReasonNoNeed        DisqualifyReason = "NO_NEED"
	ReasonNoBudget      DisqualifyReason = "NO_BUDGET"
	ReasonDuplicate     DisqualifyReason = "DUPLICATE"
	ReasonOutOfScope    DisqualifyReason = "OUT_OF_SCOPE"
	ReasonClosed        DisqualifyReason = "BUSINESS_CLOSED"
	ReasonOther         DisqualifyReason = "OTHER"
)

var validDisqualifyReasons = map[DisqualifyReason]bool{
	ReasonNotRelevant: true, ReasonInvalidContact: true, ReasonTooSmall: true, ReasonNoInterest: true,
	ReasonNoNeed: true, ReasonNoBudget: true, ReasonDuplicate: true, ReasonOutOfScope: true,
	ReasonClosed: true, ReasonOther: true,
}

func IsValidDisqualifyReason(r DisqualifyReason) bool { return validDisqualifyReasons[r] }

// Disqualification is a record of a disqualification event.
type Disqualification struct {
	ID          string           `json:"id"`
	LeadID      string           `json:"lead_id"`
	Reason      DisqualifyReason `json:"reason"`
	Explanation *string          `json:"explanation"`
	DisqualifiedBy string       `json:"disqualified_by"`
	CreatedAt   time.Time        `json:"created_at"`
}

// Rules configure qualification criteria.
type Rules struct {
	ScoreThreshold int
}

// Service performs qualification/disqualification using deterministic rules.
type Service struct {
	repo       *Repository
	leadSvc    *leads.Service
	scoreRepo  *scoring.Repository
	rules      Rules
	now        func() time.Time
}

func NewService(repo *Repository, leadSvc *leads.Service, scoreRepo *scoring.Repository, rules Rules) *Service {
	return &Service{repo: repo, leadSvc: leadSvc, scoreRepo: scoreRepo, rules: rules, now: time.Now}
}

var (
	ErrNotQualifiable = errors.New("lead does not meet qualification rules")
	ErrInvalidReason  = errors.New("invalid disqualification reason")
	ErrNotFound       = errors.New("not found")
)

// QualificationInputs are deterministic facts checked against the rules.
type QualificationInputs struct {
	BusinessRelevant          bool
	ContactInfoValid          bool
	MeaningfulContactOccurred bool
	NeedOrInterestExists      bool
	DecisionMakerIdentified   bool
	NextSalesActionExists     bool
	Score                     int
}

// ShouldQualify evaluates the configured rules deterministically.
func ShouldQualify(in QualificationInputs, rules Rules) (bool, []string) {
	var missing []string
	check := func(cond bool, rule string) {
		if !cond {
			missing = append(missing, rule)
		}
	}
	check(in.BusinessRelevant, "business_is_relevant")
	check(in.ContactInfoValid, "contact_information_valid")
	check(in.MeaningfulContactOccurred, "meaningful_contact_occurred")
	check(in.NeedOrInterestExists, "need_or_interest_exists")
	check(in.DecisionMakerIdentified, "decision_maker_or_influencer_identified")
	check(in.NextSalesActionExists, "next_sales_action_exists")
	check(in.Score >= rules.ScoreThreshold, "minimum_score_reached")
	return len(missing) == 0, missing
}

// Qualify marks a lead qualified if rules pass. Requires an explicit reason.
func (s *Service) Qualify(ctx context.Context, leadID string, inputs QualificationInputs, reason, actorID string) (*leads.Lead, error) {
	if s.rules.ScoreThreshold == 0 {
		s.rules.ScoreThreshold = 70
	}
	ok, missing := ShouldQualify(inputs, s.rules)
	if !ok {
		return nil, &NotQualifiableError{Missing: missing}
	}
	_, err := s.leadSvc.ChangeStatus(ctx, leadID, leads.StatusQualified, reason, actorID)
	if err != nil {
		return nil, err
	}
	return s.leadSvc.Get(ctx, leadID)
}

// NotQualifiableError describes which rules failed.
type NotQualifiableError struct {
	Missing []string
}

func (e *NotQualifiableError) Error() string { return "lead does not meet qualification rules" }

// Disqualify records the reason and changes status.
func (s *Service) Disqualify(ctx context.Context, leadID string, reason DisqualifyReason, explanation *string, actorID string) error {
	if !IsValidDisqualifyReason(reason) {
		return ErrInvalidReason
	}
	lead, err := s.leadSvc.Get(ctx, leadID)
	if err != nil {
		return err
	}
	if lead.Status == leads.StatusExportedMuninn {
		return errors.New("cannot disqualify an exported lead")
	}
	if err := s.repo.InsertDisqualification(ctx, leadID, reason, explanation, actorID); err != nil {
		return err
	}
	if _, err := s.leadSvc.ChangeStatus(ctx, leadID, leads.StatusDisqualified, string(reason), actorID); err != nil {
		return err
	}
	return nil
}

// Repository persists disqualifications.
type Repository struct {
	db *database.DB
}

func NewRepository(db *database.DB) *Repository { return &Repository{db: db} }

func (r *Repository) InsertDisqualification(ctx context.Context, leadID string, reason DisqualifyReason, explanation *string, actorID string) error {
	_, err := r.db.Pool().Exec(ctx, `
		INSERT INTO disqualifications (lead_id, reason, explanation, disqualified_by)
		VALUES ($1,$2,$3,$4)`, leadID, reason, explanation, actorID)
	return err
}

func (r *Repository) ListForLead(ctx context.Context, leadID string) ([]*Disqualification, error) {
	rows, err := r.db.Pool().Query(ctx, `
		SELECT id, lead_id, reason, explanation, disqualified_by, created_at
		FROM disqualifications WHERE lead_id=$1 ORDER BY created_at DESC`, leadID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []*Disqualification{}
	for rows.Next() {
		var d Disqualification
		if err := rows.Scan(&d.ID, &d.LeadID, &d.Reason, &d.Explanation, &d.DisqualifiedBy, &d.CreatedAt); err != nil {
			return nil, err
		}
		out = append(out, &d)
	}
	return out, rows.Err()
}

func (r *Repository) CountByReason(ctx context.Context, reason DisqualifyReason) (int, error) {
	var n int
	err := r.db.Pool().QueryRow(ctx, `SELECT count(*) FROM disqualifications WHERE reason=$1`, reason).Scan(&n)
	return n, err
}

func (r *Repository) CountAll(ctx context.Context) (int, error) {
	var n int
	err := r.db.Pool().QueryRow(ctx, `SELECT count(*) FROM disqualifications`).Scan(&n)
	return n, err
}
