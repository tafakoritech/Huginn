// Package leadversions implements immutable, versioned lead editing.
//
// Every edit to a lead (fields, contacts, tags, status, ownership) appends a
// new immutable snapshot to the lead_versions table while the lead row is
// locked, so the previous version is always preserved and concurrent edits
// cannot corrupt the version sequence.
package leadversions

import (
	"encoding/json"
	"time"
)

// Change describes a single field-level modification within one version.
type Change struct {
	Field string `json:"field"`
	Old   any    `json:"old,omitempty"`
	New   any    `json:"new,omitempty"`
}

// Version is one immutable snapshot/layer of a lead.
type Version struct {
	ID            string          `json:"id"`
	LeadID        string          `json:"lead_id"`
	Version       int             `json:"version"`
	Data          json.RawMessage `json:"data"`
	Changes       []Change        `json:"changes"`
	ChangedBy     *string         `json:"changed_by"`
	ChangedByRole *string         `json:"changed_by_role"`
	Note          *string         `json:"note"`
	CreatedAt     time.Time       `json:"created_at"`
}

// RecentItem is a version enriched with the lead company name for the admin
// change-history feed.
type RecentItem struct {
	Version
	LeadName string `json:"lead_name"`
}

// Snapshot is the marshallable shape of the complete lead state at a point in
// time. Lead is a map of scalar lead fields, Contacts is the ordered contact
// list, Tags the current tag names.
type Snapshot struct {
	Lead     map[string]any   `json:"lead"`
	Contacts []map[string]any `json:"contacts"`
	Tags     []string         `json:"tags"`
}

// skipLeadFields are system fields that never appear as a change record.
var skipLeadFields = map[string]bool{
	"id":         true,
	"created_at": true,
	"updated_at": true,
}

// skipContactFields are system fields on a contact that never appear as a
// change record on their own.
var skipContactFields = map[string]bool{
	"id":         true,
	"lead_id":    true,
	"created_at": true,
	"updated_at": true,
}

// HasChanges reports whether the two snapshots differ in any tracked field.
func HasChanges(oldSnap, newSnap *Snapshot) bool {
	return len(Diff(oldSnap, newSnap)) > 0
}

// Diff computes the field-level change records between two snapshots.
// An nil oldSnap (initial creation) yields a single "created" change.
func Diff(oldSnap, newSnap *Snapshot) []Change {
	if newSnap == nil {
		return nil
	}
	if oldSnap == nil {
		return []Change{{Field: "lead.created", New: newSnap.Lead["company_name"]}}
	}

	var out []Change

	// --- lead scalar fields ---
	keys := map[string]bool{}
	for k := range oldSnap.Lead {
		keys[k] = true
	}
	for k := range newSnap.Lead {
		keys[k] = true
	}
	for k := range keys {
		if skipLeadFields[k] {
			continue
		}
		oldV := oldSnap.Lead[k]
		newV := newSnap.Lead[k]
		if !valuesEqual(oldV, newV) {
			out = append(out, Change{Field: k, Old: oldV, New: newV})
		}
	}

	// --- contacts ---
	oldByID := indexContacts(oldSnap.Contacts)
	newByID := indexContacts(newSnap.Contacts)

	for id, oldC := range oldByID {
		if newC, ok := newByID[id]; ok {
			for k := range oldC {
				if skipContactFields[k] {
					continue
				}
				if !valuesEqual(oldC[k], newC[k]) {
					out = append(out, Change{Field: "contact." + id + "." + k, Old: oldC[k], New: newC[k]})
				}
			}
			continue
		}
		out = append(out, Change{Field: "contact." + id + ".removed", Old: oldC, New: nil})
	}
	for id, newC := range newByID {
		if _, ok := oldByID[id]; !ok {
			out = append(out, Change{Field: "contact." + id + ".added", Old: nil, New: newC})
		}
	}

	// --- tags ---
	if !valuesEqual(oldSnap.Tags, newSnap.Tags) {
		out = append(out, Change{Field: "tags", Old: oldSnap.Tags, New: newSnap.Tags})
	}

	return out
}

func indexContacts(list []map[string]any) map[string]map[string]any {
	out := map[string]map[string]any{}
	for _, c := range list {
		if id, ok := c["id"].(string); ok && id != "" {
			out[id] = c
		}
	}
	return out
}

func valuesEqual(a, b any) bool {
	return jsonString(a) == jsonString(b)
}

func jsonString(v any) string {
	b, err := json.Marshal(v)
	if err != nil {
		return ""
	}
	return string(b)
}
