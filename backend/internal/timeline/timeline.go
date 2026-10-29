package timeline

import (
	"context"
	"encoding/json"
	"time"

	"github.com/tafakorpardazan/huginn/internal/database"
)

// Event is a single timeline entry for a lead.
type Event struct {
	ID          string    `json:"id"`
	LeadID      string    `json:"lead_id"`
	EventType   string    `json:"event_type"`
	Title       string    `json:"title"`
	Description *string   `json:"description"`
	ActorUserID *string   `json:"actor_user_id"`
	Payload     map[string]any `json:"payload,omitempty"`
	CreatedAt   time.Time `json:"created_at"`
}

// Repository persists timeline events.
type Repository struct {
	db *database.DB
}

func NewRepository(db *database.DB) *Repository { return &Repository{db: db} }

// Add appends a timeline event.
func (r *Repository) Add(ctx context.Context, leadID, eventType, title string, description *string, actorID *string, payload map[string]any) error {
	var payloadJSON []byte
	if payload != nil {
		payloadJSON, _ = json.Marshal(payload)
	}
	_, err := r.db.Pool().Exec(ctx, `
		INSERT INTO timeline_events (lead_id, event_type, title, description, actor_user_id, payload)
		VALUES ($1,$2,$3,$4,$5,$6)`,
		leadID, eventType, title, description, actorID, payloadJSON)
	return err
}

// AddTx appends within a transaction.
func (r *Repository) AddTx(ctx context.Context, q database.Querier, leadID, eventType, title string, description *string, actorID *string, payload map[string]any) error {
	var payloadJSON []byte
	if payload != nil {
		payloadJSON, _ = json.Marshal(payload)
	}
	_, err := q.Exec(ctx, `
		INSERT INTO timeline_events (lead_id, event_type, title, description, actor_user_id, payload)
		VALUES ($1,$2,$3,$4,$5,$6)`,
		leadID, eventType, title, description, actorID, payloadJSON)
	return err
}

// List returns events for a lead, oldest first.
func (r *Repository) List(ctx context.Context, leadID string, limit int) ([]*Event, error) {
	if limit == 0 || limit > 200 {
		limit = 100
	}
	rows, err := r.db.Pool().Query(ctx, `
		SELECT id, lead_id, event_type, title, description, actor_user_id, payload, created_at
		FROM timeline_events WHERE lead_id=$1 ORDER BY created_at DESC LIMIT $2`, leadID, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []*Event{}
	for rows.Next() {
		var e Event
		var payload []byte
		if err := rows.Scan(&e.ID, &e.LeadID, &e.EventType, &e.Title, &e.Description, &e.ActorUserID, &payload, &e.CreatedAt); err != nil {
			return nil, err
		}
		if len(payload) > 0 {
			_ = json.Unmarshal(payload, &e.Payload)
		}
		out = append(out, &e)
	}
	return out, rows.Err()
}
