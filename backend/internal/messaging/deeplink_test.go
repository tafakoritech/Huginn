package messaging

import (
	"strings"
	"testing"

	"github.com/tafakorpardazan/huginn/internal/contacts"
	"github.com/tafakorpardazan/huginn/internal/interactions"
)

func strPtr(s string) *string { return &s }

func testContact() *contacts.Contact {
	return &contacts.Contact{
		Mobile:           strPtr("09357578808"),
		MobileNormalized: strPtr("+989357578808"),
		PhoneNormalized:  strPtr("+982122334455"),
		Email:            strPtr("sales@example.com"),
		Telegram:         strPtr("@tala_shop"),
		Bale:             strPtr("tala_shop"),
	}
}

func TestBuildDeepLinkCall(t *testing.T) {
	dl := BuildDeepLink(testContact(), interactions.ChannelCall, "")
	if dl.Type != "tel" || dl.Target != "tel:+989357578808" {
		t.Errorf("unexpected call deep link: %+v", dl)
	}
}

func TestBuildDeepLinkWhatsApp(t *testing.T) {
	dl := BuildDeepLink(testContact(), interactions.ChannelWhatsApp, "")
	if dl.Type != "wa.me" {
		t.Fatalf("expected wa.me, got %s", dl.Type)
	}
	if dl.Target != "https://wa.me/989357578808" {
		t.Errorf("unexpected wa.me target: %s", dl.Target)
	}
}

func TestBuildDeepLinkTelegram(t *testing.T) {
	dl := BuildDeepLink(testContact(), interactions.ChannelTelegram, "")
	if dl.Type != "username" || dl.Username != "tala_shop" {
		t.Errorf("unexpected telegram deep link: %+v", dl)
	}
	if !strings.HasPrefix(dl.Target, "https://t.me/") {
		t.Errorf("unexpected telegram target: %s", dl.Target)
	}
}

func TestBuildDeepLinkNoUsernameRequiresContactSave(t *testing.T) {
	c := &contacts.Contact{Mobile: strPtr("09357578808")}
	dl := BuildDeepLink(c, interactions.ChannelEitaa, "")
	if dl.Type != ContactSaveRequired {
		t.Errorf("expected CONTACT_SAVE_REQUIRED, got %s", dl.Type)
	}
	dl2 := BuildDeepLink(c, interactions.ChannelRubika, "")
	if dl2.Type != ContactSaveRequired {
		t.Errorf("rubika must not fabricate chat URLs, got %s", dl2.Type)
	}
}

func TestBuildDeepLinkSMS(t *testing.T) {
	dl := BuildDeepLink(testContact(), interactions.ChannelSMS, "hello")
	if dl.Type != "sms" || dl.Target != "+989357578808" || dl.Text != "hello" {
		t.Errorf("unexpected sms deep link: %+v", dl)
	}
}

func TestBuildDeepLinkEmail(t *testing.T) {
	dl := BuildDeepLink(testContact(), interactions.ChannelEmail, "")
	if dl.Type != "mailto" || dl.Target != "sales@example.com" {
		t.Errorf("unexpected email deep link: %+v", dl)
	}
}

func TestBuildVCard(t *testing.T) {
	c := &contacts.Contact{
		FullName:         strPtr("رضا محمدی"),
		MobileNormalized: strPtr("+989357578808"),
		Email:            strPtr("sales@example.com"),
	}
	v := BuildVCard(c, "بازرگانی پولیش")
	for _, expected := range []string{"BEGIN:VCARD", "FN:", "TEL;TYPE=CELL;PREF=1:+989357578808", "EMAIL;TYPE=INTERNET:sales@example.com", "ORG:", "END:VCARD"} {
		if !strings.Contains(v, expected) {
			t.Errorf("vCard missing %q:\n%s", expected, v)
		}
	}
}
