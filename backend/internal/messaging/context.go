package messaging

import (
	"context"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/tafakorpardazan/huginn/internal/campaigns"
	"github.com/tafakorpardazan/huginn/internal/contacts"
	"github.com/tafakorpardazan/huginn/internal/interactions"
	"github.com/tafakorpardazan/huginn/internal/leads"
	"github.com/tafakorpardazan/huginn/internal/notes"
	"github.com/tafakorpardazan/huginn/internal/projects"
	"github.com/tafakorpardazan/huginn/internal/reports"
)

// LeadContext is the structured, real context of a lead that is fed to the
// LLM when generating outreach. Every field is sourced from the database; the
// generator must never fabricate anything that is not present here.
type LeadContext struct {
	LeadID        string
	CompanyName   string
	DisplayName   string
	BusinessType  string
	Industry      string
	City          string
	Province      string
	Address       string
	Website       string
	Instagram     string
	Description   string
	Source        string
	Status        string
	QualityStatus string
	Tags          string
	Score         int
	// ContactName and ContactPosition are the primary contact's name and job
	// title. They are sourced from the database and drive the "Name:" and
	// "Position:" lines of the LEAD INFORMATION block fed to the LLM.
	ContactName      string
	ContactPosition  string
	ContactSummary   string
	PastInteractions string
	PreviousReports  string
	Notes            string
	// Project is the sales-initiative context that scopes the lead. When nil,
	// the lead has no project and AI generation must not proceed (the caller
	// should block scenario generation). It carries the business context
	// (product, target customer, value proposition, pain points, etc.) that
	// differentiates AI output between projects.
	Project *ProjectContext
}

// ProjectContext is the AI-relevant slice of a Project that drives scenario
// personalization. It is sourced entirely from the database; never fabricated.
type ProjectContext struct {
	ProjectID          string
	Title              string
	Description        string
	ProductInformation string
	TargetCustomer     string
	ValueProposition   string
	PainPoints         string
	SalesObjective     string
	CommonObjections   string
	Campaign           string
}

// knownFieldGroups lists the lead field groups surfaced to the LLM so the DATA
// COVERAGE block can mark exactly what is KNOWN vs UNKNOWN for the prospect.
var knownFieldGroups = []struct {
	label string
	value func(*LeadContext) string
}{
	{"Contact person", func(c *LeadContext) string { return c.ContactName }},
	{"Company name", func(c *LeadContext) string { return c.CompanyName }},
	{"Store / display name", func(c *LeadContext) string { return c.DisplayName }},
	{"Business category", func(c *LeadContext) string { return c.BusinessType }},
	{"Industry", func(c *LeadContext) string { return c.Industry }},
	{"Business description", func(c *LeadContext) string { return c.Description }},
	{"Location", func(c *LeadContext) string { return joinNonEmpty(c.City, c.Province, c.Address) }},
	{"Website", func(c *LeadContext) string { return c.Website }},
	{"Instagram", func(c *LeadContext) string { return c.Instagram }},
	{"Lead source", func(c *LeadContext) string { return c.Source }},
	{"Tags", func(c *LeadContext) string { return c.Tags }},
	{"Notes", func(c *LeadContext) string { return c.Notes }},
	{"Previous interactions", func(c *LeadContext) string { return c.PastInteractions }},
	{"Previous call reports", func(c *LeadContext) string { return c.PreviousReports }},
}

