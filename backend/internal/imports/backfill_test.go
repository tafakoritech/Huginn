package imports

import (
	"testing"

	"github.com/tafakorpardazan/huginn/internal/contacts"
)

const sadafDoc = `شرکت صدف 2
تولید اسباب بازی، تولید لوازم التحریر، کازیه سه طبقه، پایه چسب
svg تولید لوازم تحریر. اداری و تبلیغاتی و همچنین اسباب بازی
svg031-35314946
svginfo\@sadaf2.ir
svg اصفهان، بلوار ارغوانیه، کوچه ارغوان 5، پلاک 49

شرکت دوم
021-12345678
info@second.ir`

func TestFindCompanyBlockSadaf2(t *testing.T) {
	block := findCompanyBlock(sadafDoc, "شرکت صدف 2", 12)
	if block == "" {
		t.Fatal("company block not found")
	}
	// The block must be scoped to the company and not bleed into "شرکت دوم".
	det := DetectContacts(block)
	if det.Landline != "+983135314946" {
		t.Errorf("block landline = %q", det.Landline)
	}
	if det.Email != "info@sadaf2.ir" {
		t.Errorf("block email = %q", det.Email)
	}
}

func TestFindCompanyBlockSecondCompany(t *testing.T) {
	block := findCompanyBlock(sadafDoc, "شرکت دوم", 12)
	det := DetectContacts(block)
	if det.Landline != "+982112345678" {
		t.Errorf("block landline = %q", det.Landline)
	}
	if det.Email != "info@second.ir" {
		t.Errorf("block email = %q", det.Email)
	}
}

func TestFindCompanyBlockNotFound(t *testing.T) {
	if got := findCompanyBlock(sadafDoc, "شرکت ناموجود", 12); got != "" {
		t.Errorf("expected empty block, got %q", got)
	}
}

// TestBackfillFillsEmptyFieldsOnly — creates a summary contact when the lead
// has no contacts, using LLM-validated recovered values.
func TestBackfillFillsEmptyFieldsOnly(t *testing.T) {
	recovered := RecoveredContacts{Phone: "03135314946", Email: "info@sadaf2.ir"}
	res := backfillLeadContacts("lead-1", recovered, nil)
	if len(res.ToCreate) != 1 || len(res.ToUpdate) != 0 {
		t.Fatalf("expected 1 create, got create=%d update=%d", len(res.ToCreate), len(res.ToUpdate))
	}
	created := res.ToCreate[0]
	if created.Phone == nil || *created.Phone != "03135314946" {
		t.Errorf("created phone = %v", created.Phone)
	}
	if created.Email == nil || *created.Email != "info@sadaf2.ir" {
		t.Errorf("created email = %v", created.Email)
	}
	if !created.IsPrimaryContact {
		t.Error("created summary must be primary")
	}
}

func TestBackfillNoDuplicateValues(t *testing.T) {
	recovered := RecoveredContacts{Phone: "03135314946", Email: "info@sadaf2.ir"}
	existing := []*contacts.Contact{
		{
			ID:              "c1",
			PhoneNormalized: strp("+983135314946"),
			EmailNormalized: strp("info@sadaf2.ir"),
		},
	}
	res := backfillLeadContacts("lead-1", recovered, existing)
	if len(res.ToCreate) != 0 && len(res.ToUpdate) != 0 {
		t.Fatalf("values already present must not be re-added: %+v", res)
	}
}

// TestBackfillFillsMissingOnPrimary — a lead with a primary contact that lacks
// the email gets it backfilled onto that same contact (update, not create).
func TestBackfillFillsMissingOnPrimary(t *testing.T) {
	recovered := RecoveredContacts{Phone: "03135314946", Email: "info@sadaf2.ir"}
	existing := []*contacts.Contact{
		{ID: "c1", PhoneNormalized: strp("+983135314946"), IsPrimaryContact: true},
	}
	res := backfillLeadContacts("lead-1", recovered, existing)
	if len(res.ToUpdate) != 1 || len(res.ToCreate) != 0 {
		t.Fatalf("expected 1 update, got create=%d update=%d", len(res.ToCreate), len(res.ToUpdate))
	}
	upd := res.ToUpdate[0]
	if upd.Email == nil || *upd.Email != "info@sadaf2.ir" {
		t.Errorf("updated email = %v", upd.Email)
	}
	// the manually set phone must NOT be overwritten
	if upd.PhoneNormalized == nil || *upd.PhoneNormalized != "+983135314946" {
		t.Errorf("phone must not be overwritten, got %v", upd.PhoneNormalized)
	}
}

// TestBackfillDoesNotOverwriteManualValue — a manually set phone is preserved
// and a no-op update is avoided.
func TestBackfillDoesNotOverwriteManualValue(t *testing.T) {
	recovered := RecoveredContacts{Phone: "03135314946"}
	existing := []*contacts.Contact{
		{ID: "c1", Phone: strp("02112345678"), PhoneNormalized: strp("+982112345678"), IsPrimaryContact: true},
	}
	res := backfillLeadContacts("lead-1", recovered, existing)
	if len(res.ToCreate) != 0 || len(res.ToUpdate) != 0 {
		t.Fatalf("expected no changes (manual value preserved), got %+v", res)
	}
}

// TestBackfillDropsInvalidRecovered — invalid recovered values are not persisted.
func TestBackfillDropsInvalidRecovered(t *testing.T) {
	recovered := RecoveredContacts{Phone: "not-a-phone", Email: "not-an-email"}
	res := backfillLeadContacts("lead-1", recovered, nil)
	if len(res.ToCreate) != 0 {
		t.Fatalf("invalid recovered values must not be persisted, got %+v", res)
	}
}
