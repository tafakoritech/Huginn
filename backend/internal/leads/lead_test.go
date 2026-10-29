package leads

import "testing"

func TestValidTransitions(t *testing.T) {
	valid := []struct{ from, to Status }{
		{StatusNew, StatusReady},
		{StatusNew, StatusDisqualified},
		{StatusReady, StatusAssigned},
		{StatusAssigned, StatusContacting},
		{StatusAssigned, StatusQualified},
		{StatusContacting, StatusFollowUp},
		{StatusContacting, StatusDisqualified},
		{StatusFollowUp, StatusQualified},
		{StatusQualified, StatusExportedMuninn},
		{StatusQualified, StatusArchived},
		{StatusDisqualified, StatusArchived},
		{StatusExportedMuninn, StatusArchived},
	}
	for _, c := range valid {
		if !CanTransition(c.from, c.to) {
			t.Errorf("expected valid transition %s -> %s", c.from, c.to)
		}
	}
}

func TestInvalidTransitions(t *testing.T) {
	invalid := []struct{ from, to Status }{
		{StatusNew, StatusAssigned},       // must be READY first
		{StatusNew, StatusQualified},      // cannot jump to qualified
		{StatusNew, StatusExportedMuninn}, // cannot skip qualification
		{StatusArchived, StatusQualified}, // archived is terminal
		{StatusDisqualified, StatusQualified},
		{StatusContacting, StatusExportedMuninn},
	}
	for _, c := range invalid {
		if CanTransition(c.from, c.to) {
			t.Errorf("expected INVALID transition %s -> %s", c.from, c.to)
		}
	}
}

func TestSelfTransitionAllowed(t *testing.T) {
	if !CanTransition(StatusQualified, StatusQualified) {
		t.Error("self transition should be allowed")
	}
}

func TestIsValidStatus(t *testing.T) {
	if !IsValidStatus(StatusNew) || IsValidStatus("BOGUS") {
		t.Error("status validation broken")
	}
}
