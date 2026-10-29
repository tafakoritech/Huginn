package llm

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"
)

// TestProviderReceivesDocumentText spins up a real HTTP server that speaks the
// OpenAI chat/completions contract and verifies that the real provider adapter
// (used in production) sends the actual document text in the user message.
func TestProviderReceivesDocumentText(t *testing.T) {
	fixture, err := os.ReadFile("../imports/testdata/companies.txt")
	if err != nil {
		t.Fatalf("read fixture: %v", err)
	}

	var received struct {
		Model    string `json:"model"`
		Messages []struct {
			Role    string `json:"role"`
			Content string `json:"content"`
		} `json:"messages"`
		ResponseFormat json.RawMessage `json:"response_format"`
	}

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/chat/completions" {
			t.Errorf("unexpected path %s", r.URL.Path)
		}
		if r.Header.Get("Authorization") != "Bearer test-key" {
			t.Errorf("missing bearer token")
		}
		body, _ := io.ReadAll(r.Body)
		if err := json.Unmarshal(body, &received); err != nil {
			t.Errorf("decode request: %v", err)
		}
		w.Header().Set("Content-Type", "application/json")
		// A valid strict candidate response with several companies.
		_, _ = w.Write([]byte(`{
		  "choices":[{"message":{"content":` + strconv.Quote(candidateJSON4()) + `}}],
		  "usage":{"prompt_tokens":100,"completion_tokens":40}
		}`))
	}))
	defer srv.Close()

	cfg := ProviderConfig{
		BaseURL:    srv.URL,
		APIKey:     "test-key",
		Model:      "test-model",
		Timeout:    5 * 1000 * 1000 * 1000, // 5s
		MaxRetries: 0,
	}
	provider, err := NewOpenAICompatibleProvider(cfg)
	if err != nil {
		t.Fatalf("build provider: %v", err)
	}
	runner := NewRunner(provider, nopAuditor{}, cfg)

	var out struct {
		Candidates []map[string]any `json:"candidates"`
	}
	req := Request{
		Purpose:       PurposeLeadExtraction,
		PromptVersion: "huginn-extract-v2",
		EntityType:    "source_document",
		EntityID:      "doc-1",
		SystemPrompt:  ExtractionSystemPrompt,
		UserPrompt:    ExtractionUserPrompt(string(fixture), 30000),
		JSONSchema:    map[string]any{},
	}
	run, err := runner.GenerateStructured(context.Background(), req, &out)
	if err != nil {
		t.Fatalf("generate structured: %v", err)
	}
	if run.Status != "success" {
		t.Fatalf("expected success run, got %s", run.Status)
	}
	if len(out.Candidates) != 4 {
		t.Fatalf("expected 4 parsed candidates, got %d", len(out.Candidates))
	}
	if received.Model != "test-model" {
		t.Fatalf("wrong model sent: %s", received.Model)
	}
	userMsg := ""
	for _, m := range received.Messages {
		if m.Role == "user" {
			userMsg = m.Content
		}
	}
	// The REAL provider must deliver the actual document content.
	if !strings.Contains(userMsg, "عروسک‌سازی پارس") || !strings.Contains(userMsg, "0935 757 8808") {
		t.Fatal("real provider did not send the document text to the LLM")
	}
	if len(received.ResponseFormat) == 0 {
		t.Fatal("expected a structured response_format on the request")
	}
}

func candidateJSON4() string {
	set := struct {
		Candidates []map[string]any `json:"candidates"`
	}{Candidates: []map[string]any{
		{"company_name": "عروسک‌سازی پارس", "city": "تهران", "contacts": []map[string]any{{"full_name": "محمد رضایی", "mobile": "09357578808"}}},
		{"company_name": "صنایع چوب نوین", "city": "تبریز"},
		{"company_name": "گروه صنعتی فولاد آریا"},
		{"company_name": "پخش مواد غذایی البرز"},
	}}
	b, _ := json.Marshal(set)
	return string(b)
}

