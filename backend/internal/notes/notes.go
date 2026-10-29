package notes

import (
	"context"
	"time"

	"github.com/tafakorpardazan/huginn/internal/database"
)

// Note is a manual note on a lead.
type Note struct {
	ID        string    `json:"id"`
	LeadID    string    `json:"lead_id"`
	AuthorID  string    `json:"author_id"`
	Text      string    `json:"text"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// Repository persists notes with edit tracking.
type Repository struct {
	db *database.DB
}

func NewRepository(db *database.DB) *Repository { return &Repository{db: db} }

func (r *Repository) Create(ctx context.Context, n *Note) (*Note, error) {
	row := r.db.Pool().QueryRow(ctx, `
		INSERT INTO notes (lead_id, author_id, text)
		VALUES ($1,$2,$3) RETURNING id, lead_id, author_id, text, created_at, updated_at`,
		n.LeadID, n.AuthorID, n.Text)
	return scanNote(row)
}

func (r *Repository) Get(ctx context.Context, id string) (*Note, error) {
	row := r.db.Pool().QueryRow(ctx, `
		SELECT id, lead_id, author_id, text, created_at, updated_at FROM notes WHERE id=$1`, id)
	return scanNote(row)
}

// Update records the edit before overwriting.
func (r *Repository) Update(ctx context.Context, id string, newText, editorID string) (*Note, error) {
	tx, err := r.db.Pool().Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback(ctx)
	old, err := r.Get(ctx, id)
	if err != nil {
		return nil, err
	}
	if _, err := tx.Exec(ctx, `
		INSERT INTO note_edits (note_id, old_text, new_text, edited_by)
		VALUES ($1,$2,$3,$4)`, id, old.Text, newText, editorID); err != nil {
		return nil, err
	}
	if _, err := tx.Exec(ctx, `UPDATE notes SET text=$2, updated_at=now() WHERE id=$1`, id, newText); err != nil {
		return nil, err
	}
	if err := tx.Commit(ctx); err != nil {
		return nil, err
	}
	return r.Get(ctx, id)
}

func (r *Repository) ListForLead(ctx context.Context, leadID string) ([]*Note, error) {
	rows, err := r.db.Pool().Query(ctx, `
		SELECT id, lead_id, author_id, text, created_at, updated_at FROM notes
		WHERE lead_id=$1 ORDER BY created_at DESC`, leadID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []*Note{}
	for rows.Next() {
		n, err := scanNote(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, n)
	}
	return out, rows.Err()
}

func scanNote(row database.RowScanner) (*Note, error) {
	var n Note
	err := row.Scan(&n.ID, &n.LeadID, &n.AuthorID, &n.Text, &n.CreatedAt, &n.UpdatedAt)
	if err != nil {
		return nil, err
	}
	return &n, nil
}
