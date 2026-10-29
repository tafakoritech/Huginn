package imports

import (
	"io"
	"log/slog"
	"strings"
	"testing"

	"github.com/tafakorpardazan/huginn/internal/contacts"
	"github.com/tafakorpardazan/huginn/internal/leads"
	"github.com/tafakorpardazan/huginn/internal/normalize"
)

const sadaf2Block = `شرکت صدف 2
تولید اسباب بازی، تولید لوازم التحریر، کازیه سه طبقه، پایه چسب
svg تولید لوازم تحریر. اداری و تبلیغاتی و همچنین اسباب بازی
svg031-35314946
svginfo\@sadaf2.ir
svg اصفهان، بلوار ارغوانیه، کوچه ارغوان 5، پلاک 49`

func strp(s string) *string { return &s }

func testSvc() *ExtractionService {
	return &ExtractionService{logger: slog.New(slog.NewTextHandler(io.Discard, nil))}
}

// Requirement: detection still finds the Sadaf 2 landline/email, but ONLY as a
// recovery signal — it is never assigned automatically.
func TestDetectContactsSadaf2Block(t *testing.T) {
	det := DetectContacts(sadaf2Block)
	if det.Landline != "+983135314946" {
		t.Errorf("landline = %q, want +983135314946", det.Landline)
	}
	if det.Email != "info@sadaf2.ir" {
		t.Errorf("email = %q, want info@sadaf2.ir", det.Email)
	}
}

// Requirement: minimal lossless cleanup MUST keep explicit contact data visible
// to the LLM (svg031-35314946 -> 031-35314946, svginfo\@sadaf2.ir ->
// info@sadaf2.ir), preserving line/block structure.
func TestCleanSourceTextPreservesContactsForLLM(t *testing.T) {
	cleaned := normalize.CleanSourceText(sadaf2Block)
	for _, want := range []string{"031-35314946", "info@sadaf2.ir", "اصفهان، بلوار ارغوانیه", "شرکت صدف 2"} {
		if !strings.Contains(cleaned, want) {
			t.Errorf("cleaned source lost %q for the LLM:\n%s", want, cleaned)
		}
	}
	if strings.Contains(cleaned, "svg031") || strings.Contains(cleaned, `\@`) {
		t.Errorf("artifacts not cleaned:\n%s", cleaned)
	}
}

// Requirement: the LLM value is the authority. If the LLM extracted a phone,
// deterministic detection must NEVER override or replace it.
func TestNormalizeKeepsLLMValueNotDetection(t *testing.T) {
	svc := testSvc()
	c := &Candidate{
		CompanyName: "شرکت صدف 2",
		SourceBlock: strp(sadaf2Block),
		Phone:       strp("02199999999"),
		Email:       strp("wrong@sadaf2.ir"),
	}
	svc.normalizeCandidateContacts(c)
	// LLM values are kept (normalized), NOT replaced by detection.
	if c.Phone == nil || *c.Phone != "02199999999" {
		t.Errorf("phone = %v, want LLM value 02199999999 (detection must not override)", c.Phone)
	}
	if c.Email == nil || *c.Email != "wrong@sadaf2.ir" {
		t.Errorf("email = %v, want LLM value wrong@sadaf2.ir", c.Email)
	}
}

// Requirement: candidate-level phone/mobile/email are normalized to canonical
// local display form after LLM extraction.
func TestNormalizeCanonicalizesLLMValue(t *testing.T) {
	svc := testSvc()
	c := &Candidate{
		CompanyName: "شرکت صدف 2",
		Phone:       strp("031-35314946"),
		Email:       strp("INFO@Sadaf2.IR"),
	}
	svc.normalizeCandidateContacts(c)
	if c.Phone == nil || *c.Phone != "03135314946" {
		t.Errorf("phone = %v, want 03135314946", c.Phone)
	}
	if c.Email == nil || *c.Email != "info@sadaf2.ir" {
		t.Errorf("email = %v, want info@sadaf2.ir", c.Email)
	}
}

// Requirement: deterministic validation drops invalid LLM values (with a log),
// it does not invent replacements.
func TestNormalizeDropsInvalidLLMValue(t *testing.T) {
	svc := testSvc()
	c := &Candidate{
		CompanyName: "شرکت نمونه",
		Phone:       strp("not-a-phone"),
		Email:       strp("not-an-email"),
	}
	svc.normalizeCandidateContacts(c)
	if c.Phone != nil {
		t.Errorf("invalid phone must be dropped, got %v", c.Phone)
	}
	if c.Email != nil {
		t.Errorf("invalid email must be dropped, got %v", c.Email)
	}
}

