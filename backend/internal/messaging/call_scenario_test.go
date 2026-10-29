package messaging

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/tafakorpardazan/huginn/internal/llm"
)

func TestFallbackScenarioUsesRealCompany(t *testing.T) {
	ctx := &LeadContext{CompanyName: "کارخانه صدف", DisplayName: "صدف"}
	s := fallbackScenario(ctx, &SenderProfile{CompanyName: "پخش پارس"})
	if !strings.Contains(s.OpeningSentence, "کارخانه صدف") {
		t.Errorf("fallback opening must reference the real company: %q", s.OpeningSentence)
	}
	if len(s.DiscoveryQuestions) < 2 || len(s.LikelyObjections) < 2 || len(s.ObjectionResponses) != len(s.LikelyObjections) {
		t.Errorf("fallback must have 2+ questions/objections with matching responses")
	}
	if s.ClosingLine == "" || s.RecommendedNextStep == "" {
		t.Error("fallback must include a next step and closing line")
	}
}

func TestFallbackScenarioNoCompany(t *testing.T) {
	s := fallbackScenario(&LeadContext{}, nil)
	if !strings.Contains(s.OpeningSentence, "شما") {
		t.Errorf("fallback must not reference an empty company: %q", s.OpeningSentence)
	}
}

func TestValidateScenario(t *testing.T) {
	good := &CallScenario{
		OpeningSentence:     "سلام، وقتتون بخیر؛ بابت اینستاگرام تماس گرفتم",
		Introduction:        "من مشاور فروش هستم",
		ReasonForCall:       "چون از اینستاگرام آشنا شدید",
		ValueProposition:    "به رشد درآمد کمک می‌کنیم",
		DiscoveryQuestions:  []string{"کسب‌وکار شما چیست؟", "مشتری‌ها از کجا می‌آیند؟"},
		LikelyObjections:    []string{"وقت ندارم"},
		ObjectionResponses:  []string{"دو سؤال کوتاه می‌پرسم"},
		RecommendedNextStep: "هماهنگی جلسه کوتاه",
		ClosingLine:         "ممنون، روز خوبی داشته باشید",
		ContextUsed:         ContextUsed{Project: "p", Lead: "l", SalesAngle: "a"},
		Strategy:            &CallScenarioStrategy{PrimaryAngle: "angle", CallObjective: "qualify"},
	}
	val := validateScenario(good)
	if len(val.Structural) != 0 {
		t.Errorf("complete scenario must be structurally usable, got %v", val.Structural)
	}
	if val := validateScenario(nil); val.Usable() {
		t.Error("nil scenario must be structurally rejected")
	}
	empty := validateScenario(&CallScenario{})
	if len(empty.Structural) == 0 {
		t.Error("empty scenario must be structurally rejected with reasons")
	}
	missingCore := validateScenario(&CallScenario{OpeningSentence: "x", Introduction: "x", ReasonForCall: "x"})
	for _, want := range []string{"value_proposition_empty", "discovery_questions_empty", "recommended_next_step_empty"} {
		if !containsStr(missingCore.Structural, want) {
			t.Errorf("missing core scenario must report %q, got %v", want, missingCore.Structural)
		}
	}
	// Objection responses must align with objections.
	misaligned := *good
	misaligned.ObjectionResponses = []string{"r1", "r2"}
	mv := validateScenario(&misaligned)
	if len(mv.Structural) == 0 || !containsStr(mv.Structural, "objection_responses_not_aligned") {
		t.Errorf("mismatched objections/responses must be rejected with a reason, got %v", mv.Structural)
	}
	// Missing strategy block is a structural rejection (synthesized repair is
	// only applied by normalizeScenario, not by raw validation).
	noStrategy := *good
	noStrategy.Strategy = nil
	sv := validateScenario(&noStrategy)
	if !containsStr(sv.Structural, "strategy_missing") {
		t.Errorf("missing strategy must be reported, got %v", sv.Structural)
	}
}

