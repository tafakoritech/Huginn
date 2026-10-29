package duplicates

import (
	"context"
	"strings"
	"time"

	"github.com/tafakorpardazan/huginn/internal/database"
	"github.com/tafakorpardazan/huginn/internal/leads"
	"github.com/tafakorpardazan/huginn/internal/normalize"
)

// Status of a duplicate pair.
type Status string

const (
	StatusPossible Status = "POSSIBLE"
	StatusIgnored  Status = "IGNORED"
	StatusMerged   Status = "MERGED"
	StatusDistinct Status = "DISTINCT"
)

// Pair is a possible duplicate lead pair.
type Pair struct {
	ID         string    `json:"id"`
	LeadAID    string    `json:"lead_a_id"`
	LeadBID    string    `json:"lead_b_id"`
	Reason     string    `json:"reason"`
	Similarity float64   `json:"similarity"`
	Status     Status    `json:"status"`
	CreatedAt  time.Time `json:"created_at"`
}

// Repository persists duplicate pairs.
type Repository struct {
	db *database.DB
}

func NewRepository(db *database.DB) *Repository { return &Repository{db: db} }

func (r *Repository) UpsertPair(ctx context.Context, a, b string, reason string, similarity float64) error {
	if strings.Compare(a, b) > 0 {
		a, b = b, a
	}
	_, err := r.db.Pool().Exec(ctx, `
		INSERT INTO possible_duplicates (lead_a_id, lead_b_id, reason, similarity, status)
		VALUES ($1,$2,$3,$4,'POSSIBLE')
		ON CONFLICT (lead_a_id, lead_b_id) DO UPDATE SET reason=EXCLUDED.reason, similarity=GREATEST(possible_duplicates.similarity, EXCLUDED.similarity)`,
		a, b, reason, similarity)
	return err
}

func (r *Repository) MarkStatus(ctx context.Context, pairID string, status Status, resolvedBy string) error {
	_, err := r.db.Pool().Exec(ctx, `
		UPDATE possible_duplicates SET status=$2, resolved_by=$3, resolved_at=now() WHERE id=$1`,
		pairID, status, nullStr(resolvedBy))
	return err
}

func (r *Repository) ListForLead(ctx context.Context, leadID string) ([]*Pair, error) {
	rows, err := r.db.Pool().Query(ctx, `
		SELECT id, lead_a_id, lead_b_id, reason, similarity, status, created_at
		FROM possible_duplicates WHERE lead_a_id=$1 OR lead_b_id=$1 ORDER BY similarity DESC`, leadID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []*Pair{}
	for rows.Next() {
		var p Pair
		if err := rows.Scan(&p.ID, &p.LeadAID, &p.LeadBID, &p.Reason, &p.Similarity, &p.Status, &p.CreatedAt); err != nil {
			return nil, err
		}
		out = append(out, &p)
	}
	return out, rows.Err()
}

// Detector finds duplicate leads deterministically, then fuzzy with pg_trgm.
type Detector struct {
	leadRepo *leads.Repository
	dupsRepo *Repository
	db       *database.DB
}

func NewDetector(leadRepo *leads.Repository, dupsRepo *Repository, db *database.DB) *Detector {
	return &Detector{leadRepo: leadRepo, dupsRepo: dupsRepo, db: db}
}

// Candidates is a grouping of a lead and its potential duplicates.
type Candidates struct {
	LeadID string
	Pairs  []*Pair
}

// FindDeterministic returns existing lead IDs that already hold any of the
// given normalized contact values (mobile/phone/email). Used to avoid creating
// true duplicates during import.
func (d *Detector) FindDeterministic(ctx context.Context, normalizedContacts []string) ([]string, error) {
	values := []string{}
	seen := map[string]bool{}
	for _, v := range normalizedContacts {
		if v == "" || seen[v] {
			continue
		}
		seen[v] = true
		values = append(values, v)
	}
	if len(values) == 0 {
		return nil, nil
	}

	rows, err := d.db.Pool().Query(ctx, `
		SELECT DISTINCT l.id FROM leads l
		JOIN contacts c ON c.lead_id = l.id
		WHERE c.mobile_normalized = ANY($1)
		   OR c.phone_normalized = ANY($1)
		   OR c.email_normalized = ANY($1)
		LIMIT 50`, values)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []string{}
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		out = append(out, id)
	}
	return out, rows.Err()
}

