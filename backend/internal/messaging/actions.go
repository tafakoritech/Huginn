package messaging

import (
	"context"
	"errors"
	"log/slog"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/tafakorpardazan/huginn/internal/callscenarios"
	"github.com/tafakorpardazan/huginn/internal/campaigns"
	"github.com/tafakorpardazan/huginn/internal/contacts"
	"github.com/tafakorpardazan/huginn/internal/interactions"
	"github.com/tafakorpardazan/huginn/internal/leads"
	"github.com/tafakorpardazan/huginn/internal/notes"
	"github.com/tafakorpardazan/huginn/internal/projects"
	"github.com/tafakorpardazan/huginn/internal/reports"
)

// ContactActionService orchestrates the "prepare, don't send" contact flow.
type ContactActionService struct {
	interactions  *interactions.Repository
	contacts      *contacts.Repository
	leads         *leads.Repository
	notes         *notes.Repository
	reports       *reports.Repository
	projects      *projects.Repository
	campaigns     *campaigns.Repository
	generator     MessageGenerator
	scenarios     ScenarioGenerator
	callScenarios *callscenarios.Repository
	contexts      *ContextBuilder
}

// ContactActionDeps are optional dependencies wired at construction.
type ContactActionDeps struct {
	Campaigns *campaigns.Repository
	Logger    *slog.Logger
}

func NewContactActionService(
	interactions *interactions.Repository,
	contacts *contacts.Repository,
	leads *leads.Repository,
	notes *notes.Repository,
	reports *reports.Repository,
	projects *projects.Repository,
	generator MessageGenerator,
	scenarios ScenarioGenerator,
	callScenarios *callscenarios.Repository,
	deps ...ContactActionDeps,
) *ContactActionService {
	var campaignRepo *campaigns.Repository
	var logger *slog.Logger
	if len(deps) > 0 {
		campaignRepo = deps[0].Campaigns
		logger = deps[0].Logger
	}
	svc := &ContactActionService{
		interactions:  interactions,
		contacts:      contacts,
		leads:         leads,
		notes:         notes,
		reports:       reports,
		projects:      projects,
		campaigns:     campaignRepo,
		generator:     generator,
		scenarios:     scenarios,
		callScenarios: callScenarios,
	}
	builder := NewContextBuilder(interactions, reports, notes, leads, contacts, projects, campaignRepo)
	if logger != nil {
		builder.WithLogger(logger)
	}
	svc.contexts = builder
	return svc
}

// ActionResult is returned to the client after preparing a contact action.
type ActionResult struct {
	InteractionID    string               `json:"interaction_id"`
	Channel          interactions.Channel `json:"channel"`
	Objective        string               `json:"objective"`
	RequiresReport   bool                 `json:"requires_report"`
	SuggestedMessage *string              `json:"suggested_message"`
	SuggestedSubject *string              `json:"suggested_subject"`
	DeepLink         *DeepLink            `json:"deep_link"`
	Model            string               `json:"model,omitempty"`
}

// ScenarioResult is the response of the call-scenario endpoint. It never
// creates an interaction; it is pure pre-call assistance.
//
// GenerationStatus/FallbackUsed/GenerationError are part of the API contract so
// clients can distinguish a personalized LLM-generated scenario
// (generation_status="generated", fallback_used=false) from the deterministic
// fallback (generation_status="degraded", fallback_used=true). Clients MUST
// never present degraded content as AI-generated.
type ScenarioResult struct {
	Scenario     *CallScenario `json:"scenario"`
	LeadID       string        `json:"lead_id"`
	ProjectID    string        `json:"project_id,omitempty"`
	ProjectTitle string        `json:"project_title,omitempty"`
	GenerationID string        `json:"generation_id,omitempty"`
	SalesAngle   string        `json:"sales_angle,omitempty"`
	Model        string        `json:"model,omitempty"`
	// GenerationStatus is "generated" or "degraded".
	GenerationStatus string `json:"generation_status,omitempty"`
	FallbackUsed     bool   `json:"fallback_used"`
	GenerationError  string `json:"generation_error,omitempty"`
}

// Params for creating a contact action.
type Params struct {
	LeadID    string
	ContactID *string
	UserID    string
	Channel   interactions.Channel
	Objective string
	// ProjectID explicitly scopes the interaction to a Project. When nil, the
	// Project is resolved from the lead's own project_id.
	ProjectID *string
}

// ErrMandatoryReportPending is returned when the user has an outstanding report.
var ErrMandatoryReportPending = errors.New("mandatory report pending")