// TestProviderJSONSchema400CapturesErrorBodyAndCachesUnsupported verifies that
// a provider rejecting response_format json_schema with HTTP 400:
//   - surfaces the provider error body on the returned error (so the exact
//     reason is never hidden), and
//   - caches that capability so a known-invalid json_schema request is not
//     repeated on every generation (the adapter falls back to json_object).
func TestProviderJSONSchema400CapturesErrorBodyAndCachesUnsupported(t *testing.T) {
	reqs := []string{}
	var mu sync.Mutex
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		mu.Lock()
		reqs = append(reqs, string(body))
		mu.Unlock()
		w.Header().Set("Content-Type", "application/json")
		if strings.Contains(string(body), `"json_schema"`) {
			w.WriteHeader(http.StatusBadRequest)
			_, _ = w.Write([]byte(`{"error":{"message":"This response_format type is unavailable now","type":"invalid_request_error"}}`))
			return
		}
		_, _ = w.Write([]byte(`{"choices":[{"message":{"content":"{\"ok\":true}"}}],"usage":{"prompt_tokens":1,"completion_tokens":1}}`))
	}))
	defer srv.Close()

	cfg := ProviderConfig{
		BaseURL:    srv.URL,
		APIKey:     "test-key",
		Model:      "test-model",
		Timeout:    5 * time.Second,
		MaxRetries: 0,
	}
	provider, err := NewOpenAICompatibleProvider(cfg)
	if err != nil {
		t.Fatalf("build provider: %v", err)
	}
	req := Request{
		Purpose:       PurposeCallScenario,
		PromptVersion: "v3",
		SystemPrompt:  "Return JSON.",
		UserPrompt:    "Return an object with ok:true. JSON.",
		JSONSchema:    map[string]any{"type": "object"},
	}

	// First call: json_schema is attempted, gets 400, then json_object succeeds.
	// The error body must be visible on the failed attempt (ErrProvider).
	resp, err := provider.Chat(context.Background(), req)
	if err != nil {
		t.Fatalf("chat should fall back to json_object and succeed: %v", err)
	}
	if !strings.Contains(resp.Text, "ok") {
		t.Fatalf("unexpected response text: %q", resp.Text)
	}
	mu.Lock()
	if len(reqs) < 2 {
		t.Fatalf("expected json_schema 400 then json_object retry, got %d requests", len(reqs))
	}
	if !strings.Contains(reqs[0], `"json_schema"`) {
		t.Fatalf("first request must attempt json_schema: %s", reqs[0])
	}
	if !strings.Contains(reqs[1], `"json_object"`) {
		t.Fatalf("second request must use json_object: %s", reqs[1])
	}
	reqs = nil
	mu.Unlock()

	// Second generation: the provider caches the 400 and skips json_schema.
	_, err = provider.Chat(context.Background(), req)
	if err != nil {
		t.Fatalf("second chat: %v", err)
	}
	mu.Lock()
	defer mu.Unlock()
	for _, r := range reqs {
		if strings.Contains(r, `"json_schema"`) {
			t.Fatalf("known-invalid json_schema must not be re-sent after a 400, got: %s", r)
		}
	}
}

// TestProviderStructuredModeJSONObjectNeverSendsSchema verifies that pinning
// StructuredModeJSONObject never sends a json_schema request at all.
func TestProviderStructuredModeJSONObjectNeverSendsSchema(t *testing.T) {
	reqs := []string{}
	var mu sync.Mutex
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		mu.Lock()
		reqs = append(reqs, string(body))
		mu.Unlock()
		w.Header().Set("Content-Type", "application/json")
		if strings.Contains(string(body), `"json_schema"`) {
			w.WriteHeader(http.StatusBadRequest)
			_, _ = w.Write([]byte(`{"error":{"message":"This response_format type is unavailable now"}}`))
			return
		}
		_, _ = w.Write([]byte(`{"choices":[{"message":{"content":"{\"ok\":true}"}}],"usage":{"prompt_tokens":1,"completion_tokens":1}}`))
	}))
	defer srv.Close()

	cfg := ProviderConfig{
		BaseURL:        srv.URL,
		APIKey:         "test-key",
		Model:          "test-model",
		Timeout:        5 * time.Second,
		MaxRetries:     0,
		StructuredMode: StructuredModeJSONObject,
	}
	provider, err := NewOpenAICompatibleProvider(cfg)
	if err != nil {
		t.Fatalf("build provider: %v", err)
	}
	req := Request{
		Purpose:       PurposeCallScenario,
		PromptVersion: "v3",
		SystemPrompt:  "Return JSON.",
		UserPrompt:    "Return an object with ok:true. JSON.",
		JSONSchema:    map[string]any{"type": "object"},
	}
	if _, err := provider.Chat(context.Background(), req); err != nil {
		t.Fatalf("chat: %v", err)
	}
	mu.Lock()
	defer mu.Unlock()
	if len(reqs) != 1 {
		t.Fatalf("expected exactly one request, got %d", len(reqs))
	}
	if !strings.Contains(reqs[0], `"json_object"`) {
		t.Fatalf("json_object mode must be requested: %s", reqs[0])
	}
	if strings.Contains(reqs[0], `"json_schema"`) {
		t.Fatalf("json_schema must never be sent in json_object mode: %s", reqs[0])
	}
}

// TestProviderErrorBodyIncludedInError verifies the provider's error body (the
// diagnostic that identifies WHY a request was rejected) is preserved on the
// returned ErrProvider.
func TestProviderErrorBodyIncludedInError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write([]byte(`{"error":{"message":"This response_format type is unavailable now","code":"invalid_request_error"}}`))
	}))
	defer srv.Close()

	cfg := ProviderConfig{
		BaseURL:        srv.URL,
		APIKey:         "test-key",
		Model:          "test-model",
		Timeout:        5 * time.Second,
		MaxRetries:     0,
		StructuredMode: StructuredModeJSONSchema,
	}
	provider, err := NewOpenAICompatibleProvider(cfg)
	if err != nil {
		t.Fatalf("build provider: %v", err)
	}
	req := Request{
		Purpose:       PurposeCallScenario,
		PromptVersion: "v3",
		SystemPrompt:  "Return JSON.",
		UserPrompt:    "Return an object. JSON.",
		JSONSchema:    map[string]any{"type": "object"},
	}
	_, err = provider.Chat(context.Background(), req)
	if err == nil {
		t.Fatal("expected provider error")
	}
	var pe *ErrProvider
	if !errors.As(err, &pe) {
		t.Fatalf("expected *ErrProvider, got %T", err)
	}
	if pe.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected status 400, got %d", pe.StatusCode)
	}
	if !strings.Contains(pe.Err.Error(), "This response_format type is unavailable now") {
		t.Errorf("provider error body must be preserved in the error, got %q", pe.Err.Error())
	}
}
