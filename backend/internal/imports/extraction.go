package imports

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"strings"

	"github.com/tafakorpardazan/huginn/internal/contacts"
	"github.com/tafakorpardazan/huginn/internal/duplicates"
	"github.com/tafakorpardazan/huginn/internal/evidence"
	"github.com/tafakorpardazan/huginn/internal/leads"
	"github.com/tafakorpardazan/huginn/internal/llm"
	"github.com/tafakorpardazan/huginn/internal/normalize"
	"github.com/tafakorpardazan/huginn/internal/scoring"
)

// Candidate is a structured lead candidate produced by the LLM.
type Candidate struct {
	CompanyName  string  `json:"company_name"`
	DisplayName  *string `json:"display_name"`
	BusinessType *string `json:"business_type"`
	Industry     *string `json:"industry"`
	Description  *string `json:"description"`
	Province     *string `json:"province"`
	City         *string `json:"city"`
	Address      *string `json:"address"`
	Website      *string `json:"website"`
	Instagram    *string `json:"instagram"`
	// SourceBlock is the raw source excerpt the candidate was extracted from.
	// It scopes evidence and any recovery signal so one company's phone/email is
	// never attached to another company's candidate.
	SourceBlock *string `json:"source_block"`
	// Phone, Mobile and Email are the candidate-level (company contact summary)
	// values. They are set by the LLM (the sole authority) and only validated /
	// canonically normalized by deterministic logic — never backfilled or
	// overridden by regex detection.
	Phone    *string            `json:"phone"`
	Mobile   *string            `json:"mobile"`
	Email    *string            `json:"email"`
	Contacts []CandidateContact `json:"contacts"`
	Evidence CandidateEvidence  `json:"evidence"`
}

// CandidateContact is a person within a candidate.
type CandidateContact struct {
	FullName        *string `json:"full_name"`
	JobTitle        *string `json:"job_title"`
	Department      *string `json:"department"`
	Mobile          *string `json:"mobile"`
	Phone           *string `json:"phone"`
	Email           *string `json:"email"`
	WhatsApp        *string `json:"whatsapp"`
	Telegram        *string `json:"telegram"`
	Bale            *string `json:"bale"`
	Eitaa           *string `json:"eitaa"`
	Rubika          *string `json:"rubika"`
	IsDecisionMaker bool    `json:"is_decision_maker"`
}

// CandidateEvidence supports traceability.
type CandidateEvidence struct {
	FieldName     string  `json:"field_name"`
	SourceExcerpt *string `json:"source_excerpt"`
	SourceRow     *int    `json:"source_row"`
	SourcePage    *int    `json:"source_page"`
	Confidence    float64 `json:"confidence"`
}

// CandidateSet is the strict structured output.
type CandidateSet struct {
	Candidates []Candidate `json:"candidates"`
}

const extractPromptVersion = "huginn-extract-v2"

// Per-LLM-call input bounds. Instead of silently truncating a document to the
// first N characters, the text is split into overlapping chunks and each chunk
// is sent separately; candidates are merged afterwards.
const (
	maxChunkChars = 25000
	chunkOverlap  = 600
	strongNameSim = 0.7 // minimum pg_trgm similarity for a company-name duplicate
)

// CandidateOutcome describes what happened to a single candidate.
type CandidateOutcome int

const (
	OutcomeCreated CandidateOutcome = iota
	OutcomeDuplicate
	OutcomeInvalid
	OutcomeFailed
)

// DupMatch explains a duplicate decision (why a candidate was not created).
type DupMatch struct {
	MatchedLeadID string  `json:"matched_lead_id"`
	Rule          string  `json:"rule"` // exact_normalized_mobile|exact_normalized_phone|exact_normalized_email|exact_website|fuzzy_company_name
	Field         string  `json:"field"`
	Value         string  `json:"value"`
	Similarity    float64 `json:"similarity,omitempty"`
}