func TestNormalizeScenarioRepairsContextUsedAndStrategy(t *testing.T) {
	// The exact production shape before the fix: valid 9-field output with no
	// strategy and no context_used (produced by the legacy stored prompt).
	s := &CallScenario{
		OpeningSentence:     "سلام، وقتتون بخیر، اگه یک دقیقه وقت داشته باشید",
		Introduction:        "من یک مشاور فروش هستم",
		ReasonForCall:       "اطلاعات شما رو از طریق اینستاگرام دریافت کردیم",
		ValueProposition:    "کمک به رشد درآمد",
		DiscoveryQuestions:  []string{"کسب‌وکار شما چیست؟", "الان چطور مشتری جذب می‌کنید؟"},
		LikelyObjections:    []string{"الان وقت ندارم"},
		ObjectionResponses:  []string{"فقط دو سؤال کوتاه می‌پرسم"},
		RecommendedNextStep: "یک جلسه کوتاه",
		ClosingLine:         "ممنون، موفق باشید",
	}
	leadCtx := &LeadContext{LeadID: "l1", CompanyName: "Acme", Project: &ProjectContext{Title: "ERP", ValueProposition: "grow revenue 3x"}}
	val := normalizeScenario(s, leadCtx, GenerationOptions{})
	if !val.Usable() {
		t.Fatalf("legacy 9-field output must be repaired and usable, structural=%v", val.Structural)
	}
	if !containsStr(val.Semantic, "context_used_omitted_by_model") {
		t.Errorf("omitted context_used must be recorded as a semantic flag, got %v", val.Semantic)
	}
	if s.ContextUsed.Project != "ERP" || s.ContextUsed.Lead != "Acme" {
		t.Errorf("context_used must be repaired from real data, got %+v", s.ContextUsed)
	}
	if s.Strategy == nil || s.Strategy.PrimaryAngle != s.ValueProposition {
		t.Errorf("strategy must be synthesized from content, got %+v", s.Strategy)
	}
}

func TestValidateScenarioSemanticFlagsAreNonBlocking(t *testing.T) {
	s := &CallScenario{
		OpeningSentence:     "سلام، وقتتون بخیر؛ بابت اینستاگرام تماس گرفتم",
		Introduction:        "من مشاور فروش هستم",
		ReasonForCall:       "چون از اینستاگرام آشنا شدید",
		ValueProposition:    "به رشد درآمد کمک می‌کنیم",
		DiscoveryQuestions:  []string{"کسب‌وکار شما چیست؟"},
		LikelyObjections:    []string{"وقت ندارم"},
		ObjectionResponses:  []string{"دو سؤال کوتاه می‌پرسم"},
		RecommendedNextStep: "هماهنگی جلسه کوتاه",
		ClosingLine:         "ممنون",
		ContextUsed:         ContextUsed{Project: "p", Lead: "l"},
		Strategy:            &CallScenarioStrategy{PrimaryAngle: "a", CallObjective: "q"},
	}
	val := validateScenario(s)
	if !val.Usable() {
		t.Fatalf("scenario with only semantic issues must remain usable, got %v", val.Structural)
	}
	// One discovery question + very short closing line are quality flags, not
	// rejections.
	if !containsStr(val.Semantic, "discovery_questions_too_few") {
		t.Errorf("expected discovery_questions_too_few semantic flag, got %v", val.Semantic)
	}
}

func TestTemplatePlaceholderDetection(t *testing.T) {
	if !placeholderIn("سلام [اسم] هستم", "x") {
		t.Error("square-bracket placeholder must be detected")
	}
	if !placeholderIn("", "", "نام شرکت شما چیست؟", "") {
		t.Error("generic company-name stub must be detected")
	}
	if placeholderIn("یک متن عادی فارسی", "سلام") {
		t.Error("clean text must not be flagged")
	}
}

func TestEnsureStrategySynthesizesWhenMissing(t *testing.T) {
	s := &CallScenario{
		OpeningSentence: "o", Introduction: "i", ReasonForCall: "r",
		ValueProposition:    "vp",
		DiscoveryQuestions:  []string{"q"},
		LikelyObjections:    []string{"o1"},
		ObjectionResponses:  []string{"r1"},
		RecommendedNextStep: "next",
		ClosingLine:         "c",
		ContextUsed:         ContextUsed{Project: "p", Lead: "l"},
	}
	fromModel := ensureStrategy(s, &LeadContext{CompanyName: "Acme"}, GenerationOptions{})
	if fromModel {
		t.Error("missing strategy must be reported as synthesized")
	}
	if s.Strategy == nil || s.Strategy.PrimaryAngle != "vp" {
		t.Errorf("synthesized strategy must derive primary_angle from the value proposition, got %+v", s.Strategy)
	}
	// A model-provided strategy is preserved.
	modelProvided := &CallScenario{Strategy: &CallScenarioStrategy{PrimaryAngle: "chosen"}}
	if !ensureStrategy(modelProvided, nil, GenerationOptions{}) {
		t.Error("model-provided strategy must be preserved")
	}
	if modelProvided.Strategy.PrimaryAngle != "chosen" {
		t.Error("model strategy must not be overwritten")
	}
}

func TestMessageCharCap(t *testing.T) {
	long := strings.Repeat("م", 500)
	trimmed := trimRunes(long, 100)
	if runeCount(trimmed) != 100 {
		t.Fatalf("expected 100 runes, got %d", runeCount(trimmed))
	}
	if trimRunes("کوتاه", 100) != "کوتاه" {
		t.Error("short text must not be trimmed")
	}
}

