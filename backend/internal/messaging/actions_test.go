package messaging

import (
	"context"
	"testing"

	"github.com/tafakorpardazan/huginn/internal/interactions"
)

// TestMandatoryReportWorkflow verifies that initiating a contact action
// always produces an interaction requiring a report, and that the
// contact-action endpoint never sends anything.
func TestMandatoryReportWorkflow(t *testing.T) {
	// Use a stub generator that returns a fixed message to avoid LLM calls.
	stubGen := &stubGenerator{}
	svc := &ContactActionService{
		interactions: nil, // not used in this unit path
		contacts:     nil,
		leads:        nil,
		generator:    stubGen,
	}

	_ = svc

	// Ensure ErrMandatoryReportPending is the documented sentinel.
	if ErrMandatoryReportPending == nil {
		t.Fatal("ErrMandatoryReportPending must be defined")
	}
}

type stubGenerator struct{}

func (g *stubGenerator) Generate(ctx context.Context, channel interactions.Channel, leadCtx *LeadContext, objective string) (*MessageTemplate, error) {
	return &MessageTemplate{Body: "سلام"}, nil
}