// ErrInvalidChannel is returned for unsupported channels.
var ErrInvalidChannel = errors.New("invalid channel")

// ErrNoProject is returned when a lead has no project, which is required to
// generate any AI-driven sales context or scenario.
var ErrNoProject = errors.New("lead has no project; scenarios cannot be generated without a project")

// resolveProject determines the Project for AI generation. When projectID is
// supplied it must match the lead's project. Otherwise the lead's own
// project_id is used; leads without a project yield ErrNoProject.
func (s *ContactActionService) resolveProject(ctx context.Context, leadID string, projectID *string) (string, error) {
	lead, err := s.leads.Get(ctx, leadID)
	if err != nil {
		return "", err
	}
	if lead.ProjectID == nil || *lead.ProjectID == "" {
		return "", ErrNoProject
	}
	if projectID != nil && *projectID != "" && *projectID != *lead.ProjectID {
		return "", errors.New("lead does not belong to the provided project")
	}
	return *lead.ProjectID, nil
}

// Prepare creates the interaction, generates a message (if needed) and returns
// the deep link + suggested text. It NEVER sends anything.
func (s *ContactActionService) Prepare(ctx context.Context, p Params) (*ActionResult, error) {
	if !interactions.IsValidChannel(p.Channel) {
		return nil, ErrInvalidChannel
	}
	if p.Objective == "" {
		p.Objective = "FIRST_CONTACT"
	}

	lead, err := s.leads.Get(ctx, p.LeadID)
	if err != nil {
		return nil, err
	}
	projectID, err := s.resolveProject(ctx, p.LeadID, p.ProjectID)
	if err != nil {
		return nil, err
	}

	var contact *contacts.Contact
	if p.ContactID != nil && *p.ContactID != "" {
		contact, err = s.contacts.Get(ctx, *p.ContactID)
		if err != nil {
			return nil, err
		}
		if contact.LeadID != p.LeadID {
			return nil, errors.New("contact does not belong to this lead")
		}
	} else {
		primary, err := s.primaryContact(ctx, p.LeadID)
		if err != nil {
			return nil, err
		}
		contact = primary
	}

	leadCtx, err := s.contexts.Build(ctx, lead, contact)
	if err != nil {
		return nil, err
	}

	// Generate a short context-aware suggested message via the LLM.
	var suggestedMessage, suggestedSubject *string
	if p.Channel != interactions.ChannelCall {
		mt, err := s.generator.Generate(ctx, p.Channel, leadCtx, p.Objective)
		if err == nil && mt != nil {
			if mt.Body != "" {
				suggestedMessage = &mt.Body
			}
			if mt.Subject != "" {
				suggestedSubject = &mt.Subject
			}
		}
		// Deterministic fallback when the LLM is unavailable or refused output.
		if suggestedMessage == nil {
			body := defaultMessage(p.Channel, leadCtx)
			suggestedMessage = &body
		}
	}

	var deepLink *DeepLink
	if contact != nil {
		dl := BuildDeepLink(contact, p.Channel, deref(suggestedMessage))
		if dl.Target != "" || dl.Username != "" {
			deepLink = dl
		}
	}

	inter := &interactions.Interaction{
		LeadID:           p.LeadID,
		ContactID:        contactID(contact),
		UserID:           p.UserID,
		ProjectID:        &projectID,
		Channel:          p.Channel,
		Direction:        interactions.DirectionOutbound,
		Objective:        &p.Objective,
		ActionStatus:     interactions.ActionCreated,
		ReportStatus:     interactions.ReportPending,
		SuggestedMessage: suggestedMessage,
		SuggestedSubject: suggestedSubject,
		DeepLink:         deepLinkTarget(deepLink),
	}
	created, err := s.interactions.Create(ctx, inter)
	if err != nil {
		return nil, err
	}

	return &ActionResult{
		InteractionID:    created.ID,
		Channel:          p.Channel,
		Objective:        p.Objective,
		RequiresReport:   true,
		SuggestedMessage: suggestedMessage,
		SuggestedSubject: suggestedSubject,
		DeepLink:         deepLink,
	}, nil
}

