package scoring

import (
	"context"
	"strings"

	"github.com/tafakorpardazan/huginn/internal/contacts"
	"github.com/tafakorpardazan/huginn/internal/database"
	"github.com/tafakorpardazan/huginn/internal/followups"
	"github.com/tafakorpardazan/huginn/internal/leads"
	"github.com/tafakorpardazan/huginn/internal/reports"
)

// LeadScorer keeps a lead's score in sync with accumulated evidence. Every call
// report re-evaluates the lead from its full history, so the score moves after
// each report — not only the first one. It also promotes stable facts
// (decision maker, correct person) onto the lead and its contact so the next
// report can be pre-filled instead of re-asked.
type LeadScorer struct {
	db        *database.DB
	leads     *leads.Repository
	contacts  *contacts.Repository
	reports   *reports.Repository
	followups *followups.Repository
	history   *Repository
}

func NewLeadScorer(db *database.DB, leads *leads.Repository, contacts *contacts.Repository, reports *reports.Repository, followups *followups.Repository, history *Repository) *LeadScorer {
	return &LeadScorer{db: db, leads: leads, contacts: contacts, reports: reports, followups: followups, history: history}
}

// OnReportSubmitted promotes the facts carried by [rep] onto the lead/contact
// (never overwriting an established fact with a later empty value) and
// recomputes the accumulated score from the lead's base signals plus every
// historical report.
func (s *LeadScorer) OnReportSubmitted(ctx context.Context, leadID string, contactID *string, rep *reports.Report, actorID *string) error {
	lead, err := s.leads.Get(ctx, leadID)
	if err != nil {
		return err
	}
	target, err := s.pickContact(ctx, leadID, contactID)
	if err != nil {
		return err
	}

	applyReportFacts(lead, target, rep)
	if err := s.leads.UpdateReportFacts(ctx, leadID, lead.CorrectPersonReached, lead.DecisionMakerIdentified, lead.DecisionMakerName, lead.DecisionMakerRole); err != nil {
		return err
	}
	if target != nil {
		if err := s.contacts.Update(ctx, target); err != nil {
			return err
		}
	}

	reps, err := s.reports.ListForLead(ctx, leadID)
	if err != nil {
		return err
	}
	hasOpenFollowUp, err := s.hasOpenFollowUp(ctx, leadID)
	if err != nil {
		return err
	}

	res := Score(BuildSignals(lead, target, reps, s.sourceTypeRich(ctx, lead), hasOpenFollowUp))
	if err := s.leads.SetScore(ctx, leadID, res.Total); err != nil {
		return err
	}
	return s.history.Save(ctx, leadID, res, "report: "+string(rep.Outcome), actorID)
}

// pickContact returns the contact most relevant to the report: the one attached
// to the interaction if known, otherwise the decision maker, otherwise the
// primary contact, otherwise the first contact. Nil when the lead has none.
func (s *LeadScorer) pickContact(ctx context.Context, leadID string, contactID *string) (*contacts.Contact, error) {
	cs, err := s.contacts.ListForLead(ctx, leadID)
	if err != nil {
		return nil, err
	}
	if len(cs) == 0 {
		return nil, nil
	}
	if contactID != nil {
		for _, c := range cs {
			if c.ID == *contactID {
				return c, nil
			}
		}
	}
	for _, c := range cs {
		if c.IsDecisionMaker {
			return c, nil
		}
	}
	for _, c := range cs {
		if c.IsPrimaryContact {
			return c, nil
		}
	}
	return cs[0], nil
}

// applyReportFacts promotes newly-established facts onto the lead/contact.
// Empty or absent values never clear previously known facts.
func applyReportFacts(lead *leads.Lead, contact *contacts.Contact, rep *reports.Report) {
	if rep.CorrectPersonReached != nil {
		lead.CorrectPersonReached = rep.CorrectPersonReached
	}
	if rep.DecisionMakerIdentified != nil && *rep.DecisionMakerIdentified {
		lead.DecisionMakerIdentified = true
	}
	if rep.DecisionMakerName != nil && *rep.DecisionMakerName != "" {
		lead.DecisionMakerName = rep.DecisionMakerName
	}
	if rep.DecisionMakerRole != nil && *rep.DecisionMakerRole != "" {
		lead.DecisionMakerRole = rep.DecisionMakerRole
	}
	if contact == nil {
		return
	}
	if rep.DecisionMakerIdentified != nil && *rep.DecisionMakerIdentified {
		contact.IsDecisionMaker = true
	}
	if rep.DecisionMakerName != nil && *rep.DecisionMakerName != "" {
		contact.FullName = rep.DecisionMakerName
	}
	if rep.DecisionMakerRole != nil && *rep.DecisionMakerRole != "" {
		contact.JobTitle = rep.DecisionMakerRole
	}
}

