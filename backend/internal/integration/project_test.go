package integration

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/tafakorpardazan/huginn/internal/callscenarios"
	"github.com/tafakorpardazan/huginn/internal/contacts"
	"github.com/tafakorpardazan/huginn/internal/interactions"
	"github.com/tafakorpardazan/huginn/internal/leads"
	"github.com/tafakorpardazan/huginn/internal/leadversions"
	"github.com/tafakorpardazan/huginn/internal/messaging"
	"github.com/tafakorpardazan/huginn/internal/notes"
	"github.com/tafakorpardazan/huginn/internal/projects"
	"github.com/tafakorpardazan/huginn/internal/reports"
)

// scenarioStubGenerator echoes the project title/value prop from the lead
// context into the scenario, simulating an LLM that personalizes output from
// project context (no real LLM call required). It models the new reasoning
// flow: the model picks a primary_angle; when a previous angle is supplied
// (Generate Again) it picks a different one.
type scenarioStubGenerator struct{}

func (g *scenarioStubGenerator) GenerateScenario(ctx context.Context, leadCtx *messaging.LeadContext, opts messaging.GenerationOptions) (*messaging.CallScenario, error) {
	projTitle := ""
	projVP := ""
	if leadCtx.Project != nil {
		projTitle = leadCtx.Project.Title
		projVP = leadCtx.Project.ValueProposition
	}
	if projTitle == "" {
		projTitle = "NO_PROJECT"
	}
	angle := strings.TrimSpace(opts.SalesAngle)
	if angle == "" {
		angle = "stub-angle"
		if opts.PreviousAngle != "" {
			angle = "stub-angle-different-from-" + opts.PreviousAngle
		}
	}
	return &messaging.CallScenario{
		OpeningSentence:     "project=" + projTitle,
		Introduction:        "company=" + leadCtx.CompanyName,
		ReasonForCall:       "value=" + projVP,
		DiscoveryQuestions:  []string{"q1"},
		LikelyObjections:    []string{"o1"},
		ObjectionResponses:  []string{"r1"},
		RecommendedNextStep: "next",
		ClosingLine:         "bye",
		ContextUsed: messaging.ContextUsed{
			Project:    projTitle,
			Lead:       leadCtx.CompanyName,
			SalesAngle: angle,
		},
		Strategy: &messaging.CallScenarioStrategy{
			LeadSummary:   leadCtx.CompanyName,
			Fit:           "uncertain",
			FitSummary:    "stub",
			PrimaryAngle:  angle,
			CallObjective: "stub-qualify",
			LikelyPain:    "stub-pain",
		},
		IsGenerated: true,
	}, nil
}

