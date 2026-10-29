package httpapi

import (
	"context"
	"encoding/json"
	"net/http"

	"github.com/tafakorpardazan/huginn/internal/messaging"
	"github.com/tafakorpardazan/huginn/internal/scoring"
)

// settingsStore persists small JSON settings keyed by name. The table is
// created idempotently so the endpoints work without a separate migration run.
func ensureSettingsTable(ctx context.Context, h *dataHandler) error {
	_, err := h.s.DB.Pool().Exec(ctx, `
		CREATE TABLE IF NOT EXISTS app_settings (
			key TEXT PRIMARY KEY,
			value JSONB NOT NULL,
			updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
		)`)
	return err
}

func getSetting(ctx context.Context, h *dataHandler, key string) (map[string]any, bool, error) {
	var raw []byte
	err := h.s.DB.Pool().QueryRow(ctx, `SELECT value FROM app_settings WHERE key=$1`, key).Scan(&raw)
	if err != nil {
		return nil, false, nil // not found
	}
	out := map[string]any{}
	if err := json.Unmarshal(raw, &out); err != nil {
		return nil, false, err
	}
	return out, true, nil
}

func setSetting(ctx context.Context, h *dataHandler, key string, value map[string]any) error {
	raw, err := json.Marshal(value)
	if err != nil {
		return err
	}
	if _, err := h.s.DB.Pool().Exec(ctx, `
		INSERT INTO app_settings (key, value) VALUES ($1, $2)
		ON CONFLICT (key) DO UPDATE SET value=$2, updated_at=now()`, key, raw); err != nil {
		return err
	}
	return nil
}

// ---- Scoring settings ----

func (h *dataHandler) ScoringGet(w http.ResponseWriter, r *http.Request) {
	_ = ensureSettingsTable(r.Context(), h)
	weights := scoring.Weights()
	// key mapping to the dashboard's dimension names.
	order := []struct{ key, name string }{
		{"business_fit", "business_fit"},
		{"contactability", "contactability"},
		{"business_scale", "scale"},
		{"engagement", "engagement"},
		{"authority", "authority"},
		{"need", "need"},
		{"timing", "timing"},
	}
	saved, _, _ := getSetting(r.Context(), h, "scoring")
	dimensions := []map[string]any{}
	total := 0
	for _, o := range order {
		w, ok := weights[o.key]
		if !ok {
			continue
		}
		if v, ok := saved[o.name]; ok {
			if n, ok := v.(float64); ok {
				w = int(n)
			}
		}
		total += w
		dimensions = append(dimensions, map[string]any{
			"key": o.name, "name": o.name, "weight": w, "value": 0, "contribution": 0, "max": w,
		})
	}
	RespondOK(w, map[string]any{"dimensions": dimensions, "total_weight": total})
}

func (h *dataHandler) ScoringUpdate(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Weights map[string]int `json:"weights"`
	}
	if apiErr := DecodeJSON(r, &req); apiErr != nil {
		RespondError(w, http.StatusBadRequest, apiErr)
		return
	}
	_ = ensureSettingsTable(r.Context(), h)
	clean := map[string]any{}
	for k, v := range req.Weights {
		if v < 0 {
			v = 0
		}
		if v > 100 {
			v = 100
		}
		clean[k] = v
	}
	if err := setSetting(r.Context(), h, "scoring", clean); err != nil {
		internalErr(w, err)
		return
	}
	h.ScoringGet(w, r)
}

// ---- Qualification settings ----

func (h *dataHandler) QualificationGet(w http.ResponseWriter, r *http.Request) {
	_ = ensureSettingsTable(r.Context(), h)
	minScore := h.s.Config.Qualification.ScoreThreshold
	saved, _, _ := getSetting(r.Context(), h, "qualification")
	out := map[string]any{
		"minimum_score":                float64(minScore),
		"business_relevance_required":  true,
		"meaningful_contact_required":  true,
		"decision_maker_required":      false,
		"need_interest_required":       true,
		"next_action_required":         true,
	}
	for k, v := range saved {
		out[k] = v
	}
	RespondOK(w, out)
}

func (h *dataHandler) QualificationUpdate(w http.ResponseWriter, r *http.Request) {
	var req map[string]any
	if apiErr := DecodeJSON(r, &req); apiErr != nil {
		RespondError(w, http.StatusBadRequest, apiErr)
		return
	}
	_ = ensureSettingsTable(r.Context(), h)
	cur, _, _ := getSetting(r.Context(), h, "qualification")
	for k, v := range req {
		cur[k] = v
	}
	if err := setSetting(r.Context(), h, "qualification", cur); err != nil {
		internalErr(w, err)
		return
	}
	RespondOK(w, cur)
}

// ---- LLM settings (secrets never exposed) ----

func (h *dataHandler) LLMGet(w http.ResponseWriter, r *http.Request) {
	_ = ensureSettingsTable(r.Context(), h)
	saved, _, _ := getSetting(r.Context(), h, "llm")
	llm := h.s.Config.LLM
	provider := "deepseek"
	if llm.BaseURL != "" {
		provider = hostOf(llm.BaseURL)
	}
	out := map[string]any{
		"provider":                 provider,
		"model":                    llm.Model,
		"base_url":                 llm.BaseURL,
		"timeout_seconds":          float64(llm.Timeout.Seconds()),
		"prompt_version":           "current",
		"structured_output_enabled": true,
		"max_retries":              float64(llm.MaxRetries),
	}
	for k, v := range saved {
		out[k] = v
	}
	RespondOK(w, out)
}

