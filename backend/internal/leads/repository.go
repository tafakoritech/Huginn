package leads

import (
	"context"
	"fmt"
	"strings"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/tafakorpardazan/huginn/internal/database"
)

// Repository persists leads and related records.
type Repository struct {
	db *database.DB
}

func NewRepository(db *database.DB) *Repository { return &Repository{db: db} }

const leadCols = `id, company_name, display_name, business_type, industry, description, province, city, address,
	website, instagram, source, source_batch_id, source_document_id, status, quality_status, score,
	correct_person_reached, decision_maker_identified, decision_maker_name, decision_maker_role,
	owner_user_id, project_id, created_at, updated_at, archived_at`

func scanLead(row database.RowScanner) (*Lead, error) {
	var l Lead
	err := row.Scan(&l.ID, &l.CompanyName, &l.DisplayName, &l.BusinessType, &l.Industry, &l.Description,
		&l.Province, &l.City, &l.Address, &l.Website, &l.Instagram, &l.Source, &l.SourceBatchID, &l.SourceDocumentID,
		&l.Status, &l.QualityStatus, &l.Score, &l.CorrectPersonReached, &l.DecisionMakerIdentified,
		&l.DecisionMakerName, &l.DecisionMakerRole, &l.OwnerUserID, &l.ProjectID, &l.CreatedAt, &l.UpdatedAt, &l.ArchivedAt)
	if err != nil {
		return nil, err
	}
	return &l, nil
}

func (r *Repository) Create(ctx context.Context, l *Lead) (*Lead, error) {
	if l.ID == "" {
		l.ID = uuid.NewString()
	}
	if l.Status == "" {
		l.Status = StatusNew
	}
	if l.QualityStatus == "" {
		l.QualityStatus = QualityRaw
	}
	row := r.db.Pool().QueryRow(ctx, `
		INSERT INTO leads (id, company_name, display_name, business_type, industry, description, province, city,
			address, website, instagram, source, source_batch_id, source_document_id, status, quality_status, score,
			correct_person_reached, decision_maker_identified, decision_maker_name, decision_maker_role,
			owner_user_id, project_id)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15,$16,$17,$18,$19,$20,$21,$22,$23)
		RETURNING `+leadCols,
		l.ID, l.CompanyName, l.DisplayName, l.BusinessType, l.Industry, l.Description, l.Province, l.City,
		l.Address, l.Website, l.Instagram, l.Source, l.SourceBatchID, l.SourceDocumentID, l.Status, l.QualityStatus,
		l.Score, l.CorrectPersonReached, l.DecisionMakerIdentified, l.DecisionMakerName, l.DecisionMakerRole,
		l.OwnerUserID, l.ProjectID)
	return scanLead(row)
}

func (r *Repository) Get(ctx context.Context, id string) (*Lead, error) {
	row := r.db.Pool().QueryRow(ctx, `SELECT `+leadCols+` FROM leads WHERE id=$1`, id)
	return scanLead(row)
}

// GetForUpdate locks a lead row for transactional updates.
func (r *Repository) GetForUpdate(ctx context.Context, q database.Querier, id string) (*Lead, error) {
	row := q.QueryRow(ctx, `SELECT `+leadCols+` FROM leads WHERE id=$1 FOR UPDATE`, id)
	return scanLead(row)
}

func (r *Repository) Update(ctx context.Context, l *Lead) error {
	_, err := r.db.Pool().Exec(ctx, `
		UPDATE leads SET company_name=$2, display_name=$3, business_type=$4, industry=$5, description=$6,
			province=$7, city=$8, address=$9, website=$10, instagram=$11, source=$12, status=$13, quality_status=$14,
			score=$15, correct_person_reached=$16, decision_maker_identified=$17, decision_maker_name=$18,
			decision_maker_role=$19, owner_user_id=$20, project_id=$21, updated_at=now(), archived_at=$22
		WHERE id=$1`,
		l.ID, l.CompanyName, l.DisplayName, l.BusinessType, l.Industry, l.Description, l.Province, l.City,
		l.Address, l.Website, l.Instagram, l.Source, l.Status, l.QualityStatus, l.Score, l.CorrectPersonReached,
		l.DecisionMakerIdentified, l.DecisionMakerName, l.DecisionMakerRole, l.OwnerUserID, l.ProjectID, l.ArchivedAt)
	return err
}

