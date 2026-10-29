package seed

import (
	"context"
	"log/slog"
	"time"

	"github.com/tafakorpardazan/huginn/internal/auth"
	"github.com/tafakorpardazan/huginn/internal/config"
	"github.com/tafakorpardazan/huginn/internal/contacts"
	"github.com/tafakorpardazan/huginn/internal/database"
	"github.com/tafakorpardazan/huginn/internal/evidence"
	"github.com/tafakorpardazan/huginn/internal/imports"
	"github.com/tafakorpardazan/huginn/internal/interactions"
	"github.com/tafakorpardazan/huginn/internal/leads"
	"github.com/tafakorpardazan/huginn/internal/passwd"
	"github.com/tafakorpardazan/huginn/internal/rbac"
	"github.com/tafakorpardazan/huginn/internal/reports"
	"github.com/tafakorpardazan/huginn/internal/scoring"
	"github.com/tafakorpardazan/huginn/internal/users"
)

// Run seeds development data. All credentials are development defaults.
func Run(ctx context.Context, db *database.DB, cfg *config.Config, logger *slog.Logger) error {
	userRepo := users.NewRepository(db)

	adminHash, _ := passwd.HashPassword(cfg.Seed.AdminPassword)
	managerHash, _ := passwd.HashPassword(cfg.Seed.ManagerPassword)
	agentHash, _ := passwd.HashPassword(cfg.Seed.AgentPassword)

	_, err := userRepo.Create(ctx, &users.User{
		Name: "Super Admin", Username: "admin", Role: rbac.RoleSuperAdmin, IsActive: true, PasswordHash: adminHash,
	})
	if err != nil && !isConflict(err) {
		return err
	}
	manager, err := userRepo.Create(ctx, &users.User{
		Name: "Sales Manager", Username: "manager", Role: rbac.RoleSalesDevManager, IsActive: true, PasswordHash: managerHash,
	})
	if err != nil && !isConflict(err) {
		return err
	}
	if isConflict(err) {
		manager, err = userRepo.GetByUsername(ctx, "manager")
		if err != nil {
			return err
		}
	}
	agent1, err := userRepo.Create(ctx, &users.User{
		Name: "Agent One", Username: "agent1", Role: rbac.RoleSalesAgent, IsActive: true, PasswordHash: agentHash,
	})
	if err != nil && !isConflict(err) {
		return err
	}
	if isConflict(err) {
		agent1, err = userRepo.GetByUsername(ctx, "agent1")
		if err != nil {
			return err
		}
	}
	agent2, err := userRepo.Create(ctx, &users.User{
		Name: "Agent Two", Username: "agent2", Role: rbac.RoleSalesAgent, IsActive: true, PasswordHash: agentHash,
	})
	if err != nil && !isConflict(err) {
		return err
	}
	if isConflict(err) {
		agent2, err = userRepo.GetByUsername(ctx, "agent2")
		if err != nil {
			return err
		}
	}
	logger.Info("seeded users",
		"admin", "admin/"+cfg.Seed.AdminPassword,
		"manager", "manager/"+cfg.Seed.ManagerPassword,
		"agent1", "agent1/"+cfg.Seed.AgentPassword,
		"agent2", "agent2/"+cfg.Seed.AgentPassword,
	)

	// Sample import batch (idempotent: skip if a batch with this name exists)
	importRepo := imports.NewRepository(db)
	batches, _ := importRepo.ListBatches(ctx, nil, 1, 0)
	var batch *imports.Batch
	for _, b := range batches {
		if b.Name == "Toy Manufacturer Directory" {
			batch = b
			break
		}
	}
	if batch == nil {
		batch, err = importRepo.CreateBatch(ctx, &imports.Batch{
			Name: "Toy Manufacturer Directory", Description: strPtr("Sample directory of toy manufacturers"),
			SourceType: imports.SourceTXT, Status: imports.StatusCompleted, CreatedBy: &manager.ID,
		})
		if err != nil {
			return err
		}
	}
	doc, err := importRepo.CreateDocument(ctx, &imports.Document{
		BatchID: batch.ID, Status: "ANALYZED", CreatedBy: &manager.ID,
		OriginalFilename: strPtr("sample.txt"), MimeType: strPtr("text/plain"),
		ExtractedText: strPtr("تولید و پخش عمده انواع عروسک پولیشی در تهران. تماس: 09357578808"),
	})
	if err != nil {
		return err
	}

	leadRepo := leads.NewRepository(db)
	contactRepo := contacts.NewRepository(db)
	scoreRepo := scoring.NewRepository(db)
	leadSvc := leads.NewService(leadRepo)

	// Sample lead 1 - assigned to agent1 with contacts
	lead1, err := leadSvc.Create(ctx, leads.CreateParams{
		CompanyName: "بازرگانی پولیش", BusinessType: strPtr("manufacturer"),
		Industry: strPtr("toy"), City: strPtr("تهران"), Province: strPtr("تهران"),
		Description: strPtr("تولید و پخش عمده انواع عروسک پولیشی"),
		SourceBatchID: &batch.ID, SourceDocumentID: &doc.ID,
	})
	if err != nil {
		return err
	}
	_ = leadRepo.SetOwner(ctx, lead1.ID, &agent1.ID)
	_, _ = leadSvc.ChangeStatus(ctx, lead1.ID, leads.StatusReady, "seeded", manager.ID)
	_, _ = leadSvc.ChangeStatus(ctx, lead1.ID, leads.StatusAssigned, "seeded", manager.ID)

	mobile := "09357578808"
	_, err = contactRepo.Create(ctx, &contacts.Contact{
		LeadID: lead1.ID, FullName: strPtr("رضا محمدی"), JobTitle: strPtr("مدیر فروش"),
		Mobile: &mobile, IsDecisionMaker: true, IsPrimaryContact: true,
	})
	if err != nil {
		return err
	}
	res := scoring.Score(scoring.Signals{HasIndustry: true, HasDescription: true, HasBusinessType: true,
		SourceRelevant: true, HasMobile: true, HasAddress: false, HasWebsite: false})
	_ = scoreRepo.Save(ctx, lead1.ID, res, "seed", nil)
	_ = leadRepo.SetScore(ctx, lead1.ID, res.Total)

	// Sample lead 2 - unassigned
	lead2, err := leadSvc.Create(ctx, leads.CreateParams{
		CompanyName: "شیرینی سرای آرش", BusinessType: strPtr("wholesaler"),
		Industry: strPtr("food"), City: strPtr("اصفهان"), Province: strPtr("اصفهان"),
		SourceBatchID: &batch.ID, SourceDocumentID: &doc.ID,
	})
	if err != nil {
		return err
	}
	_ = scoreRepo.Save(ctx, lead2.ID, scoring.Score(scoring.Signals{HasMobile: true, HasDescription: true, SourceRelevant: true}), "seed", nil)

	// Sample interaction with pending report for agent1
	interRepo := interactions.NewRepository(db)
	inter, err := interRepo.Create(ctx, &interactions.Interaction{
		LeadID: lead1.ID, UserID: agent1.ID, Channel: interactions.ChannelCall,
		Direction: interactions.DirectionOutbound, Objective: strPtr("FIRST_CONTACT"),
		ActionStatus: interactions.ActionCompleted, ReportStatus: interactions.ReportPending,
	})
	if err != nil {
		return err
	}
	_ = inter

	// Sample completed report for lead2 (qualification path)
	inter2, err := interRepo.Create(ctx, &interactions.Interaction{
		LeadID: lead2.ID, UserID: agent2.ID, Channel: interactions.ChannelCall,
		Direction: interactions.DirectionOutbound, Objective: strPtr("FIRST_CONTACT"),
		ActionStatus: interactions.ActionCompleted, ReportStatus: interactions.ReportSubmitted,
	})
	if err != nil {
		return err
	}
	reportRepo := reports.NewRepository(db)
	high := reports.InterestHigh
	nextFollow := reports.NextFollowUp
	_, err = reportRepo.Create(ctx, &reports.Report{
		InteractionID: inter2.ID, Outcome: reports.OutcomeInterested,
		InterestLevel: &high, NextAction: &nextFollow, SubmittedBy: agent2.ID,
		DecisionMakerIdentified: boolPtr(true),
	})
	if err != nil {
		return err
	}
	_ = leadRepo.SetScore(ctx, lead2.ID, 85)

	// Sample evidence
	evRepo := evidence.NewRepository(db)
	_ = evRepo.Create(ctx, &evidence.Item{
		LeadID: lead1.ID, SourceDocumentID: &doc.ID, FieldName: "business_type",
		SourceExcerpt: strPtr("تولید و پخش عمده انواع عروسک پولیشی"), Confidence: 0.94,
	})

	// A qualified sample lead
	lead3, err := leadSvc.Create(ctx, leads.CreateParams{
		CompanyName: "پخش آراد", BusinessType: strPtr("distributor"),
		Industry: strPtr("electronics"), City: strPtr("شیراز"), Province: strPtr("فارس"),
		SourceBatchID: &batch.ID, SourceDocumentID: &doc.ID,
	})
	if err != nil {
		return err
	}
	_ = leadRepo.SetOwner(ctx, lead3.ID, &agent2.ID)
	_ = leadRepo.SetScore(ctx, lead3.ID, 92)
	_, _ = leadSvc.ChangeStatus(ctx, lead3.ID, leads.StatusReady, "seed", manager.ID)
	_, _ = leadSvc.ChangeStatus(ctx, lead3.ID, leads.StatusAssigned, "seed", manager.ID)
	_, _ = leadSvc.ChangeStatus(ctx, lead3.ID, leads.StatusContacting, "seed", manager.ID)
	_, _ = leadSvc.ChangeStatus(ctx, lead3.ID, leads.StatusQualified, "seed", manager.ID)

	logger.Info("seed complete",
		"batch_id", batch.ID,
		"lead1", lead1.ID,
		"lead2", lead2.ID,
		"qualified_lead", lead3.ID,
	)
	return nil
}

// AuthManager is unused here but kept for parity with auth wiring.
var _ = auth.RoleViewer

func strPtr(s string) *string { return &s }
func boolPtr(b bool) *bool    { return &b }

func isConflict(err error) bool {
	return err != nil && (err.Error() == users.ErrConflict.Error() || contains(err.Error(), "conflict"))
}

func contains(s, sub string) bool {
	return len(s) >= len(sub) && (s == sub || indexOf(s, sub) >= 0)
}

func indexOf(s, sub string) int {
	for i := 0; i+len(sub) <= len(s); i++ {
		if s[i:i+len(sub)] == sub {
			return i
		}
	}
	return -1
}

var _ = time.Now
