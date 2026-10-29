package projects

import (
	"testing"
)

// TestProjectModelComposition verifies the model-level field composition that
// feeds the LLM prompt (Phase 5: project.product_information, target_customer,
// value_proposition, pain_points).
func TestProjectModelComposition(t *testing.T) {
	pn := "ERP Suite"
	pd := "Enterprise resource planning for manufacturers"
	tm := "Large manufacturers"
	icp := "CIOs and operations heads"
	pp := "Legacy systems, manual reconciliation"

	p := &Project{
		ProductName:          &pn,
		ProductDescription:   &pd,
		TargetMarket:         &tm,
		IdealCustomerProfile: &icp,
		CustomerPainPoints:   &pp,
	}
	pi := p.ProductInformation()
	if pi != "ERP Suite — Enterprise resource planning for manufacturers" {
		t.Errorf("unexpected product info: %q", pi)
	}
	tc := p.TargetCustomer()
	if tc != "Large manufacturers؛ CIOs and operations heads" {
		t.Errorf("unexpected target customer: %q", tc)
	}
	// Empty fields are tolerated.
	empty := &Project{}
	if empty.ProductInformation() != "" || empty.TargetCustomer() != "" {
		t.Error("empty project should compose to empty strings")
	}
}

// TestProjectStatusDefaults verifies the active status default.
func TestProjectStatusDefaults(t *testing.T) {
	p := &Project{Title: "T", Description: "D"}
	if p.Status != "" {
		t.Error("zero-value status expected for fresh struct")
	}
}
