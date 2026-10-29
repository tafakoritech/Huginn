package analytics

import (
	"context"
	"os"
	"testing"

	"github.com/tafakorpardazan/huginn/internal/config"
	"github.com/tafakorpardazan/huginn/internal/database"
)

func TestDebugOverview(t *testing.T) {
	if os.Getenv("HUGINN_INTEGRATION") != "1" {
		t.Skip("integration disabled")
	}
	ctx := context.Background()
	cfg, _ := config.Load()
	db, err := database.New(ctx, cfg.Postgres)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	s := NewService(db)
	out, err := s.Overview(ctx, Filters{})
	t.Logf("overview err: %v", err)
	if out != nil {
		t.Logf("overview: total=%d new=%d contacted=%d reports=%d success=%d qualified=%d disq=%d fu_pending=%d fu_overdue=%d",
			out.TotalLeads, out.NewLeads, out.ContactedLeads, out.ReportsSubmitted, out.SuccessfulContacts,
			out.QualifiedLeads, out.DisqualifiedLeads, out.PendingFollowUps, out.OverdueFollowUps)
	}
	f, err := s.Funnel(ctx, Filters{})
	t.Logf("funnel err: %v", err)
	t.Logf("funnel: %+v", f)
}
