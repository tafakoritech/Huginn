package messaging

import (
	"context"
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/tafakorpardazan/huginn/internal/llm"
)

// TestScenarioOperationTimeoutWiring verifies the deliberate timeout hierarchy:
// the call-scenario operation budget must be longer than the default LLM
// per-attempt timeout (180s) and shorter than the HTTP server WriteTimeout
// (240s). The outer layer always lives longer than the inner layer.
func TestScenarioOperationTimeoutWiring(t *testing.T) {
	g := NewCallScenarioGenerator(nil, nil, nil)
	if g.opTimeout != 185*time.Second {
		t.Errorf("default operation timeout must be 185s, got %v", g.opTimeout)
	}
	if g.opTimeout <= 180*time.Second {
		t.Error("operation timeout must exceed the LLM per-attempt timeout (180s)")
	}
	if g.opTimeout >= 240*time.Second {
		t.Error("operation timeout must stay below the HTTP server WriteTimeout (240s)")
	}
	withCustom := NewCallScenarioGenerator(nil, nil, nil, WithOperationTimeout(220*time.Second))
	if withCustom.opTimeout != 220*time.Second {
		t.Errorf("WithOperationTimeout must override the default, got %v", withCustom.opTimeout)
	}
}

// TestScenarioGenerationRunsFullBudgetWithoutParentDeadline verifies the
// production guarantee: the Go HTTP server request context has NO wall-clock
// deadline (ReadTimeout/WriteTimeout never cancel the request context, and this
// codebase adds no 30s middleware deadline), so a call-scenario generation runs
// its own full operation budget instead of being killed early. Uses a fake
// runner that takes ~80ms to prove the full budget is granted.
//
// NOTE: the corollary constraint is that NO layer may add a short request
// deadline to /call-scenario — a parent deadline that fires cancels the whole
// operation (see TestScenarioOperationDeadlineDegradesGracefully). The fix for
// the 30s cancellation was to remove such a layer (client 30s read timeout),
// never to add a backend deadline.
func TestScenarioGenerationRunsFullBudgetWithoutParentDeadline(t *testing.T) {
	runner := &slowRunner{payload: validScenarioJSON, delay: 80 * time.Millisecond}
	g := NewCallScenarioGenerator(runner, &SenderProfile{CompanyName: "Co"}, nil)
	leadCtx := &LeadContext{
		LeadID:      "lead-1",
		CompanyName: "Acme",
		Project:     &ProjectContext{ProjectID: "proj-1", Title: "ERP", ValueProposition: "Reduce cost"},
	}

	// Parent carries NO deadline, mirroring the real request context.
	parent := context.Background()
	start := time.Now()
	out, err := g.GenerateScenario(parent, leadCtx, GenerationOptions{GenerationID: "gen-1", Timestamp: time.Now()})
	elapsed := time.Since(start)
	if err != nil {
		t.Fatalf("generation must not fail without a parent deadline: %v", err)
	}
	if elapsed < 80*time.Millisecond {
		t.Fatalf("generation completed too fast (%v); the fake runner was not given its full budget", elapsed)
	}
	if out == nil || out.FallbackUsed || out.GenerationStatus != GenerationStatusGenerated {
		t.Fatalf("long-running generation must still produce a generated scenario, got fallback=%v status=%q", out.FallbackUsed, out.GenerationStatus)
	}
}