// BuildSignals accumulates evidence from the lead's base facts, its contact and
// every historical report. Each report contributes positive-outcome, interest,
// need, decision-maker and purchase signals so later reports move the score.
func BuildSignals(lead *leads.Lead, contact *contacts.Contact, reps []*reports.Report, sourceTypeRich, hasOpenFollowUp bool) Signals {
	sig := Signals{}
	sig.HasIndustry = strVal(lead.Industry) != ""
	sig.HasDescription = strVal(lead.Description) != ""
	sig.HasBusinessType = strVal(lead.BusinessType) != ""
	sig.SourceRelevant = strVal(lead.Source) != ""
	sig.HasAddress = strVal(lead.Address) != ""
	sig.HasWebsite = strVal(lead.Website) != ""
	sig.SourceTypeRich = sourceTypeRich

	if contact != nil {
		sig.HasMobile = contact.Mobile != nil || contact.MobileNormalized != nil
		sig.HasPhone = contact.Phone != nil || contact.PhoneNormalized != nil
		sig.HasEmail = contact.Email != nil || contact.EmailNormalized != nil
		sig.HasSocial = contact.WhatsApp != nil || contact.Telegram != nil || contact.Bale != nil || contact.Eitaa != nil || contact.Rubika != nil
	}

	sig.InteractionCount = len(reps)
	sig.DecisionMakerIdentified = lead.DecisionMakerIdentified || (contact != nil && contact.IsDecisionMaker)
	sig.HasFollowUpScheduled = hasOpenFollowUp

	for _, rep := range reps {
		switch rep.Outcome {
		case reports.OutcomeInterested, reports.OutcomeHighlyInterested, reports.OutcomeCallbackReq, reports.OutcomeAnswered:
			sig.PositiveReports++
		}
		if rep.InterestLevel != nil {
			switch *rep.InterestLevel {
			case reports.InterestHigh:
				sig.InterestHigh = true
			case reports.InterestMedium:
				sig.InterestMedium = true
			}
		}
		if rep.NeedLevel != nil {
			switch *rep.NeedLevel {
			case reports.InterestHigh:
				sig.NeedHigh = true
			case reports.InterestMedium:
				sig.NeedMedium = true
			}
		}
		if rep.DecisionMakerIdentified != nil && *rep.DecisionMakerIdentified {
			sig.DecisionMakerIdentified = true
		}
		if purchaseTimingSoon(rep.PurchaseTiming) {
			sig.PurchaseSoon = true
		}
		if rep.EstimatedPurchaseCapacity != nil && *rep.EstimatedPurchaseCapacity != "" {
			sig.HasPurchaseCapacity = true
		}
	}
	return sig
}

func (s *LeadScorer) hasOpenFollowUp(ctx context.Context, leadID string) (bool, error) {
	fups, err := s.followups.ListForLead(ctx, leadID)
	if err != nil {
		return false, err
	}
	for _, f := range fups {
		if f.Status == followups.StatusPending || f.Status == followups.StatusSnoozed {
			return true, nil
		}
	}
	return false, nil
}

// sourceTypeRich reports whether the lead came from a rich source document
// (PDF/HTML catalogue or website), preserving the import-time signal across
// report-driven recomputes.
func (s *LeadScorer) sourceTypeRich(ctx context.Context, lead *leads.Lead) bool {
	if lead.SourceDocumentID == nil {
		return false
	}
	var mime string
	if err := s.db.Pool().QueryRow(ctx, `SELECT mime_type FROM source_documents WHERE id=$1`, *lead.SourceDocumentID).Scan(&mime); err != nil {
		return false
	}
	return mime == "application/pdf" || mime == "text/html"
}

func strVal(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}

// purchaseTimingSoon is a best-effort classification of the free-text purchase
// timing captured on a report.
func purchaseTimingSoon(t *string) bool {
	if t == nil {
		return false
	}
	switch strings.ToUpper(strings.TrimSpace(*t)) {
	case "SOON", "NOW", "IMMEDIATE", "WITHIN_1_MONTH", "WITHIN_3_MONTHS", "LESS_THAN_3_MONTHS", "HIGHLY_POTENTIAL", "HIGH":
		return true
	}
	return false
}
