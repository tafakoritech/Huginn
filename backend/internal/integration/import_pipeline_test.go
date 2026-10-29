package integration

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"testing"

	"github.com/tafakorpardazan/huginn/internal/contacts"
	"github.com/tafakorpardazan/huginn/internal/database"
	"github.com/tafakorpardazan/huginn/internal/documents"
	"github.com/tafakorpardazan/huginn/internal/duplicates"
	"github.com/tafakorpardazan/huginn/internal/evidence"
	"github.com/tafakorpardazan/huginn/internal/imports"
	"github.com/tafakorpardazan/huginn/internal/leads"
	"github.com/tafakorpardazan/huginn/internal/llm"
	"github.com/tafakorpardazan/huginn/internal/rbac"
	"github.com/tafakorpardazan/huginn/internal/scoring"
	"github.com/tafakorpardazan/huginn/internal/users"
)

// mustCreateUser creates an active agent and returns its id.
func mustCreateUser(t *testing.T, db *database.DB, name string) string {
	t.Helper()
	repo := users.NewRepository(db)
	u, err := repo.Create(context.Background(), &users.User{
		Name: name, Username: "it_imp_" + name + "_" + ts(), Role: rbac.RoleSalesAgent, IsActive: true, PasswordHash: "x",
	})
	if err != nil {
		t.Fatalf("create user: %v", err)
	}
	return u.ID
}

// fakeLLMRunner scripts a candidate list so the whole import pipeline can be
// exercised against real infrastructure without an external LLM provider.
type fakeLLMRunner struct {
	candidatesJSON string
	calls          int
	fail           bool
}

func (f *fakeLLMRunner) GenerateStructured(ctx context.Context, req llm.Request, out any) (*llm.Run, error) {
	f.calls++
	if f.fail {
		return &llm.Run{Model: "fake", Status: "failed"}, errors.New("upstream failure")
	}
	if err := json.Unmarshal([]byte(f.candidatesJSON), out); err != nil {
		return &llm.Run{Model: "fake", Status: "failed"}, &llm.ErrValidation{Purpose: req.Purpose, Err: err}
	}
	return &llm.Run{Model: "fake", Status: "success", StructuredOutput: []byte(f.candidatesJSON)}, nil
}

func (f *fakeLLMRunner) GenerateText(ctx context.Context, req llm.Request) (*llm.Run, error) {
	return nil, errors.New("unused")
}

// cleanupPipelineData removes rows created by previous runs of these tests so
// the pipeline assertions are deterministic regardless of run order.
func cleanupPipelineData(t *testing.T, db *database.DB) {
	t.Helper()
	ctx := context.Background()
	stmts := []string{
		`DELETE FROM lead_evidence WHERE lead_id IN (SELECT id FROM leads WHERE source_batch_id IN (SELECT id FROM import_batches WHERE name LIKE 'IT %'))`,
		`DELETE FROM contacts WHERE lead_id IN (SELECT id FROM leads WHERE source_batch_id IN (SELECT id FROM import_batches WHERE name LIKE 'IT %'))`,
		`DELETE FROM contacts WHERE lead_id IN (SELECT id FROM leads WHERE company_name LIKE '%(قدیمی)')`,
		`DELETE FROM possible_duplicates WHERE lead_a_id IN (SELECT id FROM leads WHERE source_batch_id IN (SELECT id FROM import_batches WHERE name LIKE 'IT %')) OR lead_b_id IN (SELECT id FROM leads WHERE source_batch_id IN (SELECT id FROM import_batches WHERE name LIKE 'IT %'))`,
		`DELETE FROM possible_duplicates WHERE lead_a_id IN (SELECT id FROM leads WHERE company_name LIKE '%(قدیمی)') OR lead_b_id IN (SELECT id FROM leads WHERE company_name LIKE '%(قدیمی)')`,
		`DELETE FROM leads WHERE source_batch_id IN (SELECT id FROM import_batches WHERE name LIKE 'IT %')`,
		`DELETE FROM leads WHERE company_name LIKE '%(قدیمی)'`,
		`DELETE FROM source_documents WHERE batch_id IN (SELECT id FROM import_batches WHERE name LIKE 'IT %')`,
		`DELETE FROM import_batches WHERE name LIKE 'IT %'`,
	}
	for _, q := range stmts {
		if _, err := db.Pool().Exec(ctx, q); err != nil {
			t.Fatalf("cleanup: %v", err)
		}
	}
}