// Render produces the RECIPIENT block of the prompt.
func (c *LeadContext) Render() string {
	var b strings.Builder
	writeKV := func(k, v string) {
		if strings.TrimSpace(v) == "" {
			return
		}
		b.WriteString(k + ": " + v + "\n")
	}
	if c.Project != nil && (c.Project.Title != "" || c.Project.Description != "") {
		b.WriteString("\n--- PROJECT CONTEXT (the sales initiative this lead belongs to) ---\n")
		writeKV("Project Title", c.Project.Title)
		writeKV("Project Description", c.Project.Description)
		writeKV("Product / Service", c.Project.ProductInformation)
		writeKV("Target Customer", c.Project.TargetCustomer)
		writeKV("Customer Pain Points", c.Project.PainPoints)
		writeKV("Value Proposition", c.Project.ValueProposition)
		writeKV("Sales Goal", c.Project.SalesObjective)
		writeKV("Common Objections", c.Project.CommonObjections)
		writeKV("Campaign", c.Project.Campaign)
		b.WriteString("--- END PROJECT CONTEXT ---\n\n")
	}
	b.WriteString("\n--- LEAD INFORMATION (the exact lead being called) ---\n")
	writeKV("Name", c.ContactName)
	writeKV("Company", c.CompanyName)
	writeKV("Industry", c.Industry)
	writeKV("Position", c.ContactPosition)
	writeKV("Notes", c.Notes)
	writeKV("Previous Interactions", c.PastInteractions)
	writeKV("Previous Call Reports", c.PreviousReports)
	writeKV("Store/display name", c.DisplayName)
	writeKV("Business category", c.BusinessType)
	writeKV("City", joinNonEmpty(c.City, c.Province))
	writeKV("Address", c.Address)
	writeKV("Website", c.Website)
	writeKV("Instagram", c.Instagram)
	writeKV("Products / services (from company description)", c.Description)
	writeKV("Lead source", c.Source)
	writeKV("Current status", c.Status)
	writeKV("Tags", c.Tags)
	writeKV("Primary contact", c.ContactSummary)
	if c.Score > 0 {
		writeKV("Lead score", fmt.Sprintf("%d", c.Score))
	}

	b.WriteString("\n--- DATA COVERAGE (what we actually know about this lead) ---\n")
	known, unknown := c.coverage()
	if len(known) > 0 {
		b.WriteString("KNOWN: " + strings.Join(known, ", ") + "\n")
	}
	if len(unknown) > 0 {
		b.WriteString("UNKNOWN: " + strings.Join(unknown, ", ") + "\n")
	}
	b.WriteString("Rule: treat UNKNOWN fields as not-yet-known. Convert them into discovery questions; never invent values for them.\n")
	b.WriteString("--- END DATA COVERAGE ---\n")
	return strings.TrimRight(b.String(), "\n")
}

// coverage splits the known field groups into present/absent labels.
func (c *LeadContext) coverage() (known, unknown []string) {
	for _, g := range knownFieldGroups {
		if strings.TrimSpace(g.value(c)) != "" {
			known = append(known, g.label)
		} else {
			unknown = append(unknown, g.label)
		}
	}
	return known, unknown
}

func joinNonEmpty(parts ...string) string {
	out := ""
	for _, p := range parts {
		if strings.TrimSpace(p) == "" {
			continue
		}
		if out != "" {
			out += ", "
		}
		out += p
	}
	return out
}

// ContextBuilder gathers the real per-lead context used for message and
// call-scenario generation. It never calls the LLM.
type ContextBuilder struct {
	interactions *interactions.Repository
	reports      *reports.Repository
	notes        *notes.Repository
	leads        *leads.Repository
	contacts     *contacts.Repository
	projects     *projects.Repository
	campaigns    *campaigns.Repository
	logger       *slog.Logger
}

// NewContextBuilder assembles a ContextBuilder. An optional logger may be
// supplied last; it is used to contextualize retrieval failures (e.g. previous
// call reports) so an error is never silently indistinguishable from a
// legitimate empty result.
func NewContextBuilder(interactions *interactions.Repository, reports *reports.Repository, notes *notes.Repository, leads *leads.Repository, contacts *contacts.Repository, projects *projects.Repository, campaignRepos ...*campaigns.Repository) *ContextBuilder {
	var campaignRepo *campaigns.Repository
	if len(campaignRepos) > 0 {
		campaignRepo = campaignRepos[0]
	}
	return &ContextBuilder{
		interactions: interactions,
		reports:      reports,
		notes:        notes,
		leads:        leads,
		contacts:     contacts,
		projects:     projects,
		campaigns:    campaignRepo,
		logger:       slog.Default(),
	}
}

// WithLogger attaches a structured logger to the builder.
func (b *ContextBuilder) WithLogger(l *slog.Logger) *ContextBuilder {
	if l != nil {
		b.logger = l
	}
	return b
}