func TestLeadContextRenderSkipsEmpty(t *testing.T) {
	ctx := &LeadContext{CompanyName: "شرکت الف", City: "تهران"}
	out := ctx.Render()
	if !strings.Contains(out, "شرکت الف") || !strings.Contains(out, "تهران") {
		t.Errorf("render must include present fields: %q", out)
	}
	if strings.Contains(out, "Products / services") {
		t.Errorf("render must skip empty fields: %q", out)
	}
}

func TestGenerateScenarioRequiresProject(t *testing.T) {
	g := NewCallScenarioGenerator(nil, nil, nil)
	ctx := context.Background()
	if _, err := g.GenerateScenario(ctx, &LeadContext{LeadID: "l1", CompanyName: "Acme"}, GenerationOptions{}); !errors.Is(err, ErrNoProject) {
		t.Fatalf("expected ErrNoProject without project context, got %v", err)
	}
	if _, err := g.GenerateScenario(ctx, &LeadContext{LeadID: "l1", CompanyName: "Acme", Project: &ProjectContext{}}, GenerationOptions{}); !errors.Is(err, ErrNoProject) {
		t.Fatalf("expected ErrNoProject for empty project, got %v", err)
	}
	if _, err := g.GenerateScenario(ctx, nil, GenerationOptions{}); !errors.Is(err, ErrNoProject) {
		t.Fatalf("expected ErrNoProject for nil lead context, got %v", err)
	}
}

func TestBuildScenarioPromptIncludesUniqueEnvelopeAndContext(t *testing.T) {
	opts := GenerationOptions{
		GenerationID:  "gen-1",
		LeadID:        "lead-1",
		ProjectID:     "proj-1",
		Timestamp:     time.Date(2026, 8, 23, 10, 0, 0, 0, time.UTC),
		SalesAngle:    "angle-x",
		PreviousAngle: "",
	}
	leadCtx := &LeadContext{
		LeadID:          "lead-1",
		CompanyName:     "کارخانه صدف",
		ContactName:     "امیر رضایی",
		ContactPosition: "مدیر فروش",
		Project: &ProjectContext{
			ProjectID:          "proj-1",
			Title:              "Enterprise ERP",
			Description:        "ERP for manufacturing",
			ProductInformation: "ERP software",
			ValueProposition:   "Reduce cost 20%",
		},
	}
	sender := &SenderProfile{CompanyName: "پخش پارس"}
	prompt := buildScenarioPrompt(sender, leadCtx, opts)
	for _, want := range []string{
		"gen-1", "proj-1", "angle-x", "GENERATION REQUEST",
		"Enterprise ERP", "Reduce cost 20%",
		"کارخانه صدف", "امیر رضایی", "مدیر فروش",
		"PROJECT CONTEXT", "LEAD INFORMATION", "DATA COVERAGE",
	} {
		if !strings.Contains(prompt, want) {
			t.Errorf("scenario prompt missing %q", want)
		}
	}
	if strings.Contains(prompt, "PREVIOUS GENERATION") {
		t.Error("no PREVIOUS GENERATION block expected without a previous angle")
	}
}

func TestBuildScenarioPromptIncludesPreviousAngleForGenerateAgain(t *testing.T) {
	opts := GenerationOptions{
		GenerationID:  "gen-2",
		LeadID:        "lead-1",
		ProjectID:     "proj-1",
		Timestamp:     time.Now(),
		PreviousAngle: "content automation",
	}
	leadCtx := &LeadContext{CompanyName: "Acme", Project: &ProjectContext{ProjectID: "proj-1", Title: "ERP"}}
	prompt := buildScenarioPrompt(nil, leadCtx, opts)
	if !strings.Contains(prompt, "PREVIOUS GENERATION") || !strings.Contains(prompt, "content automation") {
		t.Errorf("prompt must carry the previous angle for Generate Again, got: %q", prompt)
	}
	if !strings.Contains(prompt, "NEVER merely reword the previous output") {
		t.Error("prompt must instruct reasoning-based variation, not rewording")
	}
}

func TestSalesAngleFocusesAreStableVocabulary(t *testing.T) {
	if len(SalesAngleFocuses) == 0 {
		t.Fatal("focus vocabulary must not be empty")
	}
	// The vocabulary is a client hint list, NOT an auto-rotation: no function
	// should force one per generation. Verify it remains a plain list of hints.
	if len(SalesAngleFocuses) != 5 {
		t.Errorf("expected 5 focus hints, got %d", len(SalesAngleFocuses))
	}
}