// ProcessResult counts the outcome of processing a document with precise
// semantics: Extracted = raw LLM output; Valid = passed validation;
// Duplicates = valid candidates matching an existing lead (not created);
// Created = leads actually created; Failed = validation rejects + create errors.
type ProcessResult struct {
	Extracted  int `json:"extracted"`
	Valid      int `json:"valid"`
	Invalid    int `json:"invalid"`
	Duplicates int `json:"duplicates"`
	Created    int `json:"created"`
	Failed     int `json:"failed"`
}

// ExtractionDiagnostics exposes pipeline observability per document.
type ExtractionDiagnostics struct {
	ExtractedTextChars int `json:"extracted_text_chars"`
	LLMInputChars      int `json:"llm_input_text_chars"`
	Chunks             int `json:"chunks"`
	RawCandidateCount  int `json:"raw_candidate_count"`
}

// ExtractionService runs the LLM over extracted text and persists candidates.
type ExtractionService struct {
	runner    llm.Runner
	leadRepo  *leads.Repository
	contactR  *contacts.Repository
	evidenceR *evidence.Repository
	dupDetect *duplicates.Detector
	scoreRepo *scoring.Repository
	batchRepo *Repository
	logger    *slog.Logger
}

func NewExtractionService(runner llm.Runner, leadRepo *leads.Repository, contactR *contacts.Repository,
	evidenceR *evidence.Repository, dupDetect *duplicates.Detector, scoreRepo *scoring.Repository, batchRepo *Repository) *ExtractionService {
	return &ExtractionService{
		runner: runner, leadRepo: leadRepo, contactR: contactR, evidenceR: evidenceR,
		dupDetect: dupDetect, scoreRepo: scoreRepo, batchRepo: batchRepo,
		logger: slog.Default(),
	}
}

// SetLogger attaches a structured logger for pipeline observability.
func (s *ExtractionService) SetLogger(l *slog.Logger) {
	if l != nil {
		s.logger = l
	}
}

// ProcessDocumentText extracts candidates and creates leads with contacts + evidence.
func (s *ExtractionService) ProcessDocumentText(ctx context.Context, doc *Document, extractedText string) (*ProcessResult, error) {
	// Minimal lossless cleanup BEFORE the LLM: strip scrape artifacts (svg,
	// \@, zero-width chars), map digits, collapse spaces — so explicit contact
	// data stays visible to the model. The original extracted text is never
	// modified in the database.
	cleaned := normalize.CleanSourceText(extractedText)
	candidates, diag, err := s.ExtractCandidatesChunked(ctx, doc, cleaned)
	if err != nil {
		s.logger.Error("candidate extraction failed",
			"document_id", doc.ID, "batch_id", doc.BatchID, "stage", "analyze",
			"extracted_text_chars", len(extractedText), "error", err)
		return nil, err
	}

	result := s.processCandidates(ctx, doc, candidates, func(c *Candidate) (CandidateOutcome, *DupMatch, error) {
		_, outcome, dup, perr := s.persistCandidate(ctx, doc, c)
		return outcome, dup, perr
	})

	s.logger.Info("document analysis done",
		"document_id", doc.ID, "batch_id", doc.BatchID, "stage", "analyze",
		"extracted_text_chars", diag.ExtractedTextChars,
		"llm_input_text_chars", diag.LLMInputChars,
		"chunks", diag.Chunks,
		"extracted_candidates", result.Extracted,
		"valid_candidates", result.Valid,
		"invalid_candidates", result.Invalid,
		"duplicates", result.Duplicates,
		"created_leads", result.Created,
		"failed", result.Failed)
	return result, nil
}

