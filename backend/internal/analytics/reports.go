package analytics

import (
	"context"
	"fmt"
)

// ---------------------------------------------------------------------------
// Sales funnel

// FunnelStep is one stage of the lead conversion funnel.
type FunnelStep struct {
	Stage                string  `json:"stage"`
	Count                int     `json:"count"`
	ConversionFromPrevious float64 `json:"conversion_from_previous"`
	ConversionFromTotal  float64 `json:"conversion_from_total"`
}

// Funnel stages map to real domain state:
//
//	all_leads       -> non-archived leads matching filters
//	contacted       -> leads with >=1 interaction
//	reached         -> leads with a report where contact_reached = true
//	correct_person  -> lead.correct_person_reached OR any report true
//	decision_maker  -> lead.decision_maker_identified OR any report true
//	interested      -> any report interest HIGH/MEDIUM, or status FOLLOW_UP+
//	qualified       -> lead status QUALIFIED / EXPORTED_TO_MUNINN
func (s *Service) Funnel(ctx context.Context, f Filters) ([]FunnelStep, error) {
	lc := f.leadConditions("l")
	lc.sql += " AND l.status <> 'ARCHIVED'"
	countLead := func(extra string, extraArgs ...any) (int, error) {
		var n int
		args := append(append([]any{}, lc.args...), extraArgs...)
		err := s.db.Pool().QueryRow(ctx, "SELECT count(*) FROM leads l WHERE "+lc.sql+extra, args...).Scan(&n)
		return n, err
	}

	stages := []struct {
		key   string
		count func() (int, error)
	}{}
	_ = stages

	all, err := countLead("")
	if err != nil {
		return nil, err
	}

	contacted, err := s.countLeadWithInteraction(ctx, f)
	if err != nil {
		return nil, err
	}

	reached, err := s.countLeadWithReport(ctx, f, "cr.contact_reached = true")
	if err != nil {
		return nil, err
	}

	correctPerson, err := s.countLeadWithReport(ctx, f, "cr.correct_person_reached = true")
	if err != nil {
		return nil, err
	}
	// include canonical lead fact.
	if v, err := s.countLeadWithCanonical(ctx, f, "l.correct_person_reached = true"); err == nil && v > correctPerson {
		correctPerson = v
	}

	decisionMaker, err := s.countLeadWithReport(ctx, f, "cr.decision_maker_identified = true")
	if err != nil {
		return nil, err
	}
	if v, err := s.countLeadWithCanonical(ctx, f, "l.decision_maker_identified = true"); err == nil && v > decisionMaker {
		decisionMaker = v
	}

	interested, err := s.countLeadWithReport(ctx, f, "cr.interest_level IN ('HIGH','MEDIUM')")
	if err != nil {
		return nil, err
	}

	qualified, err := countLead(" AND l.status IN ('QUALIFIED','EXPORTED_TO_MUNINN')")
	if err != nil {
		return nil, err
	}

	counts := []int{all, contacted, reached, correctPerson, decisionMaker, interested, qualified}
	names := []string{"all_leads", "contacted", "reached", "correct_person", "decision_maker", "interested", "qualified"}

	out := make([]FunnelStep, 0, len(names))
	for i, name := range names {
		prev := 0
		if i > 0 {
			prev = counts[i-1]
		}
		step := FunnelStep{Stage: name, Count: counts[i]}
		if prev > 0 {
			step.ConversionFromPrevious = float64(counts[i]) / float64(prev) * 100
		}
		if all > 0 {
			step.ConversionFromTotal = float64(counts[i]) / float64(all) * 100
		}
		out = append(out, step)
	}
	return out, nil
}

func (s *Service) countLeadWithInteraction(ctx context.Context, f Filters) (int, error) {
	c := f.leadConditions("l")
	c.sql += " AND l.status <> 'ARCHIVED'"
	var n int
	err := s.db.Pool().QueryRow(ctx, `
		SELECT count(DISTINCT l.id) FROM leads l
		JOIN interactions i ON i.lead_id = l.id
		WHERE `+c.sql, c.args...).Scan(&n)
	return n, err
}

func (s *Service) countLeadWithReport(ctx context.Context, f Filters, reportCond string) (int, error) {
	c := f.leadConditions("l")
	c.sql += " AND l.status <> 'ARCHIVED'"
	var n int
	err := s.db.Pool().QueryRow(ctx, `
		SELECT count(DISTINCT l.id) FROM leads l
		JOIN interactions i ON i.lead_id = l.id
		JOIN contact_reports cr ON cr.interaction_id = i.id
		WHERE `+c.sql+` AND `+reportCond, c.args...).Scan(&n)
	return n, err
}

func (s *Service) countLeadWithCanonical(ctx context.Context, f Filters, leadCond string) (int, error) {
	c := f.leadConditions("l")
	c.sql += " AND l.status <> 'ARCHIVED'"
	var n int
	err := s.db.Pool().QueryRow(ctx, "SELECT count(*) FROM leads l WHERE "+c.sql+" AND "+leadCond, c.args...).Scan(&n)
	return n, err
}

// ---------------------------------------------------------------------------
// Activity trend

