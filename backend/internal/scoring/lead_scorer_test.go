package scoring

import (
	"testing"

	"github.com/tafakorpardazan/huginn/internal/contacts"
	"github.com/tafakorpardazan/huginn/internal/leads"
	"github.com/tafakorpardazan/huginn/internal/reports"
)

func TestBuildSignalsAccumulatesAcrossReports(t *testing.T) {
	lead := &leads.Lead{
		Industry:                strp("صنعت"),
		Description:             strp("توضیح"),
		DecisionMakerIdentified: true,
	}
	contact := &contacts.Contact{
		Mobile: strp("0912"),
		Email:  strp("a@b.c"),
	}

	high := reports.InterestHigh
	medium := reports.InterestMedium
	capacity := "مبلغ"
	soon := "SOON"
	reps := []*reports.Report{
		{Outcome: reports.OutcomeInterested, InterestLevel: &medium, DecisionMakerIdentified: boolp(true)},
		{Outcome: reports.OutcomeInterested, InterestLevel: &high, NeedLevel: &high, EstimatedPurchaseCapacity: &capacity, PurchaseTiming: &soon},
		{Outcome: reports.OutcomeAnswered},
	}

	sig := BuildSignals(lead, contact, reps, true, true)

	if sig.PositiveReports != 3 {
		t.Fatalf("expected 3 positive reports, got %d", sig.PositiveReports)
	}
	if sig.InteractionCount != 3 {
		t.Fatalf("expected 3 interactions, got %d", sig.InteractionCount)
	}
	if !sig.InterestHigh || !sig.InterestMedium {
		t.Fatal("interest signals must accumulate from any report")
	}
	if !sig.NeedHigh {
		t.Fatal("need signal must be promoted from the report that established it")
	}
	if !sig.DecisionMakerIdentified {
		t.Fatal("decision maker signal must survive even though the last report did not set it")
	}
	if !sig.HasPurchaseCapacity || !sig.PurchaseSoon {
		t.Fatal("purchase evidence must accumulate")
	}
	if !sig.HasFollowUpScheduled {
		t.Fatal("open follow-up must be reflected")
	}
	if !sig.HasMobile || !sig.HasEmail {
		t.Fatal("contact signals must be included")
	}
	if !sig.HasIndustry || !sig.HasDescription || !sig.SourceTypeRich {
		t.Fatal("lead base signals must be carried into the recompute")
	}
}

func TestBuildSignalsNoReportsScoresZeroEngagement(t *testing.T) {
	lead := &leads.Lead{}
	sig := BuildSignals(lead, nil, nil, false, false)
	if sig.PositiveReports != 0 || sig.InteractionCount != 0 {
		t.Fatalf("no reports must yield zero engagement signals, got %+v", sig)
	}
	if sig.DecisionMakerIdentified {
		t.Fatal("no decision maker evidence must not mark the lead as having one")
	}
}

func TestBuildSignalsLeadFactPowersAuthority(t *testing.T) {
	lead := &leads.Lead{DecisionMakerIdentified: true}
	sig := BuildSignals(lead, nil, nil, false, false)
	if !sig.DecisionMakerIdentified {
		t.Fatal("a previously-persisted decision maker fact must be kept")
	}
}

func TestBuildSignalsDoesNotClearFactsFromEmptyLaterReport(t *testing.T) {
	// Report 1 establishes a decision maker; report 2 is a no-answer call with
	// no decision maker data. The accumulated signals must still report the
	// decision maker (facts are promoted to the lead, not re-derived per call).
	lead := &leads.Lead{DecisionMakerIdentified: true}
	reps := []*reports.Report{
		{Outcome: reports.OutcomeNoAnswer},
	}
	sig := BuildSignals(lead, nil, reps, false, false)
	if !sig.DecisionMakerIdentified {
		t.Fatal("a later report without decision-maker data must not unset the fact")
	}
}

func TestPurchaseTimingSoonClassification(t *testing.T) {
	cases := map[string]bool{
		"SOON":            true,
		" soon ":          true,
		"NOW":             true,
		"WITHIN_3_MONTHS": true,
		"":                false,
		"سال آینده":       false,
	}
	for in, want := range cases {
		if got := purchaseTimingSoon(&in); got != want {
			t.Errorf("purchaseTimingSoon(%q) = %v, want %v", in, got, want)
		}
	}
	if purchaseTimingSoon(nil) {
		t.Error("nil timing must not be classified as soon")
	}
}

func strp(s string) *string { return &s }