// TestScenarioOperationDeadlineDegradesGracefully verifies that when the
// operation's OWN budget expires (not the parent), the generation degrades to
// the explicit fallback instead of hanging or erroring out of the handler.
func TestScenarioOperationDeadlineDegradesGracefully(t *testing.T) {
	runner := &slowRunner{payload: validScenarioJSON, delay: 200 * time.Millisecond}
	g := NewCallScenarioGenerator(runner, &SenderProfile{CompanyName: "Co"}, nil, WithOperationTimeout(50*time.Millisecond))
	leadCtx := &LeadContext{
		LeadID:      "lead-1",
		CompanyName: "Acme",
		Project:     &ProjectContext{ProjectID: "proj-1", Title: "ERP", ValueProposition: "Reduce cost"},
	}
	out, err := g.GenerateScenario(context.Background(), leadCtx, GenerationOptions{GenerationID: "gen-1", Timestamp: time.Now()})
	if err != nil {
		t.Fatalf("op deadline must degrade, not error: %v", err)
	}
	if !out.FallbackUsed || out.GenerationStatus != GenerationStatusDegraded {
		t.Fatalf("op deadline must produce explicit degraded fallback, got fallback=%v status=%q", out.FallbackUsed, out.GenerationStatus)
	}
	if out.GenerationError == "" {
		t.Error("degraded scenario must carry a generation_error")
	}
}

// TestClientCancellationSurfacesAsDegraded verifies intentional cancellation
// (user leaving the screen / network loss) propagates and is surfaced as an
// explicit degraded state — never as a successful generation.
func TestClientCancellationSurfacesAsDegraded(t *testing.T) {
	parent, cancelParent := context.WithCancel(context.Background())
	runner := &slowRunner{payload: validScenarioJSON, delay: 200 * time.Millisecond}
	g := NewCallScenarioGenerator(runner, &SenderProfile{CompanyName: "Co"}, nil)
	leadCtx := &LeadContext{
		LeadID:      "lead-1",
		CompanyName: "Acme",
		Project:     &ProjectContext{ProjectID: "proj-1", Title: "ERP", ValueProposition: "Reduce cost"},
	}
	go func() {
		time.Sleep(30 * time.Millisecond)
		cancelParent()
	}()
	out, err := g.GenerateScenario(parent, leadCtx, GenerationOptions{GenerationID: "gen-1", Timestamp: time.Now()})
	if err != nil {
		t.Fatalf("client cancellation must degrade, not error: %v", err)
	}
	if !out.FallbackUsed || out.GenerationStatus != GenerationStatusDegraded {
		t.Fatalf("client cancellation must be explicit degraded, got fallback=%v status=%q", out.FallbackUsed, out.GenerationStatus)
	}
}

// slowRunner is a stub llm.Runner that delays before returning a fixed payload,
// letting tests exercise timeout and cancellation behavior without an external
// LLM. If the context is cancelled while waiting, it fails like the provider.
type slowRunner struct {
	payload string
	delay   time.Duration
}

func (r *slowRunner) GenerateStructured(ctx context.Context, req llm.Request, out any) (*llm.Run, error) {
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	case <-time.After(r.delay):
	}
	if err := json.Unmarshal([]byte(r.payload), out); err != nil {
		return &llm.Run{Status: "failed"}, err
	}
	return &llm.Run{Status: "success"}, nil
}

func (r *slowRunner) GenerateText(ctx context.Context, req llm.Request) (*llm.Run, error) {
	return nil, nil
}

// TestSparseLeadCoverage verifies that a sparse lead (company + location +
// source only) is correctly classified: business-relevant fields are UNKNOWN,
// and the Render() DATA COVERAGE block lists them as such.
func TestSparseLeadCoverage(t *testing.T) {
	c := &LeadContext{
		CompanyName: "Acme Corp",
		City:        "تهران",
		Source:      "INSTAGRAM",
	}
	known, unknown := c.coverage()
	if !containsStr(known, "Company name") || !containsStr(known, "Location") || !containsStr(known, "Lead source") {
		t.Errorf("sparse lead must mark company/location/source KNOWN, got %v", known)
	}
	for _, want := range []string{"Business category", "Industry", "Business description", "Website", "Instagram", "Previous interactions", "Previous call reports"} {
		if !containsStr(unknown, want) {
			t.Errorf("sparse lead must mark %q UNKNOWN, got %v", want, unknown)
		}
	}
}

