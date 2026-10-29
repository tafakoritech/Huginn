package integration

import (
	"context"
	"strings"
	"testing"

	"github.com/tafakorpardazan/huginn/internal/contacts"
	"github.com/tafakorpardazan/huginn/internal/database"
	"github.com/tafakorpardazan/huginn/internal/leadversions"
	"github.com/tafakorpardazan/huginn/internal/leads"
	"github.com/tafakorpardazan/huginn/internal/passwd"
	"github.com/tafakorpardazan/huginn/internal/rbac"
	"github.com/tafakorpardazan/huginn/internal/users"
)

func seedAgent(t *testing.T, db *database.DB) string {
	t.Helper()
	userRepo := users.NewRepository(db)
	hash, _ := passwd.HashPassword("S3cret!pass")
	u, err := userRepo.Create(context.Background(), &users.User{
		Name: "IT Agent", Username: "it_ver_" + ts(), Role: rbac.RoleSalesAgent, IsActive: true, PasswordHash: hash,
	})
	if err != nil {
		t.Fatalf("create user: %v", err)
	}
	return u.ID
}

// TestVersionedLeadEditing exercises the immutable, versioned edit flow against
// a real database: every mutation appends a version while preserving prior
// state, and each change record captures the actor and old/new values.
func TestVersionedLeadEditing(t *testing.T) {
	db := setupDB(t)
	actor := seedAgent(t, db)
	leadRepo := leads.NewRepository(db)
	contactRepo := contacts.NewRepository(db)
	versionsRepo := leadversions.NewRepository(db)
	svc := leads.NewService(leadRepo, leads.Deps{Contacts: contactRepo, Versions: versionsRepo})

	// 1. Create -> version 1
	lead, err := svc.Create(context.Background(), leads.CreateParams{CompanyName: "Versioned Co"})
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	if got, _ := svc.CountVersions(context.Background(), lead.ID); got != 1 {
		t.Fatalf("expected 1 version after create, got %d", got)
	}

	// 2. Update company name -> version 2, change recorded with actor + role
	patches := map[string]*string{}
	name := "Versioned Co Ltd"
	patches["company_name"] = &name
	if _, err := svc.Update(context.Background(), lead.ID, patches, actor, "legal rename"); err != nil {
		t.Fatalf("update: %v", err)
	}
	if got, _ := svc.CountVersions(context.Background(), lead.ID); got != 2 {
		t.Fatalf("expected 2 versions, got %d", got)
	}

	// 3. Add a contact -> version 3
	mobile := "09120001122"
	c, err := svc.CreateContact(context.Background(), lead.ID, &contacts.Contact{
		FullName: strPtr("Reception"), JobTitle: strPtr("Receptionist"), Mobile: &mobile,
	}, actor, "")
	if err != nil {
		t.Fatalf("create contact: %v", err)
	}

	// 4. Promote to primary contact -> version 4, old=false/new=true recorded
	promote := map[string]any{"is_primary_contact": boolPtr(true)}
	promoted, err := svc.UpdateContact(context.Background(), lead.ID, c.ID, promote, actor, "Learned real decision maker")
	if err != nil {
		t.Fatalf("promote contact: %v", err)
	}
	if !promoted.IsPrimaryContact {
		t.Fatal("contact not promoted")
	}

	// Verify versions list (newest first: 4,3,2,1)
	versions, err := svc.ListVersions(context.Background(), lead.ID, 100, 0)
	if err != nil {
		t.Fatalf("list versions: %v", err)
	}
	if len(versions) != 4 {
		t.Fatalf("expected 4 versions, got %d", len(versions))
	}
	for i, v := range versions {
		if v.Version != 4-i {
			t.Fatalf("version order broken: %d has version %d", i, v.Version)
		}
		if v.Version == 1 {
			continue // creation has no actor
		}
		if v.ChangedBy == nil || *v.ChangedBy != actor {
			t.Fatalf("version %d missing actor", v.Version)
		}
		if v.ChangedByRole == nil || *v.ChangedByRole != string(rbac.RoleSalesAgent) {
			t.Fatalf("version %d missing role, got %v", v.Version, v.ChangedByRole)
		}
	}

	// Promote change must carry old=false new=true.
	found := false
	for _, ch := range versions[0].Changes {
		if strings.Contains(ch.Field, "is_primary_contact") {
			found = true
			if ch.Old != false || ch.New != true {
				t.Fatalf("expected is_primary_contact false->true, got %+v", ch)
			}
		}
	}
	if !found {
		t.Fatalf("missing primary contact change in %+v", versions[0].Changes)
	}

	// The update version must capture the old and new company name.
	foundCompany := false
	for _, v := range versions {
		for _, ch := range v.Changes {
			if ch.Field == "company_name" {
				foundCompany = true
				if ch.Old != "Versioned Co" || ch.New != "Versioned Co Ltd" {
					t.Fatalf("bad company change: %+v", ch)
				}
			}
		}
	}
	if !foundCompany {
		t.Fatal("missing company_name change record")
	}

	// Old values must be preserved: oldest snapshot must have original name.
	v1 := versions[3]
	if !strings.Contains(string(v1.Data), "Versioned Co") {
		t.Fatalf("version 1 snapshot should preserve original name: %s", v1.Data)
	}
}