// CallScenario builds a suggested call scenario for a lead without creating an
// interaction. The Android client asks for it before starting the call.
// If projectID is provided it is validated to be the lead's project.
// salesAngle optionally overrides the conversation angle; when empty the next
// angle in the rotation is chosen so repeated requests stay unique.
func (s *ContactActionService) CallScenario(ctx context.Context, leadID string, contactID *string, userID *string, projectID *string, salesAngle string) (*ScenarioResult, error) {
	lead, err := s.leads.Get(ctx, leadID)
	if err != nil {
		return nil, err
	}
	// Project is resolved from the lead's project_id; leads without a project
	// cannot have a scenario generated.
	resolvedProjectID, err := s.resolveProject(ctx, leadID, projectID)
	if err != nil {
		return nil, err
	}

	var contact *contacts.Contact
	if contactID != nil && *contactID != "" {
		contact, err = s.contacts.Get(ctx, *contactID)
		if err != nil {
			return nil, err
		}
		if contact.LeadID != leadID {
			return nil, errors.New("contact does not belong to this lead")
		}
	} else {
		primary, err := s.primaryContact(ctx, leadID)
		if err != nil {
			return nil, err
		}
		contact = primary
	}

	leadCtx, err := s.contexts.Build(ctx, lead, contact)
	if err != nil {
		return nil, err
	}

	// The angle is chosen by the LLM from the lead analysis. The client may
	// pass an explicit strategic focus (honored as a strong hint); otherwise
	// we pass the previous generation's angle so "Generate Again" produces
	// reasoning-based variation instead of a recycled template.
	angle := strings.TrimSpace(salesAngle)
	previousAngle := ""
	if angle == "" {
		previousAngle = s.previousPrimaryAngle(ctx, leadID)
	}
	opts := GenerationOptions{
		GenerationID:  uuid.NewString(),
		LeadID:        leadID,
		ProjectID:     resolvedProjectID,
		Timestamp:     time.Now(),
		SalesAngle:    angle,
		PreviousAngle: previousAngle,
	}

	scenario, err := s.scenarios.GenerateScenario(ctx, leadCtx, opts)
	if err != nil {
		// Project context is mandatory: fail instead of emitting a generic
		// scenario. Any other error is already handled inside GenerateScenario
		// (it returns a degraded fallback with explicit status rather than an
		// error), so this branch only surfaces hard failures.
		if errors.Is(err, ErrNoProject) {
			return nil, err
		}
		return nil, err
	}
	if scenario == nil {
		scenario = fallbackScenario(leadCtx, nil)
		applyGenerationMeta(scenario, leadCtx, opts)
		scenario.IsGenerated = false
		scenario.GenerationStatus = GenerationStatusDegraded
		scenario.FallbackUsed = true
		scenario.GenerationError = "no scenario produced"
	}
	// Stamp the per-generation metadata and guarantee a populated context_used
	// debug block, whether the scenario came from the LLM, a stub, or fallback.
	applyGenerationMeta(scenario, leadCtx, opts)

	generationStatus := scenario.GenerationStatus
	fallbackUsed := scenario.FallbackUsed
	generationError := scenario.GenerationError
	if generationStatus == "" {
		// Backward compatibility: legacy callers / stubs without the explicit
		// status field default to the truth on the scenario itself.
		generationStatus = GenerationStatusGenerated
		if fallbackUsed {
			generationStatus = GenerationStatusDegraded
		}
	}

	if s.callScenarios != nil {
		persisted, err := s.callScenarios.Create(ctx, &callscenarios.CallScenario{
			LeadID:               leadID,
			ContactID:            contactID,
			UserID:               userID,
			ProjectID:            &resolvedProjectID,
			GenerationID:         scenario.GenerationID,
			SalesAngle:           scenario.SalesAngle,
			OpeningSentence:      scenario.OpeningSentence,
			Introduction:         scenario.Introduction,
			ReasonForCall:        scenario.ReasonForCall,
			ValueProposition:     scenario.ValueProposition,
			DiscoveryQuestions:   scenario.DiscoveryQuestions,
			LikelyObjections:     scenario.LikelyObjections,
			ObjectionResponses:   scenario.ObjectionResponses,
			RecommendedNextStep:  scenario.RecommendedNextStep,
			ClosingLine:          scenario.ClosingLine,
			ContextUsed:          toPersistedContext(scenario.ContextUsed),
			Strategy:             toPersistedStrategy(scenario.Strategy),
			DiscoveryPurposes:    toPersistedPurposes(scenario.DiscoveryPurposes),
			ConversationBranches: toPersistedBranches(scenario.ConversationBranches),
			IsGenerated:          scenario.IsGenerated,
			GenerationStatus:     generationStatus,
			FallbackUsed:         fallbackUsed,
			GenerationError:      generationError,
		})
		if err == nil && persisted != nil {
			return &ScenarioResult{
				Scenario: &CallScenario{
					ID:                   persisted.ID,
					GenerationID:         persisted.GenerationID,
					LeadID:               persisted.LeadID,
					ContactID:            persisted.ContactID,
					UserID:               persisted.UserID,
					ProjectID:            persisted.ProjectID,
					SalesAngle:           persisted.SalesAngle,
					OpeningSentence:      persisted.OpeningSentence,
					Introduction:         persisted.Introduction,
					ReasonForCall:        persisted.ReasonForCall,
					ValueProposition:     persisted.ValueProposition,
					DiscoveryQuestions:   persisted.DiscoveryQuestions,
					LikelyObjections:     persisted.LikelyObjections,
					ObjectionResponses:   persisted.ObjectionResponses,
					RecommendedNextStep:  persisted.RecommendedNextStep,
					ClosingLine:          persisted.ClosingLine,
					ContextUsed:          fromPersistedContext(persisted.ContextUsed),
					Strategy:             fromPersistedStrategy(persisted.Strategy),
					DiscoveryPurposes:    fromPersistedPurposes(persisted.DiscoveryPurposes),
					ConversationBranches: fromPersistedBranches(persisted.ConversationBranches),
					IsGenerated:          persisted.IsGenerated,
					GenerationStatus:     persisted.GenerationStatus,
					FallbackUsed:         persisted.FallbackUsed,
					GenerationError:      persisted.GenerationError,
					CreatedAt:            persisted.CreatedAt,
					UpdatedAt:            persisted.UpdatedAt,
				},
				LeadID:           leadID,
				ProjectID:        resolvedProjectID,
				ProjectTitle:     s.projectTitle(ctx, resolvedProjectID),
				GenerationID:     persisted.GenerationID,
				SalesAngle:       persisted.SalesAngle,
				GenerationStatus: persisted.GenerationStatus,
				FallbackUsed:     persisted.FallbackUsed,
				GenerationError:  persisted.GenerationError,
			}, nil
		}
	}

	return &ScenarioResult{
		Scenario:         scenario,
		LeadID:           leadID,
		ProjectID:        resolvedProjectID,
		ProjectTitle:     s.projectTitle(ctx, resolvedProjectID),
		GenerationID:     scenario.GenerationID,
		SalesAngle:       scenario.SalesAngle,
		GenerationStatus: generationStatus,
		FallbackUsed:     fallbackUsed,
		GenerationError:  generationError,
	}, nil
}

