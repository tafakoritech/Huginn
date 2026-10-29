package scoring

import (
	"context"
	"encoding/json"
	"time"

	"github.com/tafakorpardazan/huginn/internal/database"
	"github.com/tafakorpardazan/huginn/internal/leads"
)

// Component names.
const (
	CompBusinessFit    = "business_fit"
	CompContactability = "contactability"
	CompBusinessScale  = "business_scale"
	CompEngagement     = "engagement"
	CompAuthority      = "authority"
	CompNeed           = "need"
	CompTiming         = "timing"
)

var defaultWeights = map[string]int{
	CompBusinessFit:    25,
	CompContactability: 15,
	CompBusinessScale:  10,
	CompEngagement:     14,
	CompAuthority:      8,
	CompNeed:           6,
	CompTiming:         4,
}

// Result is an explainable score.
type Result struct {
	Total      int            `json:"total"`
	Components map[string]int `json:"components"`
}

// Weights returns the default component weights (sum 100).
func Weights() map[string]int {
	out := map[string]int{}
	for k, v := range defaultWeights {
		out[k] = v
	}
	return out
}

// Signals describes the deterministic inputs to scoring.
type Signals struct {
	// Business fit signals
	HasIndustry     bool
	HasDescription  bool
	HasBusinessType bool
	SourceRelevant  bool
	// Contactability
	HasMobile bool
	HasPhone  bool
	HasEmail  bool
	HasSocial bool
	// Business scale
	HasAddress     bool
	HasWebsite     bool
	SourceTypeRich bool // e.g. catalogue/website suggests established business
	// Engagement
	InteractionCount int
	PositiveReports  int
	// Authority
	DecisionMakerIdentified bool
	// Need / interest
	NeedHigh       bool
	NeedMedium     bool
	InterestHigh   bool
	InterestMedium bool
	// Timing
	HasFollowUpScheduled bool
	PurchaseSoon         bool
	HasPurchaseCapacity  bool
}

// Score computes the deterministic score for the given signals.
func Score(s Signals) Result {
	comp := map[string]int{}
	capn := func(ok bool, max int) int {
		if ok {
			return max
		}
		return 0
	}

	comp[CompBusinessFit] = capn(s.HasIndustry, 10) + capn(s.HasDescription, 8) + capn(s.HasBusinessType, 4) + capn(s.SourceRelevant, 3)
	if comp[CompBusinessFit] > 25 {
		comp[CompBusinessFit] = 25
	}

	comp[CompContactability] = capn(s.HasMobile, 8) + capn(s.HasPhone, 3) + capn(s.HasEmail, 3) + capn(s.HasSocial, 1)
	if comp[CompContactability] > 15 {
		comp[CompContactability] = 15
	}

	comp[CompBusinessScale] = capn(s.HasAddress, 3) + capn(s.HasWebsite, 3) + capn(s.SourceTypeRich, 4)
	if comp[CompBusinessScale] > 10 {
		comp[CompBusinessScale] = 10
	}

	eng := 0
	if s.InteractionCount >= 1 {
		eng += 6
	}
	if s.InteractionCount >= 3 {
		eng += 4
	}
	eng += 4 * s.PositiveReports
	if eng > 14 {
		eng = 14
	}
	comp[CompEngagement] = eng

	comp[CompAuthority] = capn(s.DecisionMakerIdentified, 8)

	need := 0
	switch {
	case s.NeedHigh:
		need = 6
	case s.NeedMedium:
		need = 4
	case s.InterestHigh:
		need = 6
	case s.InterestMedium:
		need = 4
	}
	comp[CompNeed] = need

	timing := 0
	if s.HasFollowUpScheduled {
		timing += 2
	}
	if s.PurchaseSoon {
		timing += 2
	}
	if s.HasPurchaseCapacity {
		timing += 2
	}
	if timing > 4 {
		timing = 4
	}
	comp[CompTiming] = timing

	total := 0
	for _, v := range comp {
		total += v
	}
	return Result{Total: total, Components: comp}
}

// Repository persists score history.
type Repository struct {
	db *database.DB
}

func NewRepository(db *database.DB) *Repository { return &Repository{db: db} }

func (r *Repository) Save(ctx context.Context, leadID string, result Result, reason string, createdBy *string) error {
	compJSON, _ := json.Marshal(result.Components)
	_, err := r.db.Pool().Exec(ctx, `
		INSERT INTO score_history (lead_id, score, components, reason, created_by)
		VALUES ($1,$2,$3,$4,$5)`, leadID, result.Total, compJSON, nullStr(reason), createdBy)
	return err
}

func (r *Repository) History(ctx context.Context, leadID string, limit int) ([]*leads.ScoreSnapshot, error) {
	if limit == 0 {
		limit = 20
	}
	rows, err := r.db.Pool().Query(ctx, `
		SELECT score, components, reason, created_at FROM score_history
		WHERE lead_id=$1 ORDER BY created_at DESC LIMIT $2`, leadID, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []*leads.ScoreSnapshot{}
	for rows.Next() {
		s := &leads.ScoreSnapshot{}
		var comp []byte
		if err := rows.Scan(&s.Score, &comp, &s.Reason, &s.CreatedAt); err != nil {
			return nil, err
		}
		_ = json.Unmarshal(comp, &s.Components)
		out = append(out, s)
	}
	return out, rows.Err()
}

func nullStr(s string) any {
	if s == "" {
		return nil
	}
	return s
}

var _ = time.Now
