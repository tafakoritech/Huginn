package messaging

import (
	"context"
	"fmt"
	"os"
	"sort"
	"strings"
	"testing"
	"time"

	"github.com/tafakorpardazan/huginn/internal/config"
	"github.com/tafakorpardazan/huginn/internal/llm"
)

// evalProject is the single sales project used for the personalization
// comparison: an AI camera that automatically creates professional Reels.
func evalProject() *ProjectContext {
	return &ProjectContext{
		ProjectID:          "eval-project-ai-reels",
		Title:              "دوربین هوش مصنوعی تولید ریلز",
		Description:        "دوربین هوشمندی که با هوش مصنوعی لحظات مهم را به‌صورت خودکار ضبط می‌کند و آن‌ها را به ویدیوهای حرفه‌ای برای اینستاگرام و شبکه‌های اجتماعی تبدیل می‌کند.",
		ProductInformation: "دوربین هوش مصنوعی تولید ریلز — ضبط و تدوین خودکار ویدیو",
		TargetCustomer:     "کسب‌وکارهایی که به محتوای ویدیویی منظم نیاز دارند: باشگاه‌ها، رستوران‌ها، مشاوران املاک، اینفلوئنسرها و فروشگاه‌ها",
		ValueProposition:   "تولید خودکار ریلز با کیفیت حرفه‌ای بدون نیاز به تیم تولید و تدوین؛ فقط دوربین را نصب کنید و محتوا را تحویل بگیرید",
		PainPoints:         "هزینه و زمان بالای تولید محتوا؛ کیفیت نامنظم ویدیوها؛ وابستگی به تدوینگر یا فریلنسر؛ نبود محتوای روزانه",
		SalesObjective:     "فروش اشتراک سالانه و جذب مشتری برای دموی محصول",
		CommonObjections:   "قیمت بالاست؛ کیفیت خروجی خوب نیست؛ با گوشی خودم هم می‌توانم؛ نیاز به تجهیزات اضافه دارد",
		Campaign:           "کمپین بهار ۱۴۰۴ — تمرکز روی کسب‌وکارهای محلی",
	}
}

// evalLeads returns five substantially different leads under the SAME project.
func evalLeads() []*LeadContext {
	return []*LeadContext{
		{
			LeadID:       "eval-lead-trainer",
			CompanyName:  "باشگاه بدنسازی اوج",
			BusinessType: "مربی شخصی",
			Industry:     "فیتنس",
			Description:  "مربی بدنسازی و فیتنس با کلاس‌های خصوصی و گروهی؛ تمرینات در باشگاه و فضای باز",
			City:         "تهران",
			Website:      "www.oghe-fitness.ir",
			Instagram:    "@oghe.fitness",
			ContactName:  "سامان رستمی",
			Source:       "INSTAGRAM",
		},
		{
			LeadID:          "eval-lead-restaurant",
			CompanyName:     "رستوران و کافه تریا آسمان",
			BusinessType:    "رستوران",
			Industry:        "رستوران‌داری",
			Description:     "رستوران و کافه‌تریا با سه شعبه در غرب تهران؛ ارائه غذای روزانه و منوی کافه",
			City:            "تهران",
			Website:         "www.aseman-rest.ir",
			Instagram:       "@aseman.restaurant",
			ContactName:     "مریم صادقی",
			ContactPosition: "مدیر روابط عمومی",
			Source:          "WEBSITE",
		},
		{
			LeadID:          "eval-lead-estate",
			CompanyName:     "مشاور املاک ملک‌آرا",
			BusinessType:    "مشاور املاک",
			Industry:        "املاک و مستغلات",
			Description:     "مشاوره خرید، فروش و اجاره ملک در محدوده سعادت‌آباد؛ تیم ۱۲ نفره مشاور",
			City:            "تهران",
			Province:        "تهران",
			Instagram:       "@melkara.agency",
			ContactName:     "امیرعلی کاظمی",
			ContactPosition: "مدیرعامل",
			Source:          "TRADE_DIRECTORY",
		},
		{
			LeadID:          "eval-lead-influencer",
			CompanyName:     "نگار محمدی",
			DisplayName:     "نگارمحمدی",
			BusinessType:    "اینفلوئنسر",
			Industry:        "تولید محتوا",
			Description:     "اینفلوئنسر با تمرکز روی محتوای ویدیویی سفر و سبک زندگی؛ حدود ۱۲۰ هزار دنبال‌کننده",
			Instagram:       "@negar.mohammadi",
			ContactName:     "نگار محمدی",
			ContactPosition: "صاحب پیج",
			Source:          "INSTAGRAM",
		},
		{
			LeadID:          "eval-lead-retailer",
			CompanyName:     "فروشگاه موبایل و لوازم دیجیتال نواندیش",
			BusinessType:    "فروشگاه",
			Industry:        "فروش لوازم دیجیتال",
			Description:     "فروش موبایل، لپ‌تاپ و لوازم جانبی در بازار رضا؛ فروش عمده و خرده",
			City:            "تهران",
			Address:         "بازار رضا",
			Website:         "www.noavandish.ir",
			ContactName:     "حسین ترابی",
			ContactPosition: "مسئول فروش",
			Source:          "MANUAL_IMPORT",
		},
	}
}

