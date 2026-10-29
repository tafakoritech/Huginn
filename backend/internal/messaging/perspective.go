package messaging

import (
	"regexp"
	"strings"
)

// SenderProfile is OUR company identity used to write outreach messages. It is
// the ONLY source of sender identity. It must never be confused with or inferred
// from the lead's data.
type SenderProfile struct {
	CompanyName string
	Description string
	SenderName  string
	SenderTitle string
}

func (s *SenderProfile) representative() string {
	if s == nil {
		return ""
	}
	if s.SenderName != "" && s.SenderTitle != "" {
		return s.SenderName + " (" + s.SenderTitle + ")"
	}
	if s.SenderName != "" {
		return s.SenderName
	}
	return s.SenderTitle
}

func (s *SenderProfile) IsZero() bool {
	return s == nil ||
		(strings.TrimSpace(s.CompanyName) == "" &&
			strings.TrimSpace(s.Description) == "" &&
			strings.TrimSpace(s.SenderName) == "" &&
			strings.TrimSpace(s.SenderTitle) == "")
}

var companyPrefixRe = regexp.MustCompile(`^(شرکت|موسسه|گروه|کارخانه|کمپانی|سهامی|Co\.?|Inc\.?|Ltd\.?|Corp\.?|LLC)\s*`)

// normalizeCompanyName strips common legal prefixes so "شرکت صدف" and "صدف"
// are treated as the same company.
func normalizeCompanyName(s string) string {
	s = strings.TrimSpace(s)
	s = strings.ReplaceAll(s, "\u200c", " ")
	s = companyPrefixRe.ReplaceAllString(s, "")
	return strings.TrimSpace(s)
}

// first-person activity verbs that claim the speaker performs the recipient's
// business inside the recipient's company.
var membershipClaimVerbs = []string{
	"فعالیت می‌کنیم",
	"فعالیت می‌کنم",
	"فعالیت می‌کنند",
	"مشغول فعالیت",
	"کار می‌کنیم",
	"کار می‌کنم",
	"تولید می‌کنیم",
	"تولید می‌کنم",
	"عرضه می‌کنیم",
	"عرضه می‌کنم",
	"صادر می‌کنیم",
	"وارد می‌کنیم",
	"هستیم",
}

// validateMessagePerspective enforces the Sender/Recipient identity separation
// on a generated message. It returns true when the message is safe to return:
//   - the sender company is never presented as the recipient company,
//   - the sender never claims to be from the recipient's business, and
//   - no sentence implies the sender works for the recipient company.
func validateMessagePerspective(sender *SenderProfile, leadCtx *LeadContext, body string) bool {
	if strings.TrimSpace(body) == "" {
		return false
	}
	leadName := strings.TrimSpace(leadCtx.CompanyName)
	if sender != nil && normalizeCompanyName(sender.CompanyName) == normalizeCompanyName(leadName) {
		return false
	}
	senderName := ""
	if sender != nil {
		senderName = sender.CompanyName
	}
	if claimsSenderFromRecipientBusiness(senderName, body) || claimsSenderFromDescriptors(leadCtx, body) {
		return false
	}
	if leadName == "" {
		return true
	}
	return !sentenceClaimsSenderBelongsToRecipient(leadName, body)
}

// businessRoleNouns are generic business nouns a sender would never truthfully
// claim to be "from" unless impersonating the recipient's business.
var businessRoleNouns = []string{
	"تولیدکننده", "تولیدی", "کارخانه", "کارگاه", "پخش", "عرضه", "واردکننده",
	"صادرکننده", "تأمین‌کننده", "تامین‌کننده", "تولید و پخش",
}

// originVerbs are first-person verbs that mark the speaker as the origin of an
// outreach ("I am calling from ...", "I write to you from ...").
var originVerbs = []string{
	"تماس گرفتم", "تماس می‌گیرم", "تماس گرفتیم", "تماس می‌گیریم",
	"هستم", "هستیم", "می‌نویسم", "می‌نویسیم",
	"پیام می‌دهم", "پیام می‌دم", "پیام می‌دهیم",
	"فعالیت می‌کنم", "فعالیت می‌کنیم", "تولید می‌کنم", "تولید می‌کنیم",
	"عرضه می‌کنم", "عرضه می‌کنیم", "صادر می‌کنیم", "وارد می‌کنیم",
}

// claimsSenderFromRecipientBusiness rejects a message where the sender claims
// to originate from a generic business role (e.g. "از تولیدکننده ... تماس
// گرفتم"), which would make the sender sound like the recipient's own business.
// Sentences that mention the sender's own company are allowed (that is the
// sender legitimately identifying themselves).
func claimsSenderFromRecipientBusiness(senderCompany, body string) bool {
	senderNorm := normalizeCompanyName(senderCompany)
	for _, sentence := range splitSentences(body) {
		if senderNorm != "" && strings.Contains(sentence, senderNorm) {
			continue
		}
		for _, noun := range businessRoleNouns {
			if !strings.Contains(sentence, "از "+noun) {
				continue
			}
			for _, v := range originVerbs {
				if strings.Contains(sentence, v) {
					return true
				}
			}
		}
	}
	return false
}

