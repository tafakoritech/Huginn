package worker

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"testing"

	"github.com/google/uuid"
	"github.com/tafakorpardazan/huginn/internal/documents"
	"github.com/tafakorpardazan/huginn/internal/imports"
	"github.com/tafakorpardazan/huginn/internal/llm"
	"github.com/tafakorpardazan/huginn/internal/queue"
)

func discardLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(io.Discard, nil))
}

func newUUID() string { return uuid.NewString() }

// ---- fakes ----

type fakeImportStore struct {
	doc       *imports.Document
	getErr    error
	getCalls  int

	started  []string
	finished []string

	listDocs []*imports.Document
	listErr  error

	batch    *imports.Batch
	batchErr error
}

func (f *fakeImportStore) GetDocument(ctx context.Context, id string) (*imports.Document, error) {
	f.getCalls++
	if f.getErr != nil {
		return nil, f.getErr
	}
	return f.doc, nil
}

func (f *fakeImportStore) GetBatch(ctx context.Context, id string) (*imports.Batch, error) {
	if f.batchErr != nil {
		return nil, f.batchErr
	}
	return f.batch, nil
}

func (f *fakeImportStore) ListDocuments(ctx context.Context, batchID string) ([]*imports.Document, error) {
	return f.listDocs, f.listErr
}

func (f *fakeImportStore) MarkBatchStarted(ctx context.Context, id string) error {
	f.started = append(f.started, id)
	return nil
}

func (f *fakeImportStore) MaybeFinishBatch(ctx context.Context, batchID string) error {
	f.finished = append(f.finished, batchID)
	return nil
}

type fakeDocProcessor struct {
	outcome *documents.ProcessOutcome
	err     error
	calls   int
}

func (f *fakeDocProcessor) ProcessDocument(ctx context.Context, docID string) (*documents.ProcessOutcome, error) {
	f.calls++
	if f.err != nil {
		return nil, f.err
	}
	return f.outcome, nil
}

func newTestService(store *fakeImportStore, docs *fakeDocProcessor) *Service {
	if store == nil {
		store = &fakeImportStore{}
	}
	if docs == nil {
		docs = &fakeDocProcessor{}
	}
	return &Service{logger: discardLogger(), imports: store, docs: docs}
}

// ---- Case A: stale document ----

func TestStaleDocumentIsPermanentNoBatchMutation(t *testing.T) {
	store := &fakeImportStore{getErr: imports.ErrNotFound}
	docs := &fakeDocProcessor{}
	svc := newTestService(store, docs)

	err := svc.handleDocumentExtract(context.Background(), queue.Job{
		Type:    queue.JobDocumentExtract,
		ID:      "stale",
		Payload: map[string]any{"document_id": "a8c442bb-32d6-4759-973c-3071b51298fb", "batch_id": newUUID()},
	})

	if !queue.IsPermanent(err) {
		t.Fatalf("missing document must be permanent, got %v", err)
	}
	if code := queue.PermanentCode(err); code != queue.CodeDocumentNotFound {
		t.Fatalf("expected DOCUMENT_NOT_FOUND, got %s", code)
	}
	if docs.calls != 0 {
		t.Fatalf("pipeline must not run for a missing document, ran %d times", docs.calls)
	}
	if len(store.started) != 0 {
		t.Fatalf("stale job must not mark batch started: %v", store.started)
	}
	if len(store.finished) != 0 {
		t.Fatalf("stale job must not finish a batch: %v", store.finished)
	}
}

func TestStaleDocumentOneAttemptNoRequeue(t *testing.T) {
	// A single delivery: permanent classification means the queue NACKs with
	// requeue=false. This guards the "no infinite retry" contract at the
	// handler boundary.
	store := &fakeImportStore{getErr: imports.ErrNotFound}
	svc := newTestService(store, nil)

	err := svc.handleDocumentExtract(context.Background(), queue.Job{
		Type:    queue.JobDocumentExtract,
		ID:      "stale",
		Payload: map[string]any{"document_id": newUUID(), "batch_id": newUUID()},
	})
	if !queue.IsPermanent(err) {
		t.Fatal("expected permanent classification for stale document")
	}
	if store.getCalls != 1 {
		t.Fatalf("stale job must be loaded exactly once, got %d", store.getCalls)
	}
}

// ---- Case B: transient repository failure is retryable ----

func TestTransientRepositoryFailureIsRetryable(t *testing.T) {
	store := &fakeImportStore{getErr: errors.New("postgres temporarily unavailable")}
	svc := newTestService(store, nil)

	err := svc.handleDocumentExtract(context.Background(), queue.Job{
		Type:    queue.JobDocumentExtract,
		ID:      "x",
		Payload: map[string]any{"document_id": newUUID(), "batch_id": newUUID()},
	})

	if queue.IsPermanent(err) {
		t.Fatalf("transient repository failure must be retryable, got permanent: %v", err)
	}
	if !queue.IsRetryable(err) {
		t.Fatalf("expected retryable classification, got %v", err)
	}
}

