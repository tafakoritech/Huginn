package nextactions

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/tafakorpardazan/huginn/internal/config"
	"github.com/tafakorpardazan/huginn/internal/database"
	"github.com/tafakorpardazan/huginn/internal/followups"
	"github.com/tafakorpardazan/huginn/internal/interactions"
	"github.com/tafakorpardazan/huginn/internal/leads"
	"github.com/tafakorpardazan/huginn/internal/rbac"
	"github.com/tafakorpardazan/huginn/internal/users"
)

func integrationEnabled() bool { return os.Getenv("HUGINN_INTEGRATION") == "1" }

func newTestDB(t *testing.T) *database.DB {
	t.Helper()
	if !integrationEnabled() {
		t.Skip("integration tests disabled; set HUGINN_INTEGRATION=1")
	}
	cfg, err := config.Load()
	if err != nil {
		t.Fatalf("load config: %v", err)
	}
	db, err := database.New(context.Background(), cfg.Postgres)
	if err != nil {
		t.Fatalf("connect db: %v", err)
	}
	t.Cleanup(db.Close)
	return db
}

func newUser(t *testing.T, db *database.DB, name string, role rbac.Role) *users.User {
	t.Helper()
	u, err := users.NewRepository(db).Create(context.Background(), &users.User{
		ID: uuid.NewString(), Name: name, Username: name + "-" + uuid.NewString()[:8],
		PasswordHash: "x", Role: role, IsActive: true,
	})
	if err != nil {
		t.Fatalf("create user: %v", err)
	}
	return u
}

func newLead(t *testing.T, db *database.DB, company string, status leads.Status, score int, owner *string) *leads.Lead {
	t.Helper()
	l, err := leads.NewRepository(db).Create(context.Background(), &leads.Lead{
		CompanyName: company, Status: status, Score: score, OwnerUserID: owner,
	})
	if err != nil {
		t.Fatalf("create lead: %v", err)
	}
	return l
}

// TestAcceptanceScenarios covers the real-world ranking cases end to end.
func TestAcceptanceScenarios(t *testing.T) {
	db := newTestDB(t)
	ctx := context.Background()
	svc := NewService(db)
	agent := newUser(t, db, "na-agent", rbac.RoleSalesAgent)
	other := newUser(t, db, "na-other", rbac.RoleSalesAgent)

	// Scenario A: overdue follow-up on a warm lead.
	warm := newLead(t, db, "گروه ایده‌پردازان ریناس", leads.StatusFollowUp, 55, &agent.ID)
	overdue := time.Now().UTC().Add(-24 * time.Hour)
	if _, err := followups.NewService(followups.NewRepository(db)).Create(ctx, followups.CreateParams{
		LeadID: warm.ID, AssignedTo: &agent.ID, DueAt: overdue, Source: followups.SourceReport,
		Channel: strPtr("CALL"),
	}); err != nil {
		t.Fatalf("create overdue followup: %v", err)
	}

	// Scenario B: pending report for another warm lead.
	withReport := newLead(t, db, "صنایع چوب نوین", leads.StatusContacting, 60, &agent.ID)
	if _, err := interactions.NewRepository(db).Create(ctx, &interactions.Interaction{
		LeadID: withReport.ID, UserID: agent.ID, Channel: interactions.ChannelCall,
		ActionStatus: interactions.ActionCompleted, ReportStatus: interactions.ReportPending,
	}); err != nil {
		t.Fatalf("create interaction: %v", err)
	}

	// Scenario D: future follow-up (5 days out) on a hot lead — must NOT outrank
	// the overdue follow-up or pending report.
	futureLead := newLead(t, db, "فولاد آلیاژی مرکزی", leads.StatusFollowUp, 90, &agent.ID)
	if _, err := followups.NewService(followups.NewRepository(db)).Create(ctx, followups.CreateParams{
		LeadID: futureLead.ID, AssignedTo: &agent.ID, DueAt: time.Now().UTC().Add(5 * 24 * time.Hour),
		Source: followups.SourceManual, Channel: strPtr("CALL"),
	}); err != nil {
		t.Fatalf("create future followup: %v", err)
	}

	// Disqualified lead (excluded).
	disq := newLead(t, db, "شرکت رد شده", leads.StatusDisqualified, 99, &agent.ID)
	_ = disq

	// Another agent's overdue follow-up (excluded from this agent's queue).
	otherLead := newLead(t, db, "سرنخ کارشناس دیگر", leads.StatusFollowUp, 80, &other.ID)
	if _, err := followups.NewService(followups.NewRepository(db)).Create(ctx, followups.CreateParams{
		LeadID: otherLead.ID, AssignedTo: &other.ID, DueAt: time.Now().UTC().Add(-48 * time.Hour),
		Source: followups.SourceManual,
	}); err != nil {
		t.Fatalf("create other followup: %v", err)
	}

	resp, err := svc.Build(ctx, Params{UserID: agent.ID, Limit: 3})
	if err != nil {
		t.Fatalf("build: %v", err)
	}

	// The pending report must be the top action (highest precedence + urgent).
	if len(resp.Items) == 0 {
		t.Fatal("expected actionable items")
	}
	top := resp.Items[0]
	if top.SourceType != SourcePendingReport {
		t.Fatalf("expected pending report as top action, got %s", top.SourceType)
	}
	if top.Priority != TierUrgent {
		t.Fatalf("pending report must be urgent, got %s", top.Priority)
	}
	if top.CTALabel != "ثبت گزارش" {
		t.Fatalf("pending report CTA must be 'ثبت گزارش', got %q", top.CTALabel)
	}

	// The overdue follow-up must rank above the future follow-up.
	var overdueRank, futureRank = -1, -1
	for i, it := range resp.Items {
		switch it.LeadID {
		case warm.ID:
			overdueRank = i
			if !it.Overdue {
				t.Fatal("overdue follow-up must be flagged overdue")
			}
		case futureLead.ID:
			futureRank = i
		}
	}
	if overdueRank == -1 || futureRank == -1 {
		t.Fatalf("both overdue and future follow-ups should appear; overdue=%d future=%d", overdueRank, futureRank)
	}
	if futureRank < overdueRank {
		t.Fatal("future follow-up must not outrank overdue follow-up")
	}

	// The other agent's lead must not appear.
	for _, it := range resp.Items {
		if it.LeadID == otherLead.ID {
			t.Fatal("another agent's work must be excluded")
		}
	}

	// Limit respected.
	if len(resp.Items) > 3 {
		t.Fatalf("limit 3 violated: %d items", len(resp.Items))
	}
	if resp.TotalActionable < 3 {
		t.Fatalf("total actionable should be >= 3, got %d", resp.TotalActionable)
	}
}

