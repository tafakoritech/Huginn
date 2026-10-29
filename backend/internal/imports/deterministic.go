package imports

import (
	"strings"

	"github.com/tafakorpardazan/huginn/internal/normalize"
)

// DetectedContacts are contact facts that CAN be deterministically detected in
// a source block. They are a WARNING/RECOVERY SIGNAL ONLY: deterministic
// detection never assigns phone/email/website to a company. The LLM is the
// sole authority for semantic extraction and company-to-contact association.
//
// Detection is used to (1) warn when the cleaned source block appears to
// contain explicit contact data the LLM did not extract, and (2) drive an
// operator-invoked LLM re-validation (backfill) which re-asks the model before
// anything is persisted.
type DetectedContacts struct {
	Landline   string // canonical +982… or ""
	Mobile     string // canonical +989… or ""
	Email      string // normalized lowercase or ""
	Website    string // normalized domain or ""
	PostalCode string // 10-digit or ""
}

// HasAny reports whether any contact fact was detected.
func (d DetectedContacts) HasAny() bool {
	return d.Landline != "" || d.Mobile != "" || d.Email != "" || d.Website != ""
}

// DetectContacts returns contact facts detected in a source block WITHOUT
// assigning them to any company. See DetectedContacts for semantics.
func DetectContacts(block string) DetectedContacts {
	if strings.TrimSpace(block) == "" {
		return DetectedContacts{}
	}
	landline, mobile := normalize.ExtractPhonesUnique(block)
	emails := normalize.ExtractEmails(block)
	email := ""
	if len(emails) > 0 {
		email = emails[0]
	}
	return DetectedContacts{
		Landline:   landline,
		Mobile:     mobile,
		Email:      email,
		Website:    normalize.ExtractWebsite(block),
		PostalCode: normalize.ExtractPostalCode(block),
	}
}