// ---- Case C: transient failure recovers on a later attempt ----

func TestTransientFailureRecoversOnRetry(t *testing.T) {
	store := &fakeImportStore{getErr: errors.New("connection reset")}
	svc := newTestService(store, nil)

	// First attempt fails transiently.
	err1 := svc.handleDocumentExtract(context.Background(), queue.Job{
		Type:    queue.JobDocumentExtract,
		ID:      "x",
		Payload: map[string]any{"document_id": newUUID(), "batch_id": newUUID()},
	})
	if queue.IsPermanent(err1) {
		t.Fatal("first attempt must be classified transient")
	}

	// Second attempt (repository recovered) succeeds.
	store.getErr = nil
	store.doc = &imports.Document{ID: newUUID(), BatchID: newUUID(), Status: "EXTRACTED"}
	docs := &fakeDocProcessor{outcome: &documents.ProcessOutcome{}}
	svc.docs = docs

	err2 := svc.handleDocumentExtract(context.Background(), queue.Job{
		Type:    queue.JobDocumentExtract,
		ID:      "x",
		Payload: map[string]any{"document_id": store.doc.ID, "batch_id": store.doc.BatchID},
	})
	if err2 != nil {
		t.Fatalf("recovered attempt must succeed, got %v", err2)
	}
	if len(store.finished) == 0 {
		t.Fatal("successful document must finalize its batch")
	}
}

// ---- Case D: persistent transient failure retried then DLQ (queue layer) ----
// Covered by TestHandleDeliveryRetryLimitExhausted in internal/queue.

// ---- Case E: valid existing document processes normally ----

func TestValidDocumentProcessesNormally(t *testing.T) {
	doc := &imports.Document{ID: newUUID(), BatchID: "batch-1", Status: "UPLOADED"}
	store := &fakeImportStore{doc: doc}
	docs := &fakeDocProcessor{outcome: &documents.ProcessOutcome{Created: 2, Duplicates: 1, Failed: 0}}
	svc := newTestService(store, docs)

	err := svc.handleDocumentExtract(context.Background(), queue.Job{
		Type:    queue.JobDocumentExtract,
		ID:      doc.ID,
		Payload: map[string]any{"document_id": doc.ID, "batch_id": doc.BatchID},
	})

	if err != nil {
		t.Fatalf("valid document must succeed, got %v", err)
	}
	if docs.calls != 1 {
		t.Fatalf("pipeline must run once for a valid document, got %d", docs.calls)
	}
	if len(store.started) != 1 || store.started[0] != doc.BatchID {
		t.Fatalf("batch must be marked started with document's batch_id: %v", store.started)
	}
	if len(store.finished) != 1 || store.finished[0] != doc.BatchID {
		t.Fatalf("batch must be finalized with document's batch_id: %v", store.finished)
	}
}

// ---- Already-processed documents are idempotently acked ----

func TestAnalyzedDocumentSkipped(t *testing.T) {
	doc := &imports.Document{ID: newUUID(), BatchID: "batch-1", Status: "ANALYZED"}
	store := &fakeImportStore{doc: doc}
	docs := &fakeDocProcessor{}
	svc := newTestService(store, docs)

	err := svc.handleDocumentExtract(context.Background(), queue.Job{
		Type:    queue.JobDocumentExtract,
		ID:      doc.ID,
		Payload: map[string]any{"document_id": doc.ID, "batch_id": doc.BatchID},
	})

	if err != nil {
		t.Fatalf("ANALYZED document must ack cleanly, got %v", err)
	}
	if docs.calls != 0 {
		t.Fatal("pipeline must not re-run for ANALYZED documents")
	}
	if len(store.started) != 0 || len(store.finished) != 0 {
		t.Fatal("skip must not touch batch state")
	}
}

// ---- Pipeline error classification ----

func TestNoContentIsPermanent(t *testing.T) {
	store := &fakeImportStore{doc: &imports.Document{ID: newUUID(), BatchID: "b", Status: "UPLOADED"}}
	docs := &fakeDocProcessor{err: documents.ErrNoContent}
	svc := newTestService(store, docs)

	err := svc.handleDocumentExtract(context.Background(), queue.Job{
		Type:    queue.JobDocumentExtract,
		ID:      store.doc.ID,
		Payload: map[string]any{"document_id": store.doc.ID, "batch_id": store.doc.BatchID},
	})

	if code := queue.PermanentCode(err); code != queue.CodeInvalidState {
		t.Fatalf("no-content must be INVALID_DOCUMENT_STATE, got %s", code)
	}
}

