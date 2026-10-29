package leadversions

import (
	"context"
	"encoding/json"

	"github.com/google/uuid"
	"github.com/tafakorpardazan/huginn/internal/database"
)

// Repository persists lead version snapshots.
type Repository struct {
	db *database.DB
}

func NewRepository(db *database.DB) *Repository { return &Repository{db: db} }

const versionCols = `id, lead_id, version, data, changes, changed_by, changed_by_role, note, created_at`

func scanVersion(row database.RowScanner) (*Version, error) {
	var v Version
	var changes []byte
	err := row.Scan(&v.ID, &v.LeadID, &v.Version, &v.Data, &changes, &v.ChangedBy, &v.ChangedByRole, &v.Note, &v.CreatedAt)
	if err != nil {
		return nil, err
	}
	if len(changes) > 0 {
		_ = json.Unmarshal(changes, &v.Changes)
	}
	return &v, nil
}

// RecordTx appends a new immutable version inside a transaction.
//
// The caller must hold a lock on the lead row (SELECT ... FOR UPDATE) before
// calling so the version sequence cannot race. The next version number is
// derived from leads.current_version and written back atomically with the
// snapshot, guaranteeing concurrent edits never reuse or reorder versions.
func (r *Repository) RecordTx(ctx context.Context, q database.Querier, leadID string, oldSnap, newSnap *Snapshot, changedBy, changedByRole, note *string) (*Version, error) {
	var current int
	if err := q.QueryRow(ctx, `SELECT current_version FROM leads WHERE id=$1 FOR UPDATE`, leadID).Scan(&current); err != nil {
		return nil, err
	}
	version := current + 1

	data, err := json.Marshal(newSnap)
	if err != nil {
		return nil, err
	}
	changes := Diff(oldSnap, newSnap)
	changesJSON, err := json.Marshal(changes)
	if err != nil {
		return nil, err
	}

	v := &Version{
		ID:            uuid.NewString(),
		LeadID:        leadID,
		Version:       version,
		Data:          data,
		Changes:       changes,
		ChangedBy:     changedBy,
		ChangedByRole: changedByRole,
		Note:          note,
	}
	err = q.QueryRow(ctx, `
		INSERT INTO lead_versions (id, lead_id, version, data, changes, changed_by, changed_by_role, note)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8)
		RETURNING created_at`,
		v.ID, leadID, version, data, changesJSON, changedBy, changedByRole, note).Scan(&v.CreatedAt)
	if err != nil {
		return nil, err
	}
	if _, err := q.Exec(ctx, `UPDATE leads SET current_version=$2 WHERE id=$1`, leadID, version); err != nil {
		return nil, err
	}
	return v, nil
}

// List returns versions for a lead, newest first.
func (r *Repository) List(ctx context.Context, leadID string, limit, offset int) ([]*Version, error) {
	if limit <= 0 || limit > 500 {
		limit = 100
	}
	rows, err := r.db.Pool().Query(ctx, `
		SELECT `+versionCols+` FROM lead_versions
		WHERE lead_id=$1 ORDER BY version DESC LIMIT $2 OFFSET $3`, leadID, limit, offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []*Version{}
	for rows.Next() {
		v, err := scanVersion(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, v)
	}
	return out, rows.Err()
}

// Count returns the number of versions recorded for a lead.
func (r *Repository) Count(ctx context.Context, leadID string) (int, error) {
	var n int
	err := r.db.Pool().QueryRow(ctx, `SELECT count(*) FROM lead_versions WHERE lead_id=$1`, leadID).Scan(&n)
	return n, err
}

// Get returns a specific version of a lead.
func (r *Repository) Get(ctx context.Context, leadID, versionID string) (*Version, error) {
	row := r.db.Pool().QueryRow(ctx, `SELECT `+versionCols+` FROM lead_versions WHERE id=$1 AND lead_id=$2`, versionID, leadID)
	return scanVersion(row)
}

// Recent returns the latest versions across all leads with the lead company
// name, newest first. Used by the admin change-history dashboard.
func (r *Repository) Recent(ctx context.Context, limit, offset int) ([]*RecentItem, error) {
	if limit <= 0 || limit > 500 {
		limit = 100
	}
	rows, err := r.db.Pool().Query(ctx, `
		SELECT lv.id, lv.lead_id, lv.version, lv.data, lv.changes, lv.changed_by, lv.changed_by_role, lv.note, lv.created_at,
		       COALESCE(l.company_name,'')
		FROM lead_versions lv JOIN leads l ON l.id = lv.lead_id
		ORDER BY lv.created_at DESC LIMIT $1 OFFSET $2`, limit, offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []*RecentItem{}
	for rows.Next() {
		var it RecentItem
		if err := scanVersionInto(rows, &it.Version, &it.LeadName); err != nil {
			return nil, err
		}
		out = append(out, &it)
	}
	return out, rows.Err()
}

// scanVersionInto scans a version row followed by one trailing column (used by
// Recent to capture the lead name).
func scanVersionInto(row database.RowScanner, v *Version, trailing ...*string) error {
	var changes []byte
	var leadName string
	if len(trailing) > 0 {
		if err := row.Scan(&v.ID, &v.LeadID, &v.Version, &v.Data, &changes, &v.ChangedBy, &v.ChangedByRole, &v.Note, &v.CreatedAt, &leadName); err != nil {
			return err
		}
		*trailing[0] = leadName
	} else {
		return row.Scan(&v.ID, &v.LeadID, &v.Version, &v.Data, &changes, &v.ChangedBy, &v.ChangedByRole, &v.Note, &v.CreatedAt)
	}
	if len(changes) > 0 {
		_ = json.Unmarshal(changes, &v.Changes)
	}
	return nil
}