// Requirement: recovery signal warns but never mutates the candidate — the LLM
// output is left exactly as normalized, even when detection finds a phone the
// LLM omitted.
func TestRecoverySignalDoesNotAssign(t *testing.T) {
	svc := testSvc()
	c := &Candidate{
		CompanyName: "شرکت صدف 2",
		SourceBlock: strp(sadaf2Block),
		Phone:       nil,
		Email:       nil,
		Contacts:    nil,
	}
	before := *c
	svc.warnRecoverySignal(c, sadaf2Block)
	if c.Phone != nil || c.Email != nil || c.CompanyName != before.CompanyName {
		t.Errorf("recovery signal must not assign contact data: %+v", c)
	}
}

// Requirement 8 (mapping): candidate -> lead must carry website/address/city.
func TestCandidateToLeadMapping(t *testing.T) {
	c := &Candidate{
		CompanyName: "شرکت صدف 2",
		City:        strp("اصفهان"),
		Address:     strp("اصفهان، بلوار ارغوانیه، کوچه ارغوان 5، پلاک 49"),
		Website:     strp("sadaf2.ir"),
		Province:    strp("اصفهان"),
		Description: strp("تولید اسباب بازی"),
	}
	doc := &Document{ID: "doc-1", BatchID: "batch-1"}
	p := candidateToLeadParams(c, doc)
	if p.CompanyName != "شرکت صدف 2" || p.City == nil || *p.City != "اصفهان" ||
		p.Address == nil || *p.Address != "اصفهان، بلوار ارغوانیه، کوچه ارغوان 5، پلاک 49" ||
		p.Website == nil || *p.Website != "sadaf2.ir" ||
		p.SourceDocumentID == nil || *p.SourceDocumentID != "doc-1" {
		t.Errorf("lead mapping dropped fields: %+v", p)
	}
}

func TestBuildContactSummary(t *testing.T) {
	c := &Candidate{
		Mobile: strp("09151112233"),
		Phone:  strp("03135314946"),
		Email:  strp("info@sadaf2.ir"),
	}
	summary := buildContactSummary("lead-1", c, nil)
	if len(summary) != 1 {
		t.Fatalf("expected 1 summary contact, got %d", len(summary))
	}
	s := summary[0]
	if s.Phone == nil || *s.Phone != "03135314946" {
		t.Errorf("summary phone = %v", s.Phone)
	}
	if s.PhoneNormalized == nil || *s.PhoneNormalized != "+983135314946" {
		t.Errorf("summary phone_normalized = %v", s.PhoneNormalized)
	}
	if s.Mobile == nil || *s.Mobile != "09151112233" {
		t.Errorf("summary mobile = %v", s.Mobile)
	}
	if s.Email == nil || *s.Email != "info@sadaf2.ir" {
		t.Errorf("summary email = %v", s.Email)
	}
	if !s.IsPrimaryContact {
		t.Error("summary contact must be primary when there are no person contacts")
	}
}

func TestBuildContactSummaryNoDuplicate(t *testing.T) {
	c := &Candidate{
		Phone: strp("03135314946"),
		Email: strp("info@sadaf2.ir"),
	}
	person := []CandidateContact{
		{FullName: strp("علی رضایی"), Phone: strp("03135314946"), Email: strp("info@sadaf2.ir")},
	}
	if got := buildContactSummary("lead-1", c, person); len(got) != 0 {
		t.Fatalf("values already present must not be duplicated, got %d contacts", len(got))
	}
}

func TestBuildContactSummaryPartialFill(t *testing.T) {
	c := &Candidate{
		Phone: strp("03135314946"),
		Email: strp("info@sadaf2.ir"),
	}
	person := []CandidateContact{
		{FullName: strp("علی رضایی"), Phone: strp("03135314946")},
	}
	got := buildContactSummary("lead-1", c, person)
	if len(got) != 1 {
		t.Fatalf("expected 1 summary contact with the missing email, got %d", len(got))
	}
	if got[0].Email == nil || *got[0].Email != "info@sadaf2.ir" {
		t.Errorf("summary email = %v", got[0].Email)
	}
	if got[0].Phone != nil {
		t.Error("phone already present must not be re-added")
	}
	if got[0].IsPrimaryContact {
		t.Error("summary must not be primary when a person contact exists")
	}
}

var _ = contacts.Contact{}
var _ = leads.CreateParams{}