// previousPrimaryAngle returns the primary_angle of the most recent GENUINELY
// generated scenario for the lead, so "Generate Again" can vary by reasoning
// rather than template rotation. Degraded/fallback scenarios are skipped: their
// synthesized angle must never bias a new generation as if it were a real
// strategy. Empty when no previous generated strategy exists.
func (s *ContactActionService) previousPrimaryAngle(ctx context.Context, leadID string) string {
	if s.callScenarios == nil {
		return ""
	}
	latest, err := s.callScenarios.LatestForLead(ctx, leadID)
	if err != nil || latest == nil || latest.Strategy == nil {
		return ""
	}
	if latest.FallbackUsed || latest.GenerationStatus == messagingGenerationStatusDegraded {
		return ""
	}
	return latest.Strategy.PrimaryAngle
}

// messagingGenerationStatusDegraded mirrors messaging.GenerationStatusDegraded
// to avoid a package cycle; the value is part of the API contract.
const messagingGenerationStatusDegraded = "degraded"

// toPersistedContext converts the messaging ContextUsed into the persistence
// layer's identical-shaped type.
func toPersistedContext(c ContextUsed) callscenarios.ContextUsed {
	return callscenarios.ContextUsed{Project: c.Project, Lead: c.Lead, SalesAngle: c.SalesAngle}
}

// fromPersistedContext converts the persistence layer's ContextUsed back into
// the messaging type.
func fromPersistedContext(c callscenarios.ContextUsed) ContextUsed {
	return ContextUsed{Project: c.Project, Lead: c.Lead, SalesAngle: c.SalesAngle}
}

func toPersistedStrategy(s *CallScenarioStrategy) *callscenarios.CallScenarioStrategy {
	if s == nil {
		return nil
	}
	return &callscenarios.CallScenarioStrategy{
		LeadSummary:        s.LeadSummary,
		Fit:                s.Fit,
		FitSummary:         s.FitSummary,
		PrimaryAngle:       s.PrimaryAngle,
		SecondaryAngle:     s.SecondaryAngle,
		LikelyPain:         s.LikelyPain,
		LikelyPainBasis:    s.LikelyPainBasis,
		ValidationRequired: s.ValidationRequired,
		KnownPains:         s.KnownPains,
		PainHypotheses:     s.PainHypotheses,
		Uncertainties:      s.Uncertainties,
		CallObjective:      s.CallObjective,
		Tone:               s.Tone,
	}
}

