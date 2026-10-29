package imports

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/tafakorpardazan/huginn/internal/database"
)

// Status of an import batch.
type Status string

const (
	StatusUploaded   Status = "UPLOADED"
	StatusQueued     Status = "QUEUED"
	StatusProcessing Status = "PROCESSING"
	StatusReviewReq  Status = "REVIEW_REQUIRED"
	StatusCompleted  Status = "COMPLETED"
	StatusFailed     Status = "FAILED"
)

// SourceType describes the source format.
type SourceType string

const (
	SourceCSV     SourceType = "CSV"
	SourceXLSX    SourceType = "XLSX"
	SourcePDF     SourceType = "PDF"
	SourceDOCX    SourceType = "DOCX"
	SourceTXT     SourceType = "TXT"
	SourceHTML    SourceType = "HTML"
	SourceJSON    SourceType = "JSON"
	SourceImage   SourceType = "IMAGE"
	SourceRawText SourceType = "RAW_TEXT"
	SourceURL     SourceType = "URL"
)

// Batch is an import batch of market data.
type Batch struct {
	ID                   string     `json:"id"`
	Name                 string     `json:"name"`
	Description          *string    `json:"description"`
	SourceType           SourceType `json:"source_type"`
	Status               Status     `json:"status"`
	CreatedBy            *string    `json:"created_by"`
	CreatedAt            time.Time  `json:"created_at"`
	ProcessingStartedAt  *time.Time `json:"processing_started_at"`
	ProcessingFinishedAt *time.Time `json:"processing_finished_at"`
	TotalDocuments       int        `json:"total_documents"`
	TotalCandidates      int        `json:"total_candidates"`
	CreatedLeads         int        `json:"created_leads"`
	DuplicateCandidates  int        `json:"duplicate_candidates"`
	FailedItems          int        `json:"failed_items"`
	ProjectID            *string    `json:"project_id"`
}

// Document is a stored source document within a batch.
type Document struct {
	ID               string    `json:"id"`
	BatchID          string    `json:"batch_id"`
	ObjectKey        *string   `json:"object_key"`
	Bucket           *string   `json:"bucket"`
	MimeType         *string   `json:"mime_type"`
	OriginalFilename *string   `json:"original_filename"`
	SizeBytes        *int64    `json:"size_bytes"`
	Checksum         *string   `json:"checksum"`
	Status           string    `json:"status"`
	ExtractedText    *string   `json:"extracted_text"`
	ExtractionError  *string   `json:"extraction_error"`
	CreatedBy        *string   `json:"created_by"`
	CreatedAt        time.Time `json:"created_at"`
	UpdatedAt        time.Time `json:"updated_at"`
	// LeadCount is the number of leads created from this document. Populated by
	// ListDocumentsWithCounts so client tables never render an undefined value.
	LeadCount int `json:"lead_count"`
}

// Repository persists batches and documents.
type Repository struct {
	db *database.DB
}

func NewRepository(db *database.DB) *Repository { return &Repository{db: db} }

var (
	ErrNotFound = errors.New("not found")
)

const batchCols = `id, name, description, source_type, status, created_by, created_at, processing_started_at,
	processing_finished_at, total_documents, total_candidates, created_leads, duplicate_candidates, failed_items, project_id`

func scanBatch(row database.RowScanner) (*Batch, error) {
	var b Batch
	err := row.Scan(&b.ID, &b.Name, &b.Description, &b.SourceType, &b.Status, &b.CreatedBy, &b.CreatedAt,
		&b.ProcessingStartedAt, &b.ProcessingFinishedAt, &b.TotalDocuments, &b.TotalCandidates,
		&b.CreatedLeads, &b.DuplicateCandidates, &b.FailedItems, &b.ProjectID)
	if err != nil {
		return nil, err
	}
	return &b, nil
}

func (r *Repository) CreateBatch(ctx context.Context, b *Batch) (*Batch, error) {
	row := r.db.Pool().QueryRow(ctx, `
		INSERT INTO import_batches (name, description, source_type, status, created_by, project_id)
		VALUES ($1,$2,$3,$4,$5,$6) RETURNING `+batchCols,
		b.Name, b.Description, b.SourceType, b.Status, b.CreatedBy, b.ProjectID)
	return scanBatch(row)
}