func TestFallbackScenarioUsesProjectContext(t *testing.T) {
	ctx := &LeadContext{
		CompanyName: "کارخانه صدف",
		Project: &ProjectContext{
			Title:            "ERP",
			Description:      "cloud-native ERP with AI forecasting",
			ValueProposition: "cut operational cost by 30% using AI",
		},
	}
	s := fallbackScenario(ctx, &SenderProfile{CompanyName: "پخش پارس"})
	if !strings.Contains(s.Introduction, "ERP") {
		t.Errorf("fallback introduction must reference the project title: %q", s.Introduction)
	}
	if !strings.Contains(s.ReasonForCall, "cloud-native ERP") {
		t.Errorf("fallback reason must reference the project description: %q", s.ReasonForCall)
	}
	if !strings.Contains(s.ValueProposition, "cut operational cost by 30%") {
		t.Errorf("fallback value must reference the project value proposition: %q", s.ValueProposition)
	}
	if s.ContextUsed.Project != "ERP" {
		t.Errorf("fallback context_used.project must be the project title, got %q", s.ContextUsed.Project)
	}
}

func TestApplyGenerationMetaFillsContextUsed(t *testing.T) {
	s := &CallScenario{}
	leadCtx := &LeadContext{LeadID: "l1", CompanyName: "Acme", Project: &ProjectContext{Title: "ERP"}}
	opts := GenerationOptions{GenerationID: "g1", SalesAngle: "angle", Timestamp: time.Now()}
	applyGenerationMeta(s, leadCtx, opts)
	if s.GenerationID != "g1" || s.SalesAngle != "angle" || s.LeadID != "l1" {
		t.Error("generation metadata not applied")
	}
	if s.ContextUsed.Project != "ERP" || s.ContextUsed.Lead != "Acme" || s.ContextUsed.SalesAngle != "angle" {
		t.Errorf("context_used not filled from real data: %+v", s.ContextUsed)
	}
}

// captureRunner is a stub llm.Runner that records the request and returns a
// fixed structured payload, so the real CallScenarioGenerator can be tested
// without an external LLM.
type captureRunner struct {
	lastReq llm.Request
	payload string
}

func (r *captureRunner) GenerateStructured(ctx context.Context, req llm.Request, out any) (*llm.Run, error) {
	r.lastReq = req
	if err := json.Unmarshal([]byte(r.payload), out); err != nil {
		return &llm.Run{Status: "failed"}, err
	}
	return &llm.Run{Status: "success"}, nil
}

func (r *captureRunner) GenerateText(ctx context.Context, req llm.Request) (*llm.Run, error) {
	return nil, nil
}

const validScenarioJSON = `{
	"strategy": {"lead_summary": "l", "fit": "high", "fit_summary": "f", "primary_angle": "angle",
		"secondary_angle": "", "likely_pain": "p", "uncertainties": [], "call_objective": "qualify", "tone": "t"},
	"opening_sentence": "o", "introduction": "i", "reason_for_call": "r",
	"value_proposition": "v", "discovery_questions": ["q"],
	"discovery_question_purposes": [{"question": "q", "purpose": "why"}],
	"likely_objections": ["o1"], "objection_responses": ["r1"],
	"conversation_branches": [{"condition": "c", "response_strategy": "s"}],
	"recommended_next_step": "n", "closing_line": "c",
	"context_used": {"project": "ERP", "lead": "Acme", "sales_angle": "angle"}
}`

// TestCallScenarioGeneratorSendsProjectContextAndStampsMeta verifies the
// generator's LLM request carries the PROJECT CONTEXT, LEAD INFORMATION, DATA
// COVERAGE, and the unique GENERATION REQUEST envelope, and that the returned
// scenario is stamped with the generation metadata.
func TestCallScenarioGeneratorSendsProjectContextAndStampsMeta(t *testing.T) {
	runner := &captureRunner{payload: validScenarioJSON}
	g := NewCallScenarioGenerator(runner, &SenderProfile{CompanyName: "Co"}, nil)
	leadCtx := &LeadContext{
		LeadID:      "lead-1",
		CompanyName: "Acme",
		Project:     &ProjectContext{ProjectID: "proj-1", Title: "ERP", ValueProposition: "Reduce cost"},
	}
	opts := GenerationOptions{GenerationID: "gen-1", LeadID: "lead-1", ProjectID: "proj-1", Timestamp: time.Now(), SalesAngle: "angle"}

	out, err := g.GenerateScenario(context.Background(), leadCtx, opts)
	if err != nil {
		t.Fatalf("generate: %v", err)
	}
	prompt := runner.lastReq.UserPrompt
	for _, want := range []string{"ERP", "Reduce cost", "Acme", "gen-1", "proj-1", "angle", "GENERATION REQUEST", "PROJECT CONTEXT", "LEAD INFORMATION", "DATA COVERAGE"} {
		if !strings.Contains(prompt, want) {
			t.Errorf("user prompt missing %q", want)
		}
	}
	if runner.lastReq.EntityID != "lead-1" || runner.lastReq.Meta["project_id"] != "proj-1" {
		t.Errorf("request meta missing lead/project ids: entity=%q meta=%v", runner.lastReq.EntityID, runner.lastReq.Meta)
	}
	if runner.lastReq.JSONSchema == nil {
		t.Fatal("structured output schema must be supplied")
	}
	if out.GenerationID != "gen-1" || out.LeadID != "lead-1" {
		t.Errorf("metadata not stamped: gen=%q lead=%q", out.GenerationID, out.LeadID)
	}
	if out.Strategy == nil || out.Strategy.PrimaryAngle != "angle" {
		t.Errorf("model strategy must be preserved: %+v", out.Strategy)
	}
	if len(out.DiscoveryPurposes) != 1 || out.DiscoveryPurposes[0].Purpose != "why" {
		t.Errorf("discovery question purposes must be preserved: %+v", out.DiscoveryPurposes)
	}
	if len(out.ConversationBranches) != 1 || out.ConversationBranches[0].Condition != "c" {
		t.Errorf("conversation branches must be preserved: %+v", out.ConversationBranches)
	}
	if out.ContextUsed.Project != "ERP" || out.ContextUsed.Lead != "Acme" || out.ContextUsed.SalesAngle != "angle" {
		t.Errorf("context_used not preserved: %+v", out.ContextUsed)
	}
}

