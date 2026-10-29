package nextactions

import (
	"testing"
	"time"
)

func now() time.Time { return time.Date(2026, 8, 24, 12, 0, 0, 0, time.UTC) }

func urgencyReasonFor(due *time.Time, n time.Time) string {
	_, r := urgencyFor(due, n)
	return r
}

func followUpCandidate(leadID, source string, due time.Time, score int, committed bool) candidate {
	reasons := []string{}
	if ur := urgencyReasonFor(&due, now()); ur != "" {
		reasons = append(reasons, ur)
	}
	reasons = append(reasons, qualityReason(score))
	if committed {
		reasons = append(reasons, "در گزارش قبلی اقدام بعدی ثبت شده")
	}
	u, _ := urgencyFor(&due, now())
	commit := 0
	if committed {
		commit = commitmentExplicit
	}
	total := u + leadQuality(score) + commit
	it := &ActionItem{
		ID:            "follow_up:" + leadID,
		SourceType:    SourceFollowUp,
		SourceID:      leadID,
		LeadID:        leadID,
		CompanyName:   "شرکت " + leadID,
		ActionType:    ActionCall,
		DueAt:         &due,
		Overdue:       due.Before(now()),
		Priority:      TierForScore(total),
		PriorityScore: total,
		Explainability: reasons,
	}
	return candidate{rank: total, item: it}
}

func reportCandidate(leadID string) candidate {
	total := pendingReportBase + urgencyOverdueWithin
	it := &ActionItem{
		ID:             "pending_report:" + leadID,
		SourceType:     SourcePendingReport,
		SourceID:       leadID,
		LeadID:         leadID,
		CompanyName:    "شرکت " + leadID,
		ActionType:     ActionReport,
		Priority:       TierForScore(total),
		PriorityScore:  total,
		Explainability: []string{"گزارش تماس ثبت نشده"},
	}
	return candidate{rank: total, item: it}
}

func inferredCandidate(leadID string, firstContact bool, score int) candidate {
	src := SourceStaleLead
	base := staleLeadBase
	if firstContact {
		src = SourceFirstContact
		base = firstContactBase
	}
	total := base + leadQuality(score)
	it := &ActionItem{
		ID:             string(src) + ":" + leadID,
		SourceType:     src,
		SourceID:       leadID,
		LeadID:         leadID,
		CompanyName:    "شرکت " + leadID,
		Priority:       TierForScore(total),
		PriorityScore:  total,
		Explainability: []string{qualityReason(score)},
	}
	return candidate{rank: total, item: it}
}

func TestTierForScore(t *testing.T) {
	cases := []struct {
		score int
		tier  Tier
	}{
		{100, TierUrgent}, {80, TierUrgent}, {79, TierImportant},
		{50, TierImportant}, {49, TierNormal}, {0, TierNormal},
	}
	for _, c := range cases {
		if got := TierForScore(c.score); got != c.tier {
			t.Errorf("TierForScore(%d) = %s, want %s", c.score, got, c.tier)
		}
	}
}

func TestLeadQuality(t *testing.T) {
	if leadQuality(70) != qualityHot || leadQuality(85) != qualityHot {
		t.Fatal("hot leads should map to qualityHot")
	}
	if leadQuality(40) != qualityWarm || leadQuality(69) != qualityWarm {
		t.Fatal("warm leads should map to qualityWarm")
	}
	if leadQuality(39) != qualityCold || leadQuality(0) != qualityCold {
		t.Fatal("cold leads should map to qualityCold")
	}
}

func TestUrgencyBuckets(t *testing.T) {
	n := now()
	overTwoDays := n.Add(-2 * 24 * time.Hour)
	overHalfDay := n.Add(-12 * time.Hour)
	inOneHour := n.Add(time.Hour)
	todayLater := n.Add(4 * time.Hour)
	tomorrow := n.Add(30 * time.Hour)
	nextWeek := n.Add(7 * 24 * time.Hour)

	if pts, _ := urgencyFor(&overTwoDays, n); pts != urgencyOverdueOverDay {
		t.Errorf("overdue >24h: got %d", pts)
	}
	if pts, _ := urgencyFor(&overHalfDay, n); pts != urgencyOverdueWithin {
		t.Errorf("overdue <24h: got %d", pts)
	}
	if pts, _ := urgencyFor(&inOneHour, n); pts != urgencyDueSoon2h {
		t.Errorf("due within 2h: got %d", pts)
	}
	if pts, _ := urgencyFor(&todayLater, n); pts != urgencyDueToday {
		t.Errorf("due today: got %d", pts)
	}
	if pts, _ := urgencyFor(&tomorrow, n); pts != urgencyDueTomorrow {
		t.Errorf("due tomorrow: got %d", pts)
	}
	if pts, _ := urgencyFor(&nextWeek, n); pts != 0 {
		t.Errorf("future: got %d, want 0", pts)
	}
}

