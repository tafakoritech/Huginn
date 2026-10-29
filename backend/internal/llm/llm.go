package llm

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"
)

// Purpose describes why an LLM call is being made.
type Purpose string

const (
	PurposeLeadExtraction  Purpose = "lead.extract"
	PurposeLeadDedupe      Purpose = "lead.dedupe"
	PurposeLeadTagging     Purpose = "lead.tag"
	PurposeMessageGenerate Purpose = "message.generate"
	PurposeCallScenario    Purpose = "call.scenario"
	PurposeBatchAnalyze    Purpose = "batch.analyze"
	PurposeRepair          Purpose = "repair"
	PurposeClassification  Purpose = "classification"
)

// Run is a structured record of a single LLM invocation.
type Run struct {
	ID               string    `json:"id"`
	Purpose          Purpose   `json:"purpose"`
	Model            string    `json:"model"`
	PromptVersion    string    `json:"prompt_version"`
	EntityType       string    `json:"entity_type"`
	EntityID         string    `json:"entity_id"`
	RequestHash      string    `json:"request_hash"`
	StructuredOutput []byte    `json:"structured_output,omitempty"`
	LatencyMs        int64     `json:"latency_ms"`
	InputTokens      int       `json:"input_tokens"`
	OutputTokens     int       `json:"output_tokens"`
	Status           string    `json:"status"`
	Error            string    `json:"error,omitempty"`
	CreatedAt        time.Time `json:"created_at"`
}

// Request is a single LLM call input.
type Request struct {
	Purpose       Purpose
	PromptVersion string
	EntityType    string
	EntityID      string
	SystemPrompt  string
	UserPrompt    string
	JSONSchema    any // optional strict schema hint
	MaxTokens     int // 0 = provider default
	Temperature   *float64
	// Attempt is the 1-based retry attempt within the runner. Set by the runner
	// before each provider call so request logs show which attempt failed.
	Attempt int
	// Meta carries optional tracing context (e.g. document_id, batch_id) that is
	// attached to request logs. Never contains secrets.
	Meta map[string]string
}

// Structured output response_format modes supported by the provider adapter.
const (
	// StructuredModeAuto probes json_schema first, then json_object, and caches
	// a 400 so a rejected mode is not retried for the provider's lifetime.
	StructuredModeAuto = "auto"
	// StructuredModeJSONObject always requests json_object. Required for
	// providers/models that reject json_schema outright (e.g. DeepSeek
	// deepseek-v4-flash returns HTTP 400 "This response_format type is
	// unavailable now"). The prompt carries the exact JSON contract and the
	// runner validates the parsed structure, so strict mode is not lost.
	StructuredModeJSONObject = "json_object"
	// StructuredModeJSONSchema always requests json_schema. Only use it with
	// providers that support the OpenAI json_schema response_format.
	StructuredModeJSONSchema = "json_schema"
)

// ProviderConfig configures an LLM provider client.
type ProviderConfig struct {
	BaseURL        string
	APIKey         string
	Model          string
	Timeout        time.Duration
	MaxRetries     int
	RetryBaseDelay time.Duration
	// StructuredMode selects the response_format strategy for structured
	// outputs. Empty means StructuredModeAuto.
	StructuredMode string
	// SOCKS5Proxy is an optional "host:port" SOCKS5 proxy used ONLY for the LLM
	// transport. Empty means a direct connection. Never honored via the generic
	// HTTP(S)_PROXY environment variables.
	SOCKS5Proxy string
}

// Provider is the transport-level interface for talking to an LLM service.
type Provider interface {
	Chat(ctx context.Context, req Request) (*Response, error)
}

// Response is a raw provider response.
type Response struct {
	Text         string
	InputTokens  int
	OutputTokens int
}

// Runner wraps a Provider with auditing and retry semantics.
type Runner interface {
	GenerateStructured(ctx context.Context, req Request, out any) (*Run, error)
	GenerateText(ctx context.Context, req Request) (*Run, error)
}

// Auditor persists LLM run audit records.
type Auditor interface {
	Record(ctx context.Context, run *Run) error
}

// ErrValidation is returned when model output fails schema validation.
type ErrValidation struct {
	Purpose Purpose
	Err     error
}

func (e *ErrValidation) Error() string {
	return fmt.Sprintf("llm validation failed for %s: %v", e.Purpose, e.Err)
}

// ErrNotConfigured indicates the provider is missing required configuration
// (e.g. LLM_BASE_URL is empty). Retrying the same job cannot succeed, so it is
// a permanent failure.
var ErrNotConfigured = errors.New("llm provider is not configured")

// ErrProvider is returned when the upstream provider fails. StatusCode carries
// the upstream HTTP status (0 for transport-level failures) so queue consumers
// can classify the failure as transient vs permanent without parsing strings.
type ErrProvider struct {
	Purpose    Purpose
	StatusCode int
	Err        error
}

func (e *ErrProvider) Error() string {
	return fmt.Sprintf("llm provider error for %s: %v", e.Purpose, e.Err)
}

// Unwrap exposes the underlying error for errors.Is/errors.As.
func (e *ErrProvider) Unwrap() error { return e.Err }

// HTTPStatusCode satisfies queue.HTTPStatusCarrier.
func (e *ErrProvider) HTTPStatusCode() int { return e.StatusCode }

// json is re-exported here for internal use by provider implementations.
func marshalJSON(v any) ([]byte, error) { return json.Marshal(v) }