// TestImportPipelineEndToEnd verifies that real document text reaches the
// extraction layer and that candidates flow through parsing -> deduplication
// (before creation) -> lead creation with correct statistics.
func TestImportPipelineEndToEnd(t *testing.T) {
	db := setupDB(t)
	cleanupPipelineData(t, db)
	ctx := context.Background()
	cfg := mustConfig(t)

	text, err := os.ReadFile("../imports/testdata/companies.txt")
	if err != nil {
		t.Fatalf("read fixture: %v", err)
	}

	// ---- Seed two existing leads whose mobiles will be duplicate candidates.
	leadRepo := leads.NewRepository(db)
	contactRepo := contacts.NewRepository(db)
	seed := func(name, mobile string) string {
		l, err := leadRepo.Create(ctx, &leads.Lead{CompanyName: name, Status: leads.StatusNew})
		if err != nil {
			t.Fatalf("seed lead: %v", err)
		}
		if _, err := contactRepo.Create(ctx, &contacts.Contact{
			LeadID: l.ID, FullName: strP("مالک"), Mobile: strP(mobile),
		}); err != nil {
			t.Fatalf("seed contact: %v", err)
		}
		return l.ID
	}
	dupMobile1 := "09357578808" // عروسک‌سازی پارس
	dupMobile2 := "09144443322" // صنایع چوب نوین
	seed("عروسک‌سازی پارس (قدیمی)", dupMobile1)
	seed("صنایع چوب نوین (قدیمی)", dupMobile2)

	// ---- Build the pipeline with a fake LLM that returns 12 candidates,
	// two of which share the seeded mobiles (must become duplicates).
	fake := &fakeLLMRunner{candidatesJSON: twelveCandidatesJSON(t, db)}
	importRepo := imports.NewRepository(db)
	extractionSvc := imports.NewExtractionService(fake, leadRepo, contactRepo,
		evidence.NewRepository(db), duplicates.NewDetector(leadRepo, duplicates.NewRepository(db), db),
		scoring.NewRepository(db), importRepo)
	importSvc := imports.NewService(importRepo, nil, nil, cfg.Storage)
	docSvc := documents.NewService(nil, importSvc, extractionSvc)

	// ---- Create a batch and add the document as raw text (no Tika needed).
	userID := mustCreateUser(t, db, "pipeline")
	batch, err := importSvc.CreateBatch(ctx, imports.CreateBatchParams{
		Name: "IT pipeline " + ts(), SourceType: imports.SourceRawText, CreatedBy: &userID,
	})
	if err != nil {
		t.Fatalf("create batch: %v", err)
	}
	if _, err := importSvc.AddRawText(ctx, batch.ID, userID, string(text)); err != nil {
		t.Fatalf("add raw text: %v", err)
	}

	// ---- Process the document through the pipeline.
	docs, err := importRepo.ListDocuments(ctx, batch.ID)
	if err != nil || len(docs) != 1 {
		t.Fatalf("expected 1 document, got %d err=%v", len(docs), err)
	}
	outcome, err := docSvc.ProcessDocument(ctx, docs[0].ID)
	if err != nil {
		t.Fatalf("process document: %v", err)
	}

	if outcome.Candidates != 12 {
		t.Fatalf("expected 12 extracted candidates, got %d", outcome.Candidates)
	}
	if outcome.Duplicates != 2 {
		t.Fatalf("expected 2 duplicates, got %d", outcome.Duplicates)
	}
	if outcome.Created != 10 {
		t.Fatalf("expected 10 created leads, got %d", outcome.Created)
	}
	if outcome.Invalid != 0 || outcome.Failed != 0 {
		t.Fatalf("expected 0 invalid/failed, got invalid=%d failed=%d", outcome.Invalid, outcome.Failed)
	}
	if fake.calls == 0 {
		t.Fatal("the LLM/extractor was never called")
	}

	// ---- Batch counters must match the actual processing.
	b, err := importRepo.GetBatch(ctx, batch.ID)
	if err != nil {
		t.Fatalf("get batch: %v", err)
	}
	if b.TotalCandidates != 12 || b.CreatedLeads != 10 || b.DuplicateCandidates != 2 || b.FailedItems != 0 {
		t.Fatalf("batch counters mismatch: candidates=%d created=%d dups=%d failed=%d",
			b.TotalCandidates, b.CreatedLeads, b.DuplicateCandidates, b.FailedItems)
	}

	// ---- Exactly 10 leads were created from this document (duplicates were
	// NOT created), and the lead_count contract is populated.
	var docLeadCount int
	if err := db.Pool().QueryRow(ctx,
		`SELECT count(*) FROM leads WHERE source_document_id=$1`, docs[0].ID).Scan(&docLeadCount); err != nil {
		t.Fatalf("count doc leads: %v", err)
	}
	if docLeadCount != 10 {
		t.Fatalf("expected 10 leads for document, got %d (duplicates must not be created)", docLeadCount)
	}
	withCounts, err := importRepo.ListDocumentsWithCounts(ctx, batch.ID)
	if err != nil || len(withCounts) != 1 {
		t.Fatalf("list docs with counts: err=%v len=%d", err, len(withCounts))
	}
	if withCounts[0].LeadCount != 10 {
		t.Fatalf("lead_count contract wrong: %d", withCounts[0].LeadCount)
	}

	// ---- The two duplicate candidates were NOT created: no lead exists with
	// the duplicate-candidate names (dedup happens before creation). The seeds
	// carry a "(قدیمی)" suffix, so an exact-name hit means a duplicate was
	// incorrectly created.
	var dupNames int
	if err := db.Pool().QueryRow(ctx,
		`SELECT count(*) FROM leads WHERE company_name IN ('عروسک‌سازی پارس','صنایع چوب نوین')`).Scan(&dupNames); err != nil {
		t.Fatalf("count duplicate-candidate leads: %v", err)
	}
	if dupNames != 0 {
		t.Fatalf("duplicate candidates must not create leads; found %d leads with duplicate-candidate names", dupNames)
	}
}

