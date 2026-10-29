package reports

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/tafakorpardazan/huginn/internal/config"
	"github.com/tafakorpardazan/huginn/internal/database"
	"github.com/tafakorpardazan/huginn/internal/interactions"
	"github.com/tafakorpardazan/huginn/internal/leads"
	"github.com/tafakorpardazan/huginn/internal/rbac"
	"github.com/tafakorpardazan/huginn/internal/users"
)

// integrationEnabled mirrors the shared integration test gate so this test can
// run standalone (setup_test.go lives in internal/integration).
func reportsIntegrationEnabled() bool {
	return os.Getenv("HUGINN_INTEGRATION") == "1"
}

// TestListForLeadJoinWorks verifies ListForLead against real PostgreSQL with a
// real JOIN between contact_reports and interactions. Before the fix the
// unqualified `id` column made this query raise
// `column reference "id" is ambiguous`; this test proves the query runs and
// returns existing reports instead of silently yielding an empty result.
func TestListForLeadJoinWorks(t *testing.T) {
	if !reportsIntegrationEnabled() {
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
	repo := NewRepository(db)

	lead, err := leadRepo.Create(ctx, &leads.Lead{CompanyName: "شرکت آزمون گزارش"})
	if err != nil {
		t.Fatalf("create lead: %v", err)
	}
	user, err := userRepo.Create(ctx, &users.User{
		ID:           uuid.NewString(),
		Name:         "report-tester",
		Username:     "report-tester-" + uuid.NewString()[:8],
		PasswordHash: "x",
		Role:         rbac.RoleSalesAgent,
		IsActive:     true,
	})
	if err != nil {
		t.Fatalf("create user: %v", err)
	}
	inter, err := interRepo.Create(ctx, &interactions.Interaction{
		LeadID:       lead.ID,
		UserID:       user.ID,
		Channel:      interactions.ChannelCall,
		Direction:    interactions.DirectionOutbound,
		Objective:    strp("FOLLOW_UP"),
		ActionStatus: interactions.ActionCompleted,
		ReportStatus: interactions.ReportSubmitted,
	})
	if err != nil {
		t.Fatalf("create interaction: %v", err)
	}
	summary := "مشتری علاقه‌مند بود"
	created, err := repo.Create(ctx, &Report{
		InteractionID: inter.ID,
		Outcome:       OutcomeInterested,
		Summary:       &summary,
		SubmittedBy:   user.ID,
	})
	if err != nil {
		t.Fatalf("create report: %v", err)
	}

	list, err := repo.ListForLead(ctx, lead.ID)
	if err != nil {
		t.Fatalf("ListForLead must not fail with ambiguous id: %v", err)
	}
	if len(list) == 0 {
		t.Fatal("ListForLead must return the existing report, not an empty result")
	}
	found := false
	for _, rep := range list {
		if rep.ID == created.ID && rep.InteractionID == inter.ID && rep.Outcome == OutcomeInterested {
			found = true
		}
	}
	if !found {
		t.Fatalf("ListForLead did not return the created report: %+v", list)
	}

	// A lead with no reports must return an empty (not error) result.
	otherLead, err := leadRepo.Create(ctx, &leads.Lead{CompanyName: "بدون گزارش"})
	if err != nil {
		t.Fatalf("create other lead: %v", err)
	}
	empty, err := repo.ListForLead(ctx, otherLead.ID)
	if err != nil {
		t.Fatalf("ListForLead for empty lead: %v", err)
	}
	if len(empty) != 0 {
		t.Fatalf("expected empty list for lead without reports, got %d", len(empty))
	}
}

func strp(s string) *string { return &s }

// TestListForUserVisibility verifies the global submitted-reports repository
// query: joins never raise ambiguous-column errors, sales agents are scoped to
// their own reports, managers see all, VOIDED manager-override placeholders are
// excluded, and a user with no reports gets [] without error.
func TestListForUserVisibility(t *testing.T) {
	if !reportsIntegrationEnabled() {
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
	repo := NewRepository(db)

	mkUser := func(role rbac.Role) *users.User {
		u, err := userRepo.Create(ctx, &users.User{
			ID: uuid.NewString(), Name: "lv-tester-" + uuid.NewString()[:6],
			Username: "lv_" + uuid.NewString()[:8], PasswordHash: "x",
			Role: role, IsActive: true,
		})
		if err != nil {
			t.Fatalf("create user: %v", err)
		}
		return u
	}
	agentA := mkUser(rbac.RoleSalesAgent)
	agentB := mkUser(rbac.RoleSalesAgent)
	manager := mkUser(rbac.RoleSalesDevManager)

	// Agent A's real submitted report.
	leadA, err := leadRepo.Create(ctx, &leads.Lead{CompanyName: "شرکت لیست کاربر"})
	if err != nil {
		t.Fatalf("create lead A: %v", err)
	}
	interA, err := interRepo.Create(ctx, &interactions.Interaction{
		LeadID: leadA.ID, UserID: agentA.ID, Channel: interactions.ChannelCall,
		Direction: interactions.DirectionOutbound, ActionStatus: interactions.ActionCompleted,
		ReportStatus: interactions.ReportPending,
	})
	if err != nil {
		t.Fatalf("create interaction A: %v", err)
	}
	summaryA := "گزارش واقعی"
	if _, err := repo.Create(ctx, &Report{
		InteractionID: interA.ID, Outcome: OutcomeInterested, Summary: &summaryA, SubmittedBy: agentA.ID,
	}); err != nil {
		t.Fatalf("create report A: %v", err)
	}

	// Agent B's real report.
	leadB, err := leadRepo.Create(ctx, &leads.Lead{CompanyName: "شرکت لیست کاربر ۲"})
	if err != nil {
		t.Fatalf("create lead B: %v", err)
	}
	interB, err := interRepo.Create(ctx, &interactions.Interaction{
		LeadID: leadB.ID, UserID: agentB.ID, Channel: interactions.ChannelCall,
		Direction: interactions.DirectionOutbound, ActionStatus: interactions.ActionCompleted,
		ReportStatus: interactions.ReportPending,
	})
	if err != nil {
		t.Fatalf("create interaction B: %v", err)
	}
	summaryB := "گزارش دوم"
	if _, err := repo.Create(ctx, &Report{
		InteractionID: interB.ID, Outcome: OutcomeNoAnswer, Summary: &summaryB, SubmittedBy: agentB.ID,
	}); err != nil {
		t.Fatalf("create report B: %v", err)
	}

	// A VOIDED placeholder must never appear as a submitted report. It is created
	// for an interaction that had NO real report (manager voided a pending one).
	leadV, err := leadRepo.Create(ctx, &leads.Lead{CompanyName: "شرکت لیست کاربر ۳"})
	if err != nil {
		t.Fatalf("create lead V: %v", err)
	}
	interV, err := interRepo.Create(ctx, &interactions.Interaction{
		LeadID: leadV.ID, UserID: agentB.ID, Channel: interactions.ChannelCall,
		Direction: interactions.DirectionOutbound, ActionStatus: interactions.ActionCompleted,
		ReportStatus: interactions.ReportVoidedByManager,
	})
	if err != nil {
		t.Fatalf("create interaction V: %v", err)
	}
	if _, err := repo.db.Pool().Exec(ctx, `
		INSERT INTO contact_reports (interaction_id, outcome, submitted_by, manager_override, manager_override_reason, overridden_by)
		VALUES ($1,'VOIDED',$2,true,'reason',$2)`, interV.ID, agentB.ID); err != nil {
		t.Fatalf("insert voided placeholder: %v", err)
	}

	// Agent A sees exactly their own report, never agent B's and never VOIDED.
	itemsA, totalA, err := repo.ListForUser(ctx, agentA.ID, false, 50, 0)
	if err != nil {
		t.Fatalf("ListForUser(agentA) must not be ambiguous: %v", err)
	}
	if totalA != 1 || len(itemsA) != 1 {
		t.Fatalf("agent A must see exactly 1 report, got %d/%d", len(itemsA), totalA)
	}
	if itemsA[0].LeadID != leadA.ID {
		t.Fatalf("agent A report must carry its lead, got %s", itemsA[0].LeadID)
	}

	// Agent B sees their own report (the VOIDED placeholder is excluded).
	itemsB, totalB, err := repo.ListForUser(ctx, agentB.ID, false, 50, 0)
	if err != nil {
		t.Fatalf("ListForUser(agentB): %v", err)
	}
	if totalB != 1 || len(itemsB) != 1 {
		t.Fatalf("agent B must see exactly 1 real report (VOIDED excluded), got %d/%d", len(itemsB), totalB)
	}
	if itemsB[0].Outcome == Outcome("VOIDED") {
		t.Fatal("VOIDED placeholder must be excluded from submitted reports")
	}

	// Manager sees both real reports.
	itemsM, totalM, err := repo.ListForUser(ctx, manager.ID, true, 50, 0)
	if err != nil {
		t.Fatalf("ListForUser(manager): %v", err)
	}
	if totalM < 2 || len(itemsM) < 2 {
		t.Fatalf("manager must see both real reports, got %d/%d", len(itemsM), totalM)
	}

	// A brand-new agent with no reports gets [] without error.
	lonely := mkUser(rbac.RoleSalesAgent)
	itemsL, totalL, err := repo.ListForUser(ctx, lonely.ID, false, 50, 0)
	if err != nil {
		t.Fatalf("ListForUser(empty): %v", err)
	}
	if itemsL == nil || len(itemsL) != 0 || totalL != 0 {
		t.Fatalf("empty user must get items=[] total=0, got %d/%d", len(itemsL), totalL)
	}
}

var _ = time.Now
