package projects

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/tafakorpardazan/huginn/internal/database"
)

// ErrNotFound is returned when a project does not exist.
var ErrNotFound = errors.New("project not found")

const cols = `id, title, description, product_name, product_description, target_market,
	ideal_customer_profile, customer_pain_points, value_proposition, sales_objective,
	common_objections, created_by, status, campaign_id, created_at, updated_at`

// Repository persists projects and their relationship to leads (1..* via
// leads.project_id) and to campaigns (projects.campaign_id).
type Repository struct {
	db *database.DB
}

func NewRepository(db *database.DB) *Repository { return &Repository{db: db} }

func (r *Repository) DB() *database.DB { return r.db }

func scanProject(row database.RowScanner) (*Project, error) {
	var p Project
	err := row.Scan(&p.ID, &p.Title, &p.Description, &p.ProductName, &p.ProductDescription,
		&p.TargetMarket, &p.IdealCustomerProfile, &p.CustomerPainPoints, &p.ValueProposition,
		&p.SalesObjective, &p.CommonObjections, &p.CreatedBy, &p.Status, &p.CampaignID,
		&p.CreatedAt, &p.UpdatedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	if p.Status == "" {
		p.Status = StatusActive
	}
	return &p, nil
}

func (r *Repository) Create(ctx context.Context, p *Project) (*Project, error) {
	if p.ID == "" {
		p.ID = uuid.NewString()
	}
	if p.Status == "" {
		p.Status = StatusActive
	}
	row := r.db.Pool().QueryRow(ctx, `
		INSERT INTO projects (id, title, description, product_name, product_description,
			target_market, ideal_customer_profile, customer_pain_points, value_proposition,
			sales_objective, common_objections, created_by, status, campaign_id)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14)
		RETURNING `+cols,
		p.ID, p.Title, p.Description, p.ProductName, p.ProductDescription, p.TargetMarket,
		p.IdealCustomerProfile, p.CustomerPainPoints, p.ValueProposition, p.SalesObjective,
		p.CommonObjections, p.CreatedBy, p.Status, p.CampaignID)
	return scanProject(row)
}

func (r *Repository) Get(ctx context.Context, id string) (*Project, error) {
	row := r.db.Pool().QueryRow(ctx, `SELECT `+cols+` FROM projects WHERE id=$1`, id)
	return scanProject(row)
}

func (r *Repository) Update(ctx context.Context, p *Project) (*Project, error) {
	p.UpdatedAt = time.Now()
	row := r.db.Pool().QueryRow(ctx, `
		UPDATE projects SET title=$2, description=$3, product_name=$4, product_description=$5,
			target_market=$6, ideal_customer_profile=$7, customer_pain_points=$8,
			value_proposition=$9, sales_objective=$10, common_objections=$11, status=$12,
			campaign_id=$13, updated_at=now()
		WHERE id=$1
		RETURNING `+cols,
		p.ID, p.Title, p.Description, p.ProductName, p.ProductDescription, p.TargetMarket,
		p.IdealCustomerProfile, p.CustomerPainPoints, p.ValueProposition, p.SalesObjective,
		p.CommonObjections, p.Status, p.CampaignID)
	return scanProject(row)
}

func (r *Repository) Delete(ctx context.Context, id string) error {
	tag, err := r.db.Pool().Exec(ctx, `DELETE FROM projects WHERE id=$1`, id)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

// ListParams filters project listing.
type ListParams struct {
	Status     *Status
	CampaignID *string
	Page       int
	Size       int
}

type ListResult struct {
	Projects []*ProjectDetail `json:"projects"`
	Total    int              `json:"total"`
	Page     int              `json:"page"`
	Size     int              `json:"size"`
}

// ProjectDetail augments a project with the number of leads it owns.
type ProjectDetail struct {
	*Project
	LeadCount int `json:"lead_count"`
}

func (r *Repository) List(ctx context.Context, p ListParams) (*ListResult, error) {
	if p.Page < 1 {
		p.Page = 1
	}
	if p.Size < 1 || p.Size > 200 {
		p.Size = 50
	}
	var where []string
	var args []any
	argc := 1
	if p.Status != nil {
		where = append(where, "status=$"+itoa(argc))
		args = append(args, *p.Status)
		argc++
	}
	if p.CampaignID != nil {
		where = append(where, "campaign_id=$"+itoa(argc))
		args = append(args, *p.CampaignID)
		argc++
	}
	whereSQL := ""
	if len(where) > 0 {
		whereSQL = " WHERE " + joinStrings(where, " AND ")
	}
	countQ := "SELECT count(*) FROM projects" + whereSQL
	var total int
	if err := r.db.Pool().QueryRow(ctx, countQ, args...).Scan(&total); err != nil {
		return nil, err
	}
	offset := (p.Page - 1) * p.Size
	q := `SELECT ` + cols + `, COALESCE((SELECT count(*) FROM leads WHERE leads.project_id = projects.id), 0) AS lead_count FROM projects` + whereSQL +
		" ORDER BY created_at DESC LIMIT $" + itoa(argc) + " OFFSET $" + itoa(argc+1)
	args = append(args, p.Size, offset)

	rows, err := r.db.Pool().Query(ctx, q, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []*ProjectDetail{}
	for rows.Next() {
		var p Project
		var lc int
		if err := scanProjectRow(rows, &p, &lc); err != nil {
			return nil, err
		}
		if p.Status == "" {
			p.Status = StatusActive
		}
		out = append(out, &ProjectDetail{Project: &p, LeadCount: lc})
	}
	return &ListResult{Projects: out, Total: total, Page: p.Page, Size: p.Size}, rows.Err()
}

// CountLeadsByProject returns lead counts keyed by project id for the given set
// of project ids (used to refresh counts efficiently).
func (r *Repository) CountLeadsByProject(ctx context.Context, projectIDs []string) (map[string]int, error) {
	if len(projectIDs) == 0 {
		return map[string]int{}, nil
	}
	rows, err := r.db.Pool().Query(ctx, `
		SELECT project_id, count(*) FROM leads
		WHERE project_id = ANY($1) GROUP BY project_id`, projectIDs)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := map[string]int{}
	for rows.Next() {
		var id string
		var n int
		if err := rows.Scan(&id, &n); err != nil {
			return nil, err
		}
		out[id] = n
	}
	return out, rows.Err()
}

func scanProjectRow(row database.RowScanner, p *Project, leadCount *int) error {
	return row.Scan(&p.ID, &p.Title, &p.Description, &p.ProductName, &p.ProductDescription,
		&p.TargetMarket, &p.IdealCustomerProfile, &p.CustomerPainPoints, &p.ValueProposition,
		&p.SalesObjective, &p.CommonObjections, &p.CreatedBy, &p.Status, &p.CampaignID,
		&p.CreatedAt, &p.UpdatedAt, leadCount)
}

// EnsureSchema applies idempotent schema additions so the projects feature works
// even without a migration run (mirrors the leads.EnsureSchema pattern).
func (r *Repository) EnsureSchema(ctx context.Context) error {
	if _, err := r.db.Pool().Exec(ctx, `
		CREATE TABLE IF NOT EXISTS projects (
			id                  UUID PRIMARY KEY DEFAULT gen_random_uuid(),
			title               TEXT NOT NULL,
			description         TEXT NOT NULL,
			product_name        TEXT,
			product_description TEXT,
			target_market       TEXT,
			ideal_customer_profile TEXT,
			customer_pain_points TEXT,
			value_proposition   TEXT,
			sales_objective     TEXT,
			common_objections   TEXT,
			created_by          UUID REFERENCES users(id),
			status              TEXT NOT NULL DEFAULT 'ACTIVE',
			campaign_id         UUID,
			created_at          TIMESTAMPTZ NOT NULL DEFAULT now(),
			updated_at          TIMESTAMPTZ NOT NULL DEFAULT now()
		)`); err != nil {
		return err
	}
	if _, err := r.db.Pool().Exec(ctx, `CREATE INDEX IF NOT EXISTS idx_projects_created_at ON projects (created_at)`); err != nil {
		return err
	}
	if _, err := r.db.Pool().Exec(ctx, `CREATE INDEX IF NOT EXISTS idx_projects_created_by ON projects (created_by)`); err != nil {
		return err
	}
	if _, err := r.db.Pool().Exec(ctx, `ALTER TABLE projects ADD COLUMN IF NOT EXISTS campaign_id UUID`); err != nil {
		return err
	}
	if _, err := r.db.Pool().Exec(ctx, `CREATE INDEX IF NOT EXISTS idx_projects_campaign ON projects (campaign_id)`); err != nil {
		return err
	}
	if _, err := r.db.Pool().Exec(ctx, `CREATE INDEX IF NOT EXISTS idx_projects_status ON projects (status)`); err != nil {
		return err
	}
	return nil
}

func itoa(n int) string {
	return formatInt(n)
}

func joinStrings(parts []string, sep string) string {
	out := ""
	for i, p := range parts {
		if i > 0 {
			out += sep
		}
		out += p
	}
	return out
}

func formatInt(n int) string {
	if n == 0 {
		return "0"
	}
	neg := n < 0
	if neg {
		n = -n
	}
	var buf [20]byte
	i := len(buf)
	for n > 0 {
		i--
		buf[i] = byte('0' + n%10)
		n /= 10
	}
	if neg {
		i--
		buf[i] = '-'
	}
	return string(buf[i:])
}
