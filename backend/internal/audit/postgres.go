package audit

import (
	"context"
	"encoding/json"

	"github.com/google/uuid"
	"github.com/tafakorpardazan/huginn/internal/database"
)

// PostgresStore persists audit entries to the audit_log table.
type PostgresStore struct {
	db *database.DB
}

func NewPostgresStore(db *database.DB) *PostgresStore { return &PostgresStore{db: db} }

func (s *PostgresStore) Record(ctx context.Context, e *Entry) error {
	if e.ID == "" {
		e.ID = uuid.NewString()
	}
	before, err := json.Marshal(e.Before)
	if err != nil {
		return err
	}
	after, err := json.Marshal(e.After)
	if err != nil {
		return err
	}
	meta, err := json.Marshal(e.Metadata)
	if err != nil {
		return err
	}
	_, err = s.db.Pool().Exec(ctx, `
		INSERT INTO audit_log (id, actor_user_id, action, entity_type, entity_id, before_json, after_json, ip, user_agent, request_id, metadata)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11)`,
		e.ID, e.ActorUserID, e.Action, e.EntityType, e.EntityID, before, after, e.IP, e.UserAgent, e.RequestID, meta)
	return err
}

// Query returns audit entries for an entity.
func (s *PostgresStore) Query(ctx context.Context, entityType, entityID string, limit, offset int) ([]*Entry, error) {
	if limit == 0 {
		limit = 50
	}
	rows, err := s.db.Pool().Query(ctx, `
		SELECT id, actor_user_id, action, entity_type, entity_id, before_json, after_json, ip, user_agent, request_id, created_at, metadata
		FROM audit_log
		WHERE ($1::text IS NULL OR entity_type=$1) AND ($2::text IS NULL OR entity_id=$2)
		ORDER BY created_at DESC
		LIMIT $3 OFFSET $4`, entityType, entityID, limit, offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []*Entry{}
	for rows.Next() {
		var e Entry
		var before, after, meta []byte
		if err := rows.Scan(&e.ID, &e.ActorUserID, &e.Action, &e.EntityType, &e.EntityID, &before, &after, &e.IP, &e.UserAgent, &e.CreatedAt, &meta); err != nil {
			return nil, err
		}
		_ = json.Unmarshal(before, &e.Before)
		_ = json.Unmarshal(after, &e.After)
		_ = json.Unmarshal(meta, &e.Metadata)
		out = append(out, &e)
	}
	return out, rows.Err()
}
