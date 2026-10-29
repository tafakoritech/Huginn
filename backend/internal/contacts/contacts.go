package contacts

import (
	"context"
	"time"

	"github.com/tafakorpardazan/huginn/internal/database"
	"github.com/tafakorpardazan/huginn/internal/normalize"
)

// Contact is a person associated with a lead.
type Contact struct {
	ID                string     `json:"id"`
	LeadID            string     `json:"lead_id"`
	FirstName         *string    `json:"first_name"`
	LastName          *string    `json:"last_name"`
	FullName          *string    `json:"full_name"`
	JobTitle          *string    `json:"job_title"`
	Department        *string    `json:"department"`
	Mobile            *string    `json:"mobile"`
	MobileNormalized  *string    `json:"mobile_normalized"`
	Phone             *string    `json:"phone"`
	PhoneNormalized   *string    `json:"phone_normalized"`
	Email             *string    `json:"email"`
	EmailNormalized   *string    `json:"email_normalized"`
	WhatsApp          *string    `json:"whatsapp"`
	Telegram          *string    `json:"telegram"`
	Bale              *string    `json:"bale"`
	Eitaa             *string    `json:"eitaa"`
	Rubika            *string    `json:"rubika"`
	IsDecisionMaker   bool       `json:"is_decision_maker"`
	IsPrimaryContact  bool       `json:"is_primary_contact"`
	CreatedAt         time.Time  `json:"created_at"`
	UpdatedAt         time.Time  `json:"updated_at"`
	DeletedAt         *time.Time `json:"deleted_at,omitempty"`
}

// Repository persists contacts.
type Repository struct {
	db *database.DB
}

func NewRepository(db *database.DB) *Repository { return &Repository{db: db} }

const cols = `id, lead_id, first_name, last_name, full_name, job_title, department, mobile, mobile_normalized,
	phone, phone_normalized, email, email_normalized, whatsapp, telegram, bale, eitaa, rubika,
	is_decision_maker, is_primary_contact, created_at, updated_at, deleted_at`

func scan(row database.RowScanner) (*Contact, error) {
	var c Contact
	err := row.Scan(&c.ID, &c.LeadID, &c.FirstName, &c.LastName, &c.FullName, &c.JobTitle, &c.Department,
		&c.Mobile, &c.MobileNormalized, &c.Phone, &c.PhoneNormalized, &c.Email, &c.EmailNormalized,
		&c.WhatsApp, &c.Telegram, &c.Bale, &c.Eitaa, &c.Rubika,
		&c.IsDecisionMaker, &c.IsPrimaryContact, &c.CreatedAt, &c.UpdatedAt, &c.DeletedAt)
	if err != nil {
		return nil, err
	}
	return &c, nil
}

// Normalize populates the normalized fields.
func (c *Contact) Normalize() {
	if c.Mobile != nil {
		if n := normalize.NormalizeMobile(*c.Mobile); n != "" {
			c.MobileNormalized = &n
		}
	}
	if c.Phone != nil {
		if n := normalize.NormalizePhone(*c.Phone); n != "" {
			c.PhoneNormalized = &n
		}
	}
	if c.Email != nil {
		if n := normalize.NormalizeEmail(*c.Email); n != "" {
			c.EmailNormalized = &n
		}
	}
}

func (r *Repository) Create(ctx context.Context, c *Contact) (*Contact, error) {
	c.Normalize()
	row := r.db.Pool().QueryRow(ctx, `
		INSERT INTO contacts (lead_id, first_name, last_name, full_name, job_title, department,
			mobile, mobile_normalized, phone, phone_normalized, email, email_normalized,
			whatsapp, telegram, bale, eitaa, rubika, is_decision_maker, is_primary_contact)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15,$16,$17,$18,$19)
		RETURNING `+cols,
		c.LeadID, c.FirstName, c.LastName, c.FullName, c.JobTitle, c.Department,
		c.Mobile, c.MobileNormalized, c.Phone, c.PhoneNormalized, c.Email, c.EmailNormalized,
		c.WhatsApp, c.Telegram, c.Bale, c.Eitaa, c.Rubika, c.IsDecisionMaker, c.IsPrimaryContact)
	return scan(row)
}

func (r *Repository) Update(ctx context.Context, c *Contact) error {
	c.Normalize()
	_, err := r.db.Pool().Exec(ctx, `
		UPDATE contacts SET first_name=$3, last_name=$4, full_name=$5, job_title=$6, department=$7,
			mobile=$8, mobile_normalized=$9, phone=$10, phone_normalized=$11, email=$12, email_normalized=$13,
			whatsapp=$14, telegram=$15, bale=$16, eitaa=$17, rubika=$18,
			is_decision_maker=$19, is_primary_contact=$20, updated_at=now()
		WHERE id=$1 AND lead_id=$2`,
		c.ID, c.LeadID, c.FirstName, c.LastName, c.FullName, c.JobTitle, c.Department,
		c.Mobile, c.MobileNormalized, c.Phone, c.PhoneNormalized, c.Email, c.EmailNormalized,
		c.WhatsApp, c.Telegram, c.Bale, c.Eitaa, c.Rubika, c.IsDecisionMaker, c.IsPrimaryContact)
	return err
}

