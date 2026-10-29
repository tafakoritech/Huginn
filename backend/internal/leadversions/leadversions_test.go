package leadversions

import (
	"strings"
	"testing"
)

func leadMap(overrides map[string]any) map[string]any {
	m := map[string]any{
		"id": "lead-1", "company_name": "Acme", "status": "NEW", "score": 10,
		"city": "تهران", "created_at": "2026-01-01T00:00:00Z", "updated_at": "2026-01-01T00:00:00Z",
	}
	for k, v := range overrides {
		m[k] = v
	}
	return m
}

func contact(id string, overrides map[string]any) map[string]any {
	m := map[string]any{
		"id": id, "lead_id": "lead-1", "full_name": "Ali", "job_title": "Manager",
		"mobile": "09120000000", "is_primary_contact": false, "created_at": "x", "updated_at": "x",
	}
	for k, v := range overrides {
		m[k] = v
	}
	return m
}

func TestDiffLeadField(t *testing.T) {
	oldS := &Snapshot{Lead: leadMap(nil), Contacts: []map[string]any{}, Tags: []string{}}
	newS := &Snapshot{Lead: leadMap(map[string]any{"company_name": "Acme Corp"}), Contacts: []map[string]any{}, Tags: []string{}}
	changes := Diff(oldS, newS)
	if len(changes) != 1 || changes[0].Field != "company_name" {
		t.Fatalf("expected single company_name change, got %+v", changes)
	}
	if changes[0].Old != "Acme" || changes[0].New != "Acme Corp" {
		t.Fatalf("bad old/new: %+v", changes[0])
	}
}

func TestDiffPrimaryContact(t *testing.T) {
	oldS := &Snapshot{
		Lead:     leadMap(nil),
		Contacts: []map[string]any{contact("c1", map[string]any{"is_primary_contact": false})},
		Tags:     []string{},
	}
	newS := &Snapshot{
		Lead:     leadMap(nil),
		Contacts: []map[string]any{contact("c1", map[string]any{"is_primary_contact": true})},
		Tags:     []string{},
	}
	changes := Diff(oldS, newS)
	if len(changes) != 1 {
		t.Fatalf("expected one change, got %+v", changes)
	}
	c := changes[0]
	if c.Field != "contact.c1.is_primary_contact" || c.Old != false || c.New != true {
		t.Fatalf("bad primary contact change: %+v", c)
	}
}

func TestDiffContactAddedAndRemoved(t *testing.T) {
	oldS := &Snapshot{Lead: leadMap(nil), Contacts: []map[string]any{}, Tags: []string{}}
	newS := &Snapshot{Lead: leadMap(nil), Contacts: []map[string]any{contact("c1", nil)}, Tags: []string{}}
	changes := Diff(oldS, newS)
	if len(changes) != 1 || changes[0].Field != "contact.c1.added" {
		t.Fatalf("expected added change, got %+v", changes)
	}

	changes = Diff(newS, oldS)
	if len(changes) != 1 || changes[0].Field != "contact.c1.removed" {
		t.Fatalf("expected removed change, got %+v", changes)
	}
}

func TestDiffTags(t *testing.T) {
	oldS := &Snapshot{Lead: leadMap(nil), Contacts: []map[string]any{}, Tags: []string{"a", "b"}}
	newS := &Snapshot{Lead: leadMap(nil), Contacts: []map[string]any{}, Tags: []string{"a", "c"}}
	changes := Diff(oldS, newS)
	if len(changes) != 1 || changes[0].Field != "tags" {
		t.Fatalf("expected tags change, got %+v", changes)
	}
}

func TestDiffNoChanges(t *testing.T) {
	oldS := &Snapshot{Lead: leadMap(nil), Contacts: []map[string]any{contact("c1", nil)}, Tags: []string{"a"}}
	newS := &Snapshot{Lead: leadMap(nil), Contacts: []map[string]any{contact("c1", nil)}, Tags: []string{"a"}}
	if HasChanges(oldS, newS) {
		t.Fatal("identical snapshots must have no changes")
	}
}

func TestDiffCreated(t *testing.T) {
	changes := Diff(nil, &Snapshot{Lead: leadMap(nil), Contacts: []map[string]any{}, Tags: []string{}})
	if len(changes) != 1 || !strings.HasPrefix(changes[0].Field, "lead.created") {
		t.Fatalf("expected created change, got %+v", changes)
	}
}

func TestDiffIgnoresSystemFields(t *testing.T) {
	oldS := &Snapshot{Lead: leadMap(map[string]any{"score": 5}), Contacts: []map[string]any{}, Tags: []string{}}
	newS := &Snapshot{Lead: leadMap(map[string]any{"score": 5, "updated_at": "2026-01-02T00:00:00Z"}), Contacts: []map[string]any{}, Tags: []string{}}
	if HasChanges(oldS, newS) {
		t.Fatal("updated_at change must be ignored")
	}
}