// TestSystemPromptDiscoveryFirstAndLeadSourceSemantics verifies the v4 prompt
// encodes (a) discovery-first behavior for sparse leads, (b) the precise
// lead-source semantics ("does NOT mean ..."), and (c) the FACT/HYPOTHESIS/
// UNKNOWN discipline.
func TestSystemPromptDiscoveryFirstAndLeadSourceSemantics(t *testing.T) {
	sys := DefaultCallScenarioSystemPrompt
	for _, want := range []string{
		"EVIDENCE DISCIPLINE",
		"FACT: directly supported",
		"HYPOTHESIS: plausible but not confirmed",
		"UNKNOWN: no reliable evidence",
		"LEAD SOURCE SEMANTICS",
		"discovered, imported, attributed, or captured through that channel",
		"It does NOT mean",
		"DISCOVERY-FIRST QUALIFICATION",
		"Missing information is NOT a failure",
		"known_pains",
		"pain_hypotheses",
		"likely_pain_basis",
		"validation_required",
		"acknowledge → clarify → isolate the objection → discover the real reason",
		"Re-analyze the prospect from scratch",
	} {
		if !strings.Contains(sys, want) {
			t.Errorf("v4 system prompt missing %q", want)
		}
	}
}

// TestSystemPromptForbidsFabricatedSourceClaims verifies the prompt explicitly
// forbids the exact hallucination the mission flagged: turning lead_source into
// a fabricated collaboration history.
func TestSystemPromptForbidsFabricatedSourceClaims(t *testing.T) {
	sys := DefaultCallScenarioSystemPrompt
	if !strings.Contains(sys, "همکاری ما از اینستاگرام شروع شد") {
		t.Error("prompt must explicitly forbid 'همکاری ما از اینستاگرام شروع شد' without evidence")
	}
}

// TestNormalizeStrategyEvidenceDefaults verifies that an unclassified likely
// pain is normalized to a hypothesis (never assumed a fact), validation is
// required, and the pain is mirrored into pain_hypotheses.
func TestNormalizeStrategyEvidenceDefaults(t *testing.T) {
	s := &CallScenario{Strategy: &CallScenarioStrategy{LikelyPain: "احتمالاً هزینه تولید بالاست"}}
	normalizeStrategyEvidence(s)
	if s.Strategy.LikelyPainBasis != "hypothesis" {
		t.Errorf("unclassified likely_pain must default to hypothesis, got %q", s.Strategy.LikelyPainBasis)
	}
	if !s.Strategy.ValidationRequired {
		t.Error("hypothesis pain must require validation")
	}
	if len(s.Strategy.PainHypotheses) != 1 || s.Strategy.PainHypotheses[0] != s.Strategy.LikelyPain {
		t.Errorf("likely pain must be mirrored into pain_hypotheses, got %v", s.Strategy.PainHypotheses)
	}
	// A fact-labeled pain is not forced into hypotheses.
	fact := &CallScenario{Strategy: &CallScenarioStrategy{LikelyPain: "کاهش فروش در گزارش ثبت شده", LikelyPainBasis: "fact"}}
	normalizeStrategyEvidence(fact)
	if fact.Strategy.ValidationRequired {
		t.Error("fact-labeled pain must not require validation")
	}
	if len(fact.Strategy.PainHypotheses) != 0 {
		t.Errorf("fact-labeled pain must not be mirrored into pain_hypotheses, got %v", fact.Strategy.PainHypotheses)
	}
}

// TestRenderReportsIncludesObjectionsAndNextActions verifies previous call
// reports surface objections and next actions so a follow-up call is informed.
func TestRenderReportsIncludesObjectionsAndNextActions(t *testing.T) {
	rep := &ReportFixture{
		outcome:    "INTERESTED",
		objection:  "قیمت بالاست",
		nextAction: "SEND_INFORMATION",
		summary:    "مشتری در جلسه علاقه نشان داد ولی نگران قیمت است",
	}
	rendered := renderReportFixture(rep)
	if !strings.Contains(rendered, "objection=قیمت بالاست") {
		t.Errorf("renderReports must include the previous objection, got %q", rendered)
	}
	if !strings.Contains(rendered, "next_action=SEND_INFORMATION") {
		t.Errorf("renderReports must include the previous next action, got %q", rendered)
	}
	if !strings.Contains(rendered, "summary=") {
		t.Errorf("renderReports must include the summary, got %q", rendered)
	}
}

