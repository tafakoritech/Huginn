package messaging

import (
	"context"
	"encoding/json"
	"strings"

	"github.com/tafakorpardazan/huginn/internal/interactions"
	"github.com/tafakorpardazan/huginn/internal/llm"
)

// MessageTemplate is the structured output for a suggested message.
type MessageTemplate struct {
	Subject string `json:"subject,omitempty"`
	Body    string `json:"body"`
}

// MessageGenerator produces a short, context-aware suggested message via the LLM.
type MessageGenerator interface {
	Generate(ctx context.Context, channel interactions.Channel, leadCtx *LeadContext, objective string) (*MessageTemplate, error)
}

// Generator produces short context-aware suggested messages via the LLM.
type Generator struct {
	runner   llm.Runner
	sender   *SenderProfile
	prompts  PromptSettingsProvider
	maxChars int
}

func NewGenerator(runner llm.Runner, sender *SenderProfile, prompts PromptSettingsProvider) *Generator {
	return &Generator{runner: runner, sender: sender, prompts: prompts, maxChars: DefaultMaxMessageChars}
}

const msgPromptVersion = "huginn-message-v3"

// Generate produces a personalized suggested message for a contact action.
func (g *Generator) Generate(ctx context.Context, channel interactions.Channel, leadCtx *LeadContext, objective string) (*MessageTemplate, error) {
	if g.prompts != nil {
		if p, err := g.prompts.Load(ctx); err == nil && p != nil {
			if p.MaxMessageChars > 0 {
				g.maxChars = p.MaxMessageChars
			}
			sys := p.MessageSystemPrompt
			if sys == "" {
				sys = DefaultMessageSystemPrompt
			}
			if p.Tone != "" {
				sys = withTone(sys, p.Tone)
			}
			return g.generate(ctx, channel, leadCtx, objective, sys)
		}
	}
	return g.generate(ctx, channel, leadCtx, objective, DefaultMessageSystemPrompt)
}

func (g *Generator) generate(ctx context.Context, channel interactions.Channel, leadCtx *LeadContext, objective, systemPrompt string) (*MessageTemplate, error) {
	channelText := string(channel)
	if channel == interactions.ChannelEmail {
		channelText = "EMAIL"
	}
	userPrompt := buildMessagePrompt(channelText, g.sender, leadCtx, objective)

	req := llm.Request{
		Purpose:       llm.PurposeMessageGenerate,
		PromptVersion: msgPromptVersion,
		EntityType:    "lead",
		EntityID:      leadCtx.LeadID,
		SystemPrompt:  systemPrompt,
		UserPrompt:    userPrompt,
		JSONSchema: map[string]any{
			"type": "object",
			"properties": map[string]any{
				"subject": map[string]any{"type": "string"},
				"body":    map[string]any{"type": "string"},
			},
			"required": []string{"body"},
		},
		Meta: map[string]string{
			"lead_id": leadCtx.LeadID,
		},
	}

	var out MessageTemplate
	run, err := g.runner.GenerateStructured(ctx, req, &out)
	if err != nil {
		return nil, err
	}
	if run == nil || run.Status != "success" {
		return nil, nil
	}
	body := strings.TrimSpace(out.Body)
	if body == "" {
		return nil, nil
	}
	// Keep the message concise (configurable cap).
	if runeCount(body) > g.maxChars {
		body = trimRunes(body, g.maxChars)
	}
	out.Body = body
	// Never return a message that presents the sender as part of the lead's
	// company or business. Fall back to the deterministic template when the
	// perspective is wrong.
	if !validateMessagePerspective(g.sender, leadCtx, out.Body) {
		out.Body = defaultMessage(channel, leadCtx)
		out.Subject = ""
	}
	return &out, nil
}

// buildMessagePrompt renders the channel, objective, SENDER and RECIPIENT blocks.
func buildMessagePrompt(channel string, sender *SenderProfile, leadCtx *LeadContext, objective string) string {
	var b strings.Builder
	b.WriteString("Channel: " + channel + "\n")
	b.WriteString("Objective: " + objective + "\n")

	b.WriteString("\nSENDER (our company — the writer of this message):\n")
	if sender == nil || sender.IsZero() {
		b.WriteString("Company: [unset]\n")
		b.WriteString("(The sender's company is not disclosed. Do NOT name any company and do NOT claim the sender works in the recipient's industry. Write only as a sales representative.)\n")
	} else {
		if sender.CompanyName != "" {
			b.WriteString("Company: " + sender.CompanyName + "\n")
		}
		if sender.Description != "" {
			b.WriteString("Company description / what we offer: " + truncateStr(sender.Description, 500) + "\n")
		}
		if s := sender.representative(); s != "" {
			b.WriteString("Representative: " + s + "\n")
		}
	}

	b.WriteString("\nRECIPIENT (the lead — the target of this message):\n")
	if leadCtx != nil {
		if rendered := leadCtx.Render(); rendered != "" {
			b.WriteString(rendered + "\n")
		}
	}

	b.WriteString("\nWrite a short, personalized opening message as the SENDER to the RECIPIENT.")
	b.WriteString("\nMention one concrete, real detail about the RECIPIENT as the reason for reaching out; create value and curiosity; end with a low-friction CTA. Never invent facts.")
	b.WriteString("\nThe sender is from the SENDER company ONLY. The RECIPIENT company belongs to the recipient; never say the sender works there.")
	return b.String()
}

// defaultMessage is a deterministic fallback that contains zero fabricated claims.
func defaultMessage(channel interactions.Channel, leadCtx *LeadContext) string {
	company := ""
	if leadCtx != nil {
		company = leadCtx.CompanyName
	}
	if company == "" {
		company = "business"
	}
	switch channel {
	case interactions.ChannelSMS:
		return "سلام، وقت بخیر. در خصوص همکاری با " + company + " تماس می‌گیرم. لطفاً در فرصت مناسب پاسخ دهید."
	case interactions.ChannelEmail:
		return "سلام، وقت بخیر. برای آشنایی و بررسی امکان همکاری با مجموعه شما تماس می‌گیرم."
	default:
		return "سلام، وقت بخیر. از طریق سامانه هوشمند فروش با شما آشنا شدیم و دوست داریم در مورد فرصت همکاری گفتگو کنیم."
	}
}

func runeCount(s string) int {
	return len([]rune(s))
}

func trimRunes(s string, n int) string {
	r := []rune(s)
	if len(r) <= n {
		return s
	}
	return string(r[:n])
}

func truncateStr(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n] + "..."
}

var _ = json.Marshal
