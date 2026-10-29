package campaigns

import (
	"context"
	"errors"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/tafakorpardazan/huginn/internal/database"
)

// ErrNotFound is returned when a campaign does not exist.
var ErrNotFound = errors.New("campaign not found")

const cols = `id, title, description, objective, status, start_at, end_at, created_by, created_at, updated_at`

// Repository persists campaigns and their relationship to projects.
type Repository struct {
	db *database.DB
}

func NewRepository(db *database.DB) *Repository { return &Repository{db: db} }

func (r *Repository) DB() *database.DB { return r.db }

func scanCampaign(row database.RowScanner) (*Campaign, error) {
	var c Campaign
	err := row.Scan(&c.ID, &c.Title, &c.Description, &c.Objective, &c.Status,
		&c.StartAt, &c.EndAt, &c.CreatedBy, &c.CreatedAt, &c.UpdatedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	if c.Status == "" {
		c.Status = StatusActive
	}
	return &c, nil
}

func (r *Repository) Create(ctx context.Context, c *Campaign) (*Campaign, error) {
	if c.ID == "" {
		c.ID = uuid.NewString()
	}
	if c.Status == "" {
		c.Status = StatusActive
	}
	row := r.db.Pool().QueryRow(ctx, `
		INSERT INTO campaigns (id, title, description, objective, status, start_at, end_at, created_by)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8)
		RETURNING `+cols,
		c.ID, c.Title, c.Description, c.Objective, c.Status, c.StartAt, c.EndAt, c.CreatedBy)
	return scanCampaign(row)
}

func (r *Repository) Get(ctx context.Context, id string) (*Campaign, error) {
	row := r.db.Pool().QueryRow(ctx, `SELECT `+cols+` FROM campaigns WHERE id=$1`, id)
	return scanCampaign(row)
}

func (r *Repository) Update(ctx context.Context, c *Campaign) (*Campaign, error) {
	row := r.db.Pool().QueryRow(ctx, `
		UPDATE campaigns SET title=$2, description=$3, objective=$4, status=$5,
			start_at=$6, end_at=$7, updated_at=now()
		WHERE id=$1
		RETURNING `+cols,
		c.ID, c.Title, c.Description, c.Objective, c.Status, c.StartAt, c.EndAt)
	return scanCampaign(row)
}

func (r *Repository) Delete(ctx context.Context, id string) error {
	tag, err := r.db.Pool().Exec(ctx, `DELETE FROM campaigns WHERE id=$1`, id)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

// ListParams filters campaign listing.
type ListParams struct {
	Status *Status
	Page   int
	Size   int
}

// ListResult is a paginated page of campaigns.
type ListResult struct {
	Campaigns []*Campaign `json:"campaigns"`
	Total     int         `json:"total"`
	Page      int         `json:"page"`
	Size      int         `json:"size"`
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
	whereSQL := ""
	if len(where) > 0 {
		whereSQL = " WHERE " + joinStrings(where, " AND ")
	}
	countQ := "SELECT count(*) FROM campaigns" + whereSQL
	var total int
	if err := r.db.Pool().QueryRow(ctx, countQ, args...).Scan(&total); err != nil {
		return nil, err
	}
	offset := (p.Page - 1) * p.Size
	q := `SELECT ` + cols + ` FROM campaigns` + whereSQL +
		" ORDER BY created_at DESC LIMIT $" + itoa(argc) + " OFFSET $" + itoa(argc+1)
	args = append(args, p.Size, offset)

	rows, err := r.db.Pool().Query(ctx, q, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []*Campaign{}
	for rows.Next() {
		c, err := scanCampaign(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, c)
	}
	return &ListResult{Campaigns: out, Total: total, Page: p.Page, Size: p.Size}, rows.Err()
}

// ListProjects returns the Projects attached to a campaign (via campaign_id),
// with only the fields the dashboard needs.
func (r *Repository) ListProjects(ctx context.Context, campaignID string) ([]*ProjectRef, error) {
	rows, err := r.db.Pool().Query(ctx, `
		SELECT id, title, status, created_at FROM projects
		WHERE campaign_id = $1
		ORDER BY created_at DESC`, campaignID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []*ProjectRef{}
	for rows.Next() {
		var p ProjectRef
		if err := rows.Scan(&p.ID, &p.Title, &p.Status, &p.CreatedAt); err != nil {
			return nil, err
		}
		out = append(out, &p)
	}
	return out, rows.Err()
}

// CountByCampaign returns the number of projects per campaign for the given ids.
func (r *Repository) CountByCampaign(ctx context.Context, campaignIDs []string) (map[string]int, error) {
	out := map[string]int{}
	if len(campaignIDs) == 0 {
		return out, nil
	}
	rows, err := r.db.Pool().Query(ctx, `
		SELECT campaign_id, count(*) FROM projects
		WHERE campaign_id = ANY($1) GROUP BY campaign_id`, campaignIDs)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
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

// EnsureSchema applies idempotent schema additions so the campaigns feature
// works even without a migration run (mirrors the other repositories).
func (r *Repository) EnsureSchema(ctx context.Context) error {
	if _, err := r.db.Pool().Exec(ctx, `
		CREATE TABLE IF NOT EXISTS campaigns (
			id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
			title       TEXT NOT NULL,
			description TEXT,
			objective   TEXT,
			status      TEXT NOT NULL DEFAULT 'ACTIVE',
			start_at    TIMESTAMPTZ,
			end_at      TIMESTAMPTZ,
			created_by  UUID REFERENCES users(id) ON DELETE SET NULL,
			created_at  TIMESTAMPTZ NOT NULL DEFAULT now(),
			updated_at  TIMESTAMPTZ NOT NULL DEFAULT now()
		)`); err != nil {
		return err
	}
	if _, err := r.db.Pool().Exec(ctx, `CREATE INDEX IF NOT EXISTS idx_campaigns_status ON campaigns (status)`); err != nil {
		return err
	}
	if _, err := r.db.Pool().Exec(ctx, `CREATE INDEX IF NOT EXISTS idx_campaigns_created_at ON campaigns (created_at)`); err != nil {
		return err
	}
	if _, err := r.db.Pool().Exec(ctx, `CREATE INDEX IF NOT EXISTS idx_campaigns_created_by ON campaigns (created_by)`); err != nil {
		return err
	}
	if _, err := r.db.Pool().Exec(ctx, `ALTER TABLE projects ADD COLUMN IF NOT EXISTS campaign_id UUID REFERENCES campaigns(id) ON DELETE SET NULL`); err != nil {
		return err
	}
	if _, err := r.db.Pool().Exec(ctx, `CREATE INDEX IF NOT EXISTS idx_projects_campaign ON projects (campaign_id)`); err != nil {
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
