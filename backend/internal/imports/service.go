package imports

import (
	"context"
	"errors"
	"fmt"
	"io"
	"time"

	"github.com/tafakorpardazan/huginn/internal/config"
	"github.com/tafakorpardazan/huginn/internal/database"
	"github.com/tafakorpardazan/huginn/internal/queue"
	"github.com/tafakorpardazan/huginn/internal/storage"
)

// Service orchestrates import batches.
type Service struct {
	repo  *Repository
	store *storage.Store
	q     *queue.Client
	cfg   config.StorageConfig
}

func NewService(repo *Repository, store *storage.Store, q *queue.Client, cfg config.StorageConfig) *Service {
	return &Service{repo: repo, store: store, q: q, cfg: cfg}
}

var (
	ErrEmptyFile       = errors.New("no file provided")
	ErrFileTooLarge    = errors.New("file exceeds maximum allowed size")
	ErrBatchNotFound   = errors.New("batch not found")
)

// CreateBatchParams for a new import batch.
type CreateBatchParams struct {
	Name        string
	Description *string
	SourceType  SourceType
	CreatedBy   *string
	ProjectID   *string
}

// CreateBatch creates a UPLOADED batch.
func (s *Service) CreateBatch(ctx context.Context, p CreateBatchParams) (*Batch, error) {
	if p.Name == "" {
		return nil, errors.New("name is required")
	}
	b := &Batch{
		Name:       p.Name,
		Description: p.Description,
		SourceType: p.SourceType,
		Status:     StatusUploaded,
		CreatedBy:  p.CreatedBy,
		ProjectID:  p.ProjectID,
	}
	return s.repo.CreateBatch(ctx, b)
}

// AddDocument stores a file in MinIO and registers it in the batch. The
// document insert and its processing trigger (outbox) are atomic, so
// processing starts automatically after a successful upload.
func (s *Service) AddDocument(ctx context.Context, batchID, userID, filename, contentType string, size int64, r io.Reader) (*Document, error) {
	if filename == "" {
		return nil, ErrEmptyFile
	}
	if size > s.cfg.MaxUploadBytes {
		return nil, ErrFileTooLarge
	}
	if _, err := s.repo.GetBatch(ctx, batchID); err != nil {
		return nil, ErrBatchNotFound
	}

	objectKey := storage.NewObjectKey("documents", filename)
	obj, err := s.store.Put(ctx, r, size, contentType, objectKey)
	if err != nil {
		return nil, err
	}

	doc := &Document{
		BatchID:          batchID,
		ObjectKey:        &obj.Key,
		Bucket:           &obj.Bucket,
		MimeType:         &obj.MimeType,
		OriginalFilename: &filename,
		SizeBytes:        &obj.Size,
		Checksum:         &obj.Checksum,
		Status:           "UPLOADED",
		CreatedBy:        &userID,
	}
	return s.repo.CreateDocumentWithTrigger(ctx, batchID, doc)
}

// AddRawText registers a raw-text document (no file storage needed) and
// triggers automatic processing.
func (s *Service) AddRawText(ctx context.Context, batchID, userID, text string) (*Document, error) {
	if _, err := s.repo.GetBatch(ctx, batchID); err != nil {
		return nil, ErrBatchNotFound
	}
	doc := &Document{
		BatchID:       batchID,
		Status:        "EXTRACTED",
		ExtractedText: &text,
		CreatedBy:     &userID,
	}
	return s.repo.CreateDocumentWithTrigger(ctx, batchID, doc)
}

// MaybeFinishBatch transitions the batch to COMPLETED (or FAILED when any
// document failed) once every document has reached a terminal state.
func (s *Service) MaybeFinishBatch(ctx context.Context, batchID string) error {
	total, analyzed, failed, err := s.repo.BatchProgress(ctx, batchID)
	if err != nil {
		return err
	}
	if total == 0 || analyzed+failed < total {
		return nil
	}
	if failed > 0 {
		return s.repo.FinishBatch(ctx, batchID, StatusFailed)
	}
	return s.repo.FinishBatch(ctx, batchID, StatusCompleted)
}

