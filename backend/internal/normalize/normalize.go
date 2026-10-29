// Package normalize provides Iranian phone and email normalization.
package normalize

import (
	"regexp"
	"strings"
)

// Iranian mobile numbers begin with 09 (or +989, 00989, 989) followed by 9 digits.
var (
	mobilePrefixRegex = regexp.MustCompile(`^(?:\+?98|0098|98|0)?(9\d{9})$`)
	phonePrefixRegex  = regexp.MustCompile(`^(?:\+?98|0098|98|0)?([2-8]\d{9})$`)
	digitsOnly        = regexp.MustCompile(`[^0-9+]`)

	// phoneSeparator matches text that only separates digits of one phone
	// number (space, dash, en/em dash, parentheses, dot).
	phoneSeparator = regexp.MustCompile(`^[\s\-–—().]*$`)

	digitChunk = regexp.MustCompile(`[0-9]+`)

	// emailRegex matches a plain email token (after artifacts are cleaned).
	emailRegex = regexp.MustCompile(`[A-Za-z0-9._%+\-]+@[A-Za-z0-9\-]+(?:\.[A-Za-z0-9\-]+)+`)

	// websiteRegex matches a bare or scheme-qualified domain (one or more dots).
	websiteRegex = regexp.MustCompile(`(?i)(?:https?://)?(?:www\.)?[a-z0-9\-]+(?:\.[a-z0-9\-]+)*\.[a-z]{2,}(?:/[^\s]*)?`)

	// postalCodeRegex matches a standalone 10-digit Iranian postal code.
	postalCodeRegex = regexp.MustCompile(`(?:^|[^\d])(\d{10})(?:[^\d]|$)`)

	// leadingNoiseRegex strips scanning artifacts (e.g. "svg") that are glued to
	// the start of a line before contact data.
	leadingNoiseRegex = regexp.MustCompile(`^(?i:svg)+`)
)

// persianDigitReplacer maps Persian and Arabic-Indic digits to Latin digits.
var persianDigitReplacer = strings.NewReplacer(
	"۰", "0", "۱", "1", "۲", "2", "۳", "3", "۴", "4", "۵", "5", "۶", "6", "۷", "7", "۸", "8", "۹", "9",
	"٠", "0", "١", "1", "٢", "2", "٣", "3", "٤", "4", "٥", "5", "٦", "6", "٧", "7", "٨", "8", "٩", "9",
)

// zeroWidthReplacer removes invisible characters that appear in scraped text.
var zeroWidthReplacer = strings.NewReplacer(
	"\u200b", "", // ZERO WIDTH SPACE
	"\u200c", "", // ZERO WIDTH NON-JOINER
	"\u200d", "", // ZERO WIDTH JOINER
	"\ufeff", "", // BYTE ORDER MARK
	"\u2060", "", // WORD JOINER
	"\u00ad", "", // SOFT HYPHEN
)

// CleanSourceText is the minimal, lossless pre-LLM cleanup step. It runs per
// line and:
//   - strips leading scanning-artifact noise (e.g. "svg") so explicit contact
//     data stays visible to the LLM (svg031-35314946 -> 031-35314946)
//   - unescapes `\@` etc. (svginfo\@sadaf2.ir -> info@sadaf2.ir)
//   - removes zero-width / invisible characters
//   - maps Persian/Arabic digits to Latin
//   - collapses duplicated whitespace
//
// Line/block structure and ALL meaningful text are preserved. The LLM remains
// the authority for semantic extraction and company-to-contact association.
func CleanSourceText(s string) string {
	lines := strings.Split(s, "\n")
	for i := range lines {
		lines[i] = CleanScrapedText(StripContactNoise(lines[i]))
	}
	return strings.Join(lines, "\n")
}

// CleanScrapedText normalizes common scraping artifacts WITHOUT deleting
// meaningful text:
//   - unescapes `\@` (and backslashes before email punctuation) -> `@`
//   - removes zero-width / invisible characters
//   - maps Persian/Arabic digits to Latin
//   - collapses duplicated whitespace
func CleanScrapedText(s string) string {
	s = zeroWidthReplacer.Replace(s)
	s = persianDigitReplacer.Replace(s)
	s = unescapeContact(s)
	return collapseSpaces(s)
}

