package followups

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

func newTestUser(t *testing.T, db *database.DB, name string) *users.User {
	t.Helper()
	u, err := users.NewRepository(db).Create(context.Background(), &users.User{
		ID:           uuid.NewString(),
		Name:         name,
		Username:     name + "-" + uuid.NewString()[:8],
		PasswordHash: "x",
		Role:         rbac.RoleSalesAgent,
		IsActive:     true,
	})
	if err != nil {
		t.Fatalf("create user: %v", err)
	}
	return u
}

// TestFollowUpLifecycle drives create → list scopes → complete → reopen →
// reschedule → cancel across every view.
func TestFollowUpLifecycle(t *testing.T) {
	db := newTestDB(t)
	ctx := context.Background()
	repo := NewRepository(db)
	svc := NewService(repo)
	leadRepo := leads.NewRepository(db)

	agent := newTestUser(t, db, "fu-agent")
	other := newTestUser(t, db, "fu-other")

	lead, err := leadRepo.Create(ctx, &leads.Lead{CompanyName: "شرکت تست پیگیری"})
	if err != nil {
		t.Fatalf("create lead: %v", err)
	}

	now := time.Now().UTC()
	overdue := now.Add(-2 * time.Hour)
	// todayDue is guaranteed to be after now and before the end of the UTC day,
	// so it always lands in the "today" bucket regardless of when the test runs.
	endOfToday := now.Truncate(24 * time.Hour).Add(24 * time.Hour)
	todayDue := now.Add(endOfToday.Sub(now) / 2)
	upcoming := now.Add(48 * time.Hour)

	// Create an overdue follow-up assigned to the agent.
	created, err := svc.Create(ctx, CreateParams{
		LeadID: lead.ID, AssignedTo: &agent.ID, DueAt: overdue,
		Type: "FOLLOW_UP", Description: strp("قبلا موعدش گذشته"), CreatedBy: &agent.ID,
	})
	if err != nil {
		t.Fatalf("create overdue follow-up: %v", err)
	}
	if created.Status != StatusPending {
		t.Fatalf("new follow-up must be PENDING, got %s", created.Status)
	}

	// A today follow-up for a different agent.
	if _, err := svc.Create(ctx, CreateParams{LeadID: lead.ID, AssignedTo: &other.ID, DueAt: todayDue}); err != nil {
		t.Fatalf("create today follow-up: %v", err)
	}
	// An upcoming follow-up for the agent.
	if _, err := svc.Create(ctx, CreateParams{LeadID: lead.ID, AssignedTo: &agent.ID, DueAt: upcoming}); err != nil {
		t.Fatalf("create upcoming follow-up: %v", err)
	}

	// My follow-ups: only the agent's assigned, never cancelled.
	myItems, _, err := svc.List(ctx, ListParams{Scope: "my", AssignedTo: &agent.ID, LeadID: &lead.ID})
	if err != nil {
		t.Fatalf("list my: %v", err)
	}
	if len(myItems) != 2 {
		t.Fatalf("agent must have 2 assigned follow-ups, got %d", len(myItems))
	}
	for _, it := range myItems {
		if it.AssignedTo == nil || *it.AssignedTo != agent.ID {
			t.Fatalf("my list leaked another user's follow-up: %+v", it)
		}
	}

	// Overdue scope (scoped to this lead): only the overdue item.
	overdueItems, _, err := svc.List(ctx, ListParams{Scope: "overdue", LeadID: &lead.ID})
	if err != nil {
		t.Fatalf("list overdue: %v", err)
	}
	if len(overdueItems) != 1 {
		t.Fatalf("expected 1 overdue follow-up, got %d", len(overdueItems))
	}
	if !overdueItems[0].IsOverdue {
		t.Fatal("overdue list item must set is_overdue=true")
	}

	// Today scope: only the today item (not overdue, not upcoming).
	todayItems, _, err := svc.List(ctx, ListParams{Scope: "today", LeadID: &lead.ID})
	if err != nil {
		t.Fatalf("list today: %v", err)
	}
	if len(todayItems) != 1 {
		t.Fatalf("expected 1 today follow-up, got %d", len(todayItems))
	}

	// Upcoming scope.
	upcomingItems, _, err := svc.List(ctx, ListParams{Scope: "upcoming", LeadID: &lead.ID})
	if err != nil {
		t.Fatalf("list upcoming: %v", err)
	}
	if len(upcomingItems) != 1 {
		t.Fatalf("expected 1 upcoming follow-up, got %d", len(upcomingItems))
	}

	// Lead-scoped list sees all three.
	leadItems, err := svc.ListForLead(ctx, lead.ID)
	if err != nil {
		t.Fatalf("list for lead: %v", err)
	}
	if len(leadItems) != 3 {
		t.Fatalf("lead must have 3 follow-ups, got %d", len(leadItems))
	}

	// Complete the overdue one; it must leave every pending scope.
	note := "تماس گرفته شد"
	outcome := "CALL_DONE"
	if _, err := svc.Complete(ctx, created.ID, &note, &outcome); err != nil {
		t.Fatalf("complete: %v", err)
	}
	completedItems, _, _ := svc.List(ctx, ListParams{Scope: "completed", LeadID: &lead.ID})
	if len(completedItems) != 1 || completedItems[0].ID != created.ID {
		t.Fatalf("completed scope must contain the completed follow-up")
	}
	overdueItems, _, _ = svc.List(ctx, ListParams{Scope: "overdue", LeadID: &lead.ID})
	if len(overdueItems) != 0 {
		t.Fatalf("completed follow-up must never appear overdue")
	}

	// Reopen back to pending.
	if _, err := svc.Reopen(ctx, created.ID, nil); err != nil {
		t.Fatalf("reopen: %v", err)
	}
	overdueItems, _, _ = svc.List(ctx, ListParams{Scope: "overdue", LeadID: &lead.ID})
	if len(overdueItems) != 1 {
		t.Fatalf("reopened follow-up must be overdue again")
	}

	// Reschedule to upcoming → leaves overdue.
	newDue := now.Add(72 * time.Hour)
	if _, err := svc.Reschedule(ctx, created.ID, newDue); err != nil {
		t.Fatalf("reschedule: %v", err)
	}
	overdueItems, _, _ = svc.List(ctx, ListParams{Scope: "overdue", LeadID: &lead.ID})
	if len(overdueItems) != 0 {
		t.Fatalf("rescheduled follow-up must leave overdue")
	}
	upcomingItems, _, _ = svc.List(ctx, ListParams{Scope: "upcoming", LeadID: &lead.ID})
	if len(upcomingItems) != 2 {
		t.Fatalf("rescheduled follow-up must appear upcoming, got %d", len(upcomingItems))
	}

	// Cancel.
	if _, err := svc.Cancel(ctx, created.ID); err != nil {
		t.Fatalf("cancel: %v", err)
	}
	cancelled, _ := repo.Get(ctx, created.ID)
	if cancelled.Status != StatusCancelled {
		t.Fatalf("cancelled follow-up must be CANCELLED, got %s", cancelled.Status)
	}
	allItems, _, _ := svc.List(ctx, ListParams{Scope: "all", LeadID: &lead.ID})
	for _, it := range allItems {
		if it.ID == created.ID && it.Status == StatusPending {
			t.Fatal("cancelled follow-up must not be pending")
		}
	}
}