func (h *dataHandler) LLMUpdate(w http.ResponseWriter, r *http.Request) {
	var req map[string]any
	if apiErr := DecodeJSON(r, &req); apiErr != nil {
		RespondError(w, http.StatusBadRequest, apiErr)
		return
	}
	// Never accept or store API keys through the dashboard.
	for k := range req {
		if k == "api_key" || k == "apikey" || k == "secret" {
			delete(req, k)
		}
	}
	_ = ensureSettingsTable(r.Context(), h)
	cur, _, _ := getSetting(r.Context(), h, "llm")
	for k, v := range req {
		cur[k] = v
	}
	if err := setSetting(r.Context(), h, "llm", cur); err != nil {
		internalErr(w, err)
		return
	}
	RespondOK(w, cur)
}

func (h *dataHandler) LLMTest(w http.ResponseWriter, r *http.Request) {
	llm := h.s.Config.LLM
	if llm.Model == "" {
		RespondOK(w, map[string]any{"ok": false, "message": "LLM model is not configured"})
		return
	}
	RespondOK(w, map[string]any{"ok": true, "message": "ok", "model": llm.Model})
}

// ---- Muninn settings ----

func (h *dataHandler) MuninnGet(w http.ResponseWriter, r *http.Request) {
	_ = ensureSettingsTable(r.Context(), h)
	saved, _, _ := getSetting(r.Context(), h, "muninn")
	m := h.s.Config.Muninn
	out := map[string]any{
		"enabled":          m.Enabled,
		"base_url":         m.BaseURL,
		"timeout_seconds":  float64(m.Timeout.Seconds()),
		"export_endpoint":  "/api/v1/leads",
	}
	for k, v := range saved {
		out[k] = v
	}
	RespondOK(w, out)
}

func (h *dataHandler) MuninnUpdate(w http.ResponseWriter, r *http.Request) {
	var req map[string]any
	if apiErr := DecodeJSON(r, &req); apiErr != nil {
		RespondError(w, http.StatusBadRequest, apiErr)
		return
	}
	_ = ensureSettingsTable(r.Context(), h)
	cur, _, _ := getSetting(r.Context(), h, "muninn")
	for k, v := range req {
		cur[k] = v
	}
	if err := setSetting(r.Context(), h, "muninn", cur); err != nil {
		internalErr(w, err)
		return
	}
	RespondOK(w, cur)
}

// ---- Messaging / sales-outreach prompt settings ----

// MessagingGet returns the configurable sales-outreach prompt strategy. The
// generator reads these settings per request, so the strategy can be improved
// without rewriting Android or the generator.
func (h *dataHandler) MessagingGet(w http.ResponseWriter, r *http.Request) {
	_ = ensureSettingsTable(r.Context(), h)
	store := messaging.NewPostgresPromptSettings(h.s.DB)
	settings, err := store.Load(r.Context())
	if err != nil {
		internalErr(w, err)
		return
	}
	RespondOK(w, settings)
}

// MessagingUpdate patches the sales-outreach prompt strategy.
func (h *dataHandler) MessagingUpdate(w http.ResponseWriter, r *http.Request) {
	_ = ensureSettingsTable(r.Context(), h)
	var req struct {
		MessageSystemPrompt      *string `json:"message_system_prompt"`
		CallScenarioSystemPrompt *string `json:"call_scenario_system_prompt"`
		MaxMessageChars          *int    `json:"max_message_chars"`
		Tone                     *string `json:"tone"`
	}
	if apiErr := DecodeJSON(r, &req); apiErr != nil {
		RespondError(w, http.StatusBadRequest, apiErr)
		return
	}
	store := messaging.NewPostgresPromptSettings(h.s.DB)
	cur, err := store.Load(r.Context())
	if err != nil {
		internalErr(w, err)
		return
	}
	if req.MessageSystemPrompt != nil {
		cur.MessageSystemPrompt = *req.MessageSystemPrompt
	}
	if req.CallScenarioSystemPrompt != nil {
		cur.CallScenarioSystemPrompt = *req.CallScenarioSystemPrompt
	}
	if req.MaxMessageChars != nil {
		cur.MaxMessageChars = *req.MaxMessageChars
	}
	if req.Tone != nil {
		cur.Tone = *req.Tone
	}
	if err := store.Save(r.Context(), *cur); err != nil {
		internalErr(w, err)
		return
	}
	RespondOK(w, cur)
}

// ---- System ----

func (h *dataHandler) SystemGet(w http.ResponseWriter, r *http.Request) {
	cfg := h.s.Config
	RespondOK(w, map[string]any{
		"app_name":          cfg.AppName,
		"version":           "0.1.0",
		"timezone":          cfg.Timezone,
		"max_upload_bytes":  float64(cfg.Storage.MaxUploadBytes),
		"storage_engine":    "MinIO",
		"queue":             "RabbitMQ",
		"worker_count":      2,
	})
}

func hostOf(url string) string {
	for _, prefix := range []string{"https://", "http://"} {
		if len(url) >= len(prefix) && url[:len(prefix)] == prefix {
			rest := url[len(prefix):]
			for i := 0; i < len(rest); i++ {
				if rest[i] == '/' || rest[i] == ':' {
					return rest[:i]
				}
			}
			return rest
		}
	}
	return url
}
