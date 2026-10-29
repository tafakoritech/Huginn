package followups

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/tafakorpardazan/huginn/internal/database"
)

// Status of a follow-up. OVERDUE is deliberately NOT a stored status: a
// follow-up is overdue when it is pending and due_at < now() (computed), so
// completed/cancelled items never appear as overdue.
type Status string

const (
	StatusPending   Status = "PENDING"
	StatusCompleted Status = "COMPLETED"
	StatusCancelled Status = "CANCELLED"
	StatusSnoozed   Status = "SNOOZED"
)

// Priority of a follow-up.
type Priority string

const (
	PriorityLow    Priority = "LOW"
	PriorityMedium Priority = "MEDIUM"
	PriorityHigh   Priority = "HIGH"
)

// Source of a follow-up.
type Source string

const (
	SourceManual Source = "MANUAL"
	SourceReport Source = "REPORT"
	SourceSystem Source = "SYSTEM"
)

// FollowUp is an explicit scheduled follow-up tied to a lead.
type FollowUp struct {
	ID             string     `json:"id"`
	LeadID         string     `json:"lead_id"`
	ContactID      *string    `json:"contact_id,omitempty"`
	ProjectID      *string    `json:"project_id,omitempty"`
	CampaignID     *string    `json:"campaign_id,omitempty"`
	InteractionID  *string    `json:"interaction_id"`
	AssignedTo     *string    `json:"assigned_to"`
	CreatedBy      *string    `json:"created_by"`
	Title          *string    `json:"title"`
	Description    *string    `json:"description"`
	Channel        *string    `json:"channel"`
	Type           string     `json:"type"`
	Priority       Priority   `json:"priority"`
	Status         Status     `json:"status"`
	DueAt          time.Time  `json:"due_at"`
	CompletedAt    *time.Time `json:"completed_at"`
	SnoozedUntil   *time.Time `json:"snoozed_until"`
	CompletionNote *string    `json:"completion_note"`
	Outcome        *string    `json:"outcome"`
	Source         Source     `json:"source"`
	CreatedAt      time.Time  `json:"created_at"`
	UpdatedAt      time.Time  `json:"updated_at"`
	// IsOverdue is computed (pending AND due_at < now) and only set on list responses.
	IsOverdue bool `json:"is_overdue,omitempty"`
}

// ListItem enriches a FollowUp with display names for lists.
type ListItem struct {
	FollowUp
	CompanyName  string  `json:"company_name"`
	ContactName  *string `json:"contact_name"`
	AssigneeName *string `json:"assignee_name"`
}

const cols = `id, lead_id, contact_id, project_id, campaign_id, interaction_id, assigned_to, created_by,
	title, description, channel, type, priority, status, due_at, completed_at, snoozed_until,
	completion_note, outcome, source, created_at, updated_at`

const colsQualified = `f.id, f.lead_id, f.contact_id, f.project_id, f.campaign_id, f.interaction_id, f.assigned_to, f.created_by,
	f.title, f.description, f.channel, f.type, f.priority, f.status, f.due_at, f.completed_at, f.snoozed_until,
	f.completion_note, f.outcome, f.source, f.created_at, f.updated_at`

// normalizeUTC converts every timestamp to UTC so the API always serializes
// RFC3339 with the Z suffix regardless of the server's local timezone. The
// database stores TIMESTAMPTZ (UTC); clients then render in their own zone.
func (f *FollowUp) normalizeUTC() {
	f.DueAt = f.DueAt.UTC()
	f.CreatedAt = f.CreatedAt.UTC()
	f.UpdatedAt = f.UpdatedAt.UTC()
	if f.CompletedAt != nil {
		u := f.CompletedAt.UTC()
		f.CompletedAt = &u
	}
	if f.SnoozedUntil != nil {
		u := f.SnoozedUntil.UTC()
		f.SnoozedUntil = &u
	}
}

func scan(row database.RowScanner) (*FollowUp, error) {
	var f FollowUp
	err := row.Scan(&f.ID, &f.LeadID, &f.ContactID, &f.ProjectID, &f.CampaignID, &f.InteractionID,
		&f.AssignedTo, &f.CreatedBy, &f.Title, &f.Description, &f.Channel, &f.Type, &f.Priority,
		&f.Status, &f.DueAt, &f.CompletedAt, &f.SnoozedUntil, &f.CompletionNote, &f.Outcome,
		&f.Source, &f.CreatedAt, &f.UpdatedAt)
	if err != nil {
		return nil, err
	}
	f.normalizeUTC()
	return &f, nil
}

