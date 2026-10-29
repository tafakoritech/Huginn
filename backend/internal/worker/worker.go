package worker

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"time"

	"github.com/google/uuid"
	"github.com/tafakorpardazan/huginn/internal/documents"
	"github.com/tafakorpardazan/huginn/internal/imports"
	"github.com/tafakorpardazan/huginn/internal/llm"
	"github.com/tafakorpardazan/huginn/internal/queue"
)

// DocumentProcessor processes a single source document through the pipeline.
type DocumentProcessor interface {
	ProcessDocument(ctx context.Context, docID string) (*documents.ProcessOutcome, error)
}

// ImportStore is the subset of the imports service the worker needs. Keeping
// it as an interface makes the worker unit-testable without a database.
type ImportStore interface {
	GetDocument(ctx context.Context, id string) (*imports.Document, error)
	GetBatch(ctx context.Context, id string) (*imports.Batch, error)
	ListDocuments(ctx context.Context, batchID string) ([]*imports.Document, error)
	MarkBatchStarted(ctx context.Context, id string) error
	MaybeFinishBatch(ctx context.Context, batchID string) error
}

// MuninnExporter processes a single CRM export.
type MuninnExporter interface {
	ProcessExport(ctx context.Context, exportID string) error
}

// Service wires the worker's job handlers.
type Service struct {
	logger    *slog.Logger
	docs      DocumentProcessor
	imports   ImportStore
	muninn    MuninnExporter
	queue     *queue.Client
}

func NewService(logger *slog.Logger, docs DocumentProcessor, importSvc ImportStore, muninn MuninnExporter, queue *queue.Client) *Service {
	return &Service{logger: logger, docs: docs, imports: importSvc, muninn: muninn, queue: queue}
}

// Handlers returns the job type -> handler mapping.
func (s *Service) Handlers() map[string]queue.Handler {
	return map[string]queue.Handler{
		queue.JobDocumentExtract: s.handleDocumentExtract,
		queue.JobBatchAnalyze:    s.handleBatchAnalyze,
		queue.JobMuninnExport:    s.handleMuninnExport,
	}
}

// handleDocumentExtract processes a document.extract job.
//
// Classification rules:
//   - missing/invalid payload identifiers        -> permanent (INVALID_PAYLOAD/INVALID_UUID)
//   - document no longer exists in the database  -> permanent (DOCUMENT_NOT_FOUND), never requeued
//   - document exists but has no extractable content -> permanent (INVALID_DOCUMENT_STATE)
//   - LLM provider rejected the request (4xx)    -> permanent (PROVIDER_REJECTED)
//   - everything else (DB/network/MinIO/Tika/5xx)-> transient, retried with backoff
//
// Batch bookkeeping only ever runs for a document that actually exists, and it
// is keyed by the document's own batch_id — never by a potentially stale
// payload value. A stale job therefore cannot mutate an unrelated batch.
func (s *Service) handleDocumentExtract(ctx context.Context, job queue.Job) error {
	docID, _ := job.Payload["document_id"].(string)
	payloadBatchID, _ := job.Payload["batch_id"].(string)

	if docID == "" {
		s.logger.Error("document job invalid payload",
			"stage", "parse",
			"document_id", docID,
			"error_code", queue.CodeInvalidPayload,
			"retryable", false,
			"action", "dead_letter",
		)
		return queue.NewPermanentf(queue.CodeInvalidPayload, "document_id missing in job payload")
	}
	if !validUUID(docID) {
		s.logger.Error("document job invalid uuid",
			"stage", "parse",
			"document_id", docID,
			"error_code", queue.CodeInvalidUUID,
			"retryable", false,
			"action", "dead_letter",
		)
		return queue.NewPermanentf(queue.CodeInvalidUUID, "document_id %q is not a valid UUID", docID)
	}

	// Load the document BEFORE any batch bookkeeping so a stale job (document
	// already deleted) can never touch batch state.
	doc, err := s.imports.GetDocument(ctx, docID)
	if err != nil {
		if errors.Is(err, imports.ErrNotFound) {
			s.logger.Error("document job permanently failed",
				"stage", "load_document",
				"document_id", docID,
				"batch_id", payloadBatchID,
				"error_code", queue.CodeDocumentNotFound,
				"retryable", false,
				"action", "dead_letter",
				"error", err.Error(),
			)
			return queue.NewPermanent(queue.CodeDocumentNotFound, err)
		}
		// Repository/network failure: transient, retry with backoff.
		s.logger.Warn("document job transient failure",
			"stage", "load_document",
			"document_id", docID,
			"attempt", job.Retry,
			"retryable", true,
			"action", "retry",
			"error", err.Error(),
		)
		return err
	}

	if doc.Status == "ANALYZED" {
		s.logger.Info("document already processed",
			"document_id", docID,
			"batch_id", doc.BatchID,
			"stage", "skip",
			"action", "ack",
		)
		return nil
	}

	s.logger.Info("extracting document",
		"document_id", docID,
		"batch_id", doc.BatchID,
		"stage", "process",
		"attempt", job.Retry,
	)
	if err := s.imports.MarkBatchStarted(ctx, doc.BatchID); err != nil {
		s.logger.Warn("mark batch started", "batch_id", doc.BatchID, "error", err)
	}

	outcome, err := s.docs.ProcessDocument(ctx, docID)
	if err != nil {
		classErr := classifyDocumentError(err)
		if queue.IsPermanent(classErr) {
			s.logger.Error("document job permanently failed",
				"stage", "process",
				"document_id", docID,
				"batch_id", doc.BatchID,
				"error_code", queue.PermanentCode(classErr),
				"retryable", false,
				"action", "dead_letter",
				"error", err.Error(),
			)
			return classErr
		}
		s.logger.Warn("document job transient failure",
			"stage", "process",
			"document_id", docID,
			"batch_id", doc.BatchID,
			"attempt", job.Retry,
			"retryable", true,
			"action", "retry",
			"error", err.Error(),
		)
		return err
	}

	s.logger.Info("document processed",
		"document_id", docID,
		"batch_id", doc.BatchID,
		"created", outcome.Created,
		"duplicates", outcome.Duplicates,
		"failed", outcome.Failed,
	)
	if fbErr := s.imports.MaybeFinishBatch(ctx, doc.BatchID); fbErr != nil {
		s.logger.Warn("mark batch finished", "batch_id", doc.BatchID, "error", fbErr)
	}
	return nil
}