// distinctTokens tokenizes a field into small meaningful fragments so we can
// measure how much of the output is shared across different leads.
func distinctTokens(texts ...string) map[string]bool {
	out := map[string]bool{}
	for _, t := range texts {
		t = strings.ToLower(strings.ReplaceAll(t, "\u200c", " "))
		for _, tok := range strings.FieldsFunc(t, func(r rune) bool {
			return !(r > 0x0600 && r < 0x06FF) && !(r >= 'a' && r <= 'z') && !(r >= '0' && r <= '9')
		}) {
			if len([]rune(tok)) >= 3 {
				out[tok] = true
			}
		}
	}
	return out
}

func jaccard(a, b map[string]bool) float64 {
	if len(a) == 0 && len(b) == 0 {
		return 1
	}
	inter, union := 0, 0
	seen := map[string]bool{}
	for k := range a {
		seen[k] = true
	}
	for k := range b {
		seen[k] = true
	}
	for k := range a {
		if b[k] {
			inter++
		}
	}
	union = len(seen)
	if union == 0 {
		return 1
	}
	return float64(inter) / float64(union)
}

// personalizationSimilarity computes a conservative heuristic: how similar are
// two scenarios' core strategy-bearing text? Lower means more lead-specific.
func personalizationSimilarity(a, b *CallScenario) float64 {
	textA := distinctTokens(a.Strategy.PrimaryAngle, a.OpeningSentence,
		strings.Join(a.DiscoveryQuestions, " "), a.ValueProposition, a.Strategy.LikelyPain)
	textB := distinctTokens(b.Strategy.PrimaryAngle, b.OpeningSentence,
		strings.Join(b.DiscoveryQuestions, " "), b.ValueProposition, b.Strategy.LikelyPain)
	return jaccard(textA, textB)
}

// evalEnabled reports whether the live-LLM personalization eval should run.
func evalEnabled() bool {
	return os.Getenv("HUGINN_EVAL_LLM") == "1" && os.Getenv("LLM_BASE_URL") != "" && os.Getenv("LLM_API_KEY") != ""
}

// evalRunner builds a real LLM runner from the environment (DeepSeek via the
// configured transport). Skipped when the eval is not enabled.
func evalRunner(t *testing.T) llm.Runner {
	t.Helper()
	if !evalEnabled() {
		t.Skip("live personalization eval disabled; set HUGINN_EVAL_LLM=1 with LLM_BASE_URL/LLM_API_KEY")
	}
	cfg, err := config.Load()
	if err != nil {
		t.Fatalf("config: %v", err)
	}
	provCfg := llm.ProviderConfig{
		BaseURL:        cfg.LLM.BaseURL,
		APIKey:         cfg.LLM.APIKey,
		Model:          cfg.LLM.Model,
		Timeout:        180 * time.Second,
		MaxRetries:     cfg.LLM.MaxRetries,
		RetryBaseDelay: cfg.LLM.RetryBaseDelay,
		SOCKS5Proxy:    cfg.LLM.SOCKS5Proxy,
	}
	provider, err := llm.NewOpenAICompatibleProvider(provCfg)
	if err != nil {
		t.Fatalf("provider: %v", err)
	}
	return llm.NewRunner(provider, nil, provCfg)
}