// TestCallScenarioGeneratorSynthesizesStrategyWhenLLMOmitsIt verifies that a
// complete scenario without a strategy block is still returned (not replaced
// by the deterministic fallback) and that a minimal strategy is synthesized.
func TestCallScenarioGeneratorSynthesizesStrategyWhenLLMOmitsIt(t *testing.T) {
	payload := `{
		"opening_sentence": "o", "introduction": "i", "reason_for_call": "r",
		"value_proposition": "v", "discovery_questions": ["q"], "likely_objections": ["o1"],
		"objection_responses": ["r1"], "recommended_next_step": "n", "closing_line": "c",
		"context_used": {"project": "ERP", "lead": "Acme", "sales_angle": ""}
	}`
	runner := &captureRunner{payload: payload}
	g := NewCallScenarioGenerator(runner, &SenderProfile{CompanyName: "Co"}, nil)
	leadCtx := &LeadContext{
		LeadID:      "lead-1",
		CompanyName: "Acme",
		Project:     &ProjectContext{ProjectID: "proj-1", Title: "ERP", ValueProposition: "Reduce cost"},
	}
	opts := GenerationOptions{GenerationID: "gen-1", Timestamp: time.Now()}

	out, err := g.GenerateScenario(context.Background(), leadCtx, opts)
	if err != nil {
		t.Fatalf("generate: %v", err)
	}
	if out.OpeningSentence != "o" {
		t.Fatalf("LLM output must be preserved (not fallback), got opening %q", out.OpeningSentence)
	}
	if out.Strategy == nil || out.Strategy.PrimaryAngle == "" {
		t.Fatal("strategy must be synthesized when the model omits it")
	}
	// Effective sales angle falls back to the synthesized strategy's angle.
	if out.SalesAngle != out.Strategy.PrimaryAngle {
		t.Errorf("sales angle must reflect the chosen strategy angle: %q vs %q", out.SalesAngle, out.Strategy.PrimaryAngle)
	}
}

// TestCallScenarioGeneratorRepairsMissingContextUsed verifies that a complete
// scenario whose model omitted context_used is REPAIRED from real data and
// preserved (never replaced by the deterministic fallback). Cosmetic omissions
// must not discard good LLM output.
func TestCallScenarioGeneratorRepairsMissingContextUsed(t *testing.T) {
	payload := `{
		"opening_sentence": "سلام، وقتتون بخیر؛ بابت اینستاگرام تماس گرفتم",
		"introduction": "من مشاور فروش هستم",
		"reason_for_call": "چون از اینستاگرام آشنا شدید",
		"value_proposition": "به رشد درآمد کمک می‌کنیم",
		"discovery_questions": ["کسب‌وکار شما چیست؟", "مشتری‌ها از کجا می‌آیند؟"],
		"likely_objections": ["وقت ندارم"],
		"objection_responses": ["دو سؤال کوتاه می‌پرسم"],
		"recommended_next_step": "هماهنگی جلسه کوتاه",
		"closing_line": "ممنون، روز خوبی داشته باشید"
	}`
	runner := &captureRunner{payload: payload}
	g := NewCallScenarioGenerator(runner, &SenderProfile{CompanyName: "Co"}, nil)
	leadCtx := &LeadContext{
		LeadID:      "lead-1",
		CompanyName: "Acme",
		Project:     &ProjectContext{ProjectID: "proj-1", Title: "ERP", ValueProposition: "Reduce cost"},
	}
	opts := GenerationOptions{GenerationID: "gen-1", SalesAngle: "angle", Timestamp: time.Now()}

	out, err := g.GenerateScenario(context.Background(), leadCtx, opts)
	if err != nil {
		t.Fatalf("generate: %v", err)
	}
	// The LLM output must be preserved, not replaced by fallback.
	if !strings.Contains(out.Introduction, "مشاور فروش") {
		t.Errorf("LLM output must be preserved, got intro %q", out.Introduction)
	}
	// context_used is repaired from real project/lead data.
	if out.ContextUsed.Project != "ERP" || out.ContextUsed.Lead != "Acme" {
		t.Errorf("context_used must be repaired from real data: %+v", out.ContextUsed)
	}
	// The repaired scenario is a real generation, not degraded fallback.
	if out.FallbackUsed {
		t.Error("repaired LLM output must not be marked as fallback")
	}
	if out.GenerationStatus != GenerationStatusGenerated {
		t.Errorf("expected generation status %q, got %q", GenerationStatusGenerated, out.GenerationStatus)
	}
	if !out.IsGenerated {
		t.Error("repaired LLM output must be persisted as generated")
	}
	if out.Strategy == nil || out.Strategy.PrimaryAngle == "" {
		t.Error("strategy must be synthesized when the model omits it")
	}
}