// TestProjectDrivesScenarioGeneration validates the end-to-end AI integration:
//   - Test 1: the same lead company under Project A vs Project B produces
//     different scenarios because project context is part of the prompt.
//   - Test 2: a lead without a project cannot generate a scenario.
//   - Test 3: updating a project's value proposition changes future scenarios.
func TestProjectDrivesScenarioGeneration(t *testing.T) {
	db := setupDB(t)
	ctx := context.Background()

	projectRepo := projects.NewRepository(db)
	if err := projectRepo.EnsureSchema(ctx); err != nil {
		t.Fatalf("project schema: %v", err)
	}
	leadRepo := leads.NewRepository(db)
	if err := leadRepo.EnsureSchema(ctx); err != nil {
		t.Fatalf("lead schema: %v", err)
	}
	contactRepo := contacts.NewRepository(db)
	noteRepo := notes.NewRepository(db)
	reportRepo := reports.NewRepository(db)
	versionRepo := leadversions.NewRepository(db)
	interRepo := interactions.NewRepository(db)

	projectSvc := projects.NewService(projectRepo)
	gen := &scenarioStubGenerator{}
	svc := messaging.NewContactActionService(interRepo, contactRepo, leadRepo, noteRepo, reportRepo, projectRepo, nil, gen, nil)
	leadSvc := leads.NewService(leadRepo, leads.Deps{Contacts: contactRepo, Versions: versionRepo})

	// Two projects with distinct value propositions / products.
	vpA := "reduce warehouse costs by 20%"
	prodA := "warehouse management software"
	projA, err := projectSvc.Create(ctx, projects.CreateParams{
		Title:              "Warehouse WMS",
		Description:        "Warehouse management rollout",
		ProductName:        &prodA,
		ProductDescription: ptrString("Optimizes pick/pack for factories"),
		TargetMarket:       ptrString(" Factories"),
		ValueProposition:   &vpA,
		CustomerPainPoints: ptrString("manual picking errors"),
	})
	if err != nil {
		t.Fatalf("create project A: %v", err)
	}
	vpB := "grow online sales by 3x"
	projB, err := projectSvc.Create(ctx, projects.CreateParams{
		Title:              "E-commerce Growth",
		Description:        "Online store launch",
		ValueProposition:   &vpB,
		CustomerPainPoints: ptrString("no online presence"),
	})
	if err != nil {
		t.Fatalf("create project B: %v", err)
	}

	// Test 1: same company, different projects -> different scenarios.
	leadA, err := leadSvc.Create(ctx, leads.CreateParams{CompanyName: "Acme Corp", ProjectID: &projA.ID})
	if err != nil {
		t.Fatalf("create lead A: %v", err)
	}
	leadB, err := leadSvc.Create(ctx, leads.CreateParams{CompanyName: "Acme Corp", ProjectID: &projB.ID})
	if err != nil {
		t.Fatalf("create lead B: %v", err)
	}

	uid := "user-x"
	rA, err := svc.CallScenario(ctx, leadA.ID, nil, &uid, nil, "")
	if err != nil {
		t.Fatalf("scenario A: %v", err)
	}
	rB, err := svc.CallScenario(ctx, leadB.ID, nil, &uid, nil, "")
	if err != nil {
		t.Fatalf("scenario B: %v", err)
	}
	if rA.Scenario == nil || rB.Scenario == nil {
		t.Fatal("scenarios must be generated")
	}
	if rA.Scenario.OpeningSentence == rB.Scenario.OpeningSentence {
		t.Errorf("expected different scenarios for different projects; both = %q", rA.Scenario.OpeningSentence)
	}
	if !strings.Contains(rA.Scenario.OpeningSentence, "Warehouse WMS") {
		t.Errorf("scenario A must reference project A: %q", rA.Scenario.OpeningSentence)
	}
	if !strings.Contains(rB.Scenario.OpeningSentence, "E-commerce Growth") {
		t.Errorf("scenario B must reference project B: %q", rB.Scenario.OpeningSentence)
	}
	if rA.ProjectID != projA.ID || rB.ProjectID != projB.ID {
		t.Errorf("project id not surfaced: A=%q B=%q", rA.ProjectID, rB.ProjectID)
	}

	// Test 2: a lead with no project cannot generate a scenario.
	leadNoProj, err := leadRepo.Create(ctx, &leads.Lead{CompanyName: "No Project Co", Status: leads.StatusNew})
	if err != nil {
		t.Fatalf("create projectless lead: %v", err)
	}
	if _, err := svc.CallScenario(ctx, leadNoProj.ID, nil, &uid, nil, ""); !errors.Is(err, messaging.ErrNoProject) {
		t.Fatalf("expected ErrNoProject for projectless lead, got: %v", err)
	}
	// Providing a project_id that does not belong to the lead must also fail.
	if _, err := svc.CallScenario(ctx, leadNoProj.ID, nil, &uid, &projA.ID, ""); !errors.Is(err, messaging.ErrNoProject) {
		t.Fatalf("expected ErrNoProject for projectless lead even with project_id param, got: %v", err)
	}

	// Test 3: updating a project's value proposition changes future scenarios.
	updatedVP := "reduce warehouse costs by 40% with AI"
	if _, err := projectSvc.Update(ctx, projA.ID, projects.UpdateParams{
		Title:            projA.Title,
		Description:      projA.Description,
		ValueProposition: &updatedVP,
	}); err != nil {
		t.Fatalf("update project: %v", err)
	}
	rAfter, err := svc.CallScenario(ctx, leadA.ID, nil, &uid, nil, "")
	if err != nil {
		t.Fatalf("scenario after update: %v", err)
	}
	if !strings.Contains(rAfter.Scenario.ReasonForCall, "reduce warehouse costs by 40% with AI") {
		t.Errorf("future scenario must use updated value proposition, got: %q", rAfter.Scenario.ReasonForCall)
	}
	if rBefore := rA.Scenario.ReasonForCall; rBefore == rAfter.Scenario.ReasonForCall {
		t.Error("regenerated scenario must differ after project update")
	}
}