// UpdateTx updates a lead within a transaction.
func (r *Repository) UpdateTx(ctx context.Context, q database.Querier, l *Lead) error {
	_, err := q.Exec(ctx, `
		UPDATE leads SET company_name=$2, display_name=$3, business_type=$4, industry=$5, description=$6,
			province=$7, city=$8, address=$9, website=$10, instagram=$11, source=$12, status=$13, quality_status=$14,
			score=$15, correct_person_reached=$16, decision_maker_identified=$17, decision_maker_name=$18,
			decision_maker_role=$19, owner_user_id=$20, project_id=$21, updated_at=now(), archived_at=$22
		WHERE id=$1`,
		l.ID, l.CompanyName, l.DisplayName, l.BusinessType, l.Industry, l.Description, l.Province, l.City,
		l.Address, l.Website, l.Instagram, l.Source, l.Status, l.QualityStatus, l.Score, l.CorrectPersonReached,
		l.DecisionMakerIdentified, l.DecisionMakerName, l.DecisionMakerRole, l.OwnerUserID, l.ProjectID, l.ArchivedAt)
	return err
}

func (r *Repository) UpdateStatusTx(ctx context.Context, q database.Querier, id string, status Status) error {
	_, err := q.Exec(ctx, `UPDATE leads SET status=$2, updated_at=now() WHERE id=$1`, id, status)
	return err
}

func (r *Repository) SetOwner(ctx context.Context, id string, ownerID *string) error {
	_, err := r.db.Pool().Exec(ctx, `UPDATE leads SET owner_user_id=$2, updated_at=now() WHERE id=$1`, id, ownerID)
	return err
}

func (r *Repository) SetScore(ctx context.Context, id string, score int) error {
	_, err := r.db.Pool().Exec(ctx, `UPDATE leads SET score=$2, updated_at=now() WHERE id=$1`, id, score)
	return err
}

// UpdateReportFacts persists the canonical facts promoted from a call report.
// Facts are only ever written when established; NULLs leave existing values
// untouched.
func (r *Repository) UpdateReportFacts(ctx context.Context, id string, correctPersonReached *bool, decisionMakerIdentified bool, decisionMakerName, decisionMakerRole *string) error {
	_, err := r.db.Pool().Exec(ctx, `
		UPDATE leads SET
			correct_person_reached = COALESCE($2, correct_person_reached),
			decision_maker_identified = $3,
			decision_maker_name = COALESCE($4, decision_maker_name),
			decision_maker_role = COALESCE($5, decision_maker_role),
			updated_at = now()
		WHERE id=$1`, id, correctPersonReached, decisionMakerIdentified, decisionMakerName, decisionMakerRole)
	return err
}

func (r *Repository) Archive(ctx context.Context, id string) error {
	_, err := r.db.Pool().Exec(ctx, `UPDATE leads SET status='ARCHIVED', archived_at=now(), updated_at=now() WHERE id=$1`, id)
	return err
}

func (r *Repository) InsertStatusHistory(ctx context.Context, leadID string, from, to Status, reason string, changedBy *string) error {
	_, err := r.db.Pool().Exec(ctx, `
		INSERT INTO lead_status_history (lead_id, from_status, to_status, reason, changed_by)
		VALUES ($1,$2,$3,$4,$5)`, leadID, from, to, nullStr(reason), changedBy)
	return err
}

func (r *Repository) InsertStatusHistoryTx(ctx context.Context, q database.Querier, leadID string, from, to Status, reason string, changedBy *string) error {
	_, err := q.Exec(ctx, `
		INSERT INTO lead_status_history (lead_id, from_status, to_status, reason, changed_by)
		VALUES ($1,$2,$3,$4,$5)`, leadID, from, to, nullStr(reason), changedBy)
	return err
}