// TestSchemaStructContractAgreement verifies the canonical v4 output contract:
// every json-serialized field of CallScenario and CallScenarioStrategy exists in
// the JSON schema handed to the provider, and the required top-level fields are
// declared. This is the parser-contract regression test that prevents drift
// between the prompt schema and the Go structs.
func TestSchemaStructContractAgreement(t *testing.T) {
	schema := scenarioJSONSchema()
	props, ok := schema["properties"].(map[string]any)
	if !ok {
		t.Fatal("schema must have a properties object")
	}

	var strategyProp map[string]any
	if sp, ok := props["strategy"].(map[string]any); ok {
		if inner, ok := sp["properties"].(map[string]any); ok {
			strategyProp = inner
		}
	}
	if strategyProp == nil {
		t.Fatal("schema strategy must declare properties")
	}

	// Strategy fields.
	strategyStruct := CallScenarioStrategy{}
	for _, field := range jsonFields(strategyStruct) {
		if _, ok := strategyProp[field]; !ok {
			t.Errorf("strategy json field %q missing from schema", field)
		}
	}
	// Top-level scenario fields (excluding metadata stamped post-generation).
	scenario := CallScenario{}
	for _, field := range jsonFields(scenario) {
		if field == "id" || field == "lead_id" || field == "contact_id" || field == "user_id" ||
			field == "project_id" || field == "sales_angle" || field == "is_generated" ||
			field == "generation_status" || field == "fallback_used" || field == "generation_error" ||
			field == "created_at" || field == "updated_at" || field == "generation_id" {
			continue // metadata, stamped post-generation, not model output
		}
		if _, ok := props[field]; !ok {
			t.Errorf("scenario json field %q missing from schema", field)
		}
	}
	// Required list must cover the core conversation fields.
	requiredSet := map[string]bool{}
	switch req := schema["required"].(type) {
	case []any:
		for _, r := range req {
			if s, ok := r.(string); ok {
				requiredSet[s] = true
			}
		}
	case []string:
		for _, s := range req {
			requiredSet[s] = true
		}
	}
	for _, core := range []string{"opening_sentence", "introduction", "reason_for_call", "value_proposition", "discovery_questions", "likely_objections", "objection_responses", "recommended_next_step", "closing_line", "context_used", "strategy"} {
		if !requiredSet[core] {
			t.Errorf("schema required list must include %q", core)
		}
	}
}

// TestConsistencyValidationFlags verifies the strategy↔scenario consistency
// guardrails produce soft warnings for internally contradictory output.
func TestConsistencyValidationFlags(t *testing.T) {
	base := &CallScenario{
		OpeningSentence:     "سلام، وقت بخیر",
		Introduction:        "من کارشناس فروش هستم",
		ReasonForCall:       "آشنایی",
		ValueProposition:    "بررسی ظرفیت رشد",
		DiscoveryQuestions:  []string{"فعالیت شما چیست؟"},
		LikelyObjections:    []string{"وقت ندارم"},
		ObjectionResponses:  []string{"فقط یک سؤال می‌پرسم"},
		RecommendedNextStep: "یک جلسه کوتاه",
		ClosingLine:         "ممنون",
		Strategy:            &CallScenarioStrategy{PrimaryAngle: "angle", Fit: "uncertain", CallObjective: "qualify the lead"},
	}
	v := validateScenario(base)
	if !v.Usable() {
		t.Fatalf("consistent scenario must be usable, got %v", v.Structural)
	}

	// Uncertain fit + a guaranteed-outcome value proposition → warning.
	pitching := *base
	pitching.ValueProposition = "ما درآمد شما را ۳ برابر می‌کنیم"
	pv := validateScenario(&pitching)
	if !containsStr(pv.Semantic, "uncertain_fit_but_pitching_value") {
		t.Errorf("uncertain fit with guaranteed outcome must warn, got %v", pv.Semantic)
	}

	// Qualification objective + contract next step → warning.
	contract := *base
	contract.Strategy.CallObjective = "qualify and identify decision maker"
	contract.RecommendedNextStep = "ارائه پروپوزال و امضای قرارداد"
	cv := validateScenario(&contract)
	if !containsStr(cv.Semantic, "qualification_objective_but_contract_next_step") {
		t.Errorf("qualification objective with contract next step must warn, got %v", cv.Semantic)
	}

	// Unsupported pain hypothesis stated as fact → warning.
	hypo := *base
	hypo.Strategy.LikelyPainBasis = "hypothesis"
	hypo.Strategy.LikelyPain = "اتکا به روش‌های پراکنده جذب مشتری"
	hv := validateScenario(&hypo)
	if !containsStr(hv.Semantic, "unsupported_pain_hypothesis_stated_as_fact") {
		t.Errorf("unsupported pain hypothesis must warn, got %v", hv.Semantic)
	}
}

