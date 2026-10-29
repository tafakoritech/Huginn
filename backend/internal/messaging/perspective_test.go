package messaging

import (
	"strings"
	"testing"

	"github.com/tafakorpardazan/huginn/internal/leads"
)

func testSender() *SenderProfile {
	return &SenderProfile{
		CompanyName: "فروشگاه اینترنتی پارس",
		Description: "عرضه محصولات هوشمند خانگی",
		SenderName:  "علی رضایی",
		SenderTitle: "کارشناس فروش",
	}
}

func sadafLead() *leads.Lead {
	return &leads.Lead{
		CompanyName:  "شرکت صدف",
		Industry:     strPtr("عروسک‌سازی"),
		BusinessType: strPtr("manufacturer"),
	}
}

func leadContextOf(lead *leads.Lead) *LeadContext {
	return &LeadContext{
		CompanyName:    lead.CompanyName,
		BusinessType:   strVal(lead.BusinessType),
		Industry:       strVal(lead.Industry),
		ContactSummary: "سارا محمدی",
	}
}

func TestValidateMessagePerspectiveRejectsImpersonation(t *testing.T) {
	sender := testSender()

	cases := []struct {
		name string
		lead *leads.Lead
		body string
	}{
		{
			name: "sender claims to be from the lead company",
			lead: sadafLead(),
			body: "من از شرکت صدف که در زمینه عروسک‌سازی فعالیت می‌کنیم مزاحمتون می‌شم.",
		},
		{
			name: "sender works at lead company (we in)",
			lead: sadafLead(),
			body: "سلام، ما در شرکت صدف برای همکاری با شما تماس می‌گیریم.",
		},
		{
			name: "sender from stripped lead company name",
			lead: sadafLead(),
			body: "از صدف در زمینه عروسک‌سازی تماس می‌گیرم.",
		},
		{
			name: "english impersonation",
			lead: &leads.Lead{CompanyName: "Sadaf Toys"},
			body: "Hi, I'm from Sadaf Toys, we manufacture dolls.",
		},
		{
			name: "sender performs recipient business there",
			lead: sadafLead(),
			body: "ما در شرکت صدف تولید عروسک می‌کنیم.",
		},
		{
			name: "empty body",
			lead: sadafLead(),
			body: "",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if validateMessagePerspective(sender, leadContextOf(tc.lead), tc.body) {
				t.Errorf("expected message to be rejected (invalid perspective): %q", tc.body)
			}
		})
	}
}

func TestValidateMessagePerspectiveAcceptsCorrectPerspective(t *testing.T) {
	sender := testSender()
	leadCtx := leadContextOf(sadafLead())

	cases := []struct {
		name string
		body string
	}{
		{
			name: "cooperation with lead company (the regression case)",
			body: "سلام، برای همکاری با شرکت صدف در زمینه محصولات هوشمند و مرتبط با صنعت اسباب‌بازی پیام می‌دم.",
		},
		{
			name: "names own sender company and addresses recipient",
			body: "سلام، از فروشگاه اینترنتی پارس هستم. برای آشنایی و بررسی فرصت همکاری با شرکت صدف تماس می‌گیرم.",
		},
		{
			name: "no lead company present",
			body: "سلام، وقت بخیر. برای همکاری پیام می‌دم.",
		},
		{
			name: "recipient company as target with pronoun",
			body: "سلام، وقت بخیر. برای بررسی امکان همکاری با مجموعه شما تماس می‌گیرم.",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if !validateMessagePerspective(sender, leadCtx, tc.body) {
				t.Errorf("expected message to be accepted (correct perspective): %q", tc.body)
			}
		})
	}
}

func TestValidateMessagePerspectiveRejectsIndustryImpersonation(t *testing.T) {
	sender := testSender()
	ctx := &LeadContext{
		CompanyName:  "کارخانه صدف",
		BusinessType: "manufacturer",
		Industry:     "عروسک‌سازی",
		Description:  "تولید و پخش عمده انواع عروسک پولیشی",
	}
	bad := []string{
		"آقای محمدی سلام. از تولیدکننده عروسک پولیشی در تهران تماس گرفتم.",
		"سلام، از عروسک‌سازی تماس می‌گیرم.",
		"ما در زمینه تولید عروسک فعالیت می‌کنیم.",
	}
	for _, b := range bad {
		if validateMessagePerspective(sender, ctx, b) {
			t.Errorf("expected industry impersonation to be rejected: %q", b)
		}
	}
	good := "سلام، برای بررسی امکان همکاری با کارخانه صدف تماس می‌گیرم."
	if !validateMessagePerspective(sender, ctx, good) {
		t.Errorf("expected correct perspective to pass: %q", good)
	}
}

