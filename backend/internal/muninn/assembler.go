package muninn

import (
	"context"
	"time"

	"github.com/tafakorpardazan/huginn/internal/contacts"
	"github.com/tafakorpardazan/huginn/internal/database"
	"github.com/tafakorpardazan/huginn/internal/followups"
	"github.com/tafakorpardazan/huginn/internal/imports"
	"github.com/tafakorpardazan/huginn/internal/interactions"
	"github.com/tafakorpardazan/huginn/internal/leads"
	"github.com/tafakorpardazan/huginn/internal/notes"
	"github.com/tafakorpardazan/huginn/internal/reports"
)

// Assembler builds export bundles from repositories.
type Assembler struct {
	db          *database.DB
	leadRepo    *leads.Repository
	contactRepo *contacts.Repository
	noteRepo    *notes.Repository
	interRepo   *interactions.Repository
	reportRepo  *reports.Repository
	importRepo  *imports.Repository
	followRepo  *followups.Repository
	leadSvc     *leads.Service
}

// NewAssembler wires the assembler dependencies.
func NewAssembler(db *database.DB, leadRepo *leads.Repository, contactRepo *contacts.Repository,
	noteRepo *notes.Repository, interRepo *interactions.Repository, reportRepo *reports.Repository,
	importRepo *imports.Repository, followRepo *followups.Repository, leadSvc *leads.Service) *Assembler {
	return &Assembler{
		db: db, leadRepo: leadRepo, contactRepo: contactRepo, noteRepo: noteRepo,
		interRepo: interRepo, reportRepo: reportRepo, importRepo: importRepo,
		followRepo: followRepo, leadSvc: leadSvc,
	}
}

// ExportBundle assembles a full payload for a lead.
func (a *Assembler) ExportBundle(ctx context.Context, leadID string) (*Bundle, error) {
	lead, err := a.leadRepo.Get(ctx, leadID)
	if err != nil {
		return nil, err
	}
	contactsList, err := a.contactRepo.ListForLead(ctx, leadID)
	if err != nil {
		return nil, err
	}
	notesList, _ := a.noteRepo.ListForLead(ctx, leadID)
	interList, _ := a.interRepo.ListForLead(ctx, leadID)

	b := &Bundle{
		CompanyInfo: &CompanyInfo{
			CompanyName: lead.CompanyName, BusinessType: lead.BusinessType, Industry: lead.Industry,
			City: lead.City, Province: lead.Province, Address: lead.Address,
			Website: lead.Website, Instagram: lead.Instagram, Description: lead.Description,
		},
		Score: lead.Score,
	}
	for _, c := range contactsList {
		b.Contacts = append(b.Contacts, &ContactInfo{
			FullName: c.FullName, JobTitle: c.JobTitle, Mobile: c.Mobile, Phone: c.Phone,
			Email: c.Email, IsDecisionMaker: c.IsDecisionMaker,
		})
	}
	for _, n := range notesList {
		b.Notes = append(b.Notes, n.Text)
	}
	for _, i := range interList {
		var outcome *string
		if rep, err := a.reportRepo.GetByInteraction(ctx, i.ID); err == nil {
			o := string(rep.Outcome)
			outcome = &o
		}
		b.Interactions = append(b.Interactions, &InteractionInfo{
			Channel: string(i.Channel), CreatedAt: i.CreatedAt.Format(time.RFC3339),
			Summary: i.SuggestedMessage, Outcome: outcome,
		})
	}
	if lead.SourceBatchID != nil {
		if batch, err := a.importRepo.GetBatch(ctx, *lead.SourceBatchID); err == nil {
			bt := string(batch.SourceType)
			b.Source = &SourceInfo{BatchName: &batch.Name, SourceType: &bt}
		}
	}
	if fups, err := a.followRepo.ListForLead(ctx, leadID); err == nil && len(fups) > 0 {
		b.NextAction = string(fups[0].Type)
	}
	return b, nil
}