func ptrString(s string) *string { return &s }

// TestCallScenarioRegenerationVariation validates the "Generate Again"
// behavior end-to-end: repeated calls for the same lead + project must
// (1) vary by reasoning (the previous generation's angle is passed as a hint
// and the strategy picks a different angle), (2) generate a fresh generation_id
// each time, (3) persist every generation separately with its angle, strategy
// and context_used, and (4) honor an explicit client sales-angle override.
func TestCallScenarioRegenerationVariation(t *testing.T) {
	db := setupDB(t)
	ctx := context.Background()

	projectRepo := projects.NewRepository(db)
	if err := projectRepo.EnsureSchema(ctx); err != nil {
		t.Fatalf("project schema: %v", err)
	}
	leadRepo := leads.NewRepository(db)
	if err := leadRepo.EnsureSchema(ctx); err != nil {
		t.Fatalf("lead schema: %v", err)
	}
	contactRepo := contacts.NewRepository(db)
	noteRepo := notes.NewRepository(db)
	reportRepo := reports.NewRepository(db)
	versionRepo := leadversions.NewRepository(db)
	interRepo := interactions.NewRepository(db)
	scenarioRepo := callscenarios.NewRepository(db)

	projectSvc := projects.NewService(projectRepo)
	gen := &scenarioStubGenerator{}
	svc := messaging.NewContactActionService(interRepo, contactRepo, leadRepo, noteRepo, reportRepo, projectRepo, nil, gen, scenarioRepo)
	leadSvc := leads.NewService(leadRepo, leads.Deps{Contacts: contactRepo, Versions: versionRepo})

	vp := "reduce warehouse costs by 20%"
	proj, err := projectSvc.Create(ctx, projects.CreateParams{
		Title:              "Warehouse WMS",
		Description:        "Warehouse management rollout",
		ValueProposition:   &vp,
		CustomerPainPoints: ptrString("manual picking errors"),
	})
	if err != nil {
		t.Fatalf("create project: %v", err)
	}
	lead, err := leadSvc.Create(ctx, leads.CreateParams{CompanyName: "Acme Corp", ProjectID: &proj.ID})
	if err != nil {
		t.Fatalf("create lead: %v", err)
	}

	r1, err := svc.CallScenario(ctx, lead.ID, nil, nil, nil, "")
	if err != nil {
		t.Fatalf("first generation: %v", err)
	}
	r2, err := svc.CallScenario(ctx, lead.ID, nil, nil, nil, "")
	if err != nil {
		t.Fatalf("second generation: %v", err)
	}

	if r1.SalesAngle == r2.SalesAngle {
		t.Errorf("repeated generations must vary their strategy angle; both = %q", r1.SalesAngle)
	}
	if !strings.Contains(r2.SalesAngle, "different-from-"+r1.SalesAngle) {
		t.Errorf("the second generation must receive and use the previous angle hint: prev=%q cur=%q", r1.SalesAngle, r2.SalesAngle)
	}
	if r1.Scenario.GenerationID == "" {
		t.Error("every generation must carry a unique generation_id")
	}
	if r1.Scenario.GenerationID == r2.Scenario.GenerationID {
		t.Errorf("generation ids must be unique across generations: %q", r1.Scenario.GenerationID)
	}
	if r1.Scenario.ContextUsed.Project != "Warehouse WMS" || r1.Scenario.ContextUsed.Lead != "Acme Corp" {
		t.Errorf("context_used must confirm project/lead were used: %+v", r1.Scenario.ContextUsed)
	}
	if r1.Scenario.ContextUsed.SalesAngle != r1.SalesAngle {
		t.Errorf("context_used.sales_angle must match the chosen angle: %q vs %q", r1.Scenario.ContextUsed.SalesAngle, r1.SalesAngle)
	}

	// Every generation is persisted separately with its own metadata.
	list, err := scenarioRepo.ListForLead(ctx, lead.ID)
	if err != nil {
		t.Fatalf("list scenarios: %v", err)
	}
	if len(list) != 2 {
		t.Fatalf("expected 2 persisted generations, got %d", len(list))
	}
	if list[0].GenerationID == "" || list[0].SalesAngle == "" {
		t.Errorf("persisted generation missing metadata: %+v", list[0])
	}
	if list[0].GenerationID == list[1].GenerationID {
		t.Error("persisted generations must be distinct records")
	}

	// An explicit sales_angle override must be honored.
	r3, err := svc.CallScenario(ctx, lead.ID, nil, nil, nil, "focus on competitive advantage")
	if err != nil {
		t.Fatalf("overridden generation: %v", err)
	}
	if r3.SalesAngle != "focus on competitive advantage" {
		t.Errorf("sales_angle override not honored, got %q", r3.SalesAngle)
	}
	if r3.Scenario.GenerationID == "" || r3.Scenario.GenerationID == r2.Scenario.GenerationID {
		t.Error("overridden generation must also be unique")
	}
}

