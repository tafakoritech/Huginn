package imports

import (
	"encoding/json"

	"github.com/tafakorpardazan/huginn/internal/llm"
)

// parseSchema converts a JSON schema string into a Go value for the LLM request.
func parseSchema(s string) any {
	var v any
	_ = json.Unmarshal([]byte(s), &v)
	return v
}

var _ = llm.ExtractionSystemPrompt