// TestPromptContextDiffersByLead is a DETERMINISTIC regression test: the same
// project with five different leads must produce substantively different prompt
// context (data coverage + lead blocks), proving the LLM receives the per-lead
// context needed for reasoned personalization.
func TestPromptContextDiffersByLead(t *testing.T) {
	proj := evalProject()
	leads := evalLeads()
	renders := make([]string, len(leads))
	for i, l := range leads {
		l.Project = proj
		renders[i] = l.Render()
	}
	// Every lead sees the same project block but a different lead block.
	for i := 1; i < len(renders); i++ {
		if renders[i] == renders[0] {
			t.Fatalf("lead %d context must differ from lead 0", i)
		}
	}
	// The DATA COVERAGE block must be present and calibrated to each lead.
	for i, l := range leads {
		r := renders[i]
		if !strings.Contains(r, "DATA COVERAGE") {
			t.Errorf("lead %d: DATA COVERAGE block missing", i)
		}
		if strings.Contains(r, "UNKNOWN: "+strings.Join([]string{"Business description"}, ",")) {
			// description known for this lead should not be listed UNKNOWN
			if l.Description != "" && strings.Contains(r, "Business description") && !strings.Contains(r, "KNOWN:") {
				t.Errorf("lead %d: description known but DATA COVERAGE lists it UNKNOWN", i)
			}
		}
	}
	// The system prompt must contain the analysis protocol and the no-template
	// instruction.
	sys := DefaultCallScenarioSystemPrompt
	for _, want := range []string{
		"NOT a template filler",
		"MANDATORY PROTOCOL",
		"Evidence vs hypothesis",
		"Different leads in the same project MUST produce genuinely different strategies",
		"discovery_question_purposes",
		"conversation_branches",
	} {
		if !strings.Contains(sys, want) {
			t.Errorf("system prompt missing %q", want)
		}
	}
}

// TestPersonalizationDistinctiveness is the live evaluation: the SAME project
// generated for 5 substantially different leads must produce scenarios whose
// sales angle, opening, discovery questions, objections and next step differ —
// not merely the company name.
func TestPersonalizationDistinctiveness(t *testing.T) {
	runner := evalRunner(t)
	gen := NewCallScenarioGenerator(runner, &SenderProfile{
		CompanyName: "دوربین‌های هوشمند هارمون",
		Description: "تولید دوربین‌های هوش مصنوعی برای ساخت خودکار محتوای ویدیویی",
	}, nil)

	proj := evalProject()
	leads := evalLeads()
	results := make([]*CallScenario, len(leads))
	for i, l := range leads {
		l.Project = proj
		sc, err := gen.GenerateScenario(context.Background(), l, GenerationOptions{
			GenerationID: fmt.Sprintf("eval-%d-%d", i, time.Now().UnixNano()),
			LeadID:       l.LeadID,
			ProjectID:    proj.ProjectID,
			Timestamp:    time.Now(),
		})
		if err != nil {
			t.Fatalf("generate for %s: %v", l.CompanyName, err)
		}
		if sc == nil {
			t.Fatalf("scenario nil for %s", l.CompanyName)
		}
		results[i] = sc
		t.Logf("\n===== LEAD %d: %s (%s) =====\nstrategy.primary_angle: %s\nstrategy.likely_pain: %s\nstrategy.call_objective: %s\nopening: %s\nvalue: %s\ndiscovery: %v\nobjections: %v\nnext_step: %s",
			i, l.CompanyName, l.Industry,
			sc.Strategy.PrimaryAngle, sc.Strategy.LikelyPain, sc.Strategy.CallObjective,
			sc.OpeningSentence, sc.ValueProposition, sc.DiscoveryQuestions, sc.LikelyObjections, sc.RecommendedNextStep)
	}

	// Distinctiveness heuristic: pairwise similarity of strategy-bearing text
	// must stay below a generous threshold. Shallow mail-merge output would
	// produce near-identical text (similarity → 1).
	var pairs []float64
	for i := 0; i < len(results); i++ {
		for j := i + 1; j < len(results); j++ {
			sim := personalizationSimilarity(results[i], results[j])
			pairs = append(pairs, sim)
			t.Logf("similarity(%d,%d) = %.2f", i, j, sim)
		}
	}
	if len(pairs) == 0 {
		t.Fatal("no comparison pairs")
	}
	sort.Float64s(pairs)
	maxSim := pairs[len(pairs)-1]
	t.Logf("max pairwise similarity = %.2f (1.0 would mean interchangeable scripts)", maxSim)
	if maxSim > 0.85 {
		t.Errorf("scenarios too similar (max pairwise similarity %.2f > 0.85): scripts look interchangeable across leads", maxSim)
	}

	// Verify the opening is not a generic "من از طرف شرکت X تماس می‌گیرم".
	for i, sc := range results {
		if strings.Contains(sc.OpeningSentence, "محصول جدید") || strings.Contains(sc.OpeningSentence, "از طرف شرکت") {
			t.Errorf("lead %d opening is generic/templated: %q", i, sc.OpeningSentence)
		}
		if sc.Strategy == nil || sc.Strategy.PrimaryAngle == "" {
			t.Errorf("lead %d missing strategy", i)
		}
	}
}
