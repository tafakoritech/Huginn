package scoring

import "testing"

func TestScoreEmpty(t *testing.T) {
	r := Score(Signals{})
	if r.Total != 0 {
		t.Fatalf("empty signals should score 0, got %d", r.Total)
	}
	for k, v := range r.Components {
		if v != 0 {
			t.Errorf("component %s should be 0, got %d", k, v)
		}
	}
}

func TestScoreComponents(t *testing.T) {
	r := Score(Signals{
		HasIndustry: true, HasDescription: true, HasBusinessType: true, SourceRelevant: true,
		HasMobile: true, HasPhone: true, HasEmail: true,
		HasAddress: true, HasWebsite: true, SourceTypeRich: true,
		InteractionCount: 3, PositiveReports: 1,
		DecisionMakerIdentified: true,
		InterestHigh:            true,
		HasFollowUpScheduled:    true,
		PurchaseSoon:            true,
	})
	// Verify each component is capped at its weight.
	caps := map[string]int{
		CompBusinessFit: 25, CompContactability: 15, CompBusinessScale: 10,
		CompEngagement: 14, CompAuthority: 8, CompNeed: 6, CompTiming: 4,
	}
	for k, cap := range caps {
		if r.Components[k] > cap {
			t.Errorf("component %s exceeded cap %d: %d", k, cap, r.Components[k])
		}
	}
	if r.Total > 82 {
		t.Errorf("max possible score is 82, got %d", r.Total)
	}
	if r.Total < 70 {
		t.Errorf("strong lead should score >= 70, got %d", r.Total)
	}
}

func TestScoreDeterministic(t *testing.T) {
	s1 := Signalable{HasMobile: true, HasDescription: true}
	s2 := Signalable{HasMobile: true, HasDescription: true}
	r1 := Score(SignalsFrom(s1))
	r2 := Score(SignalsFrom(s2))
	if r1.Total != r2.Total {
		t.Error("scoring must be deterministic")
	}
}

// Signalable helps keep tests concise.
type Signalable struct {
	HasMobile     bool
	HasDescription bool
}

func SignalsFrom(s Signalable) Signals {
	return Signals{HasMobile: s.HasMobile, HasDescription: s.HasDescription}
}