// processCandidates validates and classifies candidates. [persist] performs the
// actual creation/duplicate decision; it is injectable so the statistics and
// decision math can be tested without a database.
func (s *ExtractionService) processCandidates(ctx context.Context, doc *Document, candidates []Candidate, persist func(*Candidate) (CandidateOutcome, *DupMatch, error)) *ProcessResult {
	result := &ProcessResult{Extracted: len(candidates)}
	for _, c := range candidates {
		if strings.TrimSpace(c.CompanyName) == "" {
			result.Invalid++
			s.logger.Warn("candidate rejected",
				"document_id", doc.ID, "batch_id", doc.BatchID, "stage", "validate",
				"reason", "missing_company_name")
			continue
		}
		result.Valid++
		outcome, dup, perr := persist(&c)
		switch outcome {
		case OutcomeCreated:
			result.Created++
		case OutcomeDuplicate:
			result.Duplicates++
			if dup != nil {
				s.logger.Info("candidate duplicate",
					"document_id", doc.ID, "batch_id", doc.BatchID, "stage", "dedup",
					"company", c.CompanyName, "matched_lead_id", dup.MatchedLeadID,
					"rule", dup.Rule, "field", dup.Field, "value", dup.Value,
					"similarity", dup.Similarity)
			}
		case OutcomeInvalid:
			result.Invalid++
			s.logger.Warn("candidate invalid",
				"document_id", doc.ID, "batch_id", doc.BatchID, "stage", "validate",
				"company", c.CompanyName, "error", perr)
		default:
			result.Failed++
			s.logger.Warn("candidate persist failed",
				"document_id", doc.ID, "batch_id", doc.BatchID, "stage", "create",
				"company", c.CompanyName, "error", perr)
		}
	}
	return result
}

// ExtractCandidatesChunked sends the document to the LLM, chunking when the
// text exceeds the per-call budget. Candidates from all chunks are merged so
// overlapping chunk boundaries never duplicate or drop records.
func (s *ExtractionService) ExtractCandidatesChunked(ctx context.Context, doc *Document, extractedText string) ([]Candidate, *ExtractionDiagnostics, error) {
	diag := &ExtractionDiagnostics{ExtractedTextChars: len(extractedText)}

	if strings.TrimSpace(extractedText) == "" {
		s.logger.Warn("extraction skipped: empty document text",
			"document_id", doc.ID, "batch_id", doc.BatchID, "stage", "analyze")
		return nil, diag, nil
	}

	chunks := llm.ChunkText(extractedText, maxChunkChars, chunkOverlap)
	diag.Chunks = len(chunks)

	var merged []Candidate
	for i, chunk := range chunks {
		diag.LLMInputChars += len(chunk)
		chunkCandidates, run, err := s.extractChunk(ctx, doc, chunk)
		if err != nil {
			s.logger.Error("llm extraction chunk failed",
				"document_id", doc.ID, "batch_id", doc.BatchID, "stage", "analyze",
				"chunk", i+1, "chunks", len(chunks), "llm_input_chars", len(chunk),
				"model", modelOf(run), "llm_status", statusOf(run),
				"llm_latency_ms", latencyOf(run), "error", err)
			return nil, nil, err
		}
		s.logger.Info("llm extraction chunk done",
			"document_id", doc.ID, "batch_id", doc.BatchID, "stage", "analyze",
			"chunk", i+1, "chunks", len(chunks), "llm_input_chars", len(chunk),
			"model", modelOf(run), "llm_status", statusOf(run),
			"llm_response_chars", responseCharsOf(run), "llm_latency_ms", latencyOf(run),
			"llm_input_tokens", inputTokensOf(run), "llm_output_tokens", outputTokensOf(run),
			"chunk_candidates", len(chunkCandidates))
		merged = append(merged, chunkCandidates...)
	}

	merged = MergeCandidates(merged)
	diag.RawCandidateCount = len(merged)
	return merged, diag, nil
}