// queryOne runs a single-row statement and scans the resulting FollowUp. It uses
// Query (not QueryRow) because the write-returning statements are re-prepared by
// pgx's statement cache and QueryRow is unreliable for parameter-type inference
// after another statement ran on the same pooled connection.
func queryOne(ctx context.Context, q database.Querier, sql string, args ...any) (*FollowUp, error) {
	rows, err := q.Query(ctx, sql, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	if !rows.Next() {
		return nil, database.ErrNoRows
	}
	return scan(rows)
}

func scanListItem(row database.RowScanner) (*ListItem, error) {
	var f FollowUp
	var company string
	var contactName, assigneeName *string
	err := row.Scan(&f.ID, &f.LeadID, &f.ContactID, &f.ProjectID, &f.CampaignID, &f.InteractionID,
		&f.AssignedTo, &f.CreatedBy, &f.Title, &f.Description, &f.Channel, &f.Type, &f.Priority,
		&f.Status, &f.DueAt, &f.CompletedAt, &f.SnoozedUntil, &f.CompletionNote, &f.Outcome,
		&f.Source, &f.CreatedAt, &f.UpdatedAt, &company, &contactName, &assigneeName)
	if err != nil {
		return nil, err
	}
	f.IsOverdue = f.Status == StatusPending && f.DueAt.Before(time.Now().UTC())
	f.normalizeUTC()
	return &ListItem{FollowUp: f, CompanyName: company, ContactName: contactName, AssigneeName: assigneeName}, nil
}

// Repository persists follow-ups.
type Repository struct {
	db *database.DB
}

func NewRepository(db *database.DB) *Repository { return &Repository{db: db} }

func (r *Repository) Create(ctx context.Context, f *FollowUp) (*FollowUp, error) {
	if f.Type == "" {
		f.Type = "FOLLOW_UP"
	}
	if f.Status == "" {
		f.Status = StatusPending
	}
	if f.Priority == "" {
		f.Priority = PriorityMedium
	}
	if f.Source == "" {
		f.Source = SourceManual
	}
	return queryOne(ctx, r.db.Pool(), `
		INSERT INTO follow_ups (lead_id, contact_id, project_id, campaign_id, interaction_id, assigned_to,
			created_by, title, description, channel, type, priority, status, due_at, snoozed_until,
			completion_note, outcome, source)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15,$16,$17,$18)
		RETURNING `+cols,
		f.LeadID, f.ContactID, f.ProjectID, f.CampaignID, f.InteractionID, f.AssignedTo,
		f.CreatedBy, f.Title, f.Description, f.Channel, f.Type, f.Priority, f.Status, f.DueAt,
		f.SnoozedUntil, f.CompletionNote, f.Outcome, f.Source)
}

func (r *Repository) Get(ctx context.Context, id string) (*FollowUp, error) {
	row := r.db.Pool().QueryRow(ctx, `SELECT `+cols+` FROM follow_ups WHERE id=$1`, id)
	return scan(row)
}

func (r *Repository) Update(ctx context.Context, f *FollowUp) (*FollowUp, error) {
	return queryOne(ctx, r.db.Pool(), `
		UPDATE follow_ups SET
			contact_id=COALESCE($2, contact_id),
			assigned_to=COALESCE($3, assigned_to),
			title=COALESCE($4, title),
			description=COALESCE($5, description),
			channel=COALESCE($6, channel),
			type=COALESCE($7, type),
			priority=COALESCE($8, priority),
			due_at=COALESCE($9, due_at),
			status=COALESCE($10::followup_status, status),
			snoozed_until=COALESCE($11, snoozed_until),
			completion_note=COALESCE($12, completion_note),
			outcome=COALESCE($13, outcome),
			completed_at=COALESCE($14, completed_at),
			updated_at=now()
		WHERE id=$1
		RETURNING `+cols,
		f.ID, f.ContactID, f.AssignedTo, f.Title, f.Description, f.Channel, f.Type, f.Priority,
		f.DueAt, f.Status, f.SnoozedUntil, f.CompletionNote, f.Outcome, f.CompletedAt)
}

func (r *Repository) Complete(ctx context.Context, id string, note, outcome *string) (*FollowUp, error) {
	return queryOne(ctx, r.db.Pool(), `
		UPDATE follow_ups SET status=$2::followup_status, completed_at=now(), completion_note=COALESCE($3, completion_note),
			outcome=COALESCE($4, outcome), snoozed_until=NULL, updated_at=now()
		WHERE id=$1 RETURNING `+cols, id, StatusCompleted, note, outcome)
}

func (r *Repository) Reopen(ctx context.Context, id string, dueAt *time.Time) (*FollowUp, error) {
	q := `UPDATE follow_ups SET status=$2::followup_status, completed_at=NULL, updated_at=now()`
	args := []any{id, StatusPending}
	if dueAt != nil {
		q += `, due_at=$3`
		args = append(args, *dueAt)
	}
	q += ` WHERE id=$1 RETURNING ` + cols
	return queryOne(ctx, r.db.Pool(), q, args...)
}

func (r *Repository) Reschedule(ctx context.Context, id string, dueAt time.Time) (*FollowUp, error) {
	return queryOne(ctx, r.db.Pool(), `
		UPDATE follow_ups SET due_at=$2, status=$3, completed_at=NULL, updated_at=now()
		WHERE id=$1 RETURNING `+cols, id, dueAt, StatusPending)
}

func (r *Repository) Cancel(ctx context.Context, id string) (*FollowUp, error) {
	return queryOne(ctx, r.db.Pool(), `
		UPDATE follow_ups SET status=$2::followup_status, completed_at=NULL, updated_at=now()
		WHERE id=$1 RETURNING `+cols, id, StatusCancelled)
}

func (r *Repository) Delete(ctx context.Context, id string) error {
	_, err := r.db.Pool().Exec(ctx, `DELETE FROM follow_ups WHERE id=$1`, id)
	return err
}

// UpsertForInteraction creates (or reschedules) the single follow-up linked to
// an interaction. The partial unique index on interaction_id guarantees that a
// report submission or a retry can never create a duplicate automatic follow-up.
func (r *Repository) UpsertForInteraction(ctx context.Context, f *FollowUp) (*FollowUp, error) {
	if f.Type == "" {
		f.Type = "FOLLOW_UP"
	}
	if f.Priority == "" {
		f.Priority = PriorityMedium
	}
	if f.Source == "" {
		f.Source = SourceReport
	}
	return queryOne(ctx, r.db.Pool(), `
		INSERT INTO follow_ups (lead_id, contact_id, project_id, campaign_id, interaction_id, assigned_to,
			created_by, title, description, channel, type, priority, status, due_at, source)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15)
		ON CONFLICT (interaction_id) WHERE interaction_id IS NOT NULL
		DO UPDATE SET due_at=EXCLUDED.due_at, title=COALESCE(EXCLUDED.title, follow_ups.title),
			description=COALESCE(EXCLUDED.description, follow_ups.description),
			channel=COALESCE(EXCLUDED.channel, follow_ups.channel),
			type=COALESCE(EXCLUDED.type, follow_ups.type),
			priority=COALESCE(EXCLUDED.priority, follow_ups.priority),
			status='PENDING', completed_at=NULL, updated_at=now()
		RETURNING `+cols,
		f.LeadID, f.ContactID, f.ProjectID, f.CampaignID, f.InteractionID, f.AssignedTo,
		f.CreatedBy, f.Title, f.Description, f.Channel, f.Type, f.Priority, StatusPending, f.DueAt, f.Source)
}

// ListParams scopes a follow-up list query.
type ListParams struct {
	// AssignedTo filters to a specific assignee. Combined with Scope="my" it is
	// mandatory (the authenticated user). When set without a scope it still
	// restricts the result.
	AssignedTo *string
	// LeadID restricts the list to a single lead.
	LeadID *string
	// Status filters to one canonical status ("" = all).
	Status *Status
	// Scope is one of: my, pending, overdue, today, upcoming, completed, all.
	Scope string
	// OverdueOnly is a shortcut for Scope "overdue".
	OverdueOnly bool
	// From/To bound due_at inclusively.
	From, To *time.Time
	Limit    int
	Offset   int
}

// List returns the matching follow-ups (soonest due first) plus the total count
// (before pagination) for client-side paging.
func (r *Repository) List(ctx context.Context, p ListParams) ([]*ListItem, int, error) {
	where := []string{"1=1"}
	args := []any{}
	argc := 1
	addCond := func(cond string) { where = append(where, cond) }
	addArg := func(cond string, val any) {
		where = append(where, fmt.Sprintf(cond, argc))
		args = append(args, val)
		argc++
	}

	switch p.Scope {
	case "my":
		if p.AssignedTo == nil {
			return nil, 0, errors.New("scope 'my' requires an assigned user")
		}
		addArg("f.assigned_to=$%d", *p.AssignedTo)
		addCond("f.status<>'CANCELLED'")
	case "pending":
		addCond("f.status='PENDING'")
	case "overdue":
		addCond("f.status='PENDING' AND f.due_at < now()")
	case "today":
		addCond("f.status='PENDING' AND f.due_at >= now() AND f.due_at < date_trunc('day', now()) + interval '1 day'")
	case "upcoming":
		addCond("f.status='PENDING' AND f.due_at >= date_trunc('day', now()) + interval '1 day'")
	case "completed":
		addCond("f.status='COMPLETED'")
	case "all", "":
		// no scope restriction
	}
	if p.OverdueOnly {
		addCond("f.status='PENDING' AND f.due_at < now()")
	}
	if p.AssignedTo != nil && p.Scope != "my" {
		addArg("f.assigned_to=$%d", *p.AssignedTo)
	}
	if p.LeadID != nil {
		addArg("f.lead_id=$%d", *p.LeadID)
	}
	if p.Status != nil {
		addArg("f.status=$%d", *p.Status)
	}
	if p.From != nil {
		addArg("f.due_at >= $%d", *p.From)
	}
	if p.To != nil {
		addArg("f.due_at <= $%d", *p.To)
	}

	limit := p.Limit
	if limit <= 0 {
		limit = 50
	}
	if limit > 200 {
		limit = 200
	}
	offset := p.Offset
	if offset < 0 {
		offset = 0
	}

	whereSQL := strings.Join(where, " AND ")
	countSQL := `SELECT count(*) FROM follow_ups f WHERE ` + whereSQL
	var total int
	if err := r.db.Pool().QueryRow(ctx, countSQL, args...).Scan(&total); err != nil {
		return nil, 0, err
	}

	itemSQL := `SELECT ` + colsQualified + `,
		COALESCE(l.company_name, ''), c.full_name, u.name
		FROM follow_ups f
		LEFT JOIN leads l ON l.id = f.lead_id
		LEFT JOIN contacts c ON c.id = f.contact_id
		LEFT JOIN users u ON u.id = f.assigned_to
		WHERE ` + whereSQL + `
		ORDER BY f.due_at ASC
		LIMIT $` + fmt.Sprintf("%d", argc) + ` OFFSET $` + fmt.Sprintf("%d", argc+1)
	args = append(args, limit, offset)

	rows, err := r.db.Pool().Query(ctx, itemSQL, args...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()
	out := []*ListItem{}
	for rows.Next() {
		item, err := scanListItem(rows)
		if err != nil {
			return nil, 0, err
		}
		out = append(out, item)
	}
	return out, total, rows.Err()
}

func (r *Repository) ListForLead(ctx context.Context, leadID string) ([]*FollowUp, error) {
	rows, err := r.db.Pool().Query(ctx, `SELECT `+cols+` FROM follow_ups WHERE lead_id=$1 ORDER BY due_at ASC`, leadID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []*FollowUp{}
	for rows.Next() {
		f, err := scan(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, f)
	}
	return out, rows.Err()
}

// CountOpenToday counts pending follow-ups due within the current UTC day.
func (r *Repository) CountOpenToday(ctx context.Context, userID string) (int, error) {
	q := `SELECT count(*) FROM follow_ups
		WHERE status='PENDING' AND due_at >= date_trunc('day', now()) AND due_at < date_trunc('day', now()) + interval '1 day'`
	args := []any{}
	if userID != "" {
		q += ` AND assigned_to=$1`
		args = append(args, userID)
	}
	var n int
	err := r.db.Pool().QueryRow(ctx, q, args...).Scan(&n)
	return n, err
}

// CountOverdue counts pending follow-ups past their due time.
func (r *Repository) CountOverdue(ctx context.Context, userID string) (int, error) {
	q := `SELECT count(*) FROM follow_ups WHERE status='PENDING' AND due_at < now()`
	args := []any{}
	if userID != "" {
		q += ` AND assigned_to=$1`
		args = append(args, userID)
	}
	var n int
	err := r.db.Pool().QueryRow(ctx, q, args...).Scan(&n)
	return n, err
}

// Service implements follow-up operations.
type Service struct {
	repo *Repository
}

func NewService(repo *Repository) *Service { return &Service{repo: repo} }

var (
	ErrNotFound = errors.New("follow-up not found")
	ErrBadInput = errors.New("invalid follow-up input")
)

type CreateParams struct {
	LeadID        string
	ContactID     *string
	ProjectID     *string
	CampaignID    *string
	InteractionID *string
	AssignedTo    *string
	CreatedBy     *string
	Title         *string
	Description   *string
	Channel       *string
	Type          string
	Priority      Priority
	DueAt         time.Time
	SnoozedUntil  *time.Time
	Source        Source
}

func (s *Service) Create(ctx context.Context, p CreateParams) (*FollowUp, error) {
	if p.LeadID == "" {
		return nil, fmt.Errorf("%w: lead_id is required", ErrBadInput)
	}
	if p.DueAt.IsZero() {
		return nil, fmt.Errorf("%w: due_at is required", ErrBadInput)
	}
	f := &FollowUp{
		LeadID:        p.LeadID,
		ContactID:     p.ContactID,
		ProjectID:     p.ProjectID,
		CampaignID:    p.CampaignID,
		InteractionID: p.InteractionID,
		AssignedTo:    p.AssignedTo,
		CreatedBy:     p.CreatedBy,
		Title:         p.Title,
		Description:   p.Description,
		Channel:       p.Channel,
		Type:          p.Type,
		Priority:      p.Priority,
		DueAt:         p.DueAt,
		SnoozedUntil:  p.SnoozedUntil,
		Source:        p.Source,
	}
	return s.repo.Create(ctx, f)
}

func (s *Service) Get(ctx context.Context, id string) (*FollowUp, error) {
	f, err := s.repo.Get(ctx, id)
	if err != nil {
		if database.IsNoRows(err) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	return f, nil
}

func (s *Service) Update(ctx context.Context, id string, u UpdateParams) (*FollowUp, error) {
	f, err := s.repo.Get(ctx, id)
	if err != nil {
		if database.IsNoRows(err) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	if u.ContactID != nil {
		f.ContactID = u.ContactID
	}
	if u.AssignedTo != nil {
		f.AssignedTo = u.AssignedTo
	}
	if u.Title != nil {
		f.Title = u.Title
	}
	if u.Description != nil {
		f.Description = u.Description
	}
	if u.Channel != nil {
		f.Channel = u.Channel
	}
	if u.Type != nil {
		f.Type = *u.Type
	}
	if u.Priority != nil {
		f.Priority = *u.Priority
	}
	if u.DueAt != nil {
		f.DueAt = *u.DueAt
	}
	return s.repo.Update(ctx, f)
}

type UpdateParams struct {
	ContactID   *string
	AssignedTo  *string
	Title       *string
	Description *string
	Channel     *string
	Type        *string
	Priority    *Priority
	DueAt       *time.Time
}

func (s *Service) Complete(ctx context.Context, id string, note, outcome *string) (*FollowUp, error) {
	f, err := s.repo.Complete(ctx, id, note, outcome)
	if err != nil {
		if database.IsNoRows(err) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	return f, nil
}

func (s *Service) Reopen(ctx context.Context, id string, dueAt *time.Time) (*FollowUp, error) {
	f, err := s.repo.Reopen(ctx, id, dueAt)
	if err != nil {
		if database.IsNoRows(err) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	return f, nil
}

func (s *Service) Reschedule(ctx context.Context, id string, dueAt time.Time) (*FollowUp, error) {
	if dueAt.IsZero() {
		return nil, fmt.Errorf("%w: due_at is required", ErrBadInput)
	}
	f, err := s.repo.Reschedule(ctx, id, dueAt)
	if err != nil {
		if database.IsNoRows(err) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	return f, nil
}

func (s *Service) Cancel(ctx context.Context, id string) (*FollowUp, error) {
	f, err := s.repo.Cancel(ctx, id)
	if err != nil {
		if database.IsNoRows(err) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	return f, nil
}

func (s *Service) Delete(ctx context.Context, id string) error {
	return s.repo.Delete(ctx, id)
}

// UpsertForInteraction creates or reschedules the follow-up linked to a report
// interaction. Idempotent: repeating the same report (or retrying) never
// creates a duplicate.
func (s *Service) UpsertForInteraction(ctx context.Context, p CreateParams) (*FollowUp, error) {
	if p.DueAt.IsZero() {
		return nil, fmt.Errorf("%w: due_at is required", ErrBadInput)
	}
	f := &FollowUp{
		LeadID:        p.LeadID,
		ContactID:     p.ContactID,
		ProjectID:     p.ProjectID,
		CampaignID:    p.CampaignID,
		InteractionID: p.InteractionID,
		AssignedTo:    p.AssignedTo,
		CreatedBy:     p.CreatedBy,
		Title:         p.Title,
		Description:   p.Description,
		Channel:       p.Channel,
		Type:          p.Type,
		Priority:      p.Priority,
		DueAt:         p.DueAt,
		Source:        SourceReport,
	}
	return s.repo.UpsertForInteraction(ctx, f)
}

func (s *Service) List(ctx context.Context, p ListParams) ([]*ListItem, int, error) {
	return s.repo.List(ctx, p)
}

func (s *Service) ListForLead(ctx context.Context, leadID string) ([]*FollowUp, error) {
	return s.repo.ListForLead(ctx, leadID)
}

var _ = time.Now