// descriptorClaimVerbs are first-person verbs that claim the speaker performs
// the recipient's business ("we produce ...", "we operate in ..."). Only used
// together with a recipient business keyword; generic verbs like "هستیم" are
// intentionally excluded to avoid false positives.
var descriptorClaimVerbs = []string{
	"تولید می‌کنم", "تولید می‌کنیم",
	"عرضه می‌کنم", "عرضه می‌کنیم",
	"فعالیت می‌کنم", "فعالیت می‌کنیم",
	"صادر می‌کنیم", "وارد می‌کنیم",
	"کار می‌کنم", "کار می‌کنیم",
}

// claimsSenderFromDescriptors rejects messages where the sender claims to be
// from or to perform the recipient's business, using the recipient's industry,
// business type or description keywords.
func claimsSenderFromDescriptors(leadCtx *LeadContext, body string) bool {
	if leadCtx == nil {
		return false
	}
	keywords := recipientKeywords(leadCtx)
	if len(keywords) == 0 {
		return false
	}
	for _, sentence := range splitSentences(body) {
		hasKeyword := false
		for _, k := range keywords {
			if strings.Contains(sentence, k) {
				hasKeyword = true
				break
			}
		}
		if !hasKeyword {
			continue
		}
		for _, v := range descriptorClaimVerbs {
			if strings.Contains(sentence, v) {
				return true
			}
		}
		for _, d := range []string{leadCtx.BusinessType, leadCtx.Industry} {
			if d == "" {
				continue
			}
			if strings.Contains(sentence, "از "+d) {
				for _, v := range originVerbs {
					if strings.Contains(sentence, v) {
						return true
					}
				}
			}
		}
	}
	return false
}

// recipientKeywords extracts distinctive words from the recipient's industry,
// business type and description so we can detect when the sender claims to
// perform the recipient's business.
func recipientKeywords(leadCtx *LeadContext) []string {
	var out []string
	seen := map[string]bool{}
	add := func(s string) {
		for _, w := range strings.Fields(s) {
			w = strings.Trim(w, "،؛.،!؟?()")
			if len([]rune(w)) < 3 || seen[w] {
				continue
			}
			seen[w] = true
			out = append(out, w)
		}
	}
	add(leadCtx.BusinessType)
	add(leadCtx.Industry)
	add(leadCtx.Description)
	return out
}

func sentenceClaimsSenderBelongsToRecipient(leadName, body string) bool {
	lead := normalizeCompanyName(leadName)
	if lead == "" {
		lead = leadName
	}
	// Hard signals: the sender explicitly says they are "from"/"at"/"we in" the
	// recipient company. Note "با شرکت X" (cooperation with X) is fine; "از
	// شرکت X" / "ما در شرکت X" is not.
	hard := []string{
		"از " + leadName,
		"از " + lead,
		"از مجموعه " + lead,
		"از گروه " + lead,
		"از طرف شرکت " + lead,
		"ما در " + leadName,
		"ما در " + lead,
		"من از " + lead,
		"همکاران ما در " + lead,
		"کارمندان ما در " + lead,
		"فروشندگان ما در " + lead,
		"from " + lead,
		"from " + leadName,
		"we are " + lead,
		"we work at " + lead,
		"at " + lead + " we",
	}
	for _, h := range hard {
		if strings.Contains(body, h) {
			return true
		}
	}

	// Soft signal: a sentence that names the recipient company AND claims the
	// sender performs the recipient's business there (first-person activity).
	for _, sentence := range splitSentences(body) {
		if !containsCompanyName(sentence, leadName) && !containsCompanyName(sentence, lead) {
			continue
		}
		for _, v := range membershipClaimVerbs {
			if strings.Contains(sentence, v) {
				return true
			}
		}
	}
	return false
}

// containsCompanyName reports whether the sentence contains the company name as
// a word, avoiding false positives from unrelated text.
func containsCompanyName(s, name string) bool {
	name = strings.TrimSpace(name)
	if name == "" {
		return false
	}
	return strings.Contains(s, name)
}

// splitSentences splits Persian/English text on sentence terminators.
func splitSentences(s string) []string {
	r := regexp.MustCompile(`[.!؟?]+|[\n]+`)
	parts := r.Split(s, -1)
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		if t := strings.TrimSpace(p); t != "" {
			out = append(out, t)
		}
	}
	return out
}