// TestCallScenarioGeneratorDegradedOnRejection verifies that when the LLM's
// output fails structural validation the returned scenario is explicitly marked
// as the deterministic fallback — it can never be mistaken for a personalized
// generation.
func TestCallScenarioGeneratorDegradedOnRejection(t *testing.T) {
	// Structurally unusable: no value proposition, no questions, no strategy
	// content to synthesize from.
	payload := `{"opening_sentence": "o", "introduction": "i", "reason_for_call": "r"}`
	runner := &captureRunner{payload: payload}
	g := NewCallScenarioGenerator(runner, &SenderProfile{CompanyName: "Co"}, nil)
	leadCtx := &LeadContext{
		LeadID:      "lead-1",
		CompanyName: "Acme",
		Project:     &ProjectContext{ProjectID: "proj-1", Title: "ERP", ValueProposition: "Reduce cost"},
	}
	opts := GenerationOptions{GenerationID: "gen-1", Timestamp: time.Now()}

	out, err := g.GenerateScenario(context.Background(), leadCtx, opts)
	if err != nil {
		t.Fatalf("generate must fall back, not error: %v", err)
	}
	if !out.FallbackUsed {
		t.Error("structurally rejected output must be replaced by fallback with fallback_used=true")
	}
	if out.GenerationStatus != GenerationStatusDegraded {
		t.Errorf("expected generation status %q, got %q", GenerationStatusDegraded, out.GenerationStatus)
	}
	if out.IsGenerated {
		t.Error("fallback must not be persisted as generated")
	}
	if out.GenerationError == "" {
		t.Error("degraded scenario must carry a generation_error reason")
	}
	if !strings.Contains(out.GenerationError, "value_proposition_empty") {
		t.Errorf("generation_error must include actionable reasons, got %q", out.GenerationError)
	}
}

// TestCallScenarioGeneratorProviderFailureDegrades verifies the provider-failure
// path (HTTP error from the LLM) also produces an explicitly degraded scenario.
func TestCallScenarioGeneratorProviderFailureDegrades(t *testing.T) {
	g := NewCallScenarioGenerator(&failingRunner{err: errors.New("llm provider error for call.scenario: upstream status 429")}, &SenderProfile{CompanyName: "Co"}, nil)
	leadCtx := &LeadContext{
		LeadID:      "lead-1",
		CompanyName: "Acme",
		Project:     &ProjectContext{ProjectID: "proj-1", Title: "ERP", ValueProposition: "Reduce cost"},
	}
	opts := GenerationOptions{GenerationID: "gen-1", Timestamp: time.Now()}

	out, err := g.GenerateScenario(context.Background(), leadCtx, opts)
	if err != nil {
		t.Fatalf("generate must fall back, not error: %v", err)
	}
	if !out.FallbackUsed || out.GenerationStatus != GenerationStatusDegraded {
		t.Errorf("provider failure must be explicit: fallback=%v status=%q", out.FallbackUsed, out.GenerationStatus)
	}
	if out.GenerationError == "" {
		t.Error("degraded scenario must carry a generation_error")
	}
}

// failingRunner is a stub llm.Runner whose GenerateStructured always errors.
type failingRunner struct {
	err error
}

func (r *failingRunner) GenerateStructured(ctx context.Context, req llm.Request, out any) (*llm.Run, error) {
	return nil, r.err
}

func (r *failingRunner) GenerateText(ctx context.Context, req llm.Request) (*llm.Run, error) {
	return nil, r.err
}

