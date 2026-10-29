package nextactions

import (
	"context"
	"sort"
	"strings"
	"time"

	"github.com/tafakorpardazan/huginn/internal/database"
)

// Service composes the Next Best Action queue from existing domain records.
//
// It deliberately uses a small number of simple, indexed queries composed at
// the domain level rather than one giant SQL statement, so the scoring and
// deduplication rules are explicit and unit-testable. The priority decision
// never calls the LLM.
type Service struct {
	db *database.DB
}

func NewService(db *database.DB) *Service { return &Service{db: db} }

// Params scopes a queue build.
type Params struct {
	// UserID filters the queue to work owned by this user.
	UserID string
	// Team, when true, returns work across every agent (manager view). It
	// requires caller-side RBAC; the service itself does not authorize.
	Team bool
	// Limit bounds the returned items (counts are computed over the full
	// actionable set regardless of Limit).
	Limit int
}

// Build returns the ranked queue for the given params.
func (s *Service) Build(ctx context.Context, p Params) (*Response, error) {
	now := time.Now().UTC()

	var candidates []candidate
	var err error

	reports, err := s.collectPendingReports(ctx, p, now)
	if err != nil {
		return nil, err
	}
	candidates = append(candidates, reports...)

	followups, err := s.collectFollowUps(ctx, p, now)
	if err != nil {
		return nil, err
	}
	candidates = append(candidates, followups...)

	inferred, err := s.collectInferredLeadActions(ctx, p, now)
	if err != nil {
		return nil, err
	}
	candidates = append(candidates, inferred...)

	deduped := dedupe(candidates)

	// Counts are computed before limiting.
	total := len(deduped)
	urgent := 0
	overdue := 0
	for _, c := range deduped {
		if c.item.Priority == TierUrgent {
			urgent++
		}
		if c.item.Overdue {
			overdue++
		}
	}

	sortCandidates(deduped)

	items := make([]*ActionItem, 0, len(deduped))
	limit := p.Limit
	if limit <= 0 || limit > len(deduped) {
		limit = len(deduped)
	}
	for _, c := range deduped[:limit] {
		items = append(items, c.item)
	}

	return &Response{
		Items:           items,
		GeneratedAt:     now,
		TotalActionable: total,
		UrgentCount:     urgent,
		OverdueCount:    overdue,
	}, nil
}

// dedupe collapses multiple candidates for the same lead into the single
// highest-value actionable item. Precedence (highest first): pending report,
// explicit follow-up, first contact, stale lead. When two candidates share a
// source, the higher-ranked one wins.
func dedupe(candidates []candidate) []candidate {
	byLead := map[string]candidate{}
	for _, c := range candidates {
		lead := c.item.LeadID
		existing, ok := byLead[lead]
		if !ok {
			byLead[lead] = c
			continue
		}
		if better(c, existing) {
			byLead[lead] = c
		}
	}
	out := make([]candidate, 0, len(byLead))
	for _, c := range byLead {
		out = append(out, c)
	}
	return out
}

// better reports whether a should replace b for the same lead.
func better(a, b candidate) bool {
	if sourceRank(a.item.SourceType) != sourceRank(b.item.SourceType) {
		return sourceRank(a.item.SourceType) < sourceRank(b.item.SourceType)
	}
	if a.rank != b.rank {
		return a.rank > b.rank
	}
	// Deterministic tie-break by earliest due time.
	return dueBefore(a.item.DueAt, b.item.DueAt)
}

func dueBefore(a, b *time.Time) bool {
	if a == nil {
		return false
	}
	if b == nil {
		return true
	}
	return a.Before(*b)
}

func sortCandidates(cs []candidate) {
	sort.SliceStable(cs, func(i, j int) bool {
		a, b := cs[i].item, cs[j].item
		if a.PriorityScore != b.PriorityScore {
			return a.PriorityScore > b.PriorityScore
		}
		if dueBefore(a.DueAt, b.DueAt) {
			return true
		}
		if dueBefore(b.DueAt, a.DueAt) {
			return false
		}
		return a.CompanyName < b.CompanyName
	})
}

// --- scoring ---------------------------------------------------------------

const (
	urgencyOverdueOverDay = 40
	urgencyOverdueWithin  = 35
	urgencyDueSoon2h      = 30
	urgencyDueToday       = 25
	urgencyDueTomorrow    = 10

	qualityHot  = 25
	qualityWarm = 15
	qualityCold = 5

	commitmentExplicit = 25

	pendingReportBase = 60
	firstContactBase  = 20
	staleLeadBase     = 15
)

// leadQuality maps a Huginn lead score to quality points. Disqualified leads
// are filtered out before scoring, never scored here.
func leadQuality(score int) int {
	switch {
	case score >= 70:
		return qualityHot
	case score >= 40:
		return qualityWarm
	default:
		return qualityCold
	}
}

// qualityReason returns the Persian reason for a lead's quality bucket.
func qualityReason(score int) string {
	switch {
	case score >= 70:
		return "سرنخ پرارزش است"
	case score >= 40:
		return "سرنخ گرم است"
	default:
		return "سرنخ سرد است"
	}
}

// urgencyFor computes urgency points from a due time and returns the points and
// the Persian reason describing why now.
func urgencyFor(due *time.Time, now time.Time) (int, string) {
	if due == nil {
		return 0, ""
	}
	d := due.UTC()
	switch {
	case d.Before(now.Add(-24 * time.Hour)):
		return urgencyOverdueOverDay, "بیش از یک روز از سررسید پیگیری گذشته است"
	case d.Before(now):
		return urgencyOverdueWithin, "از زمان پیگیری گذشته است"
	case d.Before(now.Add(2 * time.Hour)):
		return urgencyDueSoon2h, "تا دو ساعت دیگر سررسید دارد"
	case d.Before(now.Truncate(24 * time.Hour).Add(24 * time.Hour)):
		return urgencyDueToday, "سررسید پیگیری امروز است"
	case d.Before(now.Truncate(24 * time.Hour).Add(48 * time.Hour)):
		return urgencyDueTomorrow, "سررسید پیگیری فردا است"
	default:
		return 0, ""
	}
}

// promisedCommitment heuristically detects an explicit promised next action in
// a follow-up title/description, or a follow-up sourced from a report's
// recorded next_action. This outranks generic prospecting.
func promisedCommitment(source, title, description string) bool {
	if source == "REPORT" {
		return true
	}
	text := strings.ToLower(title + " " + description)
	for _, phrase := range []string{
		"تماس بگیر", "تماس بگیرید", "تماس مجدد", "اطلاعات", "بفرست", "ارسال", "پیگیری شود",
		"صحبت", "جلسه", "قرار", "نمونه", "کاتالوگ", "قیمت", "پیشنهاد",
	} {
		if strings.Contains(text, phrase) {
			return true
		}
	}
	return false
}