func TestPromisedCommitment(t *testing.T) {
	if !promisedCommitment("REPORT", "", "") {
		t.Fatal("report-sourced follow-up is an explicit commitment")
	}
	if !promisedCommitment("MANUAL", "فردا تماس بگیرید", "") {
		t.Fatal("'تماس بگیرید' should be detected")
	}
	if !promisedCommitment("MANUAL", "", "اطلاعات را بفرستید") {
		t.Fatal("'بفرستید' should be detected")
	}
	if promisedCommitment("MANUAL", "", "پیگیری کلی") {
		t.Fatal("generic description should not count as commitment")
	}
}

func TestOverdueOutranksFutureTask(t *testing.T) {
	overdue := followUpCandidate("a", "REPORT", now().Add(-2*time.Hour), 40, true)
	future := followUpCandidate("b", "REPORT", now().Add(5*24*time.Hour), 90, true)
	cs := []candidate{overdue, future}
	sortCandidates(cs)
	if cs[0].item.LeadID != "a" {
		t.Fatalf("overdue follow-up must outrank future task; got %s first", cs[0].item.LeadID)
	}
}

func TestPendingReportHasHighPriority(t *testing.T) {
	r := reportCandidate("x")
	if r.item.Priority != TierUrgent {
		t.Fatalf("pending report should be urgent, got %s (score %d)", r.item.Priority, r.item.PriorityScore)
	}
	if r.item.PriorityScore < urgentMin {
		t.Fatalf("pending report score %d below urgent threshold", r.item.PriorityScore)
	}
}

func TestWarmOutranksColdWhenEqual(t *testing.T) {
	warm := followUpCandidate("warm", "MANUAL", now().Add(24*time.Hour), 50, false)
	cold := followUpCandidate("cold", "MANUAL", now().Add(24*time.Hour), 20, false)
	if warm.item.PriorityScore <= cold.item.PriorityScore {
		t.Fatalf("warm lead must outrank cold: %d vs %d", warm.item.PriorityScore, cold.item.PriorityScore)
	}
}

func TestDedupeCollapsesSameLead(t *testing.T) {
	due := now().Add(-time.Hour)
	report := reportCandidate("lead-1")
	followup := followUpCandidate("lead-1", "REPORT", due, 60, true)
	stale := inferredCandidate("lead-1", false, 60)

	got := dedupe([]candidate{stale, followup, report})
	if len(got) != 1 {
		t.Fatalf("expected 1 item after dedup, got %d", len(got))
	}
	if got[0].item.SourceType != SourcePendingReport {
		t.Fatalf("pending report must win dedup, got %s", got[0].item.SourceType)
	}
}

func TestExplicitFollowUpBeatsInferred(t *testing.T) {
	followup := followUpCandidate("lead-1", "REPORT", now().Add(time.Hour), 50, true)
	stale := inferredCandidate("lead-1", false, 90)
	got := dedupe([]candidate{stale, followup})
	if len(got) != 1 || got[0].item.SourceType != SourceFollowUp {
		t.Fatalf("explicit follow-up must beat inferred stale-lead action")
	}
}

func TestSortDeterministic(t *testing.T) {
	base := time.Date(2026, 8, 24, 12, 0, 0, 0, time.UTC)
	a := followUpCandidate("a", "REPORT", base.Add(time.Hour), 40, true)
	b := followUpCandidate("b", "REPORT", base.Add(time.Hour), 40, true)
	cs := []candidate{a, b}
	sortCandidates(cs)
	if cs[0].item.LeadID != "a" || cs[1].item.LeadID != "b" {
		t.Fatalf("equal-score ties must be deterministic (company name); got %s, %s", cs[0].item.LeadID, cs[1].item.LeadID)
	}
}

func TestExplainabilityReasonsPresent(t *testing.T) {
	due := now().Add(-26 * time.Hour)
	c := followUpCandidate("lead-1", "REPORT", due, 75, true)
	if len(c.item.Explainability) < 2 {
		t.Fatalf("expected multiple explainability reasons, got %v", c.item.Explainability)
	}
	foundOverdue := false
	foundQuality := false
	foundCommitment := false
	for _, r := range c.item.Explainability {
		if r == "بیش از یک روز از سررسید پیگیری گذشته است" {
			foundOverdue = true
		}
		if r == "سرنخ پرارزش است" {
			foundQuality = true
		}
		if r == "در گزارش قبلی اقدام بعدی ثبت شده" {
			foundCommitment = true
		}
	}
	if !foundOverdue || !foundQuality || !foundCommitment {
		t.Fatalf("missing explainability reasons: %v", c.item.Explainability)
	}
}

func TestCTALabels(t *testing.T) {
	cases := map[ActionType]string{
		ActionCall: "تماس", ActionMessage: "پیام", ActionEmail: "ایمیل",
		ActionReport: "ثبت گزارش", ActionFollowUp: "پیگیری", ActionQualification: "ارزیابی",
	}
	for a, want := range cases {
		if got := ctaLabelFor(a); got != want {
			t.Errorf("ctaLabelFor(%s) = %q, want %q", a, got, want)
		}
	}
}