func TestFallbackScenarioIncludesStrategy(t *testing.T) {
	s := fallbackScenario(&LeadContext{
		CompanyName: "کارخانه صدف",
		Project:     &ProjectContext{Title: "ERP", ValueProposition: "cut cost"},
	}, nil)
	if s.Strategy == nil {
		t.Fatal("fallback scenario must include a strategy block")
	}
	if s.Strategy.PrimaryAngle == "" || s.Strategy.CallObjective == "" {
		t.Errorf("fallback strategy must be populated: %+v", s.Strategy)
	}
	if len(s.DiscoveryPurposes) == 0 {
		t.Error("fallback must include discovery question purposes")
	}
	if s.ContextUsed.Project != "ERP" {
		t.Errorf("fallback context_used.project must be the project title, got %q", s.ContextUsed.Project)
	}
}

// TestPersonalizationDifferentLeadsDifferentStrategies verifies the end-to-end
// personalization contract: materially different leads under the SAME project
// yield strategies that differ for substantive reasons (angle, pain, objective,
// discovery focus) — not merely by substituting the company name.
func TestPersonalizationDifferentLeadsDifferentStrategies(t *testing.T) {
	project := &ProjectContext{
		ProjectID:        "proj-1",
		Title:            "Growth Consulting",
		ValueProposition: "sales growth via paid channels and conversion",
	}

	restaurant := &LeadContext{
		LeadID:           "lead-restaurant",
		CompanyName:      "رستوران آسمان",
		Industry:         "رستوران",
		City:             "تهران",
		ContactName:      "علی",
		PastInteractions: "اولین تماس",
		Project:          project,
	}
	logistics := &LeadContext{
		LeadID:      "lead-logistics",
		CompanyName: "شرکت حمل‌ونقل تیبا",
		Industry:    "لجستیک",
		City:        "اصفهان",
		ContactName: "مریم",
		Project:     project,
	}

	// Simulate the model reasoning differently for each lead: restaurant gets a
	// table-occupancy/repeat-dining angle; logistics gets a fleet/fuel-cost
	// angle. The prompts delivered to the model differ in lead facts, and the
	// strategies differ substantively.
	runnerA := &captureRunner{payload: restaurantPayloadJSON()}
	runnerB := &captureRunner{payload: logisticsPayloadJSON()}
	gA := NewCallScenarioGenerator(runnerA, &SenderProfile{CompanyName: "Co"}, nil)
	gB := NewCallScenarioGenerator(runnerB, &SenderProfile{CompanyName: "Co"}, nil)

	outA, err := gA.GenerateScenario(context.Background(), restaurant, GenerationOptions{GenerationID: "g-a", Timestamp: time.Now()})
	if err != nil {
		t.Fatalf("generate A: %v", err)
	}
	outB, err := gB.GenerateScenario(context.Background(), logistics, GenerationOptions{GenerationID: "g-b", Timestamp: time.Now()})
	if err != nil {
		t.Fatalf("generate B: %v", err)
	}

	// Substantive difference in strategy, not company-name substitution.
	if outA.Strategy == nil || outB.Strategy == nil {
		t.Fatal("both strategies must be present")
	}
	if outA.Strategy.PrimaryAngle == outB.Strategy.PrimaryAngle {
		t.Error("primary angles must differ for materially different leads")
	}
	if outA.Strategy.LikelyPain == outB.Strategy.LikelyPain {
		t.Error("likely pain must differ for materially different leads")
	}
	if outA.Strategy.CallObjective == outB.Strategy.CallObjective {
		t.Error("call objectives must differ for materially different leads")
	}
	// Discovery questions must target the lead's own context.
	if !strings.Contains(strings.Join(outA.DiscoveryQuestions, " "), "رستوران") {
		t.Error("restaurant discovery questions must reference restaurant context")
	}
	if !strings.Contains(strings.Join(outB.DiscoveryQuestions, " "), "ناوگان") {
		t.Error("logistics discovery questions must reference logistics context")
	}
	// Both prompts carried the shared project context.
	for _, want := range []string{"Growth Consulting", "sales growth"} {
		if !strings.Contains(runnerA.lastReq.UserPrompt, want) || !strings.Contains(runnerB.lastReq.UserPrompt, want) {
			t.Errorf("prompt must carry shared project context %q", want)
		}
	}
}

