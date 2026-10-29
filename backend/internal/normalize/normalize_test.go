package normalize

import "testing"

func TestNormalizeMobile(t *testing.T) {
	cases := []struct {
		in   string
		want string
	}{
		{"09357578808", "+989357578808"},
		{"+989357578808", "+989357578808"},
		{"00989357578808", "+989357578808"},
		{"989357578808", "+989357578808"},
		{"+98 935 757 8808", "+989357578808"},
		{"0935-757-8808", "+989357578808"},
		{"(0935) 757 8808", "+989357578808"},
		{"", ""},
		{"12345", ""},
		{"02122334455", ""}, // landline is not a mobile
		{"foobar", ""},
		{"۰۹۳۵۷۵۷۸۸۰۸", ""}, // Persian digits unsupported without normalization
	}
	for _, c := range cases {
		if got := NormalizeMobile(c.in); got != c.want {
			t.Errorf("NormalizeMobile(%q) = %q, want %q", c.in, got, c.want)
		}
	}
}

func TestNormalizePhone(t *testing.T) {
	cases := []struct {
		in   string
		want string
	}{
		{"02122334455", "+982122334455"},
		{"+982122334455", "+982122334455"},
		{"00982122334455", "+982122334455"},
		{"2122334455", "+982122334455"},
		{"", ""},
		{"09357578808", ""}, // mobile is not a landline
	}
	for _, c := range cases {
		if got := NormalizePhone(c.in); got != c.want {
			t.Errorf("NormalizePhone(%q) = %q, want %q", c.in, got, c.want)
		}
	}
}

func TestLocalMobile(t *testing.T) {
	if got := LocalMobile("+989357578808"); got != "09357578808" {
		t.Errorf("LocalMobile = %q, want 09357578808", got)
	}
	if got := LocalMobile("+982122334455"); got != "02122334455" {
		t.Errorf("LocalPhone via LocalMobile = %q, want 02122334455", got)
	}
}

func TestNormalizeEmail(t *testing.T) {
	cases := []struct {
		in   string
		want string
	}{
		{"Sales@Example.COM", "sales@example.com"},
		{"  test@foo.ir  ", "test@foo.ir"},
		{"", ""},
	}
	for _, c := range cases {
		if got := NormalizeEmail(c.in); got != c.want {
			t.Errorf("NormalizeEmail(%q) = %q, want %q", c.in, got, c.want)
		}
	}
}

func TestEmailDomain(t *testing.T) {
	if got := EmailDomain("Ali@ToyMakers.com"); got != "toymakers.com" {
		t.Errorf("EmailDomain = %q, want toymakers.com", got)
	}
}

func TestNormalizeWebsite(t *testing.T) {
	cases := []struct {
		in   string
		want string
	}{
		{"https://www.example.com/", "example.com"},
		{"http://example.com", "example.com"},
		{"WWW.Example.COM", "example.com"},
		{"", ""},
	}
	for _, c := range cases {
		if got := NormalizeWebsite(c.in); got != c.want {
			t.Errorf("NormalizeWebsite(%q) = %q, want %q", c.in, got, c.want)
		}
	}
}

func TestNormalizeUsername(t *testing.T) {
	if got := NormalizeUsername("@TalaToy"); got != "talatoy" {
		t.Errorf("NormalizeUsername = %q, want talatoy", got)
	}
}

func TestNormalizeAnyPhone(t *testing.T) {
	if got := NormalizeAnyPhone("09357578808"); got != "+989357578808" {
		t.Errorf("NormalizeAnyPhone mobile = %q", got)
	}
	if got := NormalizeAnyPhone("02122334455"); got != "+982122334455" {
		t.Errorf("NormalizeAnyPhone landline = %q", got)
	}
}
