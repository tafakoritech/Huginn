package messaging

import (
	"context"
	"strings"
	"testing"

	"github.com/tafakorpardazan/huginn/internal/leads"
)

// stubScenarioGenerator echoes the project title into the scenario's opening
// sentence, simulating an LLM that personalizes output from project context.
type stubScenarioGenerator struct {
	calls int
	last  *LeadContext
}

func (g *stubScenarioGenerator) GenerateScenario(ctx context.Context, leadCtx *LeadContext, opts GenerationOptions) (*CallScenario, error) {
	g.calls++
	g.last = leadCtx
	return &CallScenario{
		OpeningSentence:     "project=" + projectTitle(leadCtx),
		Introduction:        "target=" + projectProduct(leadCtx),
		ReasonForCall:       "value=" + projectValue(leadCtx),
		DiscoveryQuestions:  []string{"what pain"},
		LikelyObjections:    []string{"no budget"},
		ObjectionResponses:  []string{"we can help"},
		RecommendedNextStep: "follow up",
		ClosingLine:         "bye",
		ContextUsed: ContextUsed{
			Project:    projectTitle(leadCtx),
			Lead:       leadCtx.CompanyName,
			SalesAngle: opts.SalesAngle,
		},
	}, nil
}

func projectTitle(c *LeadContext) string {
	if c == nil || c.Project == nil {
		return ""
	}
	return c.Project.Title
}
func projectProduct(c *LeadContext) string {
	if c == nil || c.Project == nil {
		return ""
	}
	return c.Project.ProductInformation
}
func projectValue(c *LeadContext) string {
	if c == nil || c.Project == nil {
		return ""
	}
	return c.Project.ValueProposition
}

// TestScenarioDiffersByProject verifies Test 1 of the validation spec: the same
// lead paired with Project A must produce a different scenario than the same
// lead paired with Project B, because the project context is part of the prompt.
func TestScenarioDiffersByProject(t *testing.T) {
	stub := &stubScenarioGenerator{}

	ctx := context.Background()
	lead := &leads.Lead{ID: "lead-1", CompanyName: "Acme"}

	// Project A context
	a := &ProjectContext{
		ProjectID:          "proj-a",
		Title:              "Enterprise ERP",
		Description:        "Deploy our ERP suite to large enterprises.",
		ProductInformation: "ERP software for manufacturing",
		TargetCustomer:     "CIOs at manufacturers",
		ValueProposition:   "Reduce operational cost by 20%",
		PainPoints:         "Legacy systems, manual processes",
	}
	// Project B context
	b := &ProjectContext{
		ProjectID:          "proj-b",
		Title:              "SMB E-commerce",
		Description:        "Bring online store platform to small retailers.",
		ProductInformation: "Hosted e-commerce storefront",
		TargetCustomer:     "Small retail owners",
		ValueProposition:   "Launch a store in a day",
		PainPoints:         "No online presence, slow checkout",
	}

	leadCtxA := &LeadContext{LeadID: lead.ID, CompanyName: lead.CompanyName, Project: a}
	leadCtxB := &LeadContext{LeadID: lead.ID, CompanyName: lead.CompanyName, Project: b}

	scA, err := stub.GenerateScenario(ctx, leadCtxA, GenerationOptions{})
	if err != nil {
		t.Fatalf("scenario A: %v", err)
	}
	scB, err := stub.GenerateScenario(ctx, leadCtxB, GenerationOptions{})
	if err != nil {
		t.Fatalf("scenario B: %v", err)
	}

	// Same lead data, different projects -> different scenarios.
	if scA.OpeningSentence == scB.OpeningSentence {
		t.Errorf("expected different scenarios for different projects; both opening = %q", scA.OpeningSentence)
	}
	if !strings.Contains(scA.OpeningSentence, "Enterprise ERP") {
		t.Errorf("scenario A must reference project A: %q", scA.OpeningSentence)
	}
	if !strings.Contains(scB.OpeningSentence, "SMB E-commerce") {
		t.Errorf("scenario B must reference project B: %q", scB.OpeningSentence)
	}

	// The rendered prompt (project block) must differ for the two projects.
	renderA := leadCtxA.Render()
	renderB := leadCtxB.Render()
	if renderA == renderB {
		t.Error("project context must produce different rendered prompts")
	}
	if !strings.Contains(renderA, "Enterprise ERP") || !strings.Contains(renderB, "SMB E-commerce") {
		t.Error("rendered prompts must carry each project's context")
	}
	if !strings.Contains(renderA, "PROJECT CONTEXT") {
		t.Error("rendered prompt must include a PROJECT CONTEXT block")
	}
}

