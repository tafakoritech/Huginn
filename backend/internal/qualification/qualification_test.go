package qualification

import "testing"

func TestShouldQualifyAllRules(t *testing.T) {
	in := QualificationInputs{
		BusinessRelevant: true, ContactInfoValid: true, MeaningfulContactOccurred: true,
		NeedOrInterestExists: true, DecisionMakerIdentified: true, NextSalesActionExists: true,
		Score: 75,
	}
	ok, missing := ShouldQualify(in, Rules{ScoreThreshold: 70})
	if !ok {
		t.Fatalf("expected qualify, missing: %v", missing)
	}
}

func TestShouldQualifyScoreGate(t *testing.T) {
	in := QualificationInputs{
		BusinessRelevant: true, ContactInfoValid: true, MeaningfulContactOccurred: true,
		NeedOrInterestExists: true, DecisionMakerIdentified: true, NextSalesActionExists: true,
		Score: 60,
	}
	ok, missing := ShouldQualify(in, Rules{ScoreThreshold: 70})
	if ok {
		t.Fatal("should not qualify below threshold")
	}
	found := false
	for _, m := range missing {
		if m == "minimum_score_reached" {
			found = true
		}
	}
	if !found {
		t.Fatalf("expected minimum_score_reached missing rule, got %v", missing)
	}
}

func TestShouldQualifyThresholdConfigurable(t *testing.T) {
	in := QualificationInputs{
		BusinessRelevant: true, ContactInfoValid: true, MeaningfulContactOccurred: true,
		NeedOrInterestExists: true, DecisionMakerIdentified: true, NextSalesActionExists: true,
		Score: 60,
	}
	ok, _ := ShouldQualify(in, Rules{ScoreThreshold: 50})
	if !ok {
		t.Fatal("lower threshold should qualify score 60")
	}
}

func TestShouldQualifyRequiresMeaningfulContact(t *testing.T) {
	in := QualificationInputs{
		BusinessRelevant: true, ContactInfoValid: true, NeedOrInterestExists: true,
		DecisionMakerIdentified: true, NextSalesActionExists: true, Score: 90,
	}
	ok, missing := ShouldQualify(in, Rules{ScoreThreshold: 70})
	if ok {
		t.Fatal("must require meaningful contact")
	}
	if len(missing) == 0 {
		t.Fatal("expected missing rules")
	}
}

func TestDisqualifyReasons(t *testing.T) {
	if !IsValidDisqualifyReason(ReasonNoInterest) {
		t.Error("ReasonNoInterest should be valid")
	}
	if IsValidDisqualifyReason("NOT_A_REASON") {
		t.Error("bogus reason should be invalid")
	}
}