// ListParams are query filters for lead listing.
type ListParams struct {
	Status       *Status
	Owner        *string
	Scope        string // mine | unassigned | team | all
	MinScore     *int
	MaxScore     *int
	City         *string
	Province     *string
	Industry     *string
	BusinessType *string
	Source       *string
	BatchID      *string
	DocID        *string
	Tag          *string
	Search       *string
	Qualified    *bool
	ProjectID    *string
	Page         int
	PageSize     int
	Sort         string
	Order        string
}

// ListResult contains a page of leads and total count.
type ListResult struct {
	Leads []*Lead `json:"leads"`
	Total int     `json:"total"`
	Page  int     `json:"page"`
	Size  int     `json:"size"`
}

func (r *Repository) List(ctx context.Context, p ListParams, viewerRole string, viewerID string) (*ListResult, error) {
	if p.Page < 1 {
		p.Page = 1
	}
	if p.PageSize < 1 || p.PageSize > 200 {
		p.PageSize = 50
	}

	var where []string
	var args []any
	argc := 1

	add := func(cond string, val any) {
		where = append(where, fmt.Sprintf(cond, argc))
		args = append(args, val)
		argc++
	}

	if p.Status != nil {
		add("status=$%d", *p.Status)
	}
	if p.Owner != nil {
		add("owner_user_id=$%d", *p.Owner)
	}
	if p.MinScore != nil {
		add("score>=$%d", *p.MinScore)
	}
	if p.MaxScore != nil {
		add("score<=$%d", *p.MaxScore)
	}
	if p.City != nil {
		add("city=$%d", *p.City)
	}
	if p.Province != nil {
		add("province=$%d", *p.Province)
	}
	if p.Industry != nil {
		add("industry=$%d", *p.Industry)
	}
	if p.BusinessType != nil {
		add("business_type=$%d", *p.BusinessType)
	}
	if p.Source != nil {
		add("source=$%d", *p.Source)
	}
	if p.BatchID != nil {
		add("source_batch_id=$%d", *p.BatchID)
	}
	if p.DocID != nil {
		add("source_document_id=$%d", *p.DocID)
	}
	if p.ProjectID != nil {
		add("project_id=$%d", *p.ProjectID)
	}
	if p.Qualified != nil {
		if *p.Qualified {
			where = append(where, "status='QUALIFIED' OR status='EXPORTED_TO_MUNINN'")
		} else {
			where = append(where, "status NOT IN ('QUALIFIED','EXPORTED_TO_MUNINN')")
		}
	}

	switch p.Scope {
	case "mine":
		add("owner_user_id=$%d", viewerID)
	case "unassigned":
		where = append(where, "owner_user_id IS NULL")
	case "all":
		// no restriction; everyone can see all if policy permits
	}

	if p.Search != nil {
		where = append(where, fmt.Sprintf(
			`(company_name ILIKE '%%' || $%d || '%%' OR city ILIKE '%%' || $%d || '%%' OR industry ILIKE '%%' || $%d || '%%')`,
			argc, argc+1, argc+2))
		args = append(args, *p.Search, *p.Search, *p.Search)
		argc += 3
	}
	if p.Tag != nil {
		where = append(where, fmt.Sprintf(
			`id IN (SELECT lt.lead_id FROM lead_tags lt JOIN tags t ON t.id=lt.tag_id WHERE t.name=$%d)`, argc))
		args = append(args, *p.Tag)
		argc++
	}

	if viewerRole == "sales_agent" && p.Scope != "all" {
		// agents see their own leads by default
		if p.Scope == "" {
			add("owner_user_id=$%d", viewerID)
		}
	}

	whereSQL := ""
	if len(where) > 0 {
		whereSQL = " WHERE " + strings.Join(where, " AND ")
	}

	sortCol := "created_at"
	switch p.Sort {
	case "score":
		sortCol = "score"
	case "company":
		sortCol = "company_name"
	case "updated":
		sortCol = "updated_at"
	}
	order := strings.ToUpper(p.Order)
	if order != "ASC" && order != "DESC" {
		order = "DESC"
	}

	countQ := "SELECT count(*) FROM leads" + whereSQL
	var total int
	if err := r.db.Pool().QueryRow(ctx, countQ, args...).Scan(&total); err != nil {
		return nil, err
	}

	offset := (p.Page - 1) * p.PageSize
	q := `SELECT ` + leadCols + ` FROM leads` + whereSQL +
		fmt.Sprintf(` ORDER BY %s %s LIMIT $%d OFFSET $%d`, sortCol, order, argc, argc+1)
	args = append(args, p.PageSize, offset)

	rows, err := r.db.Pool().Query(ctx, q, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := []*Lead{}
	for rows.Next() {
		l, err := scanLead(rows)
		if err != nil {
			return nil, err
		}
		tags, _ := r.TagsForLead(ctx, l.ID)
		l.Tags = tags
		out = append(out, l)
	}
	return &ListResult{Leads: out, Total: total, Page: p.Page, Size: p.PageSize}, rows.Err()
}

// CountLeads returns the number of leads matching a status.
func (r *Repository) CountLeads(ctx context.Context, status Status) (int, error) {
	var n int
	err := r.db.Pool().QueryRow(ctx, `SELECT count(*) FROM leads WHERE status=$1`, status).Scan(&n)
	return n, err
}

// ListLeadsMissingContactInfo returns leads tied to a source document that have
// no contact carrying a normalized phone/mobile/email. Used by the safe
// backfill path (recover explicit contacts from the original source).
func (r *Repository) ListLeadsMissingContactInfo(ctx context.Context, limit int) ([]*Lead, error) {
	if limit <= 0 || limit > 1000 {
		limit = 500
	}
	rows, err := r.db.Pool().Query(ctx, `
		SELECT `+leadCols+` FROM leads l
		WHERE l.source_document_id IS NOT NULL
		  AND NOT EXISTS (
		      SELECT 1 FROM contacts c
		      WHERE c.lead_id = l.id
		        AND (c.mobile_normalized IS NOT NULL
		             OR c.phone_normalized IS NOT NULL
		             OR c.email_normalized IS NOT NULL)
		  )
		ORDER BY l.created_at
		LIMIT $1`, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []*Lead{}
	for rows.Next() {
		l, err := scanLead(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, l)
	}
	return out, rows.Err()
}

func (r *Repository) CountAll(ctx context.Context) (int, error) {
	var n int
	err := r.db.Pool().QueryRow(ctx, `SELECT count(*) FROM leads`).Scan(&n)
	return n, err
}

// ---- Tags ----

func (r *Repository) TagsForLead(ctx context.Context, leadID string) ([]string, error) {
	rows, err := r.db.Pool().Query(ctx, `
		SELECT t.name FROM tags t JOIN lead_tags lt ON lt.tag_id=t.id WHERE lt.lead_id=$1 ORDER BY t.name`, leadID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []string{}
	for rows.Next() {
		var s string
		if err := rows.Scan(&s); err != nil {
			return nil, err
		}
		out = append(out, s)
	}
	return out, rows.Err()
}

func (r *Repository) AddTag(ctx context.Context, leadID, tagName, source string) error {
	tx, err := r.db.Pool().Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)
	var tagID string
	err = tx.QueryRow(ctx, `INSERT INTO tags (name) VALUES ($1) ON CONFLICT (name) DO UPDATE SET name=EXCLUDED.name RETURNING id`, tagName).Scan(&tagID)
	if err != nil {
		return err
	}
	_, err = tx.Exec(ctx, `INSERT INTO lead_tags (lead_id, tag_id, source) VALUES ($1,$2,$3) ON CONFLICT DO NOTHING`, leadID, tagID, source)
	if err != nil {
		return err
	}
	return tx.Commit(ctx)
}

// TagsForLeadTx returns tag names for a lead within a transaction.
func (r *Repository) TagsForLeadTx(ctx context.Context, q database.Querier, leadID string) ([]string, error) {
	rows, err := q.Query(ctx, `
		SELECT t.name FROM tags t JOIN lead_tags lt ON lt.tag_id=t.id WHERE lt.lead_id=$1 ORDER BY t.name`, leadID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []string{}
	for rows.Next() {
		var s string
		if err := rows.Scan(&s); err != nil {
			return nil, err
		}
		out = append(out, s)
	}
	return out, rows.Err()
}

// AddTagTx attaches a tag to a lead within a transaction. Returns whether the
// tag was newly attached.
func (r *Repository) AddTagTx(ctx context.Context, q database.Querier, leadID, tagName, source string) (bool, error) {
	var tagID string
	err := q.QueryRow(ctx, `INSERT INTO tags (name) VALUES ($1) ON CONFLICT (name) DO UPDATE SET name=EXCLUDED.name RETURNING id`, tagName).Scan(&tagID)
	if err != nil {
		return false, err
	}
	res, err := q.Exec(ctx, `INSERT INTO lead_tags (lead_id, tag_id, source) VALUES ($1,$2,$3) ON CONFLICT DO NOTHING`, leadID, tagID, source)
	if err != nil {
		return false, err
	}
	return res.RowsAffected() > 0, nil
}

// RemoveTagTx detaches a tag from a lead within a transaction. Returns whether
// a row was removed.
func (r *Repository) RemoveTagTx(ctx context.Context, q database.Querier, leadID, tagName string) (bool, error) {
	res, err := q.Exec(ctx, `
		DELETE FROM lead_tags WHERE lead_id=$1 AND tag_id IN (SELECT id FROM tags WHERE name=$2)`, leadID, tagName)
	if err != nil {
		return false, err
	}
	return res.RowsAffected() > 0, nil
}

// ---- Work Queue ----

// WorkQueue returns prioritized work items for an agent.
func (r *Repository) WorkQueue(ctx context.Context, userID string) ([]*WorkQueueItem, error) {
	rows, err := r.db.Pool().Query(ctx, `
		SELECT l.id, l.company_name, l.status, l.score,
			CASE
				WHEN f.id IS NOT NULL AND f.due_at < now() THEN 10
				WHEN f.id IS NOT NULL AND f.due_at < now() + interval '1 day' THEN 20
				WHEN l.status = 'ASSIGNED' THEN 30
				WHEN l.status = 'CONTACTING' THEN 40
				WHEN l.score >= 70 THEN 50
				ELSE 60
			END AS priority,
			CASE
				WHEN f.id IS NOT NULL AND f.due_at < now() THEN 'OVERDUE_FOLLOWUP'
				WHEN f.id IS NOT NULL AND f.due_at < now() + interval '1 day' THEN 'TODAY_FOLLOWUP'
				WHEN l.status = 'ASSIGNED' THEN 'NEW_ASSIGNED'
				WHEN l.status = 'CONTACTING' THEN 'NEEDS_ACTION'
				WHEN l.score >= 70 THEN 'HIGH_SCORE'
				ELSE 'OTHER'
			END AS reason,
			f.due_at,
			COALESCE(
				rep.next_action,
				CASE
					WHEN l.status IN ('NEW','READY') THEN 'ASSIGN'
					WHEN l.status = 'ASSIGNED' THEN 'CALL'
					WHEN l.status = 'CONTACTING' THEN 'REPORT'
					WHEN l.status = 'FOLLOW_UP' THEN 'FOLLOW_UP'
					WHEN l.status = 'QUALIFIED' THEN 'EXPORT'
					ELSE 'NONE'
				END
			) AS next_action
		FROM leads l
		LEFT JOIN LATERAL (
			SELECT f.id, f.due_at FROM follow_ups f
			WHERE f.lead_id = l.id AND f.assigned_to = $1 AND f.status = 'OPEN'
			ORDER BY f.due_at ASC LIMIT 1
		) f ON true
		LEFT JOIN LATERAL (
			SELECT cr.next_action FROM contact_reports cr
			JOIN interactions i ON i.id = cr.interaction_id
			WHERE i.lead_id = l.id AND cr.next_action IS NOT NULL AND cr.next_action <> ''
			ORDER BY cr.created_at DESC LIMIT 1
		) rep ON true
		WHERE l.owner_user_id = $1 AND l.status NOT IN ('QUALIFIED','DISQUALIFIED','EXPORTED_TO_MUNINN','ARCHIVED')
		ORDER BY priority ASC, l.score DESC
		LIMIT 100`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []*WorkQueueItem{}
	for rows.Next() {
		var it WorkQueueItem
		if err := rows.Scan(&it.LeadID, &it.CompanyName, &it.Status, &it.Score, &it.Priority, &it.Reason, &it.DueAt, &it.NextAction); err != nil {
			return nil, err
		}
		out = append(out, &it)
	}
	return out, rows.Err()
}

// TagsForLeadsBatch loads tags for multiple leads.
func (r *Repository) TagsForLeadsBatch(ctx context.Context, leadIDs []string) (map[string][]string, error) {
	if len(leadIDs) == 0 {
		return map[string][]string{}, nil
	}
	rows, err := r.db.Pool().Query(ctx, `
		SELECT lt.lead_id, t.name FROM tags t JOIN lead_tags lt ON lt.tag_id=t.id
		WHERE lt.lead_id = ANY($1) ORDER BY t.name`, leadIDs)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := map[string][]string{}
	for rows.Next() {
		var leadID, name string
		if err := rows.Scan(&leadID, &name); err != nil {
			return nil, err
		}
		out[leadID] = append(out[leadID], name)
	}
	return out, rows.Err()
}

func (r *Repository) DB() *database.DB { return r.db }

// EnsureSchema applies idempotent schema additions at startup so the leads
// source feature works without a separate migration run.
func (r *Repository) EnsureSchema(ctx context.Context) error {
	if _, err := r.db.Pool().Exec(ctx, `ALTER TABLE leads ADD COLUMN IF NOT EXISTS source TEXT`); err != nil {
		return err
	}
	// Versioned editing support (idempotent; mirrors migration 003).
	if _, err := r.db.Pool().Exec(ctx, `ALTER TABLE leads ADD COLUMN IF NOT EXISTS current_version INT NOT NULL DEFAULT 0`); err != nil {
		return err
	}
	if _, err := r.db.Pool().Exec(ctx, `ALTER TABLE contacts ADD COLUMN IF NOT EXISTS deleted_at TIMESTAMPTZ`); err != nil {
		return err
	}
	// Project association (the parent sales context for a lead). Nullable at the
	// schema level so existing rows are preserved; the service layer enforces
	// that new leads must belong to a project.
	if _, err := r.db.Pool().Exec(ctx, `DO $$ BEGIN
		IF NOT EXISTS (SELECT 1 FROM information_schema.columns WHERE table_name='leads' AND column_name='project_id') THEN
			ALTER TABLE leads ADD COLUMN project_id UUID;
		END IF;
	END $$`); err != nil {
		return err
	}
	if _, err := r.db.Pool().Exec(ctx, `CREATE INDEX IF NOT EXISTS idx_leads_project ON leads (project_id)`); err != nil {
		return err
	}
	if _, err := r.db.Pool().Exec(ctx, `
		CREATE TABLE IF NOT EXISTS lead_versions (
			id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
			lead_id UUID NOT NULL REFERENCES leads(id) ON DELETE CASCADE,
			version INT NOT NULL,
			data JSONB NOT NULL,
			changes JSONB NOT NULL DEFAULT '[]',
			changed_by UUID REFERENCES users(id),
			changed_by_role TEXT,
			note TEXT,
			created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
			UNIQUE (lead_id, version)
		)`); err != nil {
		return err
	}
	if _, err := r.db.Pool().Exec(ctx, `CREATE INDEX IF NOT EXISTS idx_lead_versions_lead ON lead_versions (lead_id, version DESC)`); err != nil {
		return err
	}
	_, err := r.db.Pool().Exec(ctx, `CREATE INDEX IF NOT EXISTS idx_leads_source ON leads (source)`)
	return err
}

func nullStr(s string) any {
	if s == "" {
		return nil
	}
	return s
}

var _ = pgx.ErrNoRows
