package integration

import (
	"context"
	"testing"

	"github.com/tafakorpardazan/huginn/internal/leads"
	"github.com/tafakorpardazan/huginn/internal/muninn"
	"github.com/tafakorpardazan/huginn/internal/reports"
)

// TestMuninnExportIdempotency verifies that a lead can only be exported once.
func TestMuninnExportIdempotency(t *testing.T) {
	db := setupDB(t)

	repo := muninn.NewRepository(db)
	leadRepo := leads.NewRepository(db)
	leadSvc := leads.NewService(leadRepo)

	lead, err := leadSvc.Create(context.Background(), leads.CreateParams{CompanyName: "Export Test " + ts()})
	if err != nil {
		t.Fatalf("create lead: %v", err)
	}

	// A non-qualified lead cannot be exported (status gate).
	svc := muninn.NewService(repo, nil, nil)
	if _, err := svc.ExportQualifiedLead(context.Background(), lead.ID, "tester"); err == nil {
		t.Fatal("exporting a non-qualified lead must fail")
	}

	// Create export record directly (as the worker would after qualification) and
	// verify the unique constraint prevents duplicates.
	exp, err := repo.CreateExport(context.Background(), lead.ID, map[string]any{"score": 85})
	if err != nil {
		t.Fatalf("create export: %v", err)
	}
	if exp.Status != "PENDING" {
		t.Fatalf("expected PENDING, got %s", exp.Status)
	}

	// duplicate creation is blocked by the unique constraint on huginn_lead_id
	if _, err := repo.CreateExport(context.Background(), lead.ID, map[string]any{"score": 85}); err == nil {
		t.Fatal("duplicate export row must be rejected")
	}

	// Mark succeeded; the same lead now returns the same export.
	if err := repo.MarkSucceeded(context.Background(), exp.ID, "muninn-123"); err != nil {
		t.Fatalf("mark succeeded: %v", err)
	}
	got, err := repo.GetExport(context.Background(), lead.ID)
	if err != nil {
		t.Fatalf("get export: %v", err)
	}
	if got.MuninnLeadID == nil || *got.MuninnLeadID != "muninn-123" {
		t.Fatalf("unexpected muninn id: %v", got.MuninnLeadID)
	}
}

func TestQualificationReportConstants(t *testing.T) {
	for _, o := range []reports.Outcome{
		reports.OutcomeNoAnswer, reports.OutcomeInterested, reports.OutcomeHighlyInterested,
		reports.OutcomeDisqualified, reports.OutcomeCallbackReq,
	} {
		if o == "" {
			t.Fatal("empty outcome constant")
		}
	}
}
