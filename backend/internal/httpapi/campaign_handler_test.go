package httpapi

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/tafakorpardazan/huginn/internal/campaigns"
	"github.com/tafakorpardazan/huginn/internal/projects"
)

// TestCampaignLifecycle exercises the campaign CRUD endpoints end-to-end:
// create → list → get (with projects) → update → delete. It guards against the
// campaigns route regressing to a 404 (as happened when the feature was dropped).
func TestCampaignLifecycle(t *testing.T) {
	db := integrationDB(t)

	campRepo := campaigns.NewRepository(db)
	campSvc := campaigns.NewService(campRepo)
	projRepo := projects.NewRepository(db)
	projSvc := projects.NewService(projRepo)

	handler := &campaignHandler{s: &Services{
		DB:        db,
		Campaigns: campSvc,
		Projects:  projSvc,
		Audit:     nil,
	}}

	r := chi.NewRouter()
	r.Post("/campaigns", handler.Create)
	r.Get("/campaigns", handler.List)
	r.Route("/campaigns/{campaignID}", func(r chi.Router) {
		r.Get("/", handler.Get)
		r.Put("/", handler.Update)
		r.Delete("/", handler.Delete)
	})

	// 1) Create.
	body := `{"title":"IT Campaign ` + ts() + `","objective":"win deals","description":"integration"}`
	req := httptest.NewRequest(http.MethodPost, "/campaigns", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)
	if rec.Code != http.StatusCreated {
		t.Fatalf("create: expected 201, got %d body=%s", rec.Code, rec.Body.String())
	}
	var created map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &created); err != nil {
		t.Fatalf("decode create body: %v", err)
	}
	cid, _ := created["id"].(string)
	if cid == "" {
		t.Fatalf("create must return an id: %v", created)
	}
	if created["status"] != "ACTIVE" {
		t.Fatalf("new campaign must default to ACTIVE: %v", created["status"])
	}

	// 2) List includes it.
	req = httptest.NewRequest(http.MethodGet, "/campaigns", nil)
	rec = httptest.NewRecorder()
	r.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("list: expected 200, got %d", rec.Code)
	}
	var list struct {
		Campaigns []map[string]any `json:"campaigns"`
		Total     int              `json:"total"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &list); err != nil {
		t.Fatalf("decode list: %v", err)
	}
	found := false
	for _, c := range list.Campaigns {
		if c["id"] == cid {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("created campaign not present in list: %v", list.Campaigns)
	}

	// 3) Get returns the campaign plus a projects array.
	req = httptest.NewRequest(http.MethodGet, "/campaigns/"+cid, nil)
	rec = httptest.NewRecorder()
	r.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("get: expected 200, got %d", rec.Code)
	}
	var detail struct {
		Campaign map[string]any   `json:"campaign"`
		Projects []map[string]any `json:"projects"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &detail); err != nil {
		t.Fatalf("decode get: %v", err)
	}
	if detail.Campaign["id"] != cid || detail.Projects == nil {
		t.Fatalf("get must return campaign + projects: %v", detail)
	}

	// 4) Update.
	req = httptest.NewRequest(http.MethodPut, "/campaigns/"+cid, strings.NewReader(`{"title":"IT Campaign Updated","status":"ARCHIVED"}`))
	req.Header.Set("Content-Type", "application/json")
	rec = httptest.NewRecorder()
	r.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("update: expected 200, got %d body=%s", rec.Code, rec.Body.String())
	}
	var updated map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &updated); err != nil {
		t.Fatalf("decode update: %v", err)
	}
	if updated["title"] != "IT Campaign Updated" || updated["status"] != "ARCHIVED" {
		t.Fatalf("update did not persist: %v", updated)
	}

	// 5) Delete → 204, then get → 404.
	req = httptest.NewRequest(http.MethodDelete, "/campaigns/"+cid, nil)
	rec = httptest.NewRecorder()
	r.ServeHTTP(rec, req)
	if rec.Code != http.StatusNoContent {
		t.Fatalf("delete: expected 204, got %d", rec.Code)
	}
	req = httptest.NewRequest(http.MethodGet, "/campaigns/"+cid, nil)
	rec = httptest.NewRecorder()
	r.ServeHTTP(rec, req)
	if rec.Code != http.StatusNotFound {
		t.Fatalf("get after delete: expected 404, got %d", rec.Code)
	}

	// 6) Create with empty title → 400 validation (not 404/500).
	req = httptest.NewRequest(http.MethodPost, "/campaigns", strings.NewReader(`{}`))
	req.Header.Set("Content-Type", "application/json")
	rec = httptest.NewRecorder()
	r.ServeHTTP(rec, req)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("create empty: expected 400, got %d body=%s", rec.Code, rec.Body.String())
	}
}