// DupEvidence is a single deterministic duplicate decision: which lead matched,
// on which normalized field, with which value.
type DupEvidence struct {
	LeadID string `json:"lead_id"`
	Field  string `json:"field"` // mobile | phone | email
	Value  string `json:"value"`
}

// FindDeterministicDetailed returns existing leads that hold any of the given
// normalized contact values, together with the exact field/value that matched.
// Used to explain every duplicate decision during import.
func (d *Detector) FindDeterministicDetailed(ctx context.Context, values []string) ([]DupEvidence, error) {
	unique := make([]string, 0, len(values))
	seen := map[string]bool{}
	for _, v := range values {
		if v == "" || seen[v] {
			continue
		}
		seen[v] = true
		unique = append(unique, v)
	}
	if len(unique) == 0 {
		return nil, nil
	}
	rows, err := d.db.Pool().Query(ctx, `
		SELECT DISTINCT ON (l.id) l.id,
			CASE WHEN c.mobile_normalized IS NOT NULL AND c.mobile_normalized = v THEN 'mobile'
			     WHEN c.phone_normalized IS NOT NULL AND c.phone_normalized = v THEN 'phone'
			     WHEN c.email_normalized IS NOT NULL AND c.email_normalized = v THEN 'email'
			     ELSE 'contact'
			END AS field,
			v AS value
		FROM unnest($1::text[]) AS v
		JOIN contacts c ON c.mobile_normalized = v OR c.phone_normalized = v OR c.email_normalized = v
		JOIN leads l ON l.id = c.lead_id
		ORDER BY l.id`, unique)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []DupEvidence{}
	for rows.Next() {
		var e DupEvidence
		if err := rows.Scan(&e.LeadID, &e.Field, &e.Value); err != nil {
			return nil, err
		}
		out = append(out, e)
	}
	return out, rows.Err()
}

// FindByName returns the strongest existing lead whose company name fuzzy-matches
// [companyName] at or above [minSimilarity]. Used to explain company-name
// duplicates during import. Returns ("", 0, nil) when no strong match exists.
func (d *Detector) FindByName(ctx context.Context, companyName string, minSimilarity float64) (string, float64, error) {
	if minSimilarity <= 0 {
		minSimilarity = 0.3
	}
	var id string
	var sim float64
	err := d.db.Pool().QueryRow(ctx, `
		SELECT id, similarity(company_name, $1) AS sim
		FROM leads
		WHERE company_name % $1
		ORDER BY sim DESC LIMIT 1`, companyName).Scan(&id, &sim)
	if err != nil {
		if database.IsNoRows(err) {
			return "", 0, nil
		}
		return "", 0, err
	}
	if sim < minSimilarity {
		return "", 0, nil
	}
	return id, sim, nil
}

// FindByWebsite returns an existing lead whose normalized website/domain equals
// the candidate's normalized website. Exact-match only.
func (d *Detector) FindByWebsite(ctx context.Context, website string) (string, error) {
	norm := normalize.NormalizeWebsite(website)
	if norm == "" {
		return "", nil
	}
	var id string
	err := d.db.Pool().QueryRow(ctx, `
		SELECT id FROM leads WHERE website = $1 LIMIT 1`, norm).Scan(&id)
	if err != nil {
		if database.IsNoRows(err) {
			return "", nil
		}
		return "", err
	}
	return id, nil
}

// Detect runs deterministic then fuzzy duplicate detection for a lead.
// Returns newly-found duplicate pairs.
func (d *Detector) Detect(ctx context.Context, leadID string) ([]*Pair, error) {
	lead, err := d.leadRepo.Get(ctx, leadID)
	if err != nil {
		return nil, err
	}

	found := []*Pair{}

	// 1. Deterministic matches via contacts (normalized mobile/phone/email)
	pairs, err := d.deterministic(ctx, lead)
	if err != nil {
		return nil, err
	}
	found = append(found, pairs...)

	// 2. Fuzzy company-name matches via pg_trgm
	pairs, err = d.fuzzyCompanyName(ctx, lead)
	if err != nil {
		return nil, err
	}
	found = append(found, pairs...)

	return found, nil
}