// TestReportGeneratedFollowUpIsIdempotent verifies that upserting a follow-up
// for the same interaction twice never creates a duplicate.
func TestReportGeneratedFollowUpIsIdempotent(t *testing.T) {
	db := newTestDB(t)
	ctx := context.Background()
	repo := NewRepository(db)
	svc := NewService(repo)
	leadRepo := leads.NewRepository(db)
	interRepo := interactions.NewRepository(db)
	agent := newTestUser(t, db, "fu-report")

	lead, err := leadRepo.Create(ctx, &leads.Lead{CompanyName: "شرکت گزارش پیگیری"})
	if err != nil {
		t.Fatalf("create lead: %v", err)
	}
	inter, err := interRepo.Create(ctx, &interactions.Interaction{
		LeadID: lead.ID, UserID: agent.ID, Channel: interactions.ChannelCall,
		Direction: interactions.DirectionOutbound, ActionStatus: interactions.ActionCompleted,
		ReportStatus: interactions.ReportPending,
	})
	if err != nil {
		t.Fatalf("create interaction: %v", err)
	}

	due := time.Now().UTC().Add(24 * time.Hour)
	desc := "پس از تماس"
	params := CreateParams{
		LeadID: lead.ID, InteractionID: &inter.ID, AssignedTo: &agent.ID,
		DueAt: due, Description: &desc, CreatedBy: &agent.ID,
	}

	first, err := svc.UpsertForInteraction(ctx, params)
	if err != nil {
		t.Fatalf("upsert #1: %v", err)
	}
	if first.Source != SourceReport {
		t.Fatalf("report follow-up source must be REPORT, got %s", first.Source)
	}

	// Resubmission: same interaction, different due time → reschedules, no dup.
	newDue := due.Add(48 * time.Hour).Truncate(time.Microsecond) // Postgres stores microseconds
	params.DueAt = newDue
	second, err := svc.UpsertForInteraction(ctx, params)
	if err != nil {
		t.Fatalf("upsert #2: %v", err)
	}
	if second.ID != first.ID {
		t.Fatalf("upsert must reuse the same follow-up: %s vs %s", first.ID, second.ID)
	}
	if !second.DueAt.Equal(newDue) {
		t.Fatalf("upsert must reschedule to the new due time: %v vs %v", second.DueAt, newDue)
	}

	all, err := repo.ListForLead(ctx, lead.ID)
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if len(all) != 1 {
		t.Fatalf("must be exactly one follow-up per interaction, got %d", len(all))
	}
}

