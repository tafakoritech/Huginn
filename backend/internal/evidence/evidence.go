package evidence

import (
	"context"
	"time"

	"github.com/tafakorpardazan/huginn/internal/database"
)

// Item is evidence for a single extracted lead field.
type Item struct {
	ID               string    `json:"id"`
	LeadID           string    `json:"lead_id"`
	SourceDocumentID *string   `json:"source_document_id"`
	FieldName        string    `json:"field_name"`
	SourcePage       *int      `json:"source_page"`
	SourceRow        *int      `json:"source_row"`
	SourceExcerpt    *string   `json:"source_excerpt"`
	Confidence       float64   `json:"confidence"`
}

// Repository persists evidence items.
type Repository struct {
	db *database.DB
}

func NewRepository(db *database.DB) *Repository { return &Repository{db: db} }

func (r *Repository) Create(ctx context.Context, it *Item) error {
	_, err := r.db.Pool().Exec(ctx, `
		INSERT INTO lead_evidence (lead_id, source_document_id, field_name, source_page, source_row, source_excerpt, confidence)
		VALUES ($1,$2,$3,$4,$5,$6,$7)`,
		it.LeadID, it.SourceDocumentID, it.FieldName, it.SourcePage, it.SourceRow, it.SourceExcerpt, it.Confidence)
	return err
}

func (r *Repository) CreateMany(ctx context.Context, items []*Item) error {
	for _, it := range items {
		if err := r.Create(ctx, it); err != nil {
			return err
		}
	}
	return nil
}

func (r *Repository) ListForLead(ctx context.Context, leadID string) ([]*Item, error) {
	rows, err := r.db.Pool().Query(ctx, `
		SELECT id, lead_id, source_document_id, field_name, source_page, source_row, source_excerpt, confidence
		FROM lead_evidence WHERE lead_id=$1 ORDER BY field_name, confidence DESC`, leadID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []*Item{}
	for rows.Next() {
		var it Item
		if err := rows.Scan(&it.ID, &it.LeadID, &it.SourceDocumentID, &it.FieldName, &it.SourcePage, &it.SourceRow, &it.SourceExcerpt, &it.Confidence); err != nil {
			return nil, err
		}
		out = append(out, &it)
	}
	return out, rows.Err()
}

var _ = time.Now