// classifyDocumentError maps a pipeline error to its retryability. Domain
// sentinels become permanent; provider/transport errors stay transient.
func classifyDocumentError(err error) error {
	switch {
	case errors.Is(err, imports.ErrNotFound):
		return queue.NewPermanent(queue.CodeDocumentNotFound, err)
	case errors.Is(err, documents.ErrNoContent):
		return queue.NewPermanent(queue.CodeInvalidState, err)
	case errors.Is(err, llm.ErrNotConfigured):
		// Missing provider configuration cannot succeed on retry.
		return queue.NewPermanent(queue.CodeProviderRejected, err)
	default:
		var ve *llm.ErrValidation
		if errors.As(err, &ve) {
			// The runner already exhausted its internal repair retries;
			// re-running the same input cannot succeed.
			return queue.NewPermanent(queue.CodeValidation, err)
		}
		if !queue.IsRetryable(err) {
			return queue.NewPermanent(queue.CodeProviderRejected, err)
		}
		return err
	}
}

func (s *Service) handleBatchAnalyze(ctx context.Context, job queue.Job) error {
	batchID, _ := job.Payload["batch_id"].(string)
	if batchID == "" {
		return queue.NewPermanentf(queue.CodeInvalidPayload, "batch_id missing in job payload")
	}
	if !validUUID(batchID) {
		return queue.NewPermanentf(queue.CodeInvalidUUID, "batch_id %q is not a valid UUID", batchID)
	}
	if _, err := s.imports.GetBatch(ctx, batchID); err != nil {
		if errors.Is(err, imports.ErrNotFound) {
			s.logger.Error("batch analyze job permanently failed",
				"stage", "load_batch",
				"batch_id", batchID,
				"error_code", queue.CodeBatchNotFound,
				"retryable", false,
				"action", "dead_letter",
				"error", err.Error(),
			)
			return queue.NewPermanent(queue.CodeBatchNotFound, err)
		}
		return err // transient repository failure
	}
	docs, err := s.imports.ListDocuments(ctx, batchID)
	if err != nil {
		return err
	}
	total := 0
	for _, d := range docs {
		if d.Status == "ANALYZED" {
			continue
		}
		job := queue.Job{
			Type:      queue.JobDocumentExtract,
			ID:        d.ID,
			Payload:   map[string]any{"document_id": d.ID, "batch_id": batchID},
			CreatedAt: time.Now(),
		}
		if err := s.queue.Publish(ctx, job); err != nil {
			return err
		}
		total++
	}
	s.logger.Info("batch analysis dispatched", "batch_id", batchID, "dispatched", total)
	return nil
}

func (s *Service) handleMuninnExport(ctx context.Context, job queue.Job) error {
	exportID, _ := job.Payload["export_id"].(string)
	if exportID == "" {
		var pl struct {
			ExportID string `json:"export_id"`
		}
		if err := json.Unmarshal([]byte(fmtJSON(job.Payload)), &pl); err == nil {
			exportID = pl.ExportID
		}
	}
	if exportID == "" {
		return queue.NewPermanentf(queue.CodeInvalidPayload, "export_id missing in job payload")
	}
	if !validUUID(exportID) {
		return queue.NewPermanentf(queue.CodeInvalidUUID, "export_id %q is not a valid UUID", exportID)
	}
	s.logger.Info("processing muninn export", "export_id", exportID)
	return s.muninn.ProcessExport(ctx, exportID)
}

func validUUID(s string) bool {
	_, err := uuid.Parse(s)
	return err == nil
}

func fmtJSON(v any) string {
	b, _ := json.Marshal(v)
	return string(b)
}