// unescapeContact unescapes backslash-escaped punctuation used in scraped
// emails (e.g. `info\@sadaf2.ir` -> `info@sadaf2.ir`) and trailing backslashes.
func unescapeContact(s string) string {
	s = strings.ReplaceAll(s, "\\@", "@")
	s = strings.ReplaceAll(s, "\\.", ".")
	s = strings.ReplaceAll(s, "\\_", "_")
	s = strings.ReplaceAll(s, "\\-", "-")
	return s
}

// StripContactNoise removes leading scraping-artifact tokens (e.g. "svg") that
// are glued to the start of a line. Only applied to the line being scanned for
// contact fields, so meaningful "svg" inside the body text is preserved.
func StripContactNoise(line string) string {
	return leadingNoiseRegex.ReplaceAllString(strings.TrimLeft(line, " \t"), "")
}

func collapseSpaces(s string) string {
	// collapse runs of horizontal whitespace but preserve newlines so block
	// (per-company) structure survives cleaning.
	return regexp.MustCompile(`[ \t]+`).ReplaceAllString(s, " ")
}

// ExtractPhones returns canonical (+98…) Iranian phone numbers found in s.
// Landlines and mobiles are both returned, already normalized.
func ExtractPhones(s string) []string {
	cleaned := CleanScrapedText(s)
	var out []string
	seen := map[string]bool{}
	for _, line := range strings.Split(cleaned, "\n") {
		line = StripContactNoise(line)
		for _, tok := range phoneTokens(line) {
			if p := NormalizePhone(tok); p != "" && !seen[p] {
				seen[p] = true
				out = append(out, p)
			} else if mb := NormalizeMobile(tok); mb != "" && !seen[mb] {
				seen[mb] = true
				out = append(out, mb)
			}
		}
	}
	return out
}

// phoneTokens extracts maximal digit sequences that may be separated only by
// phone separators (space, dash, parentheses, dot). Trailing unrelated digits
// (e.g. an internal extension separated by words) are not joined, so a valid
// number is not polluted. Over-long tokens are rejected by normalization.
func phoneTokens(s string) []string {
	matches := digitChunk.FindAllStringIndex(s, -1)
	var tokens []string
	var cur []string
	prevEnd := -1
	for i, m := range matches {
		if i > 0 {
			between := s[prevEnd:m[0]]
			if !phoneSeparator.MatchString(between) {
				if len(cur) > 0 {
					tokens = append(tokens, strings.Join(cur, ""))
					cur = nil
				}
			}
		}
		cur = append(cur, s[m[0]:m[1]])
		prevEnd = m[1]
	}
	if len(cur) > 0 {
		tokens = append(tokens, strings.Join(cur, ""))
	}
	return tokens
}

// ExtractPhonesUnique returns the first phone and first mobile (deduplicated).
func ExtractPhonesUnique(s string) (landline, mobile string) {
	for _, p := range ExtractPhones(s) {
		if strings.HasPrefix(p, "+989") {
			if mobile == "" {
				mobile = p
			}
		} else {
			if landline == "" {
				landline = p
			}
		}
	}
	return landline, mobile
}

// ExtractEmails returns normalized, validated emails found in s.
func ExtractEmails(s string) []string {
	cleaned := CleanScrapedText(s)
	var out []string
	seen := map[string]bool{}
	for _, line := range strings.Split(cleaned, "\n") {
		line = StripContactNoise(line)
		for _, m := range emailRegex.FindAllString(line, -1) {
			e := NormalizeEmail(m)
			if IsValidEmail(e) && !seen[e] {
				seen[e] = true
				out = append(out, e)
			}
		}
	}
	return out
}

// IsValidEmail reports whether e is a structurally valid email address.
func IsValidEmail(e string) bool {
	e = NormalizeEmail(e)
	if len(e) < 3 || len(e) > 254 {
		return false
	}
	at := strings.LastIndex(e, "@")
	if at <= 0 || at == len(e)-1 {
		return false
	}
	local, domain := e[:at], e[at+1:]
	if strings.ContainsAny(local, " \t\r\n") || strings.ContainsAny(domain, " \t\r\n") {
		return false
	}
	if !strings.Contains(domain, ".") {
		return false
	}
	return true
}