func restaurantPayloadJSON() string {
	return `{
		"strategy": {"lead_summary": "رستوران آسمان در تهران", "fit": "high", "fit_summary": "رستوران با ظرفیت رشد در ساعات پیک",
			"primary_angle": "افزایش سهم مشتری‌های تکراری و پر کردن ساعات کم‌تردد رستوران",
			"secondary_angle": "بهبود رزرو آنلاین", "likely_pain": "شب‌های خلوت و نرخ بازگشت پایین مشتری",
			"uncertainties": ["میزان سود هر سفارش"], "call_objective": "درک ترافیک روزانه رستوران و هماهنگی جلسه", "tone": "صمیمی"},
		"opening_sentence": "سلام، وقتتون بخیر؛ بابت رستوران آسمان در تهران تماس گرفتم",
		"introduction": "من مشاور رشد فروش هستم",
		"reason_for_call": "کمک به پر کردن ساعات خلوت رستوران شما",
		"value_proposition": "بهبود رزرو و بازگشت مشتری",
		"discovery_questions": ["ساعات خلوت رستوران شما چه وقتی است؟", "نرخ بازگشت مشتری الان چقدر است؟"],
		"discovery_question_purposes": [{"question": "ساعات خلوت رستوران شما چه وقتی است؟", "purpose": "شناخت ساعات پیک و خلوت"}],
		"likely_objections": ["الان وقت ندارم"], "objection_responses": ["فقط یک سؤال درباره ساعات کاری می‌پرسم"],
		"conversation_branches": [{"condition": "رستوران شلوغ است", "response_strategy": "روی رزرو آنلاین تمرکز کن"}],
		"recommended_next_step": "یک جلسه کوتاه برای بررسی رزرو آنلاین",
		"closing_line": "ممنون، روز خوبی داشته باشید",
		"context_used": {"project": "Growth Consulting", "lead": "رستوران آسمان", "sales_angle": "افزایش مشتری تکراری"}
	}`
}

func logisticsPayloadJSON() string {
	return `{
		"strategy": {"lead_summary": "شرکت حمل‌ونقل تیبا در اصفهان", "fit": "high", "fit_summary": "لجستیک با هزینه سوخت بالا",
			"primary_angle": "کاهش هزینه سوخت و افزایش بهره‌وری ناوگان",
			"secondary_angle": "بهبود مسیریابی", "likely_pain": "هزینه بالای سوخت و ناوگان نیمه‌بیکار",
			"uncertainties": ["تعداد ناوگان"], "call_objective": "بررسی هزینه سوخت ناوگان و هماهنگی جلسه", "tone": "رسمی"},
		"opening_sentence": "سلام، وقتتون بخیر؛ بابت ناوگان حمل‌ونقل تیبا تماس گرفتم",
		"introduction": "من مشاور بهره‌وری هستم",
		"reason_for_call": "کمک به کاهش هزینه سوخت ناوگان شما",
		"value_proposition": "بهبود بهره‌وری ناوگان",
		"discovery_questions": ["هزینه سوخت ماهانه ناوگان شما چقدر است؟", "چند کامیون در ناوگان دارید؟"],
		"discovery_question_purposes": [{"question": "هزینه سوخت ماهانه ناوگان شما چقدر است؟", "purpose": "برآورد پایه هزینه"}],
		"likely_objections": ["با تأمین‌کننده فعلی کار می‌کنیم"], "objection_responses": ["اگر روزی مقایسه خواستید در دسترس هستم"],
		"conversation_branches": [{"condition": "ناوگان کوچک است", "response_strategy": "روی بهینه‌سازی مسیر تمرکز کن"}],
		"recommended_next_step": "جلسه برای بررسی مصرف سوخت",
		"closing_line": "ممنون، موفق باشید",
		"context_used": {"project": "Growth Consulting", "lead": "شرکت حمل‌ونقل تیبا", "sales_angle": "کاهش هزینه سوخت"}
	}`
}

func containsStr(list []string, want string) bool {
	for _, s := range list {
		if s == want {
			return true
		}
	}
	return false
}

// TestFallbackVisibilitySerializationContract verifies the API-facing contract
// that Mission 5 demands: `fallback_used` is ALWAYS present in the JSON (an
// explicit false, never an absent field), so clients can never mistake a
// degraded scenario for a generated one.
func TestFallbackVisibilitySerializationContract(t *testing.T) {
	generated := &CallScenario{
		GenerationStatus: GenerationStatusGenerated,
		FallbackUsed:     false,
		OpeningSentence:  "x",
	}
	b, _ := json.Marshal(generated)
	var m map[string]any
	_ = json.Unmarshal(b, &m)
	if _, ok := m["fallback_used"]; !ok {
		t.Fatalf("fallback_used must always be serialized (explicit false), got %s", string(b))
	}
	if m["fallback_used"] != false {
		t.Fatalf("fallback_used must be false for a generated scenario, got %v", m["fallback_used"])
	}

	degraded := &CallScenario{
		GenerationStatus: GenerationStatusDegraded,
		FallbackUsed:     true,
		GenerationError:  "opening_sentence_empty",
	}
	b2, _ := json.Marshal(degraded)
	_ = json.Unmarshal(b2, &m)
	if m["fallback_used"] != true {
		t.Fatalf("fallback_used must be true for a degraded scenario, got %v", m["fallback_used"])
	}
	if m["generation_status"] != "degraded" {
		t.Fatalf("generation_status must be degraded, got %v", m["generation_status"])
	}
	if m["generation_error"] != "opening_sentence_empty" {
		t.Fatalf("generation_error must be present for a degraded scenario, got %v", m["generation_error"])
	}
}