func (s *ExtractionService) extractChunk(ctx context.Context, doc *Document, chunk string) ([]Candidate, *llm.Run, error) {
	req := llm.Request{
		Purpose:       llm.PurposeLeadExtraction,
		PromptVersion: extractPromptVersion,
		EntityType:    "source_document",
		EntityID:      doc.ID,
		SystemPrompt:  llm.ExtractionSystemPrompt,
		UserPrompt:    llm.ExtractionUserPrompt(chunk, maxChunkChars),
		JSONSchema:    parseSchema(llm.ExtractionJSONSchema),
		Meta: map[string]string{
			"document_id": doc.ID,
			"batch_id":    doc.BatchID,
		},
	}

	var set CandidateSet
	run, err := s.runner.GenerateStructured(ctx, req, &set)
	if err != nil {
		var ve *llm.ErrValidation
		if errors.As(err, &ve) {
			return nil, run, fmt.Errorf("extraction failed schema validation after retries: %w", err)
		}
		return nil, run, err
	}
	if run == nil || run.Status != "success" {
		return nil, run, errors.New("extraction produced no result")
	}
	if set.Candidates == nil {
		set.Candidates = []Candidate{}
	}
	// The LLM is the sole authority for semantic extraction and company-to-
	// contact association. Deterministic logic only VALIDATES and canonically
	// NORMALIZES the LLM-extracted values, and emits a warning (recovery signal)
	// when the cleaned source block appears to contain contact data the LLM did
	// not extract. Nothing is auto-assigned.
	for i := range set.Candidates {
		set.Candidates[i].SourceBlock = &chunk
		s.normalizeCandidateContacts(&set.Candidates[i])
		s.warnRecoverySignal(&set.Candidates[i], chunk)
	}
	return set.Candidates, run, nil
}

// normalizeCandidateContacts validates and canonically normalizes the
// LLM-extracted phone/mobile/email values (candidate level and per contact).
// Invalid values are dropped with a log; nothing is injected from regex
// detection.
func (s *ExtractionService) normalizeCandidateContacts(c *Candidate) {
	if v, ok := canonicalPhone(strOrEmpty(c.Phone)); ok {
		c.Phone = strptr(v)
	} else if c.Phone != nil && *c.Phone != "" {
		s.logger.Warn("candidate contact dropped",
			"company_name", c.CompanyName, "field", "phone", "value", *c.Phone, "reason", "invalid")
		c.Phone = nil
	}
	if v, ok := canonicalPhone(strOrEmpty(c.Mobile)); ok {
		c.Mobile = strptr(v)
	} else if c.Mobile != nil && *c.Mobile != "" {
		s.logger.Warn("candidate contact dropped",
			"company_name", c.CompanyName, "field", "mobile", "value", *c.Mobile, "reason", "invalid")
		c.Mobile = nil
	}
	if e := normalize.NormalizeEmail(strOrEmpty(c.Email)); normalize.IsValidEmail(e) {
		c.Email = strptr(e)
	} else if c.Email != nil && *c.Email != "" {
		s.logger.Warn("candidate contact dropped",
			"company_name", c.CompanyName, "field", "email", "value", *c.Email, "reason", "invalid")
		c.Email = nil
	}

	for i := range c.Contacts {
		cc := &c.Contacts[i]
		if v, ok := canonicalPhone(strOrEmpty(cc.Phone)); ok {
			cc.Phone = strptr(v)
		} else if cc.Phone != nil && *cc.Phone != "" {
			s.logger.Warn("candidate contact dropped",
				"company_name", c.CompanyName, "field", "contact.phone", "value", *cc.Phone, "reason", "invalid")
			cc.Phone = nil
		}
		if v, ok := canonicalPhone(strOrEmpty(cc.Mobile)); ok {
			cc.Mobile = strptr(v)
		} else if cc.Mobile != nil && *cc.Mobile != "" {
			s.logger.Warn("candidate contact dropped",
				"company_name", c.CompanyName, "field", "contact.mobile", "value", *cc.Mobile, "reason", "invalid")
			cc.Mobile = nil
		}
		if e := normalize.NormalizeEmail(strOrEmpty(cc.Email)); normalize.IsValidEmail(e) {
			cc.Email = strptr(e)
		} else if cc.Email != nil && *cc.Email != "" {
			s.logger.Warn("candidate contact dropped",
				"company_name", c.CompanyName, "field", "contact.email", "value", *cc.Email, "reason", "invalid")
			cc.Email = nil
		}
	}
}