// QueueBatch starts processing: marks QUEUED and publishes document.extract jobs.
func (s *Service) QueueBatch(ctx context.Context, batchID string) error {
	batch, err := s.repo.GetBatch(ctx, batchID)
	if err != nil {
		return ErrBatchNotFound
	}
	if batch.Status != StatusUploaded {
		return fmt.Errorf("batch must be UPLOADED to queue, currently %s", batch.Status)
	}
	docs, err := s.repo.ListDocuments(ctx, batchID)
	if err != nil {
		return err
	}
	if len(docs) == 0 {
		return errors.New("batch has no documents")
	}
	if err := s.repo.SetBatchQueued(ctx, batchID); err != nil {
		return err
	}
	for _, d := range docs {
		if err := s.repo.SetDocumentQueued(ctx, d.ID); err != nil {
			return err
		}
		job := queue.Job{
			Type: queue.JobDocumentExtract,
			ID:   d.ID,
			Payload: map[string]any{
				"document_id": d.ID,
				"batch_id":    batchID,
			},
			CreatedAt: time.Now(),
		}
		if err := s.q.Publish(ctx, job); err != nil {
			return err
		}
	}
	return nil
}

// GetDocument delegates.
func (s *Service) GetDocument(ctx context.Context, docID string) (*Document, error) {
	return s.repo.GetDocument(ctx, docID)
}

// GetBatch delegates.
func (s *Service) GetBatch(ctx context.Context, id string) (*Batch, error) {
	return s.repo.GetBatch(ctx, id)
}

// ListDocuments delegates.
func (s *Service) ListDocuments(ctx context.Context, batchID string) ([]*Document, error) {
	return s.repo.ListDocuments(ctx, batchID)
}

// ListBatches delegates.
func (s *Service) ListBatches(ctx context.Context, status *Status, limit, offset int) ([]*Batch, error) {
	return s.repo.ListBatches(ctx, status, limit, offset)
}

// UpdateBatchStatus delegates.
func (s *Service) UpdateBatchStatus(ctx context.Context, id string, status Status) error {
	return s.repo.UpdateBatchStatus(ctx, id, status)
}

// IncrementCounters delegates.
func (s *Service) IncrementCounters(ctx context.Context, id string, candidates, leads, dups, failed int) error {
	return s.repo.IncrementBatchCounters(ctx, id, candidates, leads, dups, failed)
}

// MarkBatchStarted delegates.
func (s *Service) MarkBatchStarted(ctx context.Context, id string) error {
	return s.repo.MarkBatchStarted(ctx, id)
}

// MarkBatchFinished delegates.
func (s *Service) MarkBatchFinished(ctx context.Context, id string) error {
	return s.repo.MarkBatchFinished(ctx, id)
}

// Repo exposes the repository for the worker.
func (s *Service) Repo() *Repository { return s.repo }

// ---- Outbox dispatch ----

func (s *Service) ListPendingOutbox(ctx context.Context, limit int) ([]*OutboxEvent, error) {
	return s.repo.ListPendingOutbox(ctx, limit)
}

func (s *Service) ClaimOutbox(ctx context.Context, id string) (bool, error) {
	return s.repo.ClaimOutbox(ctx, id)
}

func (s *Service) FinishOutbox(ctx context.Context, id string) error {
	return s.repo.FinishOutbox(ctx, id)
}

func (s *Service) FailOutbox(ctx context.Context, id, errText string, nextRetry *time.Time) error {
	return s.repo.FailOutbox(ctx, id, errText, nextRetry)
}

func (s *Service) DeadOutbox(ctx context.Context, id string) error {
	return s.repo.DeadOutbox(ctx, id)
}

// PresignedDocumentURL returns a temporary URL for a stored document.
func (s *Service) PresignedDocumentURL(ctx context.Context, docID string) (string, error) {
	doc, err := s.repo.GetDocument(ctx, docID)
	if err != nil {
		return "", err
	}
	if doc.ObjectKey == nil {
		return "", errors.New("document has no stored object")
	}
	return s.store.PresignedGetURL(ctx, *doc.ObjectKey)
}

var _ = database.ErrNoRows
