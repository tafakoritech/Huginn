package httpapi

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/tafakorpardazan/huginn/internal/campaigns"
	"github.com/tafakorpardazan/huginn/internal/leads"
	"github.com/tafakorpardazan/huginn/internal/projects"
	"github.com/tafakorpardazan/huginn/internal/timeline"
)

// TestProjectLeadAttachDetach exercises the project<->lead association endpoints:
// attaching an existing lead to a project sets leads.project_id, and detaching it
// clears the column while preserving the lead row (the mission's Phase 3 & 5).
func TestProjectLeadAttachDetach(t *testing.T) {
	db := integrationDB(t)
	ctx := context.Background()

	leadRepo := leads.NewRepository(db)
	leadSvc := leads.NewService(leadRepo)
	projRepo := projects.NewRepository(db)
	projSvc := projects.NewService(projRepo)

	handler := &projectHandler{s: &Services{
		DB:        db,
		Projects:  projSvc,
		Leads:     leadSvc,
		LeadRepo:  leadRepo,
		Timeline:  timeline.NewRepository(db),
		Audit:     nil,
	}}

	proj, err := projSvc.Create(ctx, projects.CreateParams{Title: "IT Proj " + ts(), Description: "integration"})
	if err != nil {
		t.Fatalf("create project: %v", err)
	}
	lead, err := leadSvc.Create(ctx, leads.CreateParams{CompanyName: "IT Proj Lead " + ts()})
	if err != nil {
		t.Fatalf("create lead: %v", err)
	}

	r := chi.NewRouter()
	r.Post("/projects/{projectID}/leads/{leadID}", handler.AttachLead)
	r.Delete("/projects/{projectID}/leads/{leadID}", handler.DetachLead)

	// 1) Attach existing lead to the project.
	req := httptest.NewRequest(http.MethodPost, "/projects/"+proj.ID+"/leads/"+lead.ID, nil)
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("attach: expected 200, got %d body=%s", rec.Code, rec.Body.String())
	}
	var body map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode attach body: %v", err)
	}
	if body["success"] != true || body["lead_id"] != lead.ID || body["project_id"] != proj.ID {
		t.Fatalf("unexpected attach response: %v", body)
	}

	got, err := leadSvc.Get(ctx, lead.ID)
	if err != nil {
		t.Fatalf("get lead after attach: %v", err)
	}
	if got.ProjectID == nil || *got.ProjectID != proj.ID {
		t.Fatalf("expected lead.project_id=%s after attach, got %v", proj.ID, got.ProjectID)
	}

	// 2) The lead must now appear in the project's related leads.
	detailReq := httptest.NewRequest(http.MethodGet, "/projects/"+proj.ID, nil)
	detail := handler.projectDetail(detailReq, proj)
	rl, ok := detail["related_leads"].(*leads.ListResult)
	if !ok || len(rl.Leads) != 1 || rl.Leads[0].ID != lead.ID {
		t.Fatalf("expected the lead in related_leads, got %v", detail["related_leads"])
	}

	// 3) Detach: project_id cleared, lead row preserved.
	req = httptest.NewRequest(http.MethodDelete, "/projects/"+proj.ID+"/leads/"+lead.ID, nil)
	rec = httptest.NewRecorder()
	r.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("detach: expected 200, got %d body=%s", rec.Code, rec.Body.String())
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode detach body: %v", err)
	}
	if body["success"] != true || body["project_id"] != nil {
		t.Fatalf("unexpected detach response: %v", body)
	}

	got, err = leadSvc.Get(ctx, lead.ID)
	if err != nil {
		t.Fatalf("get lead after detach: %v", err)
	}
	if got.ProjectID != nil {
		t.Fatalf("expected lead.project_id to be NULL after detach, got %v", got.ProjectID)
	}

	// 4) Attach with unknown project/lead returns a clean 404.
	req = httptest.NewRequest(http.MethodPost, "/projects/00000000-0000-0000-0000-000000000000/leads/"+lead.ID, nil)
	rec = httptest.NewRecorder()
	r.ServeHTTP(rec, req)
	if rec.Code != http.StatusNotFound {
		t.Fatalf("unknown project: expected 404, got %d", rec.Code)
	}
	req = httptest.NewRequest(http.MethodPost, "/projects/"+proj.ID+"/leads/00000000-0000-0000-0000-000000000000", nil)
	rec = httptest.NewRecorder()
	r.ServeHTTP(rec, req)
	if rec.Code != http.StatusNotFound {
		t.Fatalf("unknown lead: expected 404, got %d", rec.Code)
	}
}

