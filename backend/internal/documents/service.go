package documents

import (
	"context"
	"log/slog"

	"github.com/tafakorpardazan/huginn/internal/imports"
)

// Service orchestrates document extraction and the extraction pipeline.
type Service struct {
	extractor *Extractor
	imports   *imports.Service
	llmSvc    *imports.ExtractionService
	logger    *slog.Logger
}

func NewService(extractor *Extractor, importSvc *imports.Service, llmSvc *imports.ExtractionService) *Service {
	return &Service{extractor: extractor, imports: importSvc, llmSvc: llmSvc, logger: slog.Default()}
}

// SetLogger attaches a structured logger for pipeline observability.
func (s *Service) SetLogger(l *slog.Logger) {
	if l != nil {
		s.logger = l
	}
}

// ProcessDocument performs extraction + LLM analysis for a single document.
// Idempotent: documents already ANALYZED are skipped so retried jobs never
// create duplicate leads or double-increment batch counters.
func (s *Service) ProcessDocument(ctx context.Context, docID string) (*ProcessOutcome, error) {
	doc, err := s.imports.GetDocument(ctx, docID)
	if err != nil {
		return nil, err
	}
	if doc.Status == "ANALYZED" {
		s.logger.Info("document already processed", "document_id", docID, "batch_id", doc.BatchID, "stage", "skip")
		return &ProcessOutcome{}, nil
	}
	if doc.Status == "EXTRACTED" && doc.ExtractedText != nil {
		return s.analyze(ctx, doc)
	}
	if doc.ObjectKey == nil {
		// raw text documents have no object
		if doc.ExtractedText != nil {
			return s.analyze(ctx, doc)
		}
		return nil, ErrNoContent
	}

	s.logger.Info("extracting document text",
		"document_id", docID, "batch_id", doc.BatchID, "stage", "extract",
		"object_key", orEmpty(doc.ObjectKey), "mime_type", orEmpty(doc.MimeType))
	res, err := s.extractor.Extract(ctx, *doc.ObjectKey)
	if err != nil {
		_ = s.imports.Repo().SetDocumentFailed(ctx, docID, err.Error())
		return nil, err
	}
	if err := s.imports.Repo().SetDocumentExtracted(ctx, docID, res.Text); err != nil {
		return nil, err
	}
	s.logger.Info("document text extracted",
		"document_id", docID, "batch_id", doc.BatchID, "stage", "extract", "text_bytes", len(res.Text))
	doc.ExtractedText = &res.Text
	doc.Status = "EXTRACTED"
	return s.analyze(ctx, doc)
}

func (s *Service) analyze(ctx context.Context, doc *imports.Document) (*ProcessOutcome, error) {
	outcome, err := s.llmSvc.ProcessDocumentText(ctx, doc, stringOrEmpty(doc.ExtractedText))
	if err != nil {
		_ = s.imports.Repo().SetDocumentFailed(ctx, doc.ID, err.Error())
		return nil, err
	}
	if err := s.imports.Repo().UpdateDocumentStatus(ctx, doc.ID, "ANALYZED"); err != nil {
		return nil, err
	}
	// Batch counters use precise semantics:
	//   total_candidates    = raw candidates the LLM returned (extracted)
	//   created_leads       = leads actually created
	//   duplicate_candidates= candidates matching an existing lead (not created)
	//   failed_items        = validation rejects + creation errors
	if err := s.imports.IncrementCounters(ctx, doc.BatchID, outcome.Extracted, outcome.Created, outcome.Duplicates, outcome.Invalid+outcome.Failed); err != nil {
		s.logger.Warn("increment batch counters", "batch_id", doc.BatchID, "error", err)
	}
	return &ProcessOutcome{
		Candidates: outcome.Extracted,
		Valid:      outcome.Valid,
		Invalid:    outcome.Invalid,
		Created:    outcome.Created,
		Duplicates: outcome.Duplicates,
		Failed:     outcome.Failed,
	}, nil
}

// ProcessOutcome summarizes document processing.
type ProcessOutcome struct {
	Candidates int `json:"candidates"`
	Valid      int `json:"valid"`
	Invalid    int `json:"invalid"`
	Created    int `json:"created"`
	Duplicates int `json:"duplicates"`
	Failed     int `json:"failed"`
}

func stringOrEmpty(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}

func orEmpty(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}

type errNoContentError struct{}

func (errNoContentError) Error() string { return "document has no extractable content" }

// ErrNoContent is returned when a document exists but has no extractable
// content. It is a permanent condition: retrying cannot change the input.
var ErrNoContent error = errNoContentError{}
