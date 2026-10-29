package httpapi

import (
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/tafakorpardazan/huginn/internal/auth"
	"github.com/tafakorpardazan/huginn/internal/interactions"
	"github.com/tafakorpardazan/huginn/internal/leads"
	"github.com/tafakorpardazan/huginn/internal/rbac"
	"github.com/tafakorpardazan/huginn/internal/reports"
	"github.com/tafakorpardazan/huginn/internal/users"
)

// TestMyReportsEndToEnd verifies the full submitted-report lifecycle:
//
//	submit report -> contact_reports row persisted
//	-> no longer pending
//	-> visible via ListForLead
//	-> visible via global /me/reports for the agent + manager
//	-> hidden from another agent
//	-> empty DB yields items=[] (never an error)
//	-> joins never produce ambiguous-column errors
func TestMyReportsEndToEnd(t *testing.T) {
	db := integrationDB(t)
	ctx := context.Background()

	userRepo := users.NewRepository(db)
	leadRepo := leads.NewRepository(db)
	interRepo := interactions.NewRepository(db)
	reportRepo := reports.NewRepository(db)
	svc := reports.NewService(reportRepo, interRepo)

	mkUser := func(role rbac.Role) *users.User {
		u, err := userRepo.Create(ctx, &users.User{
			ID:           uuid.NewString(),
			Name:         "reports-it-" + ts(),
			Username:     "reports_it_" + ts(),
			PasswordHash: "x",
			Role:         role,
			IsActive:     true,
		})
		if err != nil {
			t.Fatalf("create user: %v", err)
		}
		return u
	}
	agentA := mkUser(rbac.RoleSalesAgent)
	agentB := mkUser(rbac.RoleSalesAgent)
	manager := mkUser(rbac.RoleSalesDevManager)

	mkLead := func() *leads.Lead {
		l, err := leadRepo.Create(ctx, &leads.Lead{CompanyName: "شرکت گزارش " + ts()})
		if err != nil {
			t.Fatalf("create lead: %v", err)
		}
		return l
	}

	// Agent A: interaction + real submitted report.
	leadA := mkLead()
	interA, err := interRepo.Create(ctx, &interactions.Interaction{
		LeadID: leadA.ID, UserID: agentA.ID, Channel: interactions.ChannelCall,
		Direction: interactions.DirectionOutbound, ActionStatus: interactions.ActionCompleted,
		ReportStatus: interactions.ReportPending,
	})
	if err != nil {
		t.Fatalf("create interaction A: %v", err)
	}
	summaryA := "مشتری درباره قیمت سؤال کرد"
	interestA := reports.InterestMedium
	nextA := reports.NextSendInfo
	repA, err := svc.Submit(ctx, reports.SubmitParams{
		InteractionID: interA.ID, Outcome: reports.OutcomeInterested,
		Summary: &summaryA, InterestLevel: &interestA, NextAction: &nextA,
		SubmittedBy: agentA.ID,
	})
	if err != nil {
		t.Fatalf("submit report A: %v", err)
	}

	// Agent B: interaction + real submitted report (to prove exclusion).
	leadB := mkLead()
	interB, err := interRepo.Create(ctx, &interactions.Interaction{
		LeadID: leadB.ID, UserID: agentB.ID, Channel: interactions.ChannelCall,
		Direction: interactions.DirectionOutbound, ActionStatus: interactions.ActionCompleted,
		ReportStatus: interactions.ReportPending,
	})
	if err != nil {
		t.Fatalf("create interaction B: %v", err)
	}
	summaryB := "گزارش دوم"
	repB, err := svc.Submit(ctx, reports.SubmitParams{
		InteractionID: interB.ID, Outcome: reports.OutcomeNoAnswer,
		Summary: &summaryB, SubmittedBy: agentB.ID,
	})
	if err != nil {
		t.Fatalf("submit report B: %v", err)
	}

	// 1) contact_reports row exists after submission.
	got, err := reportRepo.GetByInteraction(ctx, interA.ID)
	if err != nil {
		t.Fatalf("GetByInteraction must find the persisted report: %v", err)
	}
	if got.ID != repA.ID || got.Outcome != reports.OutcomeInterested {
		t.Fatalf("persisted report mismatch: %+v", got)
	}

	// 2) interaction no longer pending.
	updated, err := interRepo.Get(ctx, interA.ID)
	if err != nil {
		t.Fatalf("get interaction: %v", err)
	}
	if updated.ReportStatus != interactions.ReportSubmitted {
		t.Fatalf("expected report_status SUBMITTED, got %s", updated.ReportStatus)
	}
	if pend, err := interRepo.PendingReportForUser(ctx, agentA.ID); err != nil || pend != nil {
		t.Fatalf("submitted report must disappear from pending-reports, pend=%v err=%v", pend, err)
	}

	// 3) ListForLead returns the submitted report (join is not ambiguous).
	forLead, err := reportRepo.ListForLead(ctx, leadA.ID)
	if err != nil {
		t.Fatalf("ListForLead join must not be ambiguous: %v", err)
	}
	if len(forLead) != 1 || forLead[0].ID != repA.ID {
		t.Fatalf("ListForLead must return the submitted report, got %+v", forLead)
	}

	getReports := func(principal *auth.Principal) ([]map[string]any, int) {
		t.Helper()
		h := &meHandler{reports: reports.NewRepository(db), logger: slog.New(slog.NewTextHandler(io.Discard, nil))}
		r := chi.NewRouter()
		r.Use(func(next http.Handler) http.Handler {
			return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
				next.ServeHTTP(w, req.WithContext(auth.WithPrincipal(req.Context(), principal)))
			})
		})
		r.Get("/api/v1/me/reports", h.MyReports)

		req := httptest.NewRequest(http.MethodGet, "/api/v1/me/reports", nil)
		rec := httptest.NewRecorder()
		r.ServeHTTP(rec, req)
		if rec.Code != http.StatusOK {
			t.Fatalf("expected 200 for /me/reports, got %d body=%s", rec.Code, rec.Body.String())
		}
		var body struct {
			Items []map[string]any `json:"items"`
			Total int              `json:"total"`
		}
		if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
			t.Fatalf("decode: %v", err)
		}
		return body.Items, body.Total
	}

	// 4) global /me/reports returns the report for its agent.
	itemsA, totalA := getReports(&auth.Principal{UserID: agentA.ID, Role: rbac.RoleSalesAgent})
	if totalA != 1 || len(itemsA) != 1 {
		t.Fatalf("agent A should see exactly their report, items=%d total=%d", len(itemsA), totalA)
	}
	if itemsA[0]["id"] != repA.ID {
		t.Fatalf("agent A report id mismatch: %v", itemsA[0]["id"])
	}
	if itemsA[0]["lead_id"] != leadA.ID || itemsA[0]["company_name"] != leadA.CompanyName {
		t.Fatalf("agent A report missing lead context: %v", itemsA[0])
	}
	if itemsA[0]["channel"] != string(interactions.ChannelCall) {
		t.Fatalf("agent A report missing channel: %v", itemsA[0])
	}

	// 5) different agent's report is excluded.
	itemsB, totalB := getReports(&auth.Principal{UserID: agentB.ID, Role: rbac.RoleSalesAgent})
	if totalB != 1 || len(itemsB) != 1 {
		t.Fatalf("agent B should see exactly their own report, items=%d total=%d", len(itemsB), totalB)
	}
	for _, it := range itemsB {
		if it["id"] == repA.ID {
			t.Fatal("agent B must not see agent A's report")
		}
	}

	// 6) manager visibility follows RBAC: manager sees all reports including both.
	itemsM, totalM := getReports(&auth.Principal{UserID: manager.ID, Role: rbac.RoleSalesDevManager})
	if totalM < 2 || len(itemsM) < 2 {
		t.Fatalf("manager should see both reports (and the team's), items=%d total=%d", len(itemsM), totalM)
	}
	managerIDs := map[string]bool{}
	for _, it := range itemsM {
		managerIDs[it["id"].(string)] = true
	}
	if !managerIDs[repA.ID] || !managerIDs[repB.ID] {
		t.Fatal("manager must see both submitted reports in the global list")
	}

	// 7) empty database returns [] without error.
	lonely := mkUser(rbac.RoleSalesAgent)
	itemsL, totalL := getReports(&auth.Principal{UserID: lonely.ID, Role: rbac.RoleSalesAgent})
	if totalL != 0 || itemsL == nil || len(itemsL) != 0 {
		t.Fatalf("user with no reports must get items=[] total=0, got items=%v total=%d", itemsL, totalL)
	}
}