// canonicalPhone validates a raw phone/mobile and returns its local display
// form (e.g. 03135314946). Reports ok=false when it is not a valid Iranian
// phone/mobile.
func canonicalPhone(raw string) (string, bool) {
	if p := normalize.NormalizePhone(raw); p != "" {
		return normalize.LocalPhone(p), true
	}
	if m := normalize.NormalizeMobile(raw); m != "" {
		return normalize.LocalMobile(m), true
	}
	return "", false
}

// warnRecoverySignal compares deterministic detection against what the LLM
// extracted for this candidate. It only WARNS (a recovery signal) — the LLM is
// the authority, so nothing is persisted from detection alone.
func (s *ExtractionService) warnRecoverySignal(c *Candidate, fallbackBlock string) {
	block := fallbackBlock
	if c.SourceBlock != nil && strings.TrimSpace(*c.SourceBlock) != "" {
		block = *c.SourceBlock
	}
	det := DetectContacts(block)
	if !det.HasAny() {
		return
	}

	llmPhones := map[string]bool{}
	if p := normalize.NormalizePhone(strOrEmpty(c.Phone)); p != "" {
		llmPhones[p] = true
	}
	if m := normalize.NormalizeMobile(strOrEmpty(c.Mobile)); m != "" {
		llmPhones[m] = true
	}
	for _, cc := range c.Contacts {
		if p := normalize.NormalizePhone(strOrEmpty(cc.Phone)); p != "" {
			llmPhones[p] = true
		}
		if m := normalize.NormalizeMobile(strOrEmpty(cc.Mobile)); m != "" {
			llmPhones[m] = true
		}
	}
	llmEmails := map[string]bool{}
	for _, e := range []*string{c.Email} {
		if em := normalize.NormalizeEmail(strOrEmpty(e)); em != "" {
			llmEmails[em] = true
		}
	}
	for _, cc := range c.Contacts {
		if em := normalize.NormalizeEmail(strOrEmpty(cc.Email)); em != "" {
			llmEmails[em] = true
		}
	}

	if det.Landline != "" && !llmPhones[det.Landline] {
		s.logger.Warn("candidate may be missing source phone",
			"company_name", c.CompanyName, "field", "landline",
			"detected_source", normalize.LocalPhone(det.Landline),
			"action", "recovery_signal_only")
	}
	if det.Mobile != "" && !llmPhones[det.Mobile] {
		s.logger.Warn("candidate may be missing source mobile",
			"company_name", c.CompanyName, "field", "mobile",
			"detected_source", normalize.LocalMobile(det.Mobile),
			"action", "recovery_signal_only")
	}
	if det.Email != "" && !llmEmails[det.Email] {
		s.logger.Warn("candidate may be missing source email",
			"company_name", c.CompanyName, "field", "email",
			"detected_source", det.Email,
			"action", "recovery_signal_only")
	}
}

