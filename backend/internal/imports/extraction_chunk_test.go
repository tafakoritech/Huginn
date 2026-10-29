package imports

import (
	"context"
	"strings"
	"testing"

	"github.com/tafakorpardazan/huginn/internal/llm"
	"github.com/tafakorpardazan/huginn/internal/normalize"
)

func sadaf2CandidatesJSON() string {
	// The LLM omits phone and email entirely (the observed failure mode).
	return `{"candidates":[{"company_name":"شرکت صدف 2","city":"اصفهان","address":"اصفهان، بلوار ارغوانیه، کوچه ارغوان 5، پلاک 49","contacts":[]}]}`
}

// TestChunkExtractionLLMSeesCleanContacts asserts that under the LLM-authority
// architecture the cleaned source (with svg/\@ artifacts removed) is what the
// model receives, and that the candidate is NOT auto-filled with deterministic
// values when the model omits them.
func TestChunkExtractionLLMSeesCleanContacts(t *testing.T) {
	runner := &fakeRunner{
		respond: func(req llm.Request) (*llm.Response, error) {
			return &llm.Response{Text: sadaf2CandidatesJSON()}, nil
		},
	}
	svc := newService(runner)

	// ProcessDocumentText cleans the source before extraction; mirror that here.
	cleaned := normalize.CleanSourceText(sadaf2Block)
	candidates, _, err := svc.ExtractCandidatesChunked(context.Background(), doc(), cleaned)
	if err != nil {
		t.Fatalf("extract: %v", err)
	}
	if len(candidates) != 1 {
		t.Fatalf("expected 1 candidate, got %d", len(candidates))
	}
	c := candidates[0]

	// The LLM is the authority: omitted phone/email must NOT be injected.
	if c.Phone != nil || c.Email != nil {
		t.Fatalf("deterministic detection must not assign contacts, got phone=%v email=%v", c.Phone, c.Email)
	}

	// The prompt the LLM received must contain the cleaned contact data so the
	// model had the chance to extract it.
	joined := strings.Join(runner.prompts, "\n")
	if !strings.Contains(joined, "031-35314946") || !strings.Contains(joined, "info@sadaf2.ir") {
		t.Errorf("LLM prompt must contain cleaned contact data after cleanup:\n%s", joined)
	}
	// The concrete source artifact (svginfo\@...) must be gone before the LLM
	// sees it (the prompt's rules may still mention \@ as an example).
	if strings.Contains(joined, `svginfo\@`) {
		t.Errorf("source artifact must be cleaned before reaching the LLM")
	}
	if c.City == nil || *c.City != "اصفهان" {
		t.Errorf("city = %v, want اصفهان", c.City)
	}
}