// Build assembles the context for a lead. contact may be nil. When the lead
// belongs to a Project, its business context is attached so AI generation is
// project-personalized; leads without a project carry no project context and
// generation is blocked upstream (ErrNoProject).
func (b *ContextBuilder) Build(ctx context.Context, lead *leads.Lead, contact *contacts.Contact) (*LeadContext, error) {
	c := b.buildBase(ctx, lead, contact)
	if lead.ProjectID != nil && *lead.ProjectID != "" && b.projects != nil {
		if proj, err := b.projects.Get(ctx, *lead.ProjectID); err == nil && proj != nil {
			c.Project = &ProjectContext{
				ProjectID:          proj.ID,
				Title:              proj.Title,
				Description:        proj.Description,
				ProductInformation: proj.ProductInformation(),
				TargetCustomer:     proj.TargetCustomer(),
				ValueProposition:   strVal(proj.ValueProposition),
				PainPoints:         strVal(proj.CustomerPainPoints),
				SalesObjective:     strVal(proj.SalesObjective),
				CommonObjections:   strVal(proj.CommonObjections),
				Campaign:           b.campaignFor(ctx, proj.CampaignID),
			}
		}
	}
	return c, nil
}

// campaignFor renders the parent campaign (if any) as a short context string.
func (b *ContextBuilder) campaignFor(ctx context.Context, campaignID *string) string {
	if campaignID == nil || *campaignID == "" || b.campaigns == nil {
		return ""
	}
	camp, err := b.campaigns.Get(ctx, *campaignID)
	if err != nil || camp == nil {
		return ""
	}
	parts := []string{}
	if camp.Title != "" {
		parts = append(parts, camp.Title)
	}
	if camp.Objective != nil && strings.TrimSpace(*camp.Objective) != "" {
		parts = append(parts, *camp.Objective)
	}
	if camp.Description != nil && strings.TrimSpace(*camp.Description) != "" {
		parts = append(parts, *camp.Description)
	}
	return strings.Join(parts, " — ")
}

// buildBase assembles the lead, contact, tags, notes, interaction history and
// previous call reports.
func (b *ContextBuilder) buildBase(ctx context.Context, lead *leads.Lead, contact *contacts.Contact) *LeadContext {
	c := &LeadContext{
		LeadID:          lead.ID,
		CompanyName:     lead.CompanyName,
		DisplayName:     strVal(lead.DisplayName),
		BusinessType:    strVal(lead.BusinessType),
		Industry:        strVal(lead.Industry),
		City:            sanitizeLocationField(strVal(lead.City)),
		Province:        sanitizeLocationField(strVal(lead.Province)),
		Address:         sanitizeLocationField(strVal(lead.Address)),
		Website:         strVal(lead.Website),
		Instagram:       strVal(lead.Instagram),
		Description:     strVal(lead.Description),
		Source:          strVal(lead.Source),
		Status:          string(lead.Status),
		QualityStatus:   string(lead.QualityStatus),
		Score:           lead.Score,
		ContactName:     contactName(contact),
		ContactPosition: contactPosition(contact),
		ContactSummary:  summarizeContact(contact),
	}

	if tags, err := b.leads.TagsForLead(ctx, lead.ID); err == nil && len(tags) > 0 {
		c.Tags = strings.Join(tags, ", ")
	}

	if notes, err := b.notes.ListForLead(ctx, lead.ID); err == nil && len(notes) > 0 {
		var lines []string
		for _, n := range notes {
			text := strings.TrimSpace(n.Text)
			if text == "" {
				continue
			}
			if len(text) > 180 {
				text = text[:180] + "..."
			}
			lines = append(lines, "- "+text)
			if len(lines) >= 5 {
				break
			}
		}
		c.Notes = strings.Join(lines, "\n")
	}

	if b.reports != nil {
		reps, err := b.reports.ListForLead(ctx, lead.ID)
		switch {
		case err != nil:
			// A retrieval failure must never look like "reports_count=0": log it
			// separately from a legitimate empty result so operators can tell the
			// difference.
			b.logger.Warn("previous_call_reports_retrieval_failed",
				"lead_id", lead.ID,
				"error", err.Error(),
			)
		case len(reps) > 0:
			c.PreviousReports = b.renderReports(ctx, reps)
		}
	}

	interactionsList, err := b.interactions.ListForLead(ctx, lead.ID)
	switch {
	case err != nil:
		b.logger.Warn("past_interactions_retrieval_failed",
			"lead_id", lead.ID,
			"error", err.Error(),
		)
	case len(interactionsList) > 0:
		c.PastInteractions = b.renderInteractions(ctx, interactionsList)
	}
	return c
}