// persistCandidate creates a lead and its contacts + evidence, but only after a
// pre-create duplicate check against existing leads. Duplicates are never
// inserted into the leads table.
func (s *ExtractionService) persistCandidate(ctx context.Context, doc *Document, c *Candidate) (*leads.Lead, CandidateOutcome, *DupMatch, error) {
	if strings.TrimSpace(c.CompanyName) == "" {
		return nil, OutcomeInvalid, nil, errors.New("candidate missing company_name")
	}

	// 1. Deterministic dedup by exact normalized contact values (before creation).
	normValues := normalizedContactValues(c)
	if len(normValues) > 0 {
		matches, err := s.dupDetect.FindDeterministicDetailed(ctx, normValues)
		if err == nil && len(matches) > 0 {
			m := matches[0]
			return nil, OutcomeDuplicate, &DupMatch{
				MatchedLeadID: m.LeadID,
				Rule:          "exact_normalized_" + m.Field,
				Field:         m.Field,
				Value:         m.Value,
			}, nil
		}
	}

	// 2. Exact website/domain match.
	if c.Website != nil && *c.Website != "" {
		leadID, err := s.dupDetect.FindByWebsite(ctx, *c.Website)
		if err == nil && leadID != "" {
			return nil, OutcomeDuplicate, &DupMatch{
				MatchedLeadID: leadID,
				Rule:          "exact_website",
				Field:         "website",
				Value:         normalize.NormalizeWebsite(*c.Website),
			}, nil
		}
	}

	// 3. Strong normalized company-name match (>= strongNameSim). A high
	//    threshold prevents generic words / shared prefixes from producing
	//    false duplicates.
	{
		leadID, sim, err := s.dupDetect.FindByName(ctx, c.CompanyName, strongNameSim)
		if err == nil && leadID != "" {
			return nil, OutcomeDuplicate, &DupMatch{
				MatchedLeadID: leadID,
				Rule:          "fuzzy_company_name",
				Field:         "company_name",
				Value:         c.CompanyName,
				Similarity:    sim,
			}, nil
		}
	}

	// No duplicate: create the lead.
	lead, err := s.createLead(ctx, doc, c)
	if err != nil {
		return nil, OutcomeFailed, nil, err
	}
	return lead, OutcomeCreated, nil, nil
}

func (s *ExtractionService) createLead(ctx context.Context, doc *Document, c *Candidate) (*leads.Lead, error) {
	leadSvc := leads.NewService(s.leadRepo)
	// The lead is created as a GLOBAL entity: source provenance is carried on
	// the lead itself (source_batch_id / source_document_id). When the batch
	// carries a Project context it is stored directly on the lead (project_id).
	params := candidateToLeadParams(c, doc)
	params.ProjectID = s.batchProject(ctx, doc.BatchID)
	lead, err := leadSvc.Create(ctx, params)
	if err != nil {
		return nil, err
	}

	for i, cc := range c.Contacts {
		contact := &contacts.Contact{
			LeadID:           lead.ID,
			FullName:         cc.FullName,
			JobTitle:         cc.JobTitle,
			Department:       cc.Department,
			Mobile:           cc.Mobile,
			Phone:            cc.Phone,
			Email:            cc.Email,
			WhatsApp:         cc.WhatsApp,
			Telegram:         cc.Telegram,
			Bale:             cc.Bale,
			Eitaa:            cc.Eitaa,
			Rubika:           cc.Rubika,
			IsDecisionMaker:  cc.IsDecisionMaker,
			IsPrimaryContact: i == 0, // the first contact is the main one
		}
		if contact.FirstName == nil && cc.FullName != nil {
			first, last := splitFullName(*cc.FullName)
			contact.FirstName = first
			contact.LastName = last
		}
		if _, err := s.contactR.Create(ctx, contact); err != nil {
			return nil, err
		}
	}

	// Ensure the candidate's explicit contact summary (phone/mobile/email) is
	// persisted even when the LLM omitted it. buildContacts is the pure mapping
	// used here and under test (requirement: phone/mobile/email map to a lead).
	summaryContacts := buildContactSummary(lead.ID, c, c.Contacts)
	for _, sc := range summaryContacts {
		if _, err := s.contactR.Create(ctx, sc); err != nil {
			return nil, err
		}
	}

	if c.Evidence.FieldName != "" {
		_ = s.evidenceR.Create(ctx, &evidence.Item{
			LeadID:           lead.ID,
			SourceDocumentID: &doc.ID,
			FieldName:        c.Evidence.FieldName,
			SourcePage:       c.Evidence.SourcePage,
			SourceRow:        c.Evidence.SourceRow,
			SourceExcerpt:    c.Evidence.SourceExcerpt,
			Confidence:       c.Evidence.Confidence,
		})
	}

	res := scoring.Score(scoring.Signals{
		HasIndustry:     c.Industry != nil && *c.Industry != "",
		HasDescription:  c.Description != nil && *c.Description != "",
		HasBusinessType: c.BusinessType != nil && *c.BusinessType != "",
		SourceRelevant:  true,
		HasMobile:       hasAnyPhone(c.Contacts),
		HasPhone:        hasLandline(c.Contacts),
		HasEmail:        hasEmail(c.Contacts),
		HasAddress:      c.Address != nil && *c.Address != "",
		HasWebsite:      c.Website != nil && *c.Website != "",
		SourceTypeRich:  doc.MimeType != nil && (*doc.MimeType == "application/pdf" || *doc.MimeType == "text/html"),
	})
	if err := s.scoreRepo.Save(ctx, lead.ID, res, "initial extraction", nil); err == nil {
		_ = s.leadRepo.SetScore(ctx, lead.ID, res.Total)
	}
	return lead, nil
}

