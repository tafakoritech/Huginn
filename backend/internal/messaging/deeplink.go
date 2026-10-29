package messaging

import (
	"fmt"
	"net/url"
	"strings"

	"github.com/tafakorpardazan/huginn/internal/contacts"
	"github.com/tafakorpardazan/huginn/internal/interactions"
	"github.com/tafakorpardazan/huginn/internal/normalize"
)

// DeepLink describes the target the client should open.
type DeepLink struct {
	Channel  interactions.Channel `json:"channel"`
	Type     string               `json:"type"` // tel | sms | wa.me | username | contact_save_required
	Target   string               `json:"target"`
	Text     string               `json:"text,omitempty"`
	Username string               `json:"username,omitempty"`
	Note     string               `json:"note,omitempty"`
}

// ContactSaveRequired is returned when no username is available for eitaa/rubika.
const ContactSaveRequired = "CONTACT_SAVE_REQUIRED"

// BuildDeepLink constructs the appropriate deep link for a channel/contact.
func BuildDeepLink(c *contacts.Contact, channel interactions.Channel, suggestedText string) *DeepLink {
	switch channel {
	case interactions.ChannelCall:
		return &DeepLink{Channel: channel, Type: "tel", Target: telTarget(c)}
	case interactions.ChannelSMS:
		return &DeepLink{Channel: channel, Type: "sms", Target: smsTarget(c), Text: suggestedText}
	case interactions.ChannelWhatsApp:
		return &DeepLink{Channel: channel, Type: "wa.me", Target: waTarget(c)}
	case interactions.ChannelTelegram:
		if c.Telegram != nil && *c.Telegram != "" {
			return &DeepLink{Channel: channel, Type: "username", Username: normalize.NormalizeUsername(*c.Telegram), Target: "https://t.me/" + normalize.NormalizeUsername(*c.Telegram)}
		}
		return &DeepLink{Channel: channel, Type: ContactSaveRequired, Note: "no telegram username available"}
	case interactions.ChannelBale:
		if c.Bale != nil && *c.Bale != "" {
			return &DeepLink{Channel: channel, Type: "username", Username: normalize.NormalizeUsername(*c.Bale), Target: "https://ble.ir/" + normalize.NormalizeUsername(*c.Bale)}
		}
		return &DeepLink{Channel: channel, Type: ContactSaveRequired, Note: "no bale username available"}
	case interactions.ChannelEitaa:
		if c.Eitaa != nil && *c.Eitaa != "" {
			return &DeepLink{Channel: channel, Type: "username", Username: normalize.NormalizeUsername(*c.Eitaa), Target: "https://eitaa.com/" + normalize.NormalizeUsername(*c.Eitaa)}
		}
		return &DeepLink{Channel: channel, Type: ContactSaveRequired, Note: "no eitaa username available"}
	case interactions.ChannelRubika:
		if c.Rubika != nil && *c.Rubika != "" {
			return &DeepLink{Channel: channel, Type: "username", Username: normalize.NormalizeUsername(*c.Rubika), Target: "https://rubika.ir/" + normalize.NormalizeUsername(*c.Rubika)}
		}
		return &DeepLink{Channel: channel, Type: ContactSaveRequired, Note: "no rubika username available"}
	case interactions.ChannelEmail:
		return &DeepLink{Channel: channel, Type: "mailto", Target: emailTarget(c)}
	default:
		return &DeepLink{Channel: channel, Type: "other"}
	}
}

func telTarget(c *contacts.Contact) string {
	if c.MobileNormalized != nil {
		return "tel:" + *c.MobileNormalized
	}
	if c.PhoneNormalized != nil {
		return "tel:" + *c.PhoneNormalized
	}
	if c.Mobile != nil {
		if n := normalize.NormalizeMobile(*c.Mobile); n != "" {
			return "tel:" + n
		}
		return "tel:" + *c.Mobile
	}
	return ""
}