func strVal(p *string) string {
	if p == nil {
		return ""
	}
	return *p
}

// sanitizeLocationField is a light sanity check that protects the LLM from
// corrupted or misplaced source fields. Observed in production: a job title
// ("Manager") landing in the city/location column after a data import. A
// location field that reads like a person's name, a job title, or a contact
// role is treated as UNKNOWN (blanked) instead of being fed to the model as a
// fact. Pure geographic values (Tehran, تهران, اصفهان, Tehran Province, ...)
// pass through untouched.
func sanitizeLocationField(v string) string {
	v = strings.TrimSpace(v)
	if v == "" {
		return ""
	}
	lower := strings.ToLower(v)
	for _, marker := range locationSuspectMarkers {
		if strings.Contains(lower, marker) {
			return ""
		}
	}
	return v
}

// locationSuspectMarkers are job-title / person / contact-role keywords that
// must never be treated as a location. Both English and Persian variants.
var locationSuspectMarkers = []string{
	"manager", "مدیر", "owner", "مالک", "director", "رئیس", "chief",
	"supervisor", "سرپرست", "فروشنده", "sales", "مسئول", "کارشناس",
	"expert", "کارمند", "employee", "reception", "منشی", "secretary",
	"accountant", "حسابدار", "engineer", "مهندس", "مدیریت",
}

// renderInteractions renders the lead's past interactions as compact bullet
// lines so the LLM can see the outreach history.
func (b *ContextBuilder) renderInteractions(ctx context.Context, list []*interactions.Interaction) string {
	var out strings.Builder
	for _, it := range list {
		line := fmt.Sprintf("- %s (%s)", it.Channel, it.ActionStatus)
		if !it.StartedAt.IsZero() {
			line += " on " + it.StartedAt.Format(time.RFC3339)
		}
		if it.Objective != nil && *it.Objective != "" {
			line += " — " + *it.Objective
		}
		if it.SuggestedMessage != nil && *it.SuggestedMessage != "" {
			line += " — " + *it.SuggestedMessage
		}
		out.WriteString(line + "\n")
		if out.Len() > 1600 {
			out.WriteString("...")
			break
		}
	}
	return strings.TrimRight(out.String(), "\n")
}

// renderReports renders the lead's past call reports (outcomes, interest,
// objections, next actions) so a follow-up call never sounds like a cold one.
func (b *ContextBuilder) renderReports(ctx context.Context, reps []*reports.Report) string {
	var out strings.Builder
	for _, r := range reps {
		parts := []string{"outcome=" + string(r.Outcome)}
		if r.InterestLevel != nil {
			parts = append(parts, "interest="+string(*r.InterestLevel))
		}
		if r.MainObjection != nil && *r.MainObjection != "" {
			parts = append(parts, "objection="+*r.MainObjection)
		}
		if r.NextAction != nil {
			parts = append(parts, "next_action="+string(*r.NextAction))
		}
		if r.Summary != nil && *r.Summary != "" {
			if len(*r.Summary) > 220 {
				parts = append(parts, "summary="+(*r.Summary)[:220])
			} else {
				parts = append(parts, "summary="+*r.Summary)
			}
		}
		if !r.CreatedAt.IsZero() {
			parts = append(parts, "on "+r.CreatedAt.Format(time.RFC3339))
		}
		out.WriteString("- " + strings.Join(parts, " | ") + "\n")
		if out.Len() > 1600 {
			out.WriteString("...")
			break
		}
	}
	return strings.TrimRight(out.String(), "\n")
}