// normalizedContactValues returns deduplicated normalized mobile/phone/email
// values for deterministic duplicate detection. It considers both the LLM
// person contacts and the candidate-level (merged) contact summary so a lead
// carrying an explicit company phone/email is not created twice.
func normalizedContactValues(c *Candidate) []string {
	values := []string{}
	seen := map[string]bool{}
	add := func(v string) {
		if v == "" || seen[v] {
			return
		}
		seen[v] = true
		values = append(values, v)
	}
	for _, cc := range c.Contacts {
		add(normalize.NormalizeMobile(strOrEmpty(cc.Mobile)))
		add(normalize.NormalizePhone(strOrEmpty(cc.Phone)))
		add(normalize.NormalizeEmail(strOrEmpty(cc.Email)))
	}
	add(normalize.NormalizeMobile(strOrEmpty(c.Mobile)))
	add(normalize.NormalizePhone(strOrEmpty(c.Phone)))
	add(normalize.NormalizeEmail(strOrEmpty(c.Email)))
	return values
}

// MergeCandidates collapses identical candidates returned from overlapping
// chunks (same normalized company name AND a shared normalized contact/website).
func MergeCandidates(candidates []Candidate) []Candidate {
	seen := map[string]bool{}
	out := []Candidate{}
	for _, c := range candidates {
		key := candidateKey(c)
		if key == "" || seen[key] {
			continue
		}
		seen[key] = true
		out = append(out, c)
	}
	return out
}

func candidateKey(c Candidate) string {
	name := normalizeCompanyName(c.CompanyName)
	if name == "" {
		return ""
	}
	extra := ""
	for _, v := range normalizedContactValues(&c) {
		extra = v
		break
	}
	if c.Website != nil {
		extra = extra + "|" + normalize.NormalizeWebsite(*c.Website)
	}
	return name + "|" + extra
}

func normalizeCompanyName(s string) string {
	s = strings.TrimSpace(s)
	// collapse internal whitespace
	fields := strings.Fields(s)
	return strings.Join(fields, " ")
}

func splitFullName(full string) (*string, *string) {
	first := ""
	last := ""
	found := false
	for i, c := range full {
		if c == ' ' && !found {
			first = full[:i]
			last = full[i+1:]
			found = true
			break
		}
	}
	if !found {
		first = full
	}
	return strptr(first), strptr(last)
}

func hasAnyPhone(cc []CandidateContact) bool {
	for _, c := range cc {
		if c.Mobile != nil && *c.Mobile != "" {
			return true
		}
	}
	return false
}

func hasLandline(cc []CandidateContact) bool {
	for _, c := range cc {
		if c.Phone != nil && *c.Phone != "" {
			return true
		}
	}
	return false
}

func hasEmail(cc []CandidateContact) bool {
	for _, c := range cc {
		if c.Email != nil && *c.Email != "" {
			return true
		}
	}
	return false
}

func strptr(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}

func strOrEmpty(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}

func strptrOrNil(s string) *string {
	if strings.TrimSpace(s) == "" {
		return nil
	}
	return &s
}

