package normalize

import (
	"reflect"
	"testing"
)

func TestCleanScrapedText(t *testing.T) {
	cases := []struct {
		in   string
		want string
	}{
		{`info\@sadaf2.ir`, "info@sadaf2.ir"},
		{`info\@sadaf2.ir و 031-35314946`, "info@sadaf2.ir و 031-35314946"},
		{"۰۹۳۵۷۵۷۸۸۰۸", "09357578808"},
		{"٠٩٣٥٧٥٧٨٨٠٨", "09357578808"},
		{"a\u200bb\u200cc", "abc"},
		{"a\uFEFFb", "ab"},
		{"hello   world", "hello world"},
		{"\u200c۰۹۳۵۷۵۷۸۸۰۸", "09357578808"},
	}
	for _, c := range cases {
		if got := CleanScrapedText(c.in); got != c.want {
			t.Errorf("CleanScrapedText(%q) = %q, want %q", c.in, got, c.want)
		}
	}
}

func TestStripContactNoise(t *testing.T) {
	cases := []struct{ in, want string }{
		{"svg031-35314946", "031-35314946"},
		{"svginfo@sadaf2.ir", "info@sadaf2.ir"},
		{"SVG031-35314946", "031-35314946"},
		{"  svg031-35314946", "031-35314946"},
		{"031-35314946", "031-35314946"},
		{"تولید اسباب بازی svg", "تولید اسباب بازی svg"}, // interior svg kept
	}
	for _, c := range cases {
		if got := StripContactNoise(c.in); got != c.want {
			t.Errorf("StripContactNoise(%q) = %q, want %q", c.in, got, c.want)
		}
	}
}

func TestExtractPhones(t *testing.T) {
	cases := []struct {
		in      string
		phones  []string // canonical
	}{
		{"031-35314946", []string{"+983135314946"}},
		{"021-12345678", []string{"+982112345678"}},
		{"02112345678", []string{"+982112345678"}},
		{"0912 123 4567", []string{"+989121234567"}},
		{"0912-123-4567", []string{"+989121234567"}},
		{"+98 912 123 4567", []string{"+989121234567"}},
		{"svg031-35314946", []string{"+983135314946"}},
		{"شماره تماس: 03135314946 داخلی 101", []string{"+983135314946"}},
		{"کد پستی 1234567890", nil}, // postal code is not a phone
		{"اصفهان، کوچه ارغوان 5، پلاک 49", nil},
	}
	for _, c := range cases {
		got := ExtractPhones(c.in)
		if !reflect.DeepEqual(got, c.phones) {
			t.Errorf("ExtractPhones(%q) = %v, want %v", c.in, got, c.phones)
		}
	}
}

func TestExtractPhonesUnique(t *testing.T) {
	landline, mobile := ExtractPhonesUnique("031-35314946\n0912 123 4567")
	if landline != "+983135314946" || mobile != "+989121234567" {
		t.Fatalf("got landline=%s mobile=%s", landline, mobile)
	}
}

func TestExtractEmails(t *testing.T) {
	cases := []struct {
		in   string
		want []string
	}{
		{"info@sadaf2.ir", []string{"info@sadaf2.ir"}},
		{`info\@sadaf2.ir`, []string{"info@sadaf2.ir"}},
		{"svginfo@sadaf2.ir", []string{"info@sadaf2.ir"}},
		{`svginfo\@sadaf2.ir`, []string{"info@sadaf2.ir"}},
		{"Sales@Example.COM", []string{"sales@example.com"}},
		{"no email here", nil},
		{"t@t", nil}, // no TLD
	}
	for _, c := range cases {
		got := ExtractEmails(c.in)
		if !reflect.DeepEqual(got, c.want) {
			t.Errorf("ExtractEmails(%q) = %v, want %v", c.in, got, c.want)
		}
	}
}

func TestIsValidEmail(t *testing.T) {
	valid := []string{"info@sadaf2.ir", "a.b+c@example.com", "x@y.io"}
	for _, e := range valid {
		if !IsValidEmail(e) {
			t.Errorf("IsValidEmail(%q) = false", e)
		}
	}
	invalid := []string{"", "no-at", "@x.com", "a@", "a b@c.com", "a@b"}
	for _, e := range invalid {
		if IsValidEmail(e) {
			t.Errorf("IsValidEmail(%q) = true", e)
		}
	}
}

func TestExtractWebsite(t *testing.T) {
	cases := []struct{ in, want string }{
		{"https://www.example.com/", "example.com"},
		{"وب سایت: example.ir", "example.ir"},
		{"info@sadaf2.ir", ""}, // email, not a website
		{"بدون سایت", ""},
	}
	for _, c := range cases {
		if got := ExtractWebsite(c.in); got != c.want {
			t.Errorf("ExtractWebsite(%q) = %q, want %q", c.in, got, c.want)
		}
	}
}

func TestExtractPostalCode(t *testing.T) {
	if got := ExtractPostalCode("کد پستی 1234567890"); got != "1234567890" {
		t.Errorf("ExtractPostalCode = %q", got)
	}
	if got := ExtractPostalCode("03135314946"); got != "" {
		t.Errorf("phone must not be treated as postal code, got %q", got)
	}
}

// TestCleanSourceTextIrregularMixedFormats is a regression test for irregular
// and mixed source formats: companies that use svg artifacts, escaped emails,
// Persian digits, plain formats, and one with no contact data at all. The
// cleanup must be lossless (all meaningful data preserved) and LLM-visible.
func TestCleanSourceTextIrregularMixedFormats(t *testing.T) {
	src := `شرکت آلفا
svg031-35314946
svginfo\@alpha.ir

شرکت بتا
۰۲۱-۸۸۸۸-۷۷۶۶
info@beta.ir

شرکت گاما
وبسایت www.gamma.ir
0912 123 4567
پلاک ۴۹ خیابان آزادی

شرکت دلتا
فقط توضیحات بدون تماس`

	cleaned := CleanSourceText(src)
	for _, want := range []string{
		"شرکت آلفا", "031-35314946", "info@alpha.ir",
		"شرکت بتا", "021-8888-7766", "info@beta.ir",
		"شرکت گاما", "www.gamma.ir", "0912 123 4567", "پلاک 49 خیابان آزادی",
		"شرکت دلتا", "فقط توضیحات بدون تماس",
	} {
		if !contains(cleaned, want) {
			t.Errorf("cleanup lost meaningful data %q:\n%s", want, cleaned)
		}
	}
	// artifacts removed: leading svg, escaped \@, zero-width, Persian digits
	for _, bad := range []string{"svg031", `info\@alpha.ir`, "۰۲۱", "۴۹"} {
		if contains(cleaned, bad) {
			t.Errorf("artifact not cleaned: %q\n%s", bad, cleaned)
		}
	}
	// block structure (blank lines) preserved
	if !contains(cleaned, "شرکت آلفا\n031-35314946") || !contains(cleaned, "شرکت دلتا") {
		t.Errorf("block structure not preserved:\n%s", cleaned)
	}
}

func contains(s, sub string) bool {
	for i := 0; i+len(sub) <= len(s); i++ {
		if s[i:i+len(sub)] == sub {
			return true
		}
	}
	return false
}
