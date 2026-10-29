package imports

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"os"
	"strings"
	"testing"
	"unicode/utf8"

	"github.com/tafakorpardazan/huginn/internal/llm"
)

// fakeRunner simulates an LLM provider so the extraction pipeline is fully
// testable without external infrastructure or an API key.
type fakeRunner struct {
	respond func(req llm.Request) (*llm.Response, error)
	calls   int
	prompts []string
}

func (f *fakeRunner) GenerateStructured(ctx context.Context, req llm.Request, out any) (*llm.Run, error) {
	f.calls++
	f.prompts = append(f.prompts, req.UserPrompt)
	resp, err := f.respond(req)
	if err != nil {
		return &llm.Run{Model: "fake-model", Status: "failed", Error: err.Error()}, err
	}
	if err := json.Unmarshal([]byte(resp.Text), out); err != nil {
		return &llm.Run{Model: "fake-model", Status: "failed"}, &llm.ErrValidation{Purpose: req.Purpose, Err: err}
	}
	return &llm.Run{
		Model:            "fake-model",
		Status:           "success",
		StructuredOutput: []byte(resp.Text),
		InputTokens:      len(resp.Text) / 4,
		OutputTokens:     10,
	}, nil
}

func (f *fakeRunner) GenerateText(ctx context.Context, req llm.Request) (*llm.Run, error) {
	return nil, errors.New("unused")
}

func fixtureText(t *testing.T) string {
	t.Helper()
	b, err := os.ReadFile("testdata/companies.txt")
	if err != nil {
		t.Fatalf("read fixture: %v", err)
	}
	return string(b)
}

func newService(runner llm.Runner) *ExtractionService {
	return &ExtractionService{
		runner: runner,
		logger: slog.New(slog.NewTextHandler(io.Discard, nil)),
	}
}

func doc() *Document { return &Document{ID: "doc-1", BatchID: "batch-1"} }

// candidateJSON builds a strict CandidateSet response for the given companies.
func candidateJSON(companies [][2]string) string {
	set := CandidateSet{}
	for _, c := range companies {
		mobile := c[1]
		contacts := []CandidateContact{}
		if mobile != "" {
			contacts = append(contacts, CandidateContact{
				FullName: strPtr("محمد رضایی"), Mobile: strPtr(mobile),
				Email: strPtr("info@" + strings.ReplaceAll(strings.ToLower(c[0]), " ", "") + ".ir"),
			})
		}
		set.Candidates = append(set.Candidates, Candidate{
			CompanyName: c[0], BusinessType: strPtr("تولیدکننده"), City: strPtr("تهران"),
			Website: strPtr("www." + strings.ReplaceAll(strings.ToLower(c[0]), " ", "") + ".ir"),
			Contacts: contacts,
		})
	}
	b, _ := json.Marshal(set)
	return string(b)
}

func TestExtractionPipeline_ExtractsAllCandidates(t *testing.T) {
	text := fixtureText(t)
	companies := [][2]string{
		{"عروسک‌سازی پارس", "09357578808"},
		{"صنایع چوب نوین", "09144443322"},
		{"گروه صنعتی فولاد آریا", "09123334455"},
		{"پخش مواد غذایی البرز", "09197776655"},
		{"استارتاپ لجستیک هوشمند راهینو", "09126665544"},
		{"آموزشگاه زبان سفیر", "09121112233"},
		{"تولیدی پوشاک سپید", "09353336677"},
		{"کارگاه سفال و سرامیک گلستان", "09152223344"},
		{"شرکت دارویی آوا فارما", "09121112233"},
		{"مزرعه گل و گیاه بهاران", "09119988776"},
		{"شرکت نساجی ابریشم کاشان", "09134455667"},
		{"کارگاه تزئینات ساختمانی نقشین", "09127891234"},
	}
	var received string
	runner := &fakeRunner{respond: func(req llm.Request) (*llm.Response, error) {
		received = req.UserPrompt
		return &llm.Response{Text: candidateJSON(companies)}, nil
	}}

	svc := newService(runner)
	candidates, diag, err := svc.ExtractCandidatesChunked(context.Background(), doc(), text)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// More than 3 candidates must be extractable from a multi-company document.
	if len(candidates) <= 3 {
		t.Fatalf("expected >3 candidates, got %d", len(candidates))
	}
	if len(candidates) != 12 {
		t.Fatalf("expected 12 candidates, got %d", len(candidates))
	}
	if diag.ExtractedTextChars != len(text) {
		t.Fatalf("extracted_text_chars mismatch: got %d want %d", diag.ExtractedTextChars, len(text))
	}
	if diag.Chunks != 1 {
		t.Fatalf("small fixture should be a single chunk, got %d", diag.Chunks)
	}
	// The LLM must actually receive the document content.
	if !strings.Contains(received, "عروسک‌سازی پارس") || !strings.Contains(received, "0914 444 3322") {
		t.Fatal("LLM prompt does not contain the actual document text")
	}
}

