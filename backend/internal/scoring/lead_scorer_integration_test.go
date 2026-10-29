package scoring

import (
	"context"
	"os"
	"testing"

	"github.com/google/uuid"
	"github.com/tafakorpardazan/huginn/internal/config"
	"github.com/tafakorpardazan/huginn/internal/contacts"
	"github.com/tafakorpardazan/huginn/internal/database"
	"github.com/tafakorpardazan/huginn/internal/followups"
	"github.com/tafakorpardazan/huginn/internal/interactions"
	"github.com/tafakorpardazan/huginn/internal/leads"
	"github.com/tafakorpardazan/huginn/internal/rbac"
	"github.com/tafakorpardazan/huginn/internal/reports"
	"github.com/tafakorpardazan/huginn/internal/users"
)

func scorerIntegrationEnabled() bool {
	return os.Getenv("HUGINN_INTEGRATION") == "1"
}

// TestOnReportSubmittedRecomputesScoreAndPersistsFacts drives the full
// report → fact promotion → re-scoring pipeline for THREE consecutive reports
// on the same lead:
//
//  1. Report 1 establishes the decision maker + correct person and moves the
//     score away from the import-time value.
//  2. Report 2 does NOT re-ask stable facts (name/role are loaded, not
//     overwritten) and the score rises again from new evidence.
//  3. Report 3 preserves the stable facts while still allowing edits, and the
//     score reflects the accumulated evidence (not a reset).
//
// Every report remains in history unchanged.
func TestOnReportSubmittedRecomputesScoreAndPersistsFacts(t *testing.T) {
	if !scorerIntegrationEnabled() {
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

	leadRepo := leads.NewRepository(db)
	userRepo := users.NewRepository(db)
	interRepo := interactions.NewRepository(db)
	reportRepo := reports.NewRepository(db)
	contactRepo := contacts.NewRepository(db)
	followRepo := followups.NewRepository(db)
	reportSvc := reports.NewService(reportRepo, interRepo)
	scorer := NewLeadScorer(db, leadRepo, contactRepo, reportRepo, followRepo, NewRepository(db))

	industry := "لوازم خانگی"
	description := "تولیدکننده لوازم خانگی"
	lead, err := leadRepo.Create(ctx, &leads.Lead{
		CompanyName: "تولیدی آزمون امتیازدهی",
		Industry:    &industry,
		Description: &description,
	})
	if err != nil {
		t.Fatalf("create lead: %v", err)
	}
	mobile := "09121234567"
	contact, err := contactRepo.Create(ctx, &contacts.Contact{LeadID: lead.ID, Mobile: &mobile})
	if err != nil {
		t.Fatalf("create contact: %v", err)
	}

	user, err := userRepo.Create(ctx, &users.User{
		ID:           uuid.NewString(),
		Name:         "scorer-tester",
		Username:     "scorer-tester-" + uuid.NewString()[:8],
		PasswordHash: "x",
		Role:         rbac.RoleSalesAgent,
		IsActive:     true,
	})
	if err != nil {
		t.Fatalf("create user: %v", err)
	}

	low := reports.InterestLow
	medium := reports.InterestMedium
	high := reports.InterestHigh
	dmName := "علی رضایی"
	dmRole := "مدیر خرید"
	capacity := "۵۰۰ میلیون"
	soon := "SOON"
	next := reports.NextFollowUp

	submit := func(outcome reports.Outcome, interest, need *reports.Interest, dm *bool, name, role, capacity, timing *string) {
		t.Helper()
		inter, err := interRepo.Create(ctx, &interactions.Interaction{
			LeadID:       lead.ID,
			ContactID:    &contact.ID,
			UserID:       user.ID,
			Channel:      interactions.ChannelCall,
			Direction:    interactions.DirectionOutbound,
			ActionStatus: interactions.ActionCompleted,
			ReportStatus: interactions.ReportPending,
		})
		if err != nil {
			t.Fatalf("create interaction: %v", err)
		}
		rep, err := reportSvc.Submit(ctx, reports.SubmitParams{
			InteractionID:             inter.ID,
			Outcome:                   outcome,
			ContactReached:            boolp(true),
			CorrectPersonReached:      boolp(true),
			DecisionMakerIdentified:   dm,
			DecisionMakerName:         name,
			DecisionMakerRole:         role,
			InterestLevel:             interest,
			NeedLevel:                 need,
			EstimatedPurchaseCapacity: capacity,
			PurchaseTiming:            timing,
			NextAction:                &next,
			SubmittedBy:               user.ID,
		})
		if err != nil {
			t.Fatalf("submit report %s: %v", outcome, err)
		}
		if err := scorer.OnReportSubmitted(ctx, lead.ID, &contact.ID, rep, &user.ID); err != nil {
			t.Fatalf("OnReportSubmitted after %s: %v", outcome, err)
		}
	}

	// ---- Report 1: establishes decision maker + correct person ----
	dmTrue := true
	submit(reports.OutcomeInterested, &low, nil, &dmTrue, &dmName, &dmRole, nil, nil)

	after1, err := leadRepo.Get(ctx, lead.ID)
	if err != nil {
		t.Fatalf("reload lead after report 1: %v", err)
	}
	if !after1.DecisionMakerIdentified {
		t.Fatal("report 1 must persist decision_maker_identified on the lead")
	}
	if after1.DecisionMakerName == nil || *after1.DecisionMakerName != dmName {
		t.Fatalf("report 1 must persist decision_maker_name, got %v", after1.DecisionMakerName)
	}
	if after1.DecisionMakerRole == nil || *after1.DecisionMakerRole != dmRole {
		t.Fatalf("report 1 must persist decision_maker_role, got %v", after1.DecisionMakerRole)
	}
	if after1.CorrectPersonReached == nil || !*after1.CorrectPersonReached {
		t.Fatal("report 1 must persist correct_person_reached")
	}
	c1, _ := contactRepo.Get(ctx, contact.ID)
	if !c1.IsDecisionMaker || c1.FullName == nil || *c1.FullName != dmName || c1.JobTitle == nil || *c1.JobTitle != dmRole {
		t.Fatalf("report 1 must promote facts to the contact, got %+v", c1)
	}

	// ---- Report 2: new evidence, facts are NOT re-asked / re-entered ----
	submit(reports.OutcomeInterested, &medium, nil, nil, nil, nil, &capacity, nil)

	after2, err := leadRepo.Get(ctx, lead.ID)
	if err != nil {
		t.Fatalf("reload lead after report 2: %v", err)
	}
	if !after2.DecisionMakerIdentified || after2.DecisionMakerName == nil || *after2.DecisionMakerName != dmName {
		t.Fatalf("report 2 must preserve previously-known decision maker (was not re-entered)")
	}
	if after2.Score <= after1.Score {
		t.Fatalf("report 2 must raise the accumulated score: %d -> %d", after1.Score, after2.Score)
	}

	// ---- Report 3: more evidence + purchase timing; facts still preserved ----
	submit(reports.OutcomeHighlyInterested, &high, &high, nil, nil, nil, &capacity, &soon)

	after3, err := leadRepo.Get(ctx, lead.ID)
	if err != nil {
		t.Fatalf("reload lead after report 3: %v", err)
	}
	if after3.Score <= after2.Score {
		t.Fatalf("report 3 must keep raising the accumulated score: %d -> %d", after2.Score, after3.Score)
	}
	if !after3.DecisionMakerIdentified || after3.DecisionMakerName == nil || *after3.DecisionMakerName != dmName ||
		after3.DecisionMakerRole == nil || *after3.DecisionMakerRole != dmRole {
		t.Fatal("report 3 must preserve the stable decision-maker facts")
	}

	// The stored score must equal a deterministic recompute of the accumulated
	// evidence (never a reset or a partial overwrite).
	reps, err := reportRepo.ListForLead(ctx, lead.ID)
	if err != nil {
		t.Fatalf("ListForLead: %v", err)
	}
	if len(reps) != 3 {
		t.Fatalf("all three reports must remain in history, got %d", len(reps))
	}
	c3, _ := contactRepo.Get(ctx, contact.ID)
	expected := Score(BuildSignals(after3, c3, reps, false, false)).Total
	if after3.Score != expected {
		t.Fatalf("stored score %d != accumulated recompute %d", after3.Score, expected)
	}

	// Score history must contain one entry per report.
	history, err := NewRepository(db).History(ctx, lead.ID, 10)
	if err != nil {
		t.Fatalf("score history: %v", err)
	}
	if len(history) != 3 {
		t.Fatalf("expected 3 score_history entries, got %d", len(history))
	}
}

func boolp(b bool) *bool { return &b }

var _ = uuid.NewString