func TestLLMValidationIsPermanent(t *testing.T) {
	store := &fakeImportStore{doc: &imports.Document{ID: newUUID(), BatchID: "b", Status: "UPLOADED"}}
	docs := &fakeDocProcessor{err: &llm.ErrValidation{Purpose: llm.PurposeLeadExtraction, Err: errors.New("schema")}}
	svc := newTestService(store, docs)

	err := svc.handleDocumentExtract(context.Background(), queue.Job{
		Type:    queue.JobDocumentExtract,
		ID:      store.doc.ID,
		Payload: map[string]any{"document_id": store.doc.ID, "batch_id": store.doc.BatchID},
	})

	if code := queue.PermanentCode(err); code != queue.CodeValidation {
		t.Fatalf("LLM validation must be permanent, got %s", code)
	}
}

func TestLLMHTTP500IsTransient(t *testing.T) {
	store := &fakeImportStore{doc: &imports.Document{ID: newUUID(), BatchID: "b", Status: "UPLOADED"}}
	docs := &fakeDocProcessor{err: &llm.ErrProvider{Purpose: llm.PurposeLeadExtraction, StatusCode: 500, Err: errors.New("upstream")}}
	svc := newTestService(store, docs)

	err := svc.handleDocumentExtract(context.Background(), queue.Job{
		Type:    queue.JobDocumentExtract,
		ID:      store.doc.ID,
		Payload: map[string]any{"document_id": store.doc.ID, "batch_id": store.doc.BatchID},
	})

	if queue.IsPermanent(err) {
		t.Fatalf("LLM 500 must be transient, got %v", err)
	}
	if !queue.IsRetryable(err) {
		t.Fatal("LLM 500 must be retryable")
	}
}

func TestLLMHTTP401IsPermanent(t *testing.T) {
	store := &fakeImportStore{doc: &imports.Document{ID: newUUID(), BatchID: "b", Status: "UPLOADED"}}
	docs := &fakeDocProcessor{err: &llm.ErrProvider{Purpose: llm.PurposeLeadExtraction, StatusCode: 401, Err: errors.New("unauthorized")}}
	svc := newTestService(store, docs)

	err := svc.handleDocumentExtract(context.Background(), queue.Job{
		Type:    queue.JobDocumentExtract,
		ID:      store.doc.ID,
		Payload: map[string]any{"document_id": store.doc.ID, "batch_id": store.doc.BatchID},
	})

	if code := queue.PermanentCode(err); code != queue.CodeProviderRejected {
		t.Fatalf("LLM 401 must be permanently dead-lettered, got %s", code)
	}
}

// ---- Payload validation ----

func TestMissingDocumentIDIsPermanent(t *testing.T) {
	svc := newTestService(nil, nil)
	err := svc.handleDocumentExtract(context.Background(), queue.Job{
		Type: queue.JobDocumentExtract,
		ID:   "x",
	})
	if code := queue.PermanentCode(err); code != queue.CodeInvalidPayload {
		t.Fatalf("missing document_id must be INVALID_PAYLOAD, got %s", code)
	}
}

func TestMalformedDocumentIDIsPermanent(t *testing.T) {
	svc := newTestService(nil, nil)
	err := svc.handleDocumentExtract(context.Background(), queue.Job{
		Type:    queue.JobDocumentExtract,
		ID:      "x",
		Payload: map[string]any{"document_id": "not-a-uuid", "batch_id": newUUID()},
	})
	if code := queue.PermanentCode(err); code != queue.CodeInvalidUUID {
		t.Fatalf("malformed document_id must be INVALID_UUID, got %s", code)
	}
}

// ---- Batch analyze ----

func TestBatchAnalyzeMissingBatchIsPermanent(t *testing.T) {
	store := &fakeImportStore{batchErr: imports.ErrNotFound}
	svc := newTestService(store, nil)
	err := svc.handleBatchAnalyze(context.Background(), queue.Job{
		Type:    queue.JobBatchAnalyze,
		ID:      "b",
		Payload: map[string]any{"batch_id": newUUID()},
	})
	if code := queue.PermanentCode(err); code != queue.CodeBatchNotFound {
		t.Fatalf("missing batch must be BATCH_NOT_FOUND, got %s", code)
	}
}

func TestBatchAnalyzeInvalidPayloadIsPermanent(t *testing.T) {
	svc := newTestService(nil, nil)
	err := svc.handleBatchAnalyze(context.Background(), queue.Job{Type: queue.JobBatchAnalyze})
	if code := queue.PermanentCode(err); code != queue.CodeInvalidPayload {
		t.Fatalf("missing batch_id must be INVALID_PAYLOAD, got %s", code)
	}
}
