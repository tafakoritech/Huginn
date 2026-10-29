package httpapi

import (
	"context"

	"github.com/tafakorpardazan/huginn/internal/database"
	"github.com/tafakorpardazan/huginn/internal/leads"
)

// populateOwnerNames fills Lead.OwnerName for a batch of leads with a single
// users query, so list responses show assignee names instead of raw IDs.
func populateOwnerNames(ctx context.Context, db *database.DB, list []*leads.Lead) {
	ids := map[string]string{}
	var missing []string
	for _, l := range list {
		if l == nil || l.OwnerUserID == nil {
			continue
		}
		if _, ok := ids[*l.OwnerUserID]; !ok {
			ids[*l.OwnerUserID] = ""
			missing = append(missing, *l.OwnerUserID)
		}
	}
	if len(missing) == 0 {
		return
	}
	rows, err := db.Pool().Query(ctx, `SELECT id, name FROM users WHERE id = ANY($1)`, missing)
	if err != nil {
		return
	}
	defer rows.Close()
	for rows.Next() {
		var id, name string
		if err := rows.Scan(&id, &name); err != nil {
			continue
		}
		ids[id] = name
	}
	for _, l := range list {
		if l == nil || l.OwnerUserID == nil {
			continue
		}
		if n, ok := ids[*l.OwnerUserID]; ok && n != "" {
			l.OwnerName = &n
		}
	}
}