func smsTarget(c *contacts.Contact) string {
	if c.MobileNormalized != nil {
		return *c.MobileNormalized
	}
	if c.Mobile != nil {
		if n := normalize.NormalizeMobile(*c.Mobile); n != "" {
			return n
		}
		return *c.Mobile
	}
	return ""
}

func waTarget(c *contacts.Contact) string {
	num := ""
	if c.MobileNormalized != nil {
		num = *c.MobileNormalized
	} else if c.Mobile != nil {
		num = normalize.NormalizeMobile(*c.Mobile)
	}
	num = strings.TrimPrefix(num, "+")
	if num == "" {
		return ""
	}
	return "https://wa.me/" + num
}

func emailTarget(c *contacts.Contact) string {
	if c.Email != nil {
		return *c.Email
	}
	return ""
}

// BuildVCard returns a vCard 3.0 string for a contact.
func BuildVCard(c *contacts.Contact, leadName string) string {
	var b strings.Builder
	b.WriteString("BEGIN:VCARD\r\n")
	b.WriteString("VERSION:3.0\r\n")
	full := ""
	if c.FullName != nil {
		full = *c.FullName
	} else {
		parts := []string{}
		if c.FirstName != nil {
			parts = append(parts, *c.FirstName)
		}
		if c.LastName != nil {
			parts = append(parts, *c.LastName)
		}
		full = strings.Join(parts, " ")
	}
	if full == "" {
		full = leadName
	}
	if full != "" {
		b.WriteString("FN:" + escapeVCard(full) + "\r\n")
		b.WriteString("N:" + vcardName(full) + "\r\n")
	}
	if c.JobTitle != nil && *c.JobTitle != "" {
		b.WriteString("TITLE:" + escapeVCard(*c.JobTitle) + "\r\n")
	}
	if leadName != "" {
		b.WriteString("ORG:" + escapeVCard(leadName) + "\r\n")
	}
	if c.MobileNormalized != nil {
		b.WriteString("TEL;TYPE=CELL;PREF=1:" + *c.MobileNormalized + "\r\n")
	} else if c.Mobile != nil {
		b.WriteString("TEL;TYPE=CELL;PREF=1:" + *c.Mobile + "\r\n")
	}
	if c.PhoneNormalized != nil {
		b.WriteString("TEL;TYPE=VOICE:" + *c.PhoneNormalized + "\r\n")
	} else if c.Phone != nil {
		b.WriteString("TEL;TYPE=VOICE:" + *c.Phone + "\r\n")
	}
	if c.EmailNormalized != nil {
		b.WriteString("EMAIL;TYPE=INTERNET:" + *c.EmailNormalized + "\r\n")
	} else if c.Email != nil {
		b.WriteString("EMAIL;TYPE=INTERNET:" + *c.Email + "\r\n")
	}
	if c.Telegram != nil && *c.Telegram != "" {
		b.WriteString("X-SOCIALPROFILE;TYPE=telegram:" + normalize.NormalizeUsername(*c.Telegram) + "\r\n")
	}
	b.WriteString("END:VCARD\r\n")
	return b.String()
}

func vcardName(full string) string {
	parts := strings.Fields(full)
	if len(parts) == 0 {
		return ";;;;"
	}
	last := parts[len(parts)-1]
	first := strings.Join(parts[:len(parts)-1], " ")
	return escapeVCard(last) + ";" + escapeVCard(first) + ";;;"
}

func escapeVCard(s string) string {
	s = strings.ReplaceAll(s, "\\", "\\\\")
	s = strings.ReplaceAll(s, ";", "\\;")
	s = strings.ReplaceAll(s, ",", "\\,")
	s = strings.ReplaceAll(s, "\r\n", " ")
	s = strings.ReplaceAll(s, "\n", " ")
	return s
}

// ValidateDeepLinkText is a no-op placeholder to keep url import if unused later.
var _ = fmt.Sprintf
var _ = url.Values{}