func TestValidateMessagePerspectiveSenderEqualsRecipient(t *testing.T) {
	// Identity collision: our company must never be the same as the lead company.
	leadCtx := leadContextOf(sadafLead())
	sender := &SenderProfile{CompanyName: "شرکت صدف"}
	if validateMessagePerspective(sender, leadCtx, "سلام، برای همکاری با شما تماس می‌گیرم.") {
		t.Error("expected sender==recipient to be rejected")
	}
}

func TestValidateMessagePerspectiveNoLeadCompany(t *testing.T) {
	ctx := &LeadContext{BusinessType: "manufacturer"}
	if !validateMessagePerspective(testSender(), ctx, "سلام، از شرکت صدف هستم") {
		t.Error("without a recipient company there is nothing to impersonate; message should pass")
	}
	// But industry impersonation is still rejected even without a company name.
	if validateMessagePerspective(testSender(), ctx, "از تولیدکننده تماس می‌گیرم") {
		t.Error("industry impersonation must be rejected even without a company name")
	}
}

func TestBuildMessagePromptSeparatesSenderAndRecipient(t *testing.T) {
	sender := testSender()
	leadCtx := leadContextOf(sadafLead())

	p := buildMessagePrompt("SMS", sender, leadCtx, "FIRST_CONTACT")

	if !strings.Contains(p, "SENDER (our company") {
		t.Errorf("prompt must label the SENDER block:\n%s", p)
	}
	if !strings.Contains(p, "RECIPIENT (the lead") {
		t.Errorf("prompt must label the RECIPIENT block:\n%s", p)
	}
	// The sender block must contain OUR company, the recipient block the lead's.
	senderBlock := p[strings.Index(p, "SENDER"):strings.Index(p, "RECIPIENT")]
	recipientBlock := p[strings.Index(p, "RECIPIENT"):]
	if !strings.Contains(senderBlock, "فروشگاه اینترنتی پارس") {
		t.Errorf("sender block must contain our company name:\n%s", senderBlock)
	}
	if strings.Contains(senderBlock, "شرکت صدف") {
		t.Errorf("sender block must never contain the lead company:\n%s", senderBlock)
	}
	if !strings.Contains(recipientBlock, "شرکت صدف") {
		t.Errorf("recipient block must contain the lead company:\n%s", recipientBlock)
	}
	if strings.Contains(recipientBlock, "فروشگاه اینترنتی پارس") {
		t.Errorf("recipient block must never contain our company name:\n%s", recipientBlock)
	}
	if !strings.Contains(p, "never say the sender works there") {
		t.Errorf("prompt must enforce the sender/recipient separation rule:\n%s", p)
	}
}

func TestBuildMessagePromptNoSenderProfile(t *testing.T) {
	p := buildMessagePrompt("SMS", nil, leadContextOf(sadafLead()), "FIRST_CONTACT")
	if !strings.Contains(p, "SENDER (our company") || !strings.Contains(p, "RECIPIENT (the lead") {
		t.Errorf("prompt must still separate sender and recipient blocks:\n%s", p)
	}
}

func TestBuildMessagePromptIncludesRichContext(t *testing.T) {
	ctx := &LeadContext{
		CompanyName:      "کارخانه صدف",
		BusinessType:     "manufacturer",
		Industry:         "اسباب‌بازی",
		City:             "تهران",
		Description:      "تولید عروسک پولیشی",
		Source:           "لیست تولیدکنندگان",
		Status:           "CONTACTING",
		Tags:             "اولویت بالا",
		ContactSummary:   "سارا محمدی، مدیر خرید",
		PastInteractions: "- CALL (2026-01-01), outcome=INTERESTED",
		Notes:            "- علاقه‌مند به نمونه محصول",
	}
	p := buildMessagePrompt("WHATSAPP", testSender(), ctx, "FIRST_CONTACT")
	for _, want := range []string{"کارخانه صدف", "تهران", "تولید عروسک پولیشی", "لیست تولیدکنندگان", "CONTACTING", "سارا محمدی", "INTERESTED", "علاقه‌مند به نمونه محصول"} {
		if !strings.Contains(p, want) {
			t.Errorf("prompt must include the real lead context %q:\n%s", want, p)
		}
	}
}
