package callscenarios

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/tafakorpardazan/huginn/internal/config"
	"github.com/tafakorpardazan/huginn/internal/database"
	"github.com/tafakorpardazan/huginn/internal/leads"
)

func TestCallScenarioDefaults(t *testing.T) {
	sc := &CallScenario{
		LeadID:              "lead-1",
		OpeningSentence:     "Opening",
		Introduction:        "Intro",
		ReasonForCall:       "Reason",
		ValueProposition:    "Value",
		DiscoveryQuestions:  []string{"Q1", "Q2"},
		LikelyObjections:    []string{"O1"},
		ObjectionResponses:  []string{"R1"},
		RecommendedNextStep: "Next",
		ClosingLine:         "Close",
	}

	if sc.LeadID != "lead-1" {
		t.Errorf("expected LeadID to be lead-1, got %s", sc.LeadID)
	}
	if len(sc.DiscoveryQuestions) != 2 {
		t.Errorf("expected 2 questions, got %d", len(sc.DiscoveryQuestions))
	}
}

// TestPersistenceRoundTripGenerationStatus verifies that the explicit
// generation-provenance fields (generation_status / fallback_used /
// generation_error) survive Create -> Get round-trip against real PostgreSQL,
// so clients retrieving /call-scenarios can always tell fallback from a real
// generation.
func TestPersistenceRoundTripGenerationStatus(t *testing.T) {
	if os.Getenv("HUGINN_INTEGRATION") != "1" {
		t.Skip("integration tests disabled; set HUGINN_INTEGRATION=1")
	}
	ctx := context.Background()
	cfg, err := config.Load()
	if err != nil {
		t.Fatalf("load config: %v", err)
	}
	db, err := database.New(ctx, cfg.Postgres)
	if err != nil {
		t.Fatalf("connect db: %v", err)
	}
	defer db.Close()

	repo := NewRepository(db)
	if err := repo.EnsureSchema(ctx); err != nil {
		t.Fatalf("ensure schema: %v", err)
	}
	leadRepo := leads.NewRepository(db)
	lead, err := leadRepo.Create(ctx, &leads.Lead{CompanyName: "شرکت آزمون سناریو"})
	if err != nil {
		t.Fatalf("create lead: %v", err)
	}
	leadID := lead.ID

	created, err := repo.Create(ctx, &CallScenario{
		LeadID:              leadID,
		GenerationID:        "11111111-1111-1111-1111-111111111111",
		OpeningSentence:     "Opening",
		Introduction:        "Intro",
		ReasonForCall:       "Reason",
		ValueProposition:    "Value",
		DiscoveryQuestions:  []string{"Q1"},
		LikelyObjections:    []string{"O1"},
		ObjectionResponses:  []string{"R1"},
		RecommendedNextStep: "Next",
		ClosingLine:         "Close",
		ContextUsed:         ContextUsed{Project: "p", Lead: "l"},
		IsGenerated:         true,
		GenerationStatus:    "generated",
		FallbackUsed:        false,
	})
	if err != nil {
		t.Fatalf("create: %v", err)
	}

	got, err := repo.Get(ctx, leadID, created.ID)
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	if got.GenerationStatus != "generated" || got.FallbackUsed {
		t.Errorf("generated status must round-trip, got status=%q fallback=%v", got.GenerationStatus, got.FallbackUsed)
	}
	if !got.IsGenerated {
		t.Error("is_generated must round-trip as true")
	}

	degraded, err := repo.Create(ctx, &CallScenario{
		LeadID:              leadID,
		GenerationID:        "22222222-2222-2222-2222-222222222222",
		OpeningSentence:     "Opening",
		Introduction:        "Intro",
		ReasonForCall:       "Reason",
		ValueProposition:    "Value",
		DiscoveryQuestions:  []string{"Q1"},
		LikelyObjections:    []string{"O1"},
		ObjectionResponses:  []string{"R1"},
		RecommendedNextStep: "Next",
		ClosingLine:         "Close",
		ContextUsed:         ContextUsed{Project: "p", Lead: "l"},
		IsGenerated:         false,
		GenerationStatus:    "degraded",
		FallbackUsed:        true,
		GenerationError:     "opening_sentence_empty",
	})
	if err != nil {
		t.Fatalf("create degraded: %v", err)
	}
	gotDegraded, err := repo.Get(ctx, leadID, degraded.ID)
	if err != nil {
		t.Fatalf("get degraded: %v", err)
	}
	if gotDegraded.GenerationStatus != "degraded" || !gotDegraded.FallbackUsed {
		t.Errorf("degraded status must round-trip, got status=%q fallback=%v", gotDegraded.GenerationStatus, gotDegraded.FallbackUsed)
	}
	if gotDegraded.GenerationError != "opening_sentence_empty" {
		t.Errorf("generation_error must round-trip, got %q", gotDegraded.GenerationError)
	}
	if gotDegraded.IsGenerated {
		t.Error("fallback must round-trip with is_generated=false")
	}

	// Clean up so the test is idempotent.
	_ = repo.Delete(ctx, leadID, created.ID)
	_ = repo.Delete(ctx, leadID, degraded.ID)
}

var _ = time.Now