func (d *Detector) deterministic(ctx context.Context, lead *leads.Lead) ([]*Pair, error) {
	rows, err := d.db.Pool().Query(ctx, `
		SELECT DISTINCT l.id FROM contacts c
		JOIN contacts other ON
			(c.mobile_normalized IS NOT NULL AND c.mobile_normalized = other.mobile_normalized AND c.mobile_normalized != '')
			OR (c.phone_normalized IS NOT NULL AND c.phone_normalized = other.phone_normalized AND c.phone_normalized != '')
			OR (c.email_normalized IS NOT NULL AND c.email_normalized = other.email_normalized AND c.email_normalized != '')
		JOIN leads l ON l.id = other.lead_id AND l.id != $1
		WHERE c.lead_id = $1`, lead.ID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var found []*Pair
	for rows.Next() {
		var otherID string
		if err := rows.Scan(&otherID); err != nil {
			return nil, err
		}
		if err := d.dupsRepo.UpsertPair(ctx, lead.ID, otherID, "deterministic_contact_match", 1.0); err != nil {
			return nil, err
		}
		p := &Pair{LeadAID: lead.ID, LeadBID: otherID, Reason: "deterministic_contact_match", Similarity: 1.0, Status: StatusPossible}
		found = append(found, p)
	}
	return found, rows.Err()
}

func (d *Detector) fuzzyCompanyName(ctx context.Context, lead *leads.Lead) ([]*Pair, error) {
	rows, err := d.db.Pool().Query(ctx, `
		SELECT id, company_name, similarity(company_name, $1) AS sim
		FROM leads
		WHERE id != $2 AND company_name % $1
		ORDER BY sim DESC LIMIT 10`, lead.CompanyName, lead.ID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var found []*Pair
	for rows.Next() {
		var otherID, otherName string
		var sim float64
		if err := rows.Scan(&otherID, &otherName, &sim); err != nil {
			return nil, err
		}
		if sim < 0.3 {
			continue
		}
		if err := d.dupsRepo.UpsertPair(ctx, lead.ID, otherID, "fuzzy_company_name", sim); err != nil {
			return nil, err
		}
		found = append(found, &Pair{LeadAID: lead.ID, LeadBID: otherID, Reason: "fuzzy_company_name", Similarity: sim, Status: StatusPossible})
	}
	return found, rows.Err()
}

// Merge merges source into target, archiving source. All merges must be audited by the caller.
func (d *Detector) Merge(ctx context.Context, targetID, sourceID, actorID string) error {
	return d.db.WithTx(ctx, func(q database.Querier) error {
		// move contacts
		if _, err := q.Exec(ctx, `UPDATE contacts SET lead_id=$1 WHERE lead_id=$2`, targetID, sourceID); err != nil {
			return err
		}
		// move interactions
		if _, err := q.Exec(ctx, `UPDATE interactions SET lead_id=$1 WHERE lead_id=$2`, targetID, sourceID); err != nil {
			return err
		}
		// move follow_ups
		if _, err := q.Exec(ctx, `UPDATE follow_ups SET lead_id=$1 WHERE lead_id=$2`, targetID, sourceID); err != nil {
			return err
		}
		// move evidence
		if _, err := q.Exec(ctx, `UPDATE lead_evidence SET lead_id=$1 WHERE lead_id=$2`, targetID, sourceID); err != nil {
			return err
		}
		// move tags
		if _, err := q.Exec(ctx, `UPDATE lead_tags SET lead_id=$1 WHERE lead_id=$2`, targetID, sourceID); err != nil {
			return err
		}
		// archive source
		if _, err := q.Exec(ctx, `UPDATE leads SET status='ARCHIVED', archived_at=now(), updated_at=now() WHERE id=$1`, sourceID); err != nil {
			return err
		}
		// mark pair as merged
		if _, err := q.Exec(ctx, `UPDATE possible_duplicates SET status='MERGED', resolved_by=$3, resolved_at=now()
			WHERE (lead_a_id=$1 AND lead_b_id=$2) OR (lead_a_id=$2 AND lead_b_id=$1)`,
			targetID, sourceID, nullStr(actorID)); err != nil {
			return err
		}
		return nil
	})
}

func nullStr(s string) any {
	if s == "" {
		return nil
	}
	return s
}

var _ = normalize.NormalizeEmail
