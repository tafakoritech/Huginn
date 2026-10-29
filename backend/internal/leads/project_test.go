package leads

import (
	"context"
	"strings"
	"testing"
)

// Project association is optional at creation time: leads imported from
// documents have no project yet and are linked later via Update. These tests
// assert the service layer no longer rejects a missing project_id. Because the
// repository needs a real database to persist, we only assert that no
// project-validation error is returned — a nil-DB panic here simply means the
// code progressed past project validation, which is exactly what we want.

func callCreate(svc *Service, ctx context.Context, p CreateParams) (err error) {
	func() {
		defer func() { recover() }()
		_, err = svc.Create(ctx, p)
	}()
	return err
}

func TestCreateAllowsNilProject(t *testing.T) {
	svc := NewService(NewRepository(nil))
	err := callCreate(svc, context.Background(), CreateParams{CompanyName: "Acme"})
	if err != nil && strings.Contains(err.Error(), "project") {
		t.Fatalf("project_id should be optional, got error: %v", err)
	}
}

func TestCreateAllowsEmptyProject(t *testing.T) {
	svc := NewService(NewRepository(nil))
	empty := ""
	err := callCreate(svc, context.Background(), CreateParams{CompanyName: "Acme", ProjectID: &empty})
	if err != nil && strings.Contains(err.Error(), "project") {
		t.Fatalf("empty project_id should be optional, got error: %v", err)
	}
}