// candidateToLeadParams maps a candidate's identity/geo fields onto the lead
// creation parameters. This pure mapping is exercised by tests to guarantee
// website/address/city are never silently dropped. Project membership is set
// by the caller from the batch's project_id.
func candidateToLeadParams(c *Candidate, doc *Document) leads.CreateParams {
	return leads.CreateParams{
		CompanyName:      c.CompanyName,
		DisplayName:      c.DisplayName,
		BusinessType:     c.BusinessType,
		Industry:         c.Industry,
		Description:      c.Description,
		Province:         c.Province,
		City:             c.City,
		Address:          c.Address,
		Website:          c.Website,
		Instagram:        c.Instagram,
		SourceBatchID:    &doc.BatchID,
		SourceDocumentID: &doc.ID,
	}
}

// batchProject returns the project_id a batch was created with, or nil. Lookup
// errors are swallowed so extraction never fails on a missing/corrupt batch.
func (s *ExtractionService) batchProject(ctx context.Context, batchID string) *string {
	if batchID == "" {
		return nil
	}
	b, err := s.batchRepo.GetBatch(ctx, batchID)
	if err != nil || b == nil {
		return nil
	}
	return b.ProjectID
}

// buildContactSummary returns the contact record(s) that carry the candidate's
// explicit contact summary (phone/mobile/email). It only adds values NOT
// already present on the LLM person contacts so explicit source data is never
// duplicated, but is always represented on the lead.
//
// personContacts may be nil when the candidate had no contacts; the pure
// mapping is exercised directly by tests.
func buildContactSummary(leadID string, c *Candidate, personContacts []CandidateContact) []*contacts.Contact {
	// canonical values already carried by person contacts
	present := map[string]bool{}
	for _, cc := range personContacts {
		for _, v := range []string{
			normalize.NormalizeMobile(strOrEmpty(cc.Mobile)),
			normalize.NormalizePhone(strOrEmpty(cc.Phone)),
			normalize.NormalizeEmail(strOrEmpty(cc.Email)),
		} {
			if v != "" {
				present[v] = true
			}
		}
	}

	summary := &contacts.Contact{LeadID: leadID}
	if mob := normalize.NormalizeMobile(strOrEmpty(c.Mobile)); mob != "" && !present[mob] {
		summary.Mobile = strptr(normalize.LocalMobile(mob))
		summary.MobileNormalized = strptr(mob)
	}
	if ph := normalize.NormalizePhone(strOrEmpty(c.Phone)); ph != "" && !present[ph] {
		summary.Phone = strptr(normalize.LocalPhone(ph))
		summary.PhoneNormalized = strptr(ph)
	}
	if em := normalize.NormalizeEmail(strOrEmpty(c.Email)); em != "" && !present[em] {
		summary.Email = strptr(em)
		summary.EmailNormalized = strptr(em)
	}

	if summary.Mobile == nil && summary.Phone == nil && summary.Email == nil {
		return nil // nothing new to persist
	}

	// A summary contact is primary when there are no person contacts at all.
	summary.IsPrimaryContact = len(personContacts) == 0
	return []*contacts.Contact{summary}
}

// ---- diagnostics helpers (never expose secrets) ----

func modelOf(run *llm.Run) string {
	if run == nil {
		return ""
	}
	return run.Model
}

func statusOf(run *llm.Run) string {
	if run == nil {
		return ""
	}
	return run.Status
}

func latencyOf(run *llm.Run) int64 {
	if run == nil {
		return 0
	}
	return run.LatencyMs
}

func responseCharsOf(run *llm.Run) int {
	if run == nil {
		return 0
	}
	return len(run.StructuredOutput)
}

func inputTokensOf(run *llm.Run) int {
	if run == nil {
		return 0
	}
	return run.InputTokens
}

func outputTokensOf(run *llm.Run) int {
	if run == nil {
		return 0
	}
	return run.OutputTokens
}