func (r *Repository) Get(ctx context.Context, id string) (*Contact, error) {
	row := r.db.Pool().QueryRow(ctx, `SELECT `+cols+` FROM contacts WHERE id=$1`, id)
	return scan(row)
}

func (r *Repository) ListForLead(ctx context.Context, leadID string) ([]*Contact, error) {
	rows, err := r.db.Pool().Query(ctx, `SELECT `+cols+` FROM contacts WHERE lead_id=$1 AND deleted_at IS NULL ORDER BY is_primary_contact DESC, created_at`, leadID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []*Contact{}
	for rows.Next() {
		c, err := scan(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, c)
	}
	return out, rows.Err()
}

// Delete soft-deletes a contact so it remains visible in historical versions.
func (r *Repository) Delete(ctx context.Context, id string) error {
	_, err := r.db.Pool().Exec(ctx, `UPDATE contacts SET deleted_at=now() WHERE id=$1 AND deleted_at IS NULL`, id)
	return err
}

// ---- Transactional variants (used by the versioned lead editing flow) ----

// ListForLeadTx returns the active (non-deleted) contacts for a lead within a
// transaction.
func (r *Repository) ListForLeadTx(ctx context.Context, q database.Querier, leadID string) ([]*Contact, error) {
	rows, err := q.Query(ctx, `SELECT `+cols+` FROM contacts WHERE lead_id=$1 AND deleted_at IS NULL ORDER BY is_primary_contact DESC, created_at`, leadID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []*Contact{}
	for rows.Next() {
		c, err := scan(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, c)
	}
	return out, rows.Err()
}

// CreateTx inserts a contact within a transaction.
func (r *Repository) CreateTx(ctx context.Context, q database.Querier, c *Contact) (*Contact, error) {
	c.Normalize()
	row := q.QueryRow(ctx, `
		INSERT INTO contacts (lead_id, first_name, last_name, full_name, job_title, department,
			mobile, mobile_normalized, phone, phone_normalized, email, email_normalized,
			whatsapp, telegram, bale, eitaa, rubika, is_decision_maker, is_primary_contact)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15,$16,$17,$18,$19)
		RETURNING `+cols,
		c.LeadID, c.FirstName, c.LastName, c.FullName, c.JobTitle, c.Department,
		c.Mobile, c.MobileNormalized, c.Phone, c.PhoneNormalized, c.Email, c.EmailNormalized,
		c.WhatsApp, c.Telegram, c.Bale, c.Eitaa, c.Rubika, c.IsDecisionMaker, c.IsPrimaryContact)
	return scan(row)
}

// UpdateTx updates a contact within a transaction.
func (r *Repository) UpdateTx(ctx context.Context, q database.Querier, c *Contact) error {
	c.Normalize()
	_, err := q.Exec(ctx, `
		UPDATE contacts SET first_name=$3, last_name=$4, full_name=$5, job_title=$6, department=$7,
			mobile=$8, mobile_normalized=$9, phone=$10, phone_normalized=$11, email=$12, email_normalized=$13,
			whatsapp=$14, telegram=$15, bale=$16, eitaa=$17, rubika=$18,
			is_decision_maker=$19, is_primary_contact=$20, updated_at=now()
		WHERE id=$1 AND lead_id=$2 AND deleted_at IS NULL`,
		c.ID, c.LeadID, c.FirstName, c.LastName, c.FullName, c.JobTitle, c.Department,
		c.Mobile, c.MobileNormalized, c.Phone, c.PhoneNormalized, c.Email, c.EmailNormalized,
		c.WhatsApp, c.Telegram, c.Bale, c.Eitaa, c.Rubika, c.IsDecisionMaker, c.IsPrimaryContact)
	return err
}

// SoftDeleteTx marks a contact deleted within a transaction.
func (r *Repository) SoftDeleteTx(ctx context.Context, q database.Querier, id, leadID string) error {
	_, err := q.Exec(ctx, `UPDATE contacts SET deleted_at=now() WHERE id=$1 AND lead_id=$2 AND deleted_at IS NULL`, id, leadID)
	return err
}

// FindByNormalizedMobile finds contacts by canonical number (for dedup).
func (r *Repository) FindByNormalizedMobile(ctx context.Context, normalized string) ([]*Contact, error) {
	rows, err := r.db.Pool().Query(ctx, `SELECT `+cols+` FROM contacts WHERE mobile_normalized=$1`, normalized)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []*Contact{}
	for rows.Next() {
		c, err := scan(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, c)
	}
	return out, rows.Err()
}
