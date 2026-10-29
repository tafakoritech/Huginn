package audit

import (
	"context"
	"time"
)

// Entry is an immutable audit record.
type Entry struct {
	ID         string          `json:"id"`
	ActorUserID *string        `json:"actor_user_id"`
	Action     string          `json:"action"`
	EntityType string          `json:"entity_type"`
	EntityID   string          `json:"entity_id"`
	Before     map[string]any  `json:"before_json,omitempty"`
	After      map[string]any  `json:"after_json,omitempty"`
	IP         string          `json:"ip,omitempty"`
	UserAgent  string          `json:"user_agent,omitempty"`
	RequestID  string          `json:"request_id,omitempty"`
	CreatedAt  time.Time       `json:"created_at"`
	Metadata   map[string]any  `json:"metadata,omitempty"`
}

// Store persists audit entries.
type Store interface {
	Record(ctx context.Context, e *Entry) error
}

// Logger is a convenience wrapper used by services to write audit entries.
type Logger struct {
	store Store
}

func NewLogger(store Store) *Logger {
	return &Logger{store: store}
}

func (l *Logger) Record(ctx context.Context, e *Entry) {
	if l == nil || l.store == nil || e == nil {
		return
	}
	_ = l.store.Record(ctx, e)
}
