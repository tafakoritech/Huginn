package httpapi

import (
	"net/http"

	"github.com/tafakorpardazan/huginn/internal/audit"
	"github.com/tafakorpardazan/huginn/internal/auth"
)

// actorEntry builds an audit entry from the request context.
func actorEntry(r *http.Request, action, entityType, entityID string, before, after map[string]any) *audit.Entry {
	p, _ := auth.FromContext(r.Context())
	return &audit.Entry{
		ActorUserID: idPtr(p),
		Action:      action,
		EntityType:  entityType,
		EntityID:    entityID,
		Before:      before,
		After:       after,
		IP:          clientIP(r),
		UserAgent:   r.UserAgent(),
		RequestID:   RequestIDFromContext(r.Context()),
	}
}

func idPtr(p *auth.Principal) *string {
	if p == nil {
		return nil
	}
	return &p.UserID
}

func beforeMap(v any) map[string]any {
	return nil
}