func TestExtractionPipeline_ChunkingSendsEveryPart(t *testing.T) {
	base := fixtureText(t)
	// Build a document well beyond the per-chunk budget.
	big := strings.Repeat(base, 12)
	if len(big) <= 30000 {
		t.Fatalf("test fixture not large enough: %d", len(big))
	}
	// markers prove head and tail both reach the LLM
	big = "MARKER_START\n" + big + "\nMARKER_END"

	runner := &fakeRunner{respond: func(req llm.Request) (*llm.Response, error) {
		// Each chunk returns the same single candidate; merge must collapse it.
		return &llm.Response{Text: candidateJSON([][2]string{{"همین شرکت تکراری", "09121112233"}})}, nil
	}}

	svc := newService(runner)
	candidates, diag, err := svc.ExtractCandidatesChunked(context.Background(), doc(), big)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if diag.Chunks < 2 {
		t.Fatalf("expected >=2 chunks, got %d", diag.Chunks)
	}
	if runner.calls != diag.Chunks {
		t.Fatalf("runner calls (%d) != chunks (%d)", runner.calls, diag.Chunks)
	}
	// No silent truncation: combined LLM input must be at least the full text.
	if diag.LLMInputChars < len(big) {
		t.Fatalf("LLM received fewer chars (%d) than the document (%d)", diag.LLMInputChars, len(big))
	}
	if !strings.Contains(runner.prompts[0], "MARKER_START") {
		t.Fatal("document head was not sent to the LLM")
	}
	if !strings.Contains(runner.prompts[len(runner.prompts)-1], "MARKER_END") {
		t.Fatal("document tail was not sent to the LLM")
	}
	// Overlapping duplicate candidate merged to a single one.
	if len(candidates) != 1 {
		t.Fatalf("expected merged single candidate, got %d", len(candidates))
	}
	if diag.RawCandidateCount != 1 {
		t.Fatalf("expected raw count 1 after merge, got %d", diag.RawCandidateCount)
	}
}

func TestExtractionPipeline_LLMFailurePropagates(t *testing.T) {
	text := fixtureText(t)
	runner := &fakeRunner{respond: func(req llm.Request) (*llm.Response, error) {
		return nil, errors.New("upstream 500")
	}}
	svc := newService(runner)
	_, _, err := svc.ExtractCandidatesChunked(context.Background(), doc(), text)
	if err == nil {
		t.Fatal("LLM failure must propagate as an error, not a fake empty success")
	}
}

func TestExtractionPipeline_ParsingKeepsCandidates(t *testing.T) {
	text := fixtureText(t)
	// A candidate with ONLY a name (every optional field null) plus a candidate
	// with two contacts (multiple phones) must both survive parsing.
	runner := &fakeRunner{respond: func(req llm.Request) (*llm.Response, error) {
		set := CandidateSet{Candidates: []Candidate{
			{CompanyName: "فقط یک نام، بدون تماس"},
			{CompanyName: "شرکت چند مخاطب", Contacts: []CandidateContact{
				{FullName: strPtr("مدیرعامل"), Mobile: strPtr("09121112233")},
				{FullName: strPtr("منشی"), Mobile: strPtr("09357778899"), Email: strPtr("X@Y.IR")},
			}},
		}}
		b, _ := json.Marshal(set)
		return &llm.Response{Text: string(b)}, nil
	}}
	svc := newService(runner)
	candidates, _, err := svc.ExtractCandidatesChunked(context.Background(), doc(), text)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(candidates) != 2 {
		t.Fatalf("expected 2 parsed candidates, got %d", len(candidates))
	}
	if len(candidates[1].Contacts) != 2 {
		t.Fatalf("expected 2 contacts (multiple phones), got %d", len(candidates[1].Contacts))
	}
}