// TestProjectCampaignAssociation exercises the campaign_id association end to
// end: a project created under a campaign persists the reference, the project
// GET and campaign GET read paths both reflect it, reassignment works via PUT,
// and invalid/nonexistent campaign ids are rejected.
func TestProjectCampaignAssociation(t *testing.T) {
	db := integrationDB(t)
	ctx := context.Background()

	campRepo := campaigns.NewRepository(db)
	campSvc := campaigns.NewService(campRepo)
	leadRepo := leads.NewRepository(db)
	leadSvc := leads.NewService(leadRepo)
	projRepo := projects.NewRepository(db)
	projSvc := projects.NewService(projRepo)

	handler := &projectHandler{s: &Services{
		DB:        db,
		Projects:  projSvc,
		Campaigns: campSvc,
		Leads:     leadSvc,
		LeadRepo:  leadRepo,
		Timeline:  timeline.NewRepository(db),
		Audit:     nil,
	}}
	campH := newCampaignHandler(&Services{Campaigns: campSvc})

	campA, err := campSvc.Create(ctx, campaigns.CreateParams{Title: "IT Campaign A " + ts()})
	if err != nil {
		t.Fatalf("create campaign A: %v", err)
	}
	campB, err := campSvc.Create(ctx, campaigns.CreateParams{Title: "IT Campaign B " + ts()})
	if err != nil {
		t.Fatalf("create campaign B: %v", err)
	}

	r := chi.NewRouter()
	r.Post("/projects", handler.Create)
	r.Get("/projects/{projectID}", handler.Get)
	r.Put("/projects/{projectID}", handler.Update)
	r.Get("/campaigns/{campaignID}", campH.Get)

	type projectBody struct {
		Project struct {
			ID         string  `json:"id"`
			Title      string  `json:"title"`
			CampaignID *string `json:"campaign_id"`
		} `json:"project"`
	}

	// 1) Create with a valid campaign → 201, campaign_id returned.
	body := fmt.Sprintf(`{"campaign_id": %q, "title": "IT Proj %s", "description": "integration", "product_name": "ریلز رولز", "sales_objective": "فروش ۵۰۰ دستگاه"}`, campA.ID, ts())
	req := httptest.NewRequest(http.MethodPost, "/projects", strings.NewReader(body))
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)
	if rec.Code != http.StatusCreated {
		t.Fatalf("create with campaign: expected 201, got %d body=%s", rec.Code, rec.Body.String())
	}
	var created projectBody
	if err := json.Unmarshal(rec.Body.Bytes(), &created); err != nil {
		t.Fatalf("decode create response: %v", err)
	}
	if created.Project.CampaignID == nil || *created.Project.CampaignID != campA.ID {
		t.Fatalf("create must return campaign_id=%s, got %v", campA.ID, created.Project.CampaignID)
	}
	projID := created.Project.ID
	if projID == "" {
		t.Fatal("create must return a project id")
	}

	// 2) GET project returns the persisted campaign association.
	req = httptest.NewRequest(http.MethodGet, "/projects/"+projID, nil)
	rec = httptest.NewRecorder()
	r.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("get project: expected 200, got %d body=%s", rec.Code, rec.Body.String())
	}
	var got projectBody
	if err := json.Unmarshal(rec.Body.Bytes(), &got); err != nil {
		t.Fatalf("decode get response: %v", err)
	}
	if got.Project.CampaignID == nil || *got.Project.CampaignID != campA.ID {
		t.Fatalf("get project must return campaign_id=%s, got %v", campA.ID, got.Project.CampaignID)
	}

	// 3) GET campaign lists the newly created project under the campaign.
	var campDetail struct {
		Projects []struct {
			ID string `json:"id"`
		} `json:"projects"`
	}
	req = httptest.NewRequest(http.MethodGet, "/campaigns/"+campA.ID, nil)
	rec = httptest.NewRecorder()
	r.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("get campaign: expected 200, got %d", rec.Code)
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &campDetail); err != nil {
		t.Fatalf("decode campaign detail: %v", err)
	}
	found := false
	for _, p := range campDetail.Projects {
		if p.ID == projID {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("campaign detail must list the project; got %v", campDetail.Projects)
	}

	// 4) Create with an invalid UUID → 400 validation error.
	req = httptest.NewRequest(http.MethodPost, "/projects", strings.NewReader(`{"campaign_id":"not-a-uuid","title":"IT Bad","description":"x"}`))
	rec = httptest.NewRecorder()
	r.ServeHTTP(rec, req)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("invalid campaign uuid: expected 400, got %d body=%s", rec.Code, rec.Body.String())
	}

	// 5) Create with a nonexistent campaign → 404.
	req = httptest.NewRequest(http.MethodPost, "/projects", strings.NewReader(`{"campaign_id":"00000000-0000-0000-0000-000000000000","title":"IT Missing","description":"x"}`))
	rec = httptest.NewRecorder()
	r.ServeHTTP(rec, req)
	if rec.Code != http.StatusNotFound {
		t.Fatalf("nonexistent campaign: expected 404, got %d body=%s", rec.Code, rec.Body.String())
	}

	// 6) Reassign the project to another campaign via PUT.
	req = httptest.NewRequest(http.MethodPut, "/projects/"+projID, strings.NewReader(fmt.Sprintf(`{"title":"IT Proj updated","description":"integration","campaign_id":%q}`, campB.ID)))
	rec = httptest.NewRecorder()
	r.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("reassign campaign: expected 200, got %d body=%s", rec.Code, rec.Body.String())
	}
	var updated projectBody
	if err := json.Unmarshal(rec.Body.Bytes(), &updated); err != nil {
		t.Fatalf("decode update response: %v", err)
	}
	if updated.Project.CampaignID == nil || *updated.Project.CampaignID != campB.ID {
		t.Fatalf("update must return campaign_id=%s, got %v", campB.ID, updated.Project.CampaignID)
	}

	// 7) The reassigned project now appears under campaign B (and no longer A).
	req = httptest.NewRequest(http.MethodGet, "/campaigns/"+campB.ID, nil)
	rec = httptest.NewRecorder()
	r.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("get campaign B: expected 200, got %d", rec.Code)
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &campDetail); err != nil {
		t.Fatalf("decode campaign B detail: %v", err)
	}
	found = false
	for _, p := range campDetail.Projects {
		if p.ID == projID {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("campaign B must list the reassigned project; got %v", campDetail.Projects)
	}
}