// TestFollowUpCreateFieldsAndUTC verifies that creating a follow-up preserves
// the type, description and due date, and that the stored timestamp round-trips
// as UTC. The Jalali (Shamsi) conversion happens only in the presentation layer,
// so the API must keep the original UTC instant intact.
func TestFollowUpCreateFieldsAndUTC(t *testing.T) {
	db := newTestDB(t)
	ctx := context.Background()
	repo := NewRepository(db)
	svc := NewService(repo)
	leadRepo := leads.NewRepository(db)
	agent := newTestUser(t, db, "fu-fields")

	lead, err := leadRepo.Create(ctx, &leads.Lead{CompanyName: "شرکت تست فیلدهای پیگیری"})
	if err != nil {
		t.Fatalf("create lead: %v", err)
	}

	due := time.Date(2026, 8, 24, 0, 0, 0, 0, time.UTC)
	desc := "تماس با مدیر مجموعه برای ارائه پیشنهاد قیمت"
	title := "پیشنهاد قیمت"

	for _, tt := range []struct {
		name string
		typ  string
	}{
		{"call", "CALL"},
		{"follow-up", "FOLLOW_UP"},
		{"message", "MESSAGE"},
		{"email", "EMAIL"},
	} {
		created, err := svc.Create(ctx, CreateParams{
			LeadID: lead.ID, AssignedTo: &agent.ID, DueAt: due,
			Type: tt.typ, Title: &title, Description: &desc, CreatedBy: &agent.ID,
		})
		if err != nil {
			t.Fatalf("create %s follow-up: %v", tt.name, err)
		}
		if created.Type != tt.typ {
			t.Fatalf("%s follow-up type must be preserved, got %q", tt.name, created.Type)
		}
		if created.Description == nil || *created.Description != desc {
			t.Fatalf("%s follow-up description must be preserved, got %v", tt.name, created.Description)
		}
		if created.DueAt.Location() != time.UTC {
			t.Fatalf("%s follow-up due_at must be UTC, got %v (%s)", tt.name, created.DueAt, created.DueAt.Location())
		}
		if !created.DueAt.Equal(due) {
			t.Fatalf("%s follow-up due_at must round-trip exactly, got %v", tt.name, created.DueAt)
		}
		if created.Status != StatusPending {
			t.Fatalf("%s follow-up must start PENDING, got %s", tt.name, created.Status)
		}
		if created.Priority != PriorityMedium {
			t.Fatalf("%s follow-up must default to MEDIUM priority, got %s", tt.name, created.Priority)
		}
		if created.Source != SourceManual {
			t.Fatalf("%s follow-up must default to MANUAL source, got %s", tt.name, created.Source)
		}
	}

	// Every created follow-up must surface with the same UTC instant.
	items, err := svc.ListForLead(ctx, lead.ID)
	if err != nil {
		t.Fatalf("list for lead: %v", err)
	}
	if len(items) != 4 {
		t.Fatalf("lead must have 4 follow-ups, got %d", len(items))
	}
	for _, it := range items {
		if !it.DueAt.Equal(due) {
			t.Fatalf("stored due_at must stay UTC (%v)", it.DueAt)
		}
		if it.DueAt.Location() != time.UTC {
			t.Fatalf("stored due_at must be UTC, got %v", it.DueAt.Location())
		}
	}
}

func strp(s string) *string { return &s }
