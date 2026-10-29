package httpapi

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/tafakorpardazan/huginn/internal/config"
	"github.com/tafakorpardazan/huginn/internal/database"
	"github.com/tafakorpardazan/huginn/internal/interactions"
	"github.com/tafakorpardazan/huginn/internal/leads"
	"github.com/tafakorpardazan/huginn/internal/rbac"
	"github.com/tafakorpardazan/huginn/internal/reports"
	"github.com/tafakorpardazan/huginn/internal/users"
)

func ts() string { return fmt.Sprintf("%d", time.Now().UnixNano()) }

// These tests hit a real database and are skipped unless HUGINN_INTEGRATION=1,
// matching the convention in internal/integration.
func integrationDB(t *testing.T) *database.DB {
	t.Helper()
	if os.Getenv("HUGINN_INTEGRATION") != "1" {
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

// TestReportGetRegression is a regression test for the dashboard report-detail
// 404: ReportGet used COALESCE(cr.overridden_by,'') on a uuid column, which
// raises "invalid input syntax for type uuid" whenever overridden_by is NULL
// (i.e. every report that was not manager-overridden), surfacing as a false 404.
func TestReportGetRegression(t *testing.T) {
	db := integrationDB(t)
	ctx := context.Background()

	userRepo := users.NewRepository(db)
	leadRepo := leads.NewRepository(db)
	interRepo := interactions.NewRepository(db)
	reportRepo := reports.NewRepository(db)

	agent, err := userRepo.Create(ctx, &users.User{
		Name: "IT Report Agent", Username: "it_report_" + ts(), Role: rbac.RoleSalesAgent,
		IsActive: true, PasswordHash: "x",
	})
	if err != nil {
		t.Fatalf("create agent: %v", err)
	}

	lead, err := leadRepo.Create(ctx, &leads.Lead{CompanyName: "IT Report Lead " + ts(), Status: leads.StatusNew})
	if err != nil {
		t.Fatalf("create lead: %v", err)
	}

	inter, err := interRepo.Create(ctx, &interactions.Interaction{
		LeadID: lead.ID, UserID: agent.ID, Channel: interactions.ChannelCall,
		Direction: interactions.DirectionOutbound, ActionStatus: interactions.ActionCreated,
		ReportStatus: interactions.ReportPending,
	})
	if err != nil {
		t.Fatalf("create interaction: %v", err)
	}

	// A report without a manager override => overridden_by is NULL (the bug path).
	summary := "IT regression summary"
	rep, err := reportRepo.Create(ctx, &reports.Report{
		InteractionID: inter.ID, Outcome: reports.OutcomeNoAnswer, SubmittedBy: agent.ID, Summary: &summary,
	})
	if err != nil {
		t.Fatalf("create report: %v", err)
	}

	handler := &dataHandler{s: &Services{DB: db}}
	r := chi.NewRouter()
	r.Get("/reports/{reportID}", handler.ReportGet)

	// 1) Existing report with NULL overridden_by must return 200 with the report.
	req := httptest.NewRequest(http.MethodGet, "/reports/"+rep.ID, nil)
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200 for existing report, got %d body=%s", rec.Code, rec.Body.String())
	}
	var body map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode body: %v", err)
	}
	if body["id"] != rep.ID {
		t.Fatalf("expected id %s, got %v", rep.ID, body["id"])
	}
	if ov, ok := body["overridden_by"].(string); !ok || ov != "" {
		t.Fatalf("expected overridden_by to be empty string, got %#v", body["overridden_by"])
	}
	interaction, ok := body["interaction"].(map[string]any)
	if !ok {
		t.Fatalf("missing interaction in response: %v", body)
	}
	if interaction["lead_id"] != lead.ID {
		t.Fatalf("expected lead_id %s, got %v", lead.ID, interaction["lead_id"])
	}

	// 2) Unknown report id must return a clean 404 NOT_FOUND (not a 500).
	req = httptest.NewRequest(http.MethodGet, "/reports/00000000-0000-0000-0000-000000000000", nil)
	rec = httptest.NewRecorder()
	r.ServeHTTP(rec, req)
	if rec.Code != http.StatusNotFound {
		t.Fatalf("expected 404 for missing report, got %d body=%s", rec.Code, rec.Body.String())
	}
	var errBody map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &errBody); err != nil {
		t.Fatalf("decode error body: %v", err)
	}
	if errBody["error"] == nil {
		t.Fatalf("expected an error envelope, got %v", errBody)
	}
}