func (r *Repository) GetBatch(ctx context.Context, id string) (*Batch, error) {
	row := r.db.Pool().QueryRow(ctx, `SELECT `+batchCols+` FROM import_batches WHERE id=$1`, id)
	b, err := scanBatch(row)
	if err != nil {
		if database.IsNoRows(err) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	return b, nil
}

func (r *Repository) ListBatches(ctx context.Context, status *Status, limit, offset int) ([]*Batch, error) {
	q := `SELECT ` + batchCols + ` FROM import_batches`
	args := []any{}
	argc := 1
	if status != nil {
		q += fmt.Sprintf(` WHERE status=$%d`, argc)
		args = append(args, *status)
		argc++
	}
	q += fmt.Sprintf(` ORDER BY created_at DESC LIMIT $%d OFFSET $%d`, argc, argc+1)
	args = append(args, limit, offset)
	rows, err := r.db.Pool().Query(ctx, q, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []*Batch{}
	for rows.Next() {
		b, err := scanBatch(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, b)
	}
	return out, rows.Err()
}

func (r *Repository) UpdateBatchStatus(ctx context.Context, id string, status Status) error {
	_, err := r.db.Pool().Exec(ctx, `UPDATE import_batches SET status=$2 WHERE id=$1`, id, status)
	return err
}

func (r *Repository) MarkBatchStarted(ctx context.Context, id string) error {
	_, err := r.db.Pool().Exec(ctx, `UPDATE import_batches SET status='PROCESSING', processing_started_at=now() WHERE id=$1`, id)
	return err
}

func (r *Repository) MarkBatchFinished(ctx context.Context, id string) error {
	_, err := r.db.Pool().Exec(ctx, `UPDATE import_batches SET status='COMPLETED', processing_finished_at=now() WHERE id=$1`, id)
	return err
}

func (r *Repository) IncrementBatchCounters(ctx context.Context, id string, candidates, leads, dups, failed int) error {
	_, err := r.db.Pool().Exec(ctx, `
		UPDATE import_batches
		SET total_candidates = total_candidates + $2,
		    created_leads = created_leads + $3,
		    duplicate_candidates = duplicate_candidates + $4,
		    failed_items = failed_items + $5
		WHERE id=$1`, id, candidates, leads, dups, failed)
	return err
}

func (r *Repository) UpdateDocumentCount(ctx context.Context, id string) error {
	_, err := r.db.Pool().Exec(ctx, `
		UPDATE import_batches SET total_documents = (SELECT count(*) FROM source_documents WHERE batch_id=$1) WHERE id=$1`, id)
	return err
}

// ---- Documents ----

const docCols = `id, batch_id, object_key, bucket, mime_type, original_filename, size_bytes, checksum,
	status, extracted_text, extraction_error, created_by, created_at, updated_at`

func scanDoc(row database.RowScanner) (*Document, error) {
	var d Document
	err := row.Scan(&d.ID, &d.BatchID, &d.ObjectKey, &d.Bucket, &d.MimeType, &d.OriginalFilename,
		&d.SizeBytes, &d.Checksum, &d.Status, &d.ExtractedText, &d.ExtractionError,
		&d.CreatedBy, &d.CreatedAt, &d.UpdatedAt)
	if err != nil {
		return nil, err
	}
	return &d, nil
}

func (r *Repository) CreateDocument(ctx context.Context, d *Document) (*Document, error) {
	row := r.db.Pool().QueryRow(ctx, `
		INSERT INTO source_documents (batch_id, object_key, bucket, mime_type, original_filename, size_bytes, checksum, status, created_by)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9) RETURNING `+docCols,
		d.BatchID, d.ObjectKey, d.Bucket, d.MimeType, d.OriginalFilename, d.SizeBytes, d.Checksum, d.Status, d.CreatedBy)
	return scanDoc(row)
}

// CreateDocumentWithTrigger atomically persists the source document and
// enqueues a durable processing trigger (outbox) in a single transaction.
// If the queue publish later fails, the outbox event remains PENDING and the
// dispatcher retries — the processing trigger can never silently disappear.
func (r *Repository) CreateDocumentWithTrigger(ctx context.Context, batchID string, d *Document) (*Document, error) {
	var created *Document
	err := r.db.WithTx(ctx, func(q database.Querier) error {
		row := q.QueryRow(ctx, `
			INSERT INTO source_documents (batch_id, object_key, bucket, mime_type, original_filename, size_bytes, checksum, status, extracted_text, created_by)
			VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10) RETURNING `+docCols,
			d.BatchID, d.ObjectKey, d.Bucket, d.MimeType, d.OriginalFilename, d.SizeBytes, d.Checksum, d.Status, d.ExtractedText, d.CreatedBy)
		c, err := scanDoc(row)
		if err != nil {
			return err
		}
		payload, _ := json.Marshal(map[string]any{"document_id": c.ID, "batch_id": batchID})
		if _, err := q.Exec(ctx, `
			INSERT INTO outbox_events (event_type, payload)
			VALUES ('import.document.extract', $1)`, payload); err != nil {
			return err
		}
		// First document moves the batch into QUEUED; later documents keep it there.
		if _, err := q.Exec(ctx, `
			UPDATE import_batches SET status='QUEUED' WHERE id=$1 AND status='UPLOADED'`, batchID); err != nil {
			return err
		}
		if _, err := q.Exec(ctx, `
			UPDATE import_batches SET total_documents = (SELECT count(*) FROM source_documents WHERE batch_id=$1) WHERE id=$1`, batchID); err != nil {
			return err
		}
		created = c
		return nil
	})
	return created, err
}

// FinishBatch transitions a batch to a terminal status with a finished timestamp.
func (r *Repository) FinishBatch(ctx context.Context, id string, status Status) error {
	_, err := r.db.Pool().Exec(ctx, `
		UPDATE import_batches SET status=$2, processing_finished_at=now() WHERE id=$1`, id, status)
	return err
}

// BatchProgress reports how many documents in a batch are terminal.
func (r *Repository) BatchProgress(ctx context.Context, batchID string) (total, analyzed, failed int, err error) {
	err = r.db.Pool().QueryRow(ctx, `
		SELECT count(*),
		       count(*) FILTER (WHERE status='ANALYZED'),
		       count(*) FILTER (WHERE status='FAILED')
		FROM source_documents WHERE batch_id=$1`, batchID).Scan(&total, &analyzed, &failed)
	return
}

func (r *Repository) GetDocument(ctx context.Context, id string) (*Document, error) {
	row := r.db.Pool().QueryRow(ctx, `SELECT `+docCols+` FROM source_documents WHERE id=$1`, id)
	d, err := scanDoc(row)
	if err != nil {
		if database.IsNoRows(err) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	return d, nil
}

func (r *Repository) ListDocuments(ctx context.Context, batchID string) ([]*Document, error) {
	rows, err := r.db.Pool().Query(ctx, `SELECT `+docCols+` FROM source_documents WHERE batch_id=$1 ORDER BY created_at`, batchID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []*Document{}
	for rows.Next() {
		d, err := scanDoc(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, d)
	}
	return out, rows.Err()
}

// ListDocumentsWithCounts returns a batch's documents enriched with the number
// of leads created from each (lead_count). Used by the API contract so clients
// never receive a missing lead_count.
func (r *Repository) ListDocumentsWithCounts(ctx context.Context, batchID string) ([]*Document, error) {
	rows, err := r.db.Pool().Query(ctx, `
		SELECT d.id, d.batch_id, d.object_key, d.bucket, d.mime_type, d.original_filename, d.size_bytes,
		       d.checksum, d.status, d.extracted_text, d.extraction_error, d.created_by, d.created_at, d.updated_at,
		       (SELECT count(*) FROM leads l WHERE l.source_document_id = d.id)
		FROM source_documents d WHERE d.batch_id=$1 ORDER BY d.created_at`, batchID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []*Document{}
	for rows.Next() {
		var d Document
		if err := rows.Scan(&d.ID, &d.BatchID, &d.ObjectKey, &d.Bucket, &d.MimeType, &d.OriginalFilename,
			&d.SizeBytes, &d.Checksum, &d.Status, &d.ExtractedText, &d.ExtractionError,
			&d.CreatedBy, &d.CreatedAt, &d.UpdatedAt, &d.LeadCount); err != nil {
			return nil, err
		}
		out = append(out, &d)
	}
	return out, rows.Err()
}

func (r *Repository) UpdateDocumentStatus(ctx context.Context, id, status string) error {
	_, err := r.db.Pool().Exec(ctx, `UPDATE source_documents SET status=$2, updated_at=now() WHERE id=$1`, id, status)
	return err
}

func (r *Repository) SetDocumentExtracted(ctx context.Context, id, text string) error {
	_, err := r.db.Pool().Exec(ctx, `UPDATE source_documents SET status='EXTRACTED', extracted_text=$2, extraction_error=NULL, updated_at=now() WHERE id=$1`, id, text)
	return err
}

func (r *Repository) SetDocumentFailed(ctx context.Context, id, errText string) error {
	_, err := r.db.Pool().Exec(ctx, `UPDATE source_documents SET status='FAILED', extraction_error=$2, updated_at=now() WHERE id=$1`, id, errText)
	return err
}

func (r *Repository) SetDocumentQueued(ctx context.Context, id string) error {
	_, err := r.db.Pool().Exec(ctx, `UPDATE source_documents SET status='QUEUED', updated_at=now() WHERE id=$1`, id)
	return err
}

func (r *Repository) SetBatchQueued(ctx context.Context, id string) error {
	_, err := r.db.Pool().Exec(ctx, `UPDATE import_batches SET status='QUEUED' WHERE id=$1`, id)
	return err
}

func (r *Repository) DB() *database.DB { return r.db }

// ---- Outbox (durable processing triggers) ----

// OutboxEvent is a persisted processing trigger.
type OutboxEvent struct {
	ID        string
	EventType string
	Payload   map[string]any
	Status    string
	Attempts  int
	NextRetry *time.Time
	LastError *string
	CreatedAt time.Time
}

func (r *Repository) ListPendingOutbox(ctx context.Context, limit int) ([]*OutboxEvent, error) {
	if limit <= 0 || limit > 200 {
		limit = 50
	}
	rows, err := r.db.Pool().Query(ctx, `
		SELECT id, event_type, payload, status, attempts, next_retry_at, last_error, created_at, updated_at
		FROM outbox_events
		WHERE status='PENDING' AND (next_retry_at IS NULL OR next_retry_at <= now())
		ORDER BY created_at LIMIT $1`, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []*OutboxEvent{}
	for rows.Next() {
		var e OutboxEvent
		var payload []byte
		var updatedAt time.Time
		if err := rows.Scan(&e.ID, &e.EventType, &payload, &e.Status, &e.Attempts, &e.NextRetry, &e.LastError, &e.CreatedAt, &updatedAt); err != nil {
			return nil, err
		}
		_ = json.Unmarshal(payload, &e.Payload)
		out = append(out, &e)
	}
	return out, rows.Err()
}

func (r *Repository) ClaimOutbox(ctx context.Context, id string) (bool, error) {
	tag, err := r.db.Pool().Exec(ctx, `
		UPDATE outbox_events SET status='PROCESSING', attempts=attempts+1, updated_at=now()
		WHERE id=$1 AND status='PENDING'`, id)
	if err != nil {
		return false, err
	}
	return tag.RowsAffected() == 1, nil
}

func (r *Repository) FinishOutbox(ctx context.Context, id string) error {
	_, err := r.db.Pool().Exec(ctx, `UPDATE outbox_events SET status='DONE', updated_at=now() WHERE id=$1`, id)
	return err
}

func (r *Repository) FailOutbox(ctx context.Context, id, errText string, nextRetry *time.Time) error {
	_, err := r.db.Pool().Exec(ctx, `
		UPDATE outbox_events SET status='PENDING', last_error=$2, next_retry_at=$3, updated_at=now()
		WHERE id=$1`, id, errText, nextRetry)
	return err
}

func (r *Repository) DeadOutbox(ctx context.Context, id string) error {
	_, err := r.db.Pool().Exec(ctx, `UPDATE outbox_events SET status='DEAD', updated_at=now() WHERE id=$1`, id)
	return err
}

// EnsureSchema applies idempotent schema additions at startup. Import batches
// carry an optional project so all leads extracted from a batch inherit the
// chosen project as AI sales context.
func (r *Repository) EnsureSchema(ctx context.Context) error {
	if _, err := r.db.Pool().Exec(ctx, `ALTER TABLE import_batches ADD COLUMN IF NOT EXISTS project_id UUID`); err != nil {
		return err
	}
	_, err := r.db.Pool().Exec(ctx, `CREATE INDEX IF NOT EXISTS idx_import_batches_project ON import_batches (project_id)`)
	return err
}