// TestVersionedDeleteContact preserves removed contacts in history.
func TestVersionedDeleteContact(t *testing.T) {
	db := setupDB(t)
	actor := seedAgent(t, db)
	leadRepo := leads.NewRepository(db)
	contactRepo := contacts.NewRepository(db)
	versionsRepo := leadversions.NewRepository(db)
	svc := leads.NewService(leadRepo, leads.Deps{Contacts: contactRepo, Versions: versionsRepo})

	lead, err := svc.Create(context.Background(), leads.CreateParams{CompanyName: "Delete Co"})
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	c, err := svc.CreateContact(context.Background(), lead.ID, &contacts.Contact{FullName: strPtr("Someone")}, actor, "")
	if err != nil {
		t.Fatalf("create contact: %v", err)
	}
	if err := svc.DeleteContact(context.Background(), lead.ID, c.ID, actor, "removed"); err != nil {
		t.Fatalf("delete contact: %v", err)
	}

	versions, _ := svc.ListVersions(context.Background(), lead.ID, 100, 0)
	// versions: create(1), add contact(2), delete contact(3)
	if len(versions) != 3 {
		t.Fatalf("expected 3 versions, got %d", len(versions))
	}
	foundRemoved := false
	for _, ch := range versions[0].Changes {
		if strings.Contains(ch.Field, ".removed") {
			foundRemoved = true
			if ch.Old == nil {
				t.Fatal("removed contact change must carry the old value")
			}
		}
	}
	if !foundRemoved {
		t.Fatalf("expected contact removed change, got %+v", versions[0].Changes)
	}

	// Historical version still contains the contact data.
	if !strings.Contains(string(versions[1].Data), "Someone") {
		t.Fatalf("historical version should preserve the contact: %s", versions[1].Data)
	}

	// Live contact list no longer includes it.
	list, err := contactRepo.ListForLead(context.Background(), lead.ID)
	if err != nil {
		t.Fatalf("list contacts: %v", err)
	}
	if len(list) != 0 {
		t.Fatalf("expected no active contacts, got %d", len(list))
	}
}

// TestConcurrentVersionedEdits ensures concurrent edits serialize on the lead
// lock and never produce duplicate or corrupted version numbers.
func TestConcurrentVersionedEdits(t *testing.T) {
	db := setupDB(t)
	actor := seedAgent(t, db)
	leadRepo := leads.NewRepository(db)
	contactRepo := contacts.NewRepository(db)
	versionsRepo := leadversions.NewRepository(db)
	svc := leads.NewService(leadRepo, leads.Deps{Contacts: contactRepo, Versions: versionsRepo})

	lead, err := svc.Create(context.Background(), leads.CreateParams{CompanyName: "Concurrent Co"})
	if err != nil {
		t.Fatalf("create: %v", err)
	}

	const workers = 8
	errs := make(chan error, workers)
	done := make(chan struct{}, workers)
	for i := 0; i < workers; i++ {
		go func(i int) {
			defer func() { done <- struct{}{} }()
			city := "City " + string(rune('A'+i))
			_, err := svc.Update(context.Background(), lead.ID, map[string]*string{"city": &city}, actor, "")
			errs <- err
		}(i)
	}
	for i := 0; i < workers; i++ {
		<-done
	}
	close(errs)
	for err := range errs {
		if err != nil {
			t.Fatalf("concurrent update failed: %v", err)
		}
	}

	versions, err := svc.ListVersions(context.Background(), lead.ID, 200, 0)
	if err != nil {
		t.Fatalf("list versions: %v", err)
	}
	// create + 8 updates
	if len(versions) != workers+1 {
		t.Fatalf("expected %d versions, got %d", workers+1, len(versions))
	}
	seen := map[int]bool{}
	for _, v := range versions {
		if seen[v.Version] {
			t.Fatalf("duplicate version number %d", v.Version)
		}
		seen[v.Version] = true
	}
	for i := 1; i <= workers+1; i++ {
		if !seen[i] {
			t.Fatalf("missing version number %d (seen %v)", i, seen)
		}
	}

	// Latest version must equal the current lead state.
	l, err := svc.Get(context.Background(), lead.ID)
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	if got, _ := svc.CountVersions(context.Background(), lead.ID); got != versions[0].Version {
		t.Fatalf("current_version mismatch: count=%d latest=%d", got, versions[0].Version)
	}
	_ = l
}

func strPtr(s string) *string { return &s }
func boolPtr(b bool) *bool    { return &b }