func TestExtractionPipeline_StatisticsMatchProcessing(t *testing.T) {
	// 13 candidates from the LLM: 12 valid + 1 missing company_name.
	candidates := []Candidate{}
	names := []string{
		"عروسک‌سازی پارس", "صنایع چوب نوین", "گروه صنعتی فولاد آریا", "پخش مواد غذایی البرز",
		"استارتاپ لجستیک هوشمند", "آموزشگاه زبان سفیر", "تولیدی پوشاک سپید", "کارگاه سفال گلستان",
		"شرکت دارویی آوا فارما", "مزرعه گل بهاران", "شرکت نساجی ابریشم کاشان", "کارگاه تزئینات نقشین",
	}
	for _, n := range names {
		candidates = append(candidates, Candidate{CompanyName: n, Contacts: []CandidateContact{{Mobile: strPtr("0912" + n[:3] + "1234")}}})
	}
	candidates = append(candidates, Candidate{}) // invalid: no company_name

	// Two candidates are duplicates: their mobile matches a "seed" lead.
	seedDupes := map[string]bool{"09121111234": true, "09122222345": true}
	candidates[0].Contacts[0].Mobile = strPtr("09121111234") // duplicate 1
	candidates[1].Contacts[0].Mobile = strPtr("09122222345") // duplicate 2

	svc := newService(nil)
	result := svc.processCandidates(context.Background(), doc(), candidates, func(c *Candidate) (CandidateOutcome, *DupMatch, error) {
		if seedDupes[normalizedMobile(t, c)] {
			return OutcomeDuplicate, &DupMatch{
				MatchedLeadID: "lead-seed", Rule: "exact_normalized_mobile",
				Field: "mobile", Value: normalizedMobile(t, c),
			}, nil
		}
		return OutcomeCreated, nil, nil
	})

	if result.Extracted != 13 {
		t.Fatalf("extracted want 13 got %d", result.Extracted)
	}
	if result.Valid != 12 {
		t.Fatalf("valid want 12 got %d", result.Valid)
	}
	if result.Invalid != 1 {
		t.Fatalf("invalid want 1 got %d", result.Invalid)
	}
	if result.Duplicates != 2 {
		t.Fatalf("duplicates want 2 got %d", result.Duplicates)
	}
	if result.Created != 10 {
		t.Fatalf("created want 10 got %d", result.Created)
	}
	if result.Failed != 0 {
		t.Fatalf("failed want 0 got %d", result.Failed)
	}
	// Invariant: every valid candidate was either created, duplicated, or failed.
	if result.Created+result.Duplicates+result.Failed != result.Valid {
		t.Fatalf("stats invariant broken: %d+%d+%d != %d", result.Created, result.Duplicates, result.Failed, result.Valid)
	}
	// Duplicate decisions must be explainable (rule + field + value).
	dup := &DupMatch{MatchedLeadID: "lead-seed", Rule: "exact_normalized_mobile", Field: "mobile", Value: "x"}
	if dup.Rule == "" || dup.Field == "" || dup.Value == "" || dup.MatchedLeadID == "" {
		t.Fatal("duplicate explanation must include matched lead, rule, field and value")
	}
}

func TestChunkText_SplitsWithoutLosingContent(t *testing.T) {
	text := strings.Repeat("خط شرکت یک\nتلفن: 021-11112222\nایمیل: a@b.ir\n\n", 6000)
	chunks := llm.ChunkText(text, 25000, 600)

	if len(chunks) < 2 {
		t.Fatalf("expected multiple chunks, got %d", len(chunks))
	}
	total := 0
	for i, c := range chunks {
		if len(c) > 25000+utf8.UTFMax {
			t.Fatalf("chunk %d exceeds budget: %d", i, len(c))
		}
		if !utf8.ValidString(c) {
			t.Fatalf("chunk %d is not valid UTF-8", i)
		}
		total += len(c)
	}
	if total < len(text) {
		t.Fatalf("chunked text smaller than input: %d < %d", total, len(text))
	}
	// Every original character must appear in some chunk (no content loss).
	joined := strings.Join(chunks, "")
	for _, r := range text {
		if !strings.ContainsRune(joined, r) {
			t.Fatalf("character %q lost during chunking", r)
		}
	}
}

func TestChunkText_SingleChunkWhenSmall(t *testing.T) {
	text := "شرکت کوچک\nتلفن: 021-11112222\n"
	chunks := llm.ChunkText(text, 25000, 600)
	if len(chunks) != 1 || chunks[0] != text {
		t.Fatalf("small text must stay a single chunk, got %d", len(chunks))
	}
}

func normalizedMobile(t *testing.T, c *Candidate) string {
	t.Helper()
	if len(c.Contacts) == 0 || c.Contacts[0].Mobile == nil {
		return ""
	}
	return *c.Contacts[0].Mobile
}

func strPtr(s string) *string { return &s }