// ExtractWebsite returns the first bare/scheme-qualified domain in s, or "".
func ExtractWebsite(s string) string {
	cleaned := CleanScrapedText(s)
	for _, idx := range websiteRegex.FindAllStringIndex(cleaned, -1) {
		m := cleaned[idx[0]:idx[1]]
		if strings.Contains(m, "@") {
			continue // an email, not a website
		}
		// skip when this is the domain of an email (preceded by '@')
		if idx[0] > 0 && cleaned[idx[0]-1] == '@' {
			continue
		}
		return NormalizeWebsite(m)
	}
	return ""
}

// ExtractPostalCode returns a standalone 10-digit postal code, or "".
func ExtractPostalCode(s string) string {
	cleaned := CleanScrapedText(s)
	for _, m := range postalCodeRegex.FindAllStringSubmatch(cleaned, -1) {
		if len(m) == 2 {
			// must not itself be a valid phone
			if NormalizePhone(m[1]) == "" && NormalizeMobile(m[1]) == "" {
				return m[1]
			}
		}
	}
	return ""
}

// NormalizeMobile returns the canonical +98 mobile representation or "" if invalid.
// Accepts 09357578808, +989357578808, 00989357578808, 989357578808.
func NormalizeMobile(raw string) string {
	cleaned := strings.TrimSpace(raw)
	if cleaned == "" {
		return ""
	}
	// strip common separators like 09-357-578-808, (0935) ...
	cleaned = digitsOnly.ReplaceAllString(cleaned, "")
	if cleaned == "" {
		return ""
	}
	if m := mobilePrefixRegex.FindStringSubmatch(cleaned); m != nil {
		return "+98" + m[1]
	}
	return ""
}

// NormalizePhone returns canonical +98 for landlines, "" if invalid.
func NormalizePhone(raw string) string {
	cleaned := strings.TrimSpace(raw)
	if cleaned == "" {
		return ""
	}
	cleaned = digitsOnly.ReplaceAllString(cleaned, "")
	if cleaned == "" {
		return ""
	}
	if m := phonePrefixRegex.FindStringSubmatch(cleaned); m != nil {
		return "+98" + m[1]
	}
	return ""
}

// NormalizeAnyPhone detects mobile vs landline.
func NormalizeAnyPhone(raw string) string {
	if m := NormalizeMobile(raw); m != "" {
		return m
	}
	return NormalizePhone(raw)
}

// LocalMobile returns the local display form 09357578808 from a canonical +989... number.
func LocalMobile(canonical string) string {
	if strings.HasPrefix(canonical, "+98") && len(canonical) == 13 {
		return "0" + canonical[3:]
	}
	return ""
}

// LocalPhone returns the local display form for landlines (0...).
func LocalPhone(canonical string) string {
	if strings.HasPrefix(canonical, "+98") && len(canonical) == 13 {
		return "0" + canonical[3:]
	}
	return ""
}

// NormalizeEmail lowercases and trims whitespace.
func NormalizeEmail(raw string) string {
	email := strings.TrimSpace(raw)
	if email == "" {
		return ""
	}
	return strings.ToLower(email)
}

// EmailDomain extracts the domain from a normalized email.
func EmailDomain(email string) string {
	email = NormalizeEmail(email)
	idx := strings.LastIndex(email, "@")
	if idx < 0 {
		return ""
	}
	return strings.ToLower(email[idx+1:])
}

// NormalizeWebsite strips scheme and trailing slash for comparison.
func NormalizeWebsite(raw string) string {
	s := strings.TrimSpace(strings.ToLower(raw))
	s = strings.TrimPrefix(s, "https://")
	s = strings.TrimPrefix(s, "http://")
	s = strings.TrimPrefix(s, "www.")
	s = strings.TrimSuffix(s, "/")
	return s
}

// NormalizeUsername returns a normalized handle (lowercase, no @).
func NormalizeUsername(raw string) string {
	return strings.TrimPrefix(strings.TrimSpace(strings.ToLower(raw)), "@")
}