func TestImportPipelineLLMFailureReportsFailure(t *testing.T) {
	db := setupDB(t)
	cleanupPipelineData(t, db)
	ctx := context.Background()
	cfg := mustConfig(t)

	fake := &fakeLLMRunner{fail: true}
	leadRepo := leads.NewRepository(db)
	contactRepo := contacts.NewRepository(db)
	importRepo := imports.NewRepository(db)
	extractionSvc := imports.NewExtractionService(fake, leadRepo, contactRepo,
		evidence.NewRepository(db), duplicates.NewDetector(leadRepo, duplicates.NewRepository(db), db),
		scoring.NewRepository(db), importRepo)
	importSvc := imports.NewService(importRepo, nil, nil, cfg.Storage)
	docSvc := documents.NewService(nil, importSvc, extractionSvc)

	userID := mustCreateUser(t, db, "fail")
	batch, err := importSvc.CreateBatch(ctx, imports.CreateBatchParams{Name: "IT fail " + ts(), SourceType: imports.SourceRawText, CreatedBy: &userID})
	if err != nil {
		t.Fatalf("create batch: %v", err)
	}
	if _, err := importSvc.AddRawText(ctx, batch.ID, userID, "شرکت تست یک"); err != nil {
		t.Fatalf("add raw text: %v", err)
	}
	docs, _ := importRepo.ListDocuments(ctx, batch.ID)

	if _, err := docSvc.ProcessDocument(ctx, docs[0].ID); err == nil {
		t.Fatal("LLM failure must fail the document, not produce a fake success")
	}
	doc, err := importRepo.GetDocument(ctx, docs[0].ID)
	if err != nil || doc.Status != "FAILED" {
		t.Fatalf("document must be FAILED after LLM failure, got %+v err=%v", doc, err)
	}
	// The worker calls MaybeFinishBatch after a document failure.
	if err := importSvc.MaybeFinishBatch(ctx, batch.ID); err != nil {
		t.Fatalf("maybe finish batch: %v", err)
	}
	b, err := importRepo.GetBatch(ctx, batch.ID)
	if err != nil || b.Status != imports.StatusFailed {
		t.Fatalf("batch must be FAILED after LLM failure, got %v err=%v", b.Status, err)
	}
}

func twelveCandidatesJSON(t *testing.T, db *database.DB) string {
	t.Helper()
	ctx := context.Background()
	// Pick 12 candidate mobiles that do NOT collide with any pre-existing lead
	// contact, so only the two seeded duplicates match deterministically.
	taken := map[string]bool{}
	rows, err := db.Pool().Query(ctx, `SELECT DISTINCT mobile_normalized FROM contacts WHERE mobile_normalized IS NOT NULL`)
	if err != nil {
		t.Fatalf("query taken mobiles: %v", err)
	}
	for rows.Next() {
		var v string
		_ = rows.Scan(&v)
		taken[v] = true
	}
	rows.Close()
	taken["+989357578808"] = true // seeded duplicate 1
	taken["+989144443322"] = true // seeded duplicate 2

	mobiles := []string{}
	base := 9100000000
	for i := 0; i < 2000 && len(mobiles) < 12; i++ {
		candidate := fmt.Sprintf("+98%d", base+i)
		if !taken[candidate] {
			mobiles = append(mobiles, candidate)
		}
	}
	if len(mobiles) < 12 {
		t.Fatalf("could not find 12 unused mobiles")
	}

	names := []string{
		"عروسک‌سازی پارس", "صنایع چوب نوین", "گروه صنعتی فولاد آریا", "پخش مواد غذایی البرز",
		"استارتاپ لجستیک هوشمند راهینو", "آموزشگاه زبان سفیر", "تولیدی پوشاک سپید", "کارگاه سفال گلستان",
		"شرکت دارویی آوا فارما", "مزرعه گل بهاران", "شرکت نساجی ابریشم کاشان", "کارگاه تزئینات نقشین",
	}
	set := struct {
		Candidates []imports.Candidate `json:"candidates"`
	}{}
	for i, n := range names {
		// the first two candidates reuse the seeded mobiles -> expected duplicates
		used := mobiles[i]
		if i == 0 {
			used = "09357578808"
		}
		if i == 1 {
			used = "09144443322"
		}
		set.Candidates = append(set.Candidates, imports.Candidate{
			CompanyName:  n,
			BusinessType: strP("تولیدکننده"),
			City:         strP("تهران"),
			Contacts: []imports.CandidateContact{{
				FullName: strP("محمد رضایی"),
				Mobile:   strP(used),
				Email:    strP(fmt.Sprintf("info@c%d.ir", i)),
			}},
		})
	}
	b, _ := json.Marshal(set)
	return string(b)
}

func strP(s string) *string { return &s }