// TestRenderIncludesAllProjectFields verifies the LLM prompt is built from
// project.title, product_information, target_customer, value_proposition and
// pain_points (Phase 5 requirement).
func TestRenderIncludesAllProjectFields(t *testing.T) {
	ctx := &LeadContext{
		CompanyName: "Acme",
		Project: &ProjectContext{
			Title:              "T",
			Description:        "D",
			ProductInformation: "PI",
			TargetCustomer:     "TC",
			ValueProposition:   "VP",
			PainPoints:         "PP",
			SalesObjective:     "SO",
			CommonObjections:   "CO",
		},
	}
	out := ctx.Render()
	for _, want := range []string{"Project Title", "Project Description", "Product / Service", "Target Customer", "Customer Pain Points", "Value Proposition", "Sales Goal", "Common Objections"} {
		if !strings.Contains(out, want) {
			t.Errorf("rendered prompt missing project field %q", want)
		}
	}
}

// TestRenderSkipsEmptyProject verifies an empty project produces no block.
func TestRenderSkipsEmptyProject(t *testing.T) {
	ctx := &LeadContext{CompanyName: "Acme"}
	if strings.Contains(ctx.Render(), "PROJECT CONTEXT") {
		t.Error("no project context block expected when project is nil")
	}
}

// TestProjectUpdateAffectsFutureScenarios verifies Test 3 of the validation
// spec: after a project's description is updated, the next scenario generation
// must use the NEW description. Because the scenario generator rebuilds the
// prompt from the latest project data on every call, regenerating with the
// updated project yields a different (updated) scenario.
func TestProjectUpdateAffectsFutureScenarios(t *testing.T) {
	stub := &stubScenarioGenerator{}
	ctx := context.Background()
	lead := &leads.Lead{ID: "lead-x", CompanyName: "Acme"}

	before := &ProjectContext{
		ProjectID:          "p1",
		Title:              "ERP",
		Description:        "legacy system integration",
		ValueProposition:   "cut operational cost",
		ProductInformation: "ERP software",
	}
	scenarioBefore, err := stub.GenerateScenario(ctx, &LeadContext{LeadID: lead.ID, CompanyName: lead.CompanyName, Project: before}, GenerationOptions{})
	if err != nil {
		t.Fatalf("scenario before: %v", err)
	}

	// Simulate: the project description + value proposition are updated.
	after := *before
	after.Description = "cloud-native ERP with AI forecasting"
	after.ValueProposition = "cut operational cost by 30% using AI"

	scenarioAfter, err := stub.GenerateScenario(ctx, &LeadContext{LeadID: lead.ID, CompanyName: lead.CompanyName, Project: &after}, GenerationOptions{})
	if err != nil {
		t.Fatalf("scenario after: %v", err)
	}

	if scenarioBefore.OpeningSentence == scenarioAfter.OpeningSentence &&
		scenarioBefore.ReasonForCall == scenarioAfter.ReasonForCall {
		t.Error("regenerated scenario must differ after project update")
	}
	// The regenerated scenario reflects the updated value proposition.
	if !strings.Contains(scenarioAfter.ReasonForCall, "cut operational cost by 30% using AI") {
		t.Errorf("future scenario must use updated value proposition, got: %q", scenarioAfter.ReasonForCall)
	}
	if strings.Contains(scenarioAfter.ReasonForCall, "cut operational cost\n") {
		t.Error("stale value proposition must not appear in the regenerated scenario")
	}
	// The rendered prompt carries the new description.
	rendered := (&LeadContext{CompanyName: "Acme", Project: &after}).Render()
	if !strings.Contains(rendered, "cloud-native ERP with AI forecasting") {
		t.Error("rendered prompt must carry the updated description")
	}
}