// TestProjectIDMustMatchLeadProject verifies that AI generation resolves the
// project from the lead's project_id and never silently uses a different one:
// providing a mismatched project fails, while the correct project generates a
// project-scoped scenario.
func TestProjectIDMustMatchLeadProject(t *testing.T) {
	db := setupDB(t)
	ctx := context.Background()

	projectRepo := projects.NewRepository(db)
	if err := projectRepo.EnsureSchema(ctx); err != nil {
		t.Fatalf("project schema: %v", err)
	}
	leadRepo := leads.NewRepository(db)
	if err := leadRepo.EnsureSchema(ctx); err != nil {
		t.Fatalf("lead schema: %v", err)
	}
	contactRepo := contacts.NewRepository(db)
	noteRepo := notes.NewRepository(db)
	reportRepo := reports.NewRepository(db)
	versionRepo := leadversions.NewRepository(db)
	interRepo := interactions.NewRepository(db)

	projectSvc := projects.NewService(projectRepo)
	gen := &scenarioStubGenerator{}
	svc := messaging.NewContactActionService(interRepo, contactRepo, leadRepo, noteRepo, reportRepo, projectRepo, nil, gen, nil)
	leadSvc := leads.NewService(leadRepo, leads.Deps{Contacts: contactRepo, Versions: versionRepo})

	vpA := "reduce cost by 20%"
	projA, err := projectSvc.Create(ctx, projects.CreateParams{Title: "Proj A", Description: "A", ValueProposition: &vpA})
	if err != nil {
		t.Fatalf("create project A: %v", err)
	}
	vpB := "grow revenue 3x"
	projB, err := projectSvc.Create(ctx, projects.CreateParams{Title: "Proj B", Description: "B", ValueProposition: &vpB})
	if err != nil {
		t.Fatalf("create project B: %v", err)
	}

	lead, err := leadSvc.Create(ctx, leads.CreateParams{CompanyName: "Acme Corp", ProjectID: &projA.ID})
	if err != nil {
		t.Fatalf("create lead: %v", err)
	}

	// An explicit project that is not the lead's project must fail.
	if _, err := svc.CallScenario(ctx, lead.ID, nil, nil, &projB.ID, ""); err == nil {
		t.Fatal("expected error when project_id param does not match the lead's project, got nil")
	}

	// The correct project generates a project-scoped scenario.
	r, err := svc.CallScenario(ctx, lead.ID, nil, nil, &projA.ID, "")
	if err != nil {
		t.Fatalf("scenario for the lead's own project: %v", err)
	}
	if r.ProjectID != projA.ID {
		t.Fatalf("project id not resolved: %q", r.ProjectID)
	}
	if !strings.Contains(r.Scenario.OpeningSentence, "Proj A") {
		t.Errorf("scenario must reference the lead's project: %q", r.Scenario.OpeningSentence)
	}
}