// TestSanitizeLocationField verifies the field-mapping sanity check: a job
// title that landed in a location column ("Manager") is treated as UNKNOWN,
// while genuine geography passes through.
func TestSanitizeLocationField(t *testing.T) {
	for _, bad := range []string{"Manager", "مدیر", "Owner", "مسئول فروش", "Sales Manager", "کارشناس بازاریابی"} {
		if got := sanitizeLocationField(bad); got != "" {
			t.Errorf("job-title-like location %q must be blanked, got %q", bad, got)
		}
	}
	for _, good := range []string{"تهران", "Tehran", "اصفهان", "آذربایجان شرقی", "خ کاشانی، پلاک 12"} {
		if got := sanitizeLocationField(good); got != good {
			t.Errorf("geographic location %q must pass through, got %q", good, got)
		}
	}
}

// TestFallbackNeverMasquerades verifies that any fallback path is unmistakable:
// fallback_used=true, generation_status=degraded, is_generated=false.
func TestFallbackNeverMasquerades(t *testing.T) {
	g := NewCallScenarioGenerator(&failingRunner{err: context.DeadlineExceeded}, &SenderProfile{CompanyName: "Co"}, nil)
	leadCtx := &LeadContext{
		LeadID:      "lead-1",
		CompanyName: "Acme",
		Project:     &ProjectContext{ProjectID: "proj-1", Title: "ERP", ValueProposition: "Reduce cost"},
	}
	out, err := g.GenerateScenario(context.Background(), leadCtx, GenerationOptions{GenerationID: "gen-1", Timestamp: time.Now()})
	if err != nil {
		t.Fatalf("fallback must not error: %v", err)
	}
	if !out.FallbackUsed || out.GenerationStatus != GenerationStatusDegraded || out.IsGenerated {
		t.Fatalf("fallback must be unmistakable: fallback=%v status=%q isGenerated=%v", out.FallbackUsed, out.GenerationStatus, out.IsGenerated)
	}
}

// ---- helpers ----

// ReportFixture mirrors the fields used by renderReports so the test does not
// depend on the reports package.
type ReportFixture struct {
	outcome    string
	objection  string
	nextAction string
	summary    string
}

func renderReportFixture(r *ReportFixture) string {
	parts := []string{"outcome=" + r.outcome}
	if r.objection != "" {
		parts = append(parts, "objection="+r.objection)
	}
	if r.nextAction != "" {
		parts = append(parts, "next_action="+r.nextAction)
	}
	if r.summary != "" {
		parts = append(parts, "summary="+r.summary)
	}
	return strings.Join(parts, " | ")
}

// jsonFields returns the JSON field names (json tags) of a struct.
func jsonFields(v any) []string {
	b, err := json.Marshal(v)
	if err != nil {
		return nil
	}
	var m map[string]any
	if err := json.Unmarshal(b, &m); err != nil {
		return nil
	}
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	return out
}