// TestDedupeAcrossSources ensures a lead with an overdue follow-up, a stale
// state and a first-contact state collapses into a single card.
func TestDedupeAcrossSources(t *testing.T) {
	db := newTestDB(t)
	ctx := context.Background()
	svc := NewService(db)
	agent := newUser(t, db, "na-dedupe", rbac.RoleSalesAgent)

	l := newLead(t, db, "شرکت چندگانه", leads.StatusFollowUp, 75, &agent.ID)
	if _, err := followups.NewService(followups.NewRepository(db)).Create(ctx, followups.CreateParams{
		LeadID: l.ID, AssignedTo: &agent.ID, DueAt: time.Now().UTC().Add(-time.Hour),
		Source: followups.SourceManual, Channel: strPtr("CALL"),
	}); err != nil {
		t.Fatalf("create followup: %v", err)
	}

	resp, err := svc.Build(ctx, Params{UserID: agent.ID, Limit: 200})
	if err != nil {
		t.Fatalf("build: %v", err)
	}
	count := 0
	for _, it := range resp.Items {
		if it.LeadID == l.ID {
			count++
		}
	}
	if count != 1 {
		t.Fatalf("lead must appear exactly once, got %d", count)
	}
}

// TestEmptyQueue ensures no work yields an intentional empty list, never a
// fabricated item.
func TestEmptyQueue(t *testing.T) {
	db := newTestDB(t)
	ctx := context.Background()
	svc := NewService(db)
	agent := newUser(t, db, "na-empty", rbac.RoleSalesAgent)

	// Only a cold, unassigned-ish lead with no follow-up/report should not be
	// surfaced (score below the inferred-action threshold).
	_ = newLead(t, db, "شرکت سرد", leads.StatusAssigned, 10, &agent.ID)

	resp, err := svc.Build(ctx, Params{UserID: agent.ID, Limit: 3})
	if err != nil {
		t.Fatalf("build: %v", err)
	}
	if len(resp.Items) != 0 || resp.TotalActionable != 0 {
		t.Fatalf("expected empty queue, got %d items", len(resp.Items))
	}
}

func strPtr(s string) *string { return &s }