// ActivityPoint is one bucket of sales activity.
type ActivityPoint struct {
	Label               string `json:"label"`
	Interactions        int    `json:"interactions"`
	Reports             int    `json:"reports"`
	SuccessfulContacts  int    `json:"successful_contacts"`
	FollowUpsCreated    int    `json:"followups_created"`
}

// Activity returns a time series. bucket is "day" (default) or "week". The
// series covers the filter range, or the last 30 days when unbounded.
func (s *Service) Activity(ctx context.Context, f Filters, bucket string) ([]ActivityPoint, error) {
	trunc, interval := "day", "30 days"
	if bucket == "week" {
		trunc, interval = "week", "12 weeks"
	}
	from := "now() - interval '" + interval + "'"
	to := "now()"
	if f.From != nil {
		from = "'" + f.From.UTC().Format("2006-01-02T15:04:05Z") + "'"
	}
	if f.To != nil {
		to = "'" + f.To.UTC().Format("2006-01-02T15:04:05Z") + "'"
	}

	ic := f.interactionConditions()
	ic.sql = stringsReplaceFirst(ic.sql, "1=1 AND ", "")
	ic.sql = stringsReplaceFirst(ic.sql, "1=1", "")
	_ = ic

	// Interactions / reports / successful contacts per bucket.
	rows, err := s.db.Pool().Query(ctx, fmt.Sprintf(`
		WITH series AS (
			SELECT generate_series(date_trunc('%s', %s), date_trunc('%s', %s), interval '1 %s') AS bucket
		)
		SELECT to_char(s.bucket, 'YYYY-MM-DD'),
		       count(DISTINCT i.id) FILTER (WHERE i.started_at >= %s AND i.started_at < %s + interval '1 %s'),
		       count(DISTINCT cr.id),
		       count(DISTINCT cr.id) FILTER (WHERE cr.contact_reached = true)
		FROM series s
		LEFT JOIN interactions i ON date_trunc('%s', i.started_at) = s.bucket
		LEFT JOIN contact_reports cr ON cr.interaction_id = i.id
		WHERE 1=1 %s
		GROUP BY 1 ORDER BY 1`,
		trunc, from, trunc, to, trunc, from, to, trunc, trunc, ic.sql),
		ic.args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	points := []ActivityPoint{}
	for rows.Next() {
		var p ActivityPoint
		if err := rows.Scan(&p.Label, &p.Interactions, &p.Reports, &p.SuccessfulContacts); err != nil {
			return nil, err
		}
		points = append(points, p)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	// Follow-ups created per bucket.
	fc := f.followUpConditions()
	fc.sql = stringsReplaceFirst(fc.sql, "1=1 AND ", "")
	_ = fc
	frows, err := s.db.Pool().Query(ctx, fmt.Sprintf(`
		WITH series AS (
			SELECT generate_series(date_trunc('%s', %s), date_trunc('%s', %s), interval '1 %s') AS bucket
		)
		SELECT to_char(s.bucket, 'YYYY-MM-DD'), count(f.id)
		FROM series s LEFT JOIN follow_ups f ON date_trunc('%s', f.created_at) = s.bucket
		WHERE 1=1
		GROUP BY 1 ORDER BY 1`, trunc, from, trunc, to, trunc, trunc),
	)
	if err != nil {
		return nil, err
	}
	defer frows.Close()
	fuByLabel := map[string]int{}
	for frows.Next() {
		var label string
		var n int
		if err := frows.Scan(&label, &n); err != nil {
			return nil, err
		}
		fuByLabel[label] = n
	}
	for i := range points {
		points[i].FollowUpsCreated = fuByLabel[points[i].Label]
	}
	return points, frows.Err()
}

func stringsReplaceFirst(s, old, new string) string {
	idx := 0
	for i := 0; i < len(s)-len(old); i++ {
		if s[i:i+len(old)] == old {
			idx = i
			break
		}
	}
	return s[:idx] + new + s[idx+len(old):]
}

// ---------------------------------------------------------------------------
// Score distribution

// ScoreBucket is one range of lead scores.
type ScoreBucket struct {
	Label string `json:"label"`
	Count int    `json:"count"`
}

// ScoreDistribution returns a histogram over the full filtered lead set plus
// average / median / high-quality (>=70) counts.
func (s *Service) ScoreDistribution(ctx context.Context, f Filters) ([]ScoreBucket, float64, float64, int, error) {
	lc := f.leadConditions("l")
	lc.sql += " AND l.status <> 'ARCHIVED'"
	rows, err := s.db.Pool().Query(ctx, `
		SELECT CASE
			WHEN l.score BETWEEN 0 AND 20 THEN '0-20'
			WHEN l.score BETWEEN 21 AND 40 THEN '21-40'
			WHEN l.score BETWEEN 41 AND 60 THEN '41-60'
			WHEN l.score BETWEEN 61 AND 80 THEN '61-80'
			ELSE '81-100' END AS bucket,
			count(*)
		FROM leads l WHERE `+lc.sql+`
		GROUP BY 1 ORDER BY 1`, lc.args...)
	if err != nil {
		return nil, 0, 0, 0, err
	}
	defer rows.Close()
	out := []ScoreBucket{}
	for rows.Next() {
		var b ScoreBucket
		if err := rows.Scan(&b.Label, &b.Count); err != nil {
			return nil, 0, 0, 0, err
		}
		out = append(out, b)
	}
	if err := rows.Err(); err != nil {
		return nil, 0, 0, 0, err
	}

	var avg float64
	var median float64
	var high int
	err = s.db.Pool().QueryRow(ctx, `
		SELECT COALESCE(avg(l.score),0),
		       COALESCE(percentile_cont(0.5) WITHIN GROUP (ORDER BY l.score),0),
		       count(*) FILTER (WHERE l.score >= 70)
		FROM leads l WHERE `+lc.sql, lc.args...).Scan(&avg, &median, &high)
	if err != nil {
		return nil, 0, 0, 0, err
	}
	return out, avg, median, high, nil
}

// ---------------------------------------------------------------------------
// Objections

// Objection aggregates the free-text main objection from call reports.
type Objection struct {
	Objection  string  `json:"objection"`
	Count      int     `json:"count"`
	Percentage float64 `json:"percentage"`
}

func (s *Service) Objections(ctx context.Context, f Filters, limit int) ([]Objection, int, error) {
	if limit <= 0 {
		limit = 10
	}
	ic := f.interactionConditions()
	var total int
	if err := s.db.Pool().QueryRow(ctx, `
		SELECT count(*) FROM contact_reports cr JOIN interactions i ON i.id = cr.interaction_id
		WHERE `+ic.sql+` AND cr.main_objection IS NOT NULL AND cr.main_objection <> ''`, ic.args...).Scan(&total); err != nil {
		return nil, 0, err
	}
	rows, err := s.db.Pool().Query(ctx, `
		SELECT cr.main_objection, count(*)
		FROM contact_reports cr JOIN interactions i ON i.id = cr.interaction_id
		WHERE `+ic.sql+` AND cr.main_objection IS NOT NULL AND cr.main_objection <> ''
		GROUP BY 1 ORDER BY 2 DESC LIMIT $`+itoa(len(ic.args)+1), append(ic.args, limit)...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()
	out := []Objection{}
	for rows.Next() {
		var o Objection
		if err := rows.Scan(&o.Objection, &o.Count); err != nil {
			return nil, 0, err
		}
		if total > 0 {
			o.Percentage = float64(o.Count) / float64(total) * 100
		}
		out = append(out, o)
	}
	return out, total, rows.Err()
}

// ---------------------------------------------------------------------------
// Outcomes and interest

// OutcomeCount aggregates report outcomes.
type OutcomeCount struct {
	Outcome    string  `json:"outcome"`
	Count      int     `json:"count"`
	Percentage float64 `json:"percentage"`
}

// InterestCount aggregates report interest levels.
type InterestCount struct {
	Interest   string  `json:"interest"`
	Count      int     `json:"count"`
	Percentage float64 `json:"percentage"`
}

func (s *Service) Outcomes(ctx context.Context, f Filters) ([]OutcomeCount, error) {
	ic := f.interactionConditions()
	var total int
	if err := s.db.Pool().QueryRow(ctx, `
		SELECT count(*) FROM contact_reports cr JOIN interactions i ON i.id = cr.interaction_id
		WHERE `+ic.sql, ic.args...).Scan(&total); err != nil {
		return nil, err
	}
	rows, err := s.db.Pool().Query(ctx, `
		SELECT cr.outcome, count(*) FROM contact_reports cr JOIN interactions i ON i.id = cr.interaction_id
		WHERE `+ic.sql+` GROUP BY 1 ORDER BY 2 DESC`, ic.args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []OutcomeCount{}
	for rows.Next() {
		var o OutcomeCount
		if err := rows.Scan(&o.Outcome, &o.Count); err != nil {
			return nil, err
		}
		if total > 0 {
			o.Percentage = float64(o.Count) / float64(total) * 100
		}
		out = append(out, o)
	}
	return out, rows.Err()
}

func (s *Service) Interest(ctx context.Context, f Filters) ([]InterestCount, error) {
	ic := f.interactionConditions()
	var total int
	if err := s.db.Pool().QueryRow(ctx, `
		SELECT count(*) FROM contact_reports cr JOIN interactions i ON i.id = cr.interaction_id
		WHERE `+ic.sql+` AND cr.interest_level IS NOT NULL`, ic.args...).Scan(&total); err != nil {
		return nil, err
	}
	rows, err := s.db.Pool().Query(ctx, `
		SELECT COALESCE(cr.interest_level,'NONE'), count(*)
		FROM contact_reports cr JOIN interactions i ON i.id = cr.interaction_id
		WHERE `+ic.sql+` GROUP BY 1 ORDER BY 2 DESC`, ic.args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []InterestCount{}
	for rows.Next() {
		var o InterestCount
		if err := rows.Scan(&o.Interest, &o.Count); err != nil {
			return nil, err
		}
		if total > 0 {
			o.Percentage = float64(o.Count) / float64(total) * 100
		}
		out = append(out, o)
	}
	return out, rows.Err()
}