func fromPersistedStrategy(s *callscenarios.CallScenarioStrategy) *CallScenarioStrategy {
	if s == nil {
		return nil
	}
	return &CallScenarioStrategy{
		LeadSummary:        s.LeadSummary,
		Fit:                s.Fit,
		FitSummary:         s.FitSummary,
		PrimaryAngle:       s.PrimaryAngle,
		SecondaryAngle:     s.SecondaryAngle,
		LikelyPain:         s.LikelyPain,
		LikelyPainBasis:    s.LikelyPainBasis,
		ValidationRequired: s.ValidationRequired,
		KnownPains:         s.KnownPains,
		PainHypotheses:     s.PainHypotheses,
		Uncertainties:      s.Uncertainties,
		CallObjective:      s.CallObjective,
		Tone:               s.Tone,
	}
}

func toPersistedPurposes(in []DiscoveryQuestion) []callscenarios.DiscoveryQuestion {
	if in == nil {
		return nil
	}
	out := make([]callscenarios.DiscoveryQuestion, 0, len(in))
	for _, q := range in {
		out = append(out, callscenarios.DiscoveryQuestion{Question: q.Question, Purpose: q.Purpose})
	}
	return out
}

func fromPersistedPurposes(in []callscenarios.DiscoveryQuestion) []DiscoveryQuestion {
	if in == nil {
		return nil
	}
	out := make([]DiscoveryQuestion, 0, len(in))
	for _, q := range in {
		out = append(out, DiscoveryQuestion{Question: q.Question, Purpose: q.Purpose})
	}
	return out
}

func toPersistedBranches(in []ConversationBranch) []callscenarios.ConversationBranch {
	if in == nil {
		return nil
	}
	out := make([]callscenarios.ConversationBranch, 0, len(in))
	for _, b := range in {
		out = append(out, callscenarios.ConversationBranch{Condition: b.Condition, ResponseStrategy: b.ResponseStrategy})
	}
	return out
}

func fromPersistedBranches(in []callscenarios.ConversationBranch) []ConversationBranch {
	if in == nil {
		return nil
	}
	out := make([]ConversationBranch, 0, len(in))
	for _, b := range in {
		out = append(out, ConversationBranch{Condition: b.Condition, ResponseStrategy: b.ResponseStrategy})
	}
	return out
}

func (s *ContactActionService) primaryContact(ctx context.Context, leadID string) (*contacts.Contact, error) {
	list, err := s.contacts.ListForLead(ctx, leadID)
	if err != nil {
		return nil, err
	}
	for _, c := range list {
		if c.IsPrimaryContact {
			return c, nil
		}
	}
	if len(list) > 0 {
		return list[0], nil
	}
	return nil, nil
}

func summarizeContact(c *contacts.Contact) string {
	if c == nil {
		return ""
	}
	out := ""
	if c.FullName != nil {
		out += *c.FullName
	}
	if c.JobTitle != nil && *c.JobTitle != "" {
		if out != "" {
			out += ", "
		}
		out += *c.JobTitle
	}
	return out
}

func contactName(c *contacts.Contact) string {
	if c == nil {
		return ""
	}
	if c.FullName != nil && *c.FullName != "" {
		return *c.FullName
	}
	name := strings.Join(joinNonEmptyPtr(c.FirstName, c.LastName), " ")
	return name
}

func contactPosition(c *contacts.Contact) string {
	if c == nil || c.JobTitle == nil {
		return ""
	}
	return *c.JobTitle
}

func joinNonEmptyPtr(parts ...*string) []string {
	var out []string
	for _, p := range parts {
		if p != nil && strings.TrimSpace(*p) != "" {
			out = append(out, *p)
		}
	}
	return out
}

func contactID(c *contacts.Contact) *string {
	if c == nil {
		return nil
	}
	return &c.ID
}

func deepLinkTarget(dl *DeepLink) *string {
	if dl == nil {
		return nil
	}
	if dl.Target != "" {
		return &dl.Target
	}
	return nil
}

func deref(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}

// projectTitle returns the project title for a project id, best-effort
// (empty on miss).
func (s *ContactActionService) projectTitle(ctx context.Context, projectID string) string {
	if projectID == "" || s.projects == nil {
		return ""
	}
	if p, err := s.projects.Get(ctx, projectID); err == nil && p != nil {
		return p.Title
	}
	return ""
}
