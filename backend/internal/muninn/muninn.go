package muninn

import (
	"context"
	"encoding/json"
	"time"

	"github.com/tafakorpardazan/huginn/internal/config"
	"github.com/tafakorpardazan/huginn/internal/database"
)

// Status of an export.
type Status string

const (
	StatusPending    Status = "PENDING"
	StatusInProgress Status = "IN_PROGRESS"
	StatusSucceeded  Status = "SUCCEEDED"
	StatusFailed     Status = "FAILED"
	StatusSkipped    Status = "SKIPPED"
)

// Export is the Huginn-side record of an export to Muninn.
type Export struct {
	ID           string     `json:"id"`
	HuginnLeadID string     `json:"huginn_lead_id"`
	MuninnLeadID *string    `json:"muninn_lead_id"`
	Status       Status     `json:"status"`
	AttemptCount int        `json:"attempt_count"`
	LastError    *string    `json:"last_error"`
	ExportedAt   *time.Time `json:"exported_at"`
	CreatedAt    time.Time  `json:"created_at"`
	UpdatedAt    time.Time  `json:"updated_at"`
}

// OutboxEvent is a persistent event for the outbox pattern.
type OutboxEvent struct {
	ID         string          `json:"id"`
	EventType  string          `json:"event_type"`
	Payload    map[string]any  `json:"payload"`
	Status     string          `json:"status"` // PENDING | PROCESSING | DONE | DEAD
	Attempts   int             `json:"attempts"`
	NextRetry  *time.Time      `json:"next_retry_at"`
	LastError  *string         `json:"last_error"`
	CreatedAt  time.Time       `json:"created_at"`
	UpdatedAt  time.Time       `json:"updated_at"`
}

// Repository persists exports and outbox events.
type Repository struct {
	db *database.DB
}

func NewRepository(db *database.DB) *Repository { return &Repository{db: db} }

func (r *Repository) GetExport(ctx context.Context, leadID string) (*Export, error) {
	row := r.db.Pool().QueryRow(ctx, `
		SELECT id, huginn_lead_id, muninn_lead_id, status, attempt_count, last_error, exported_at, created_at, updated_at
		FROM crm_exports WHERE huginn_lead_id=$1`, leadID)
	return scanExport(row)
}

func (r *Repository) CreateExport(ctx context.Context, leadID string, payload any) (*Export, error) {
	payloadJSON, _ := json.Marshal(payload)
	row := r.db.Pool().QueryRow(ctx, `
		INSERT INTO crm_exports (huginn_lead_id, status, payload)
		VALUES ($1,'PENDING',$2)
		ON CONFLICT (huginn_lead_id) DO NOTHING
		RETURNING id, huginn_lead_id, muninn_lead_id, status, attempt_count, last_error, exported_at, created_at, updated_at`,
		leadID, payloadJSON)
	return scanExport(row)
}

func scanExport(row database.RowScanner) (*Export, error) {
	var e Export
	err := row.Scan(&e.ID, &e.HuginnLeadID, &e.MuninnLeadID, &e.Status, &e.AttemptCount, &e.LastError, &e.ExportedAt, &e.CreatedAt, &e.UpdatedAt)
	if err != nil {
		return nil, err
	}
	return &e, nil
}

// MarkInProgress atomically claims an export for a worker.
func (r *Repository) MarkInProgress(ctx context.Context, id string) (bool, error) {
	tag, err := r.db.Pool().Exec(ctx, `
		UPDATE crm_exports SET status='IN_PROGRESS', attempt_count = attempt_count + 1, updated_at=now()
		WHERE id=$1 AND status IN ('PENDING','FAILED')`, id)
	if err != nil {
		return false, err
	}
	return tag.RowsAffected() == 1, nil
}

func (r *Repository) MarkSucceeded(ctx context.Context, id, muninnLeadID string) error {
	_, err := r.db.Pool().Exec(ctx, `
		UPDATE crm_exports SET status='SUCCEEDED', muninn_lead_id=$2, exported_at=now(), last_error=NULL, updated_at=now()
		WHERE id=$1`, id, muninnLeadID)
	return err
}

func (r *Repository) MarkFailed(ctx context.Context, id, errText string) error {
	_, err := r.db.Pool().Exec(ctx, `
		UPDATE crm_exports SET status='FAILED', last_error=$2, updated_at=now() WHERE id=$1`, id, errText)
	return err
}

func (r *Repository) ListPending(ctx context.Context, limit int) ([]*Export, error) {
	if limit == 0 {
		limit = 20
	}
	rows, err := r.db.Pool().Query(ctx, `
		SELECT id, huginn_lead_id, muninn_lead_id, status, attempt_count, last_error, exported_at, created_at, updated_at
		FROM crm_exports WHERE status IN ('PENDING','FAILED') AND attempt_count < 5 ORDER BY created_at LIMIT $1`, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []*Export{}
	for rows.Next() {
		e, err := scanExport(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, e)
	}
	return out, rows.Err()
}

// ---- Outbox ----

func (r *Repository) EnqueueOutbox(ctx context.Context, eventType string, payload any) (*OutboxEvent, error) {
	payloadJSON, _ := json.Marshal(payload)
	row := r.db.Pool().QueryRow(ctx, `
		INSERT INTO outbox_events (event_type, payload)
		VALUES ($1,$2)
		RETURNING id, event_type, payload, status, attempts, next_retry_at, last_error, created_at, updated_at`,
		eventType, payloadJSON)
	return scanOutbox(row)
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

func (r *Repository) ListPendingOutbox(ctx context.Context, limit int) ([]*OutboxEvent, error) {
	if limit == 0 {
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
		e, err := scanOutbox(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, e)
	}
	return out, rows.Err()
}

func scanOutbox(row database.RowScanner) (*OutboxEvent, error) {
	var e OutboxEvent
	var payload []byte
	if err := row.Scan(&e.ID, &e.EventType, &payload, &e.Status, &e.Attempts, &e.NextRetry, &e.LastError, &e.CreatedAt, &e.UpdatedAt); err != nil {
		return nil, err
	}
	_ = json.Unmarshal(payload, &e.Payload)
	return &e, nil
}

// Exporter is the Muninn CRM integration boundary. Huginn business logic never
// lives here; only HTTP transport to the external CRM API.
type Exporter interface {
	ExportQualifiedLead(ctx context.Context, payload *ExportPayload) (muninnLeadID string, err error)
}

// ExportPayload is what Huginn sends to Muninn.
type ExportPayload struct {
	HuginnLeadID    string             `json:"huginn_lead_id"`
	Company         *CompanyInfo       `json:"company"`
	Contacts        []*ContactInfo     `json:"contacts"`
	Qualification   *QualificationInfo `json:"qualification"`
	ImportantNotes  []string           `json:"important_notes"`
	Interactions    []*InteractionInfo `json:"interactions"`
	Source          *SourceInfo        `json:"source"`
	NextAction      string             `json:"next_action"`
	Score           int                `json:"score"`
}

type CompanyInfo struct {
	CompanyName  string  `json:"company_name"`
	BusinessType *string `json:"business_type"`
	Industry     *string `json:"industry"`
	City         *string `json:"city"`
	Province     *string `json:"province"`
	Address      *string `json:"address"`
	Website      *string `json:"website"`
	Instagram    *string `json:"instagram"`
	Description  *string `json:"description"`
}

type ContactInfo struct {
	FullName       *string `json:"full_name"`
	JobTitle       *string `json:"job_title"`
	Mobile         *string `json:"mobile"`
	Phone          *string `json:"phone"`
	Email          *string `json:"email"`
	IsDecisionMaker bool    `json:"is_decision_maker"`
}

type QualificationInfo struct {
	Score               int     `json:"score"`
	QualifiedAt         string  `json:"qualified_at"`
	Reason              string  `json:"reason"`
	NextAction          *string `json:"next_action"`
	QualificationNote   *string `json:"qualification_note"`
}

type InteractionInfo struct {
	Channel     string  `json:"channel"`
	CreatedAt   string  `json:"created_at"`
	Summary     *string `json:"summary"`
	Outcome     *string `json:"outcome"`
}

type SourceInfo struct {
	BatchName   *string `json:"batch_name"`
	SourceType  *string `json:"source_type"`
	DocumentName *string `json:"document_name"`
}

// HTTPExporter implements Exporter against Muninn's REST API.
type HTTPExporter struct {
	cfg config.MuninnConfig
	client *httpClient
}

// Service enforces export idempotency and queues exports.
type Service struct {
	repo     *Repository
	exporter Exporter
	leadSvc  LeadLister
}

// LeadLister provides lead data for the export payload.
type LeadLister interface {
	ExportBundle(ctx context.Context, leadID string) (*Bundle, error)
}

// ExportGuard provides the lead status for the export gate.
type ExportGuard interface {
	LeadStatus(ctx context.Context, leadID string) (string, error)
}

// Bundle is assembled lead data for export.
type Bundle struct {
	CompanyInfo   *CompanyInfo
	Contacts      []*ContactInfo
	Notes         []string
	Interactions  []*InteractionInfo
	Qualification *QualificationInfo
	Source        *SourceInfo
	NextAction    string
	Score         int
}

// Company returns the company payload.
func (b *Bundle) Company() *CompanyInfo { return b.CompanyInfo }

// NewService wires the export service.
func NewService(repo *Repository, exporter Exporter, leadSvc LeadLister) *Service {
	return &Service{repo: repo, exporter: exporter, leadSvc: leadSvc}
}

var (
	ErrAlreadyExported = errAlreadyExported
	ErrNotQualified    = errNotQualified
)

var (
	errAlreadyExported = &ExportError{code: "ALREADY_EXPORTED", msg: "lead has already been exported to Muninn"}
	errNotQualified    = &ExportError{code: "NOT_QUALIFIED", msg: "only qualified leads can be exported"}
)

// ExportError is a typed export error.
type ExportError struct {
	code string
	msg  string
}

func (e *ExportError) Error() string { return e.code + ": " + e.msg }
func (e *ExportError) Code() string  { return e.code }

// ExportQualifiedLead queues a qualified lead for export. Idempotent.
func (s *Service) ExportQualifiedLead(ctx context.Context, leadID, actorID string) (*Export, error) {
	existing, err := s.repo.GetExport(ctx, leadID)
	if err != nil && !database.IsNoRows(err) {
		return nil, err
	}
	if existing != nil {
		if existing.Status == StatusSucceeded {
			return existing, errAlreadyExported
		}
		return existing, nil // already queued
	}

	// Only qualified leads may enter the export pipeline.
	status, err := s.leadStatus(ctx, leadID)
	if err != nil {
		return nil, err
	}
	if status != "QUALIFIED" {
		return nil, errNotQualified
	}

	bundle, err := s.leadSvc.ExportBundle(ctx, leadID)
	if err != nil {
		return nil, err
	}
	if bundle.Score <= 0 {
		return nil, errNotQualified
	}

	payload := &ExportPayload{
		HuginnLeadID:   leadID,
		Company:        bundle.Company(),
		Contacts:       bundle.Contacts,
		Qualification:  bundle.Qualification,
		ImportantNotes: bundle.Notes,
		Interactions:   bundle.Interactions,
		Source:         bundle.Source,
		NextAction:     bundle.NextAction,
		Score:          bundle.Score,
	}

	exp, err := s.repo.CreateExport(ctx, leadID, payload)
	if err != nil {
		return nil, err
	}
	if _, err := s.repo.EnqueueOutbox(ctx, "muninn.export", map[string]any{
		"export_id": exp.ID,
		"lead_id":   leadID,
	}); err != nil {
		return nil, err
	}
	return exp, nil
}

// leadStatus reads the lead's current status from the source of truth.
func (s *Service) leadStatus(ctx context.Context, leadID string) (string, error) {
	var status string
	err := s.repo.db.Pool().QueryRow(ctx, `SELECT status FROM leads WHERE id=$1`, leadID).Scan(&status)
	if err != nil {
		return "", err
	}
	return status, nil
}

// ProcessExport performs a single export attempt with idempotency.
func (s *Service) ProcessExport(ctx context.Context, exportID string) error {
	// claim atomically
	claimed, err := s.claimExport(ctx, exportID)
	if err != nil || !claimed {
		return err
	}
	return s.processClaimed(ctx, exportID)
}
