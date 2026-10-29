package muninn

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/tafakorpardazan/huginn/internal/config"
)

type httpClient struct {
	client  *http.Client
	baseURL string
	apiKey  string
}

// NewHTTPExporter builds an Exporter that posts to Muninn's API.
func NewHTTPExporter(cfg config.MuninnConfig) Exporter {
	return &HTTPExporter{
		cfg: cfg,
		client: &httpClient{
			client:  &http.Client{Timeout: cfg.Timeout},
			baseURL: cfg.BaseURL,
			apiKey:  cfg.APIKey,
		},
	}
}

// ExportQualifiedLead POSTs the qualified lead to Muninn.
func (e *HTTPExporter) ExportQualifiedLead(ctx context.Context, payload *ExportPayload) (string, error) {
	if e.cfg.BaseURL == "" {
		return "", fmt.Errorf("muninn exporter: MUNINN_BASE_URL not configured")
	}
	body, err := json.Marshal(payload)
	if err != nil {
		return "", err
	}
	url := trimSlash(e.cfg.BaseURL) + "/api/v1/leads"
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/json")
	if e.cfg.APIKey != "" {
		req.Header.Set("Authorization", "Bearer "+e.cfg.APIKey)
	}

	resp, err := e.client.client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	respBody, _ := io.ReadAll(io.LimitReader(resp.Body, 1<<20))

	if resp.StatusCode != http.StatusCreated && resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("muninn exporter: status %d: %s", resp.StatusCode, truncate(string(respBody), 300))
	}

	// Try to extract muninn lead id
	var out struct {
		ID string `json:"id"`
	}
	_ = json.Unmarshal(respBody, &out)
	if out.ID != "" {
		return out.ID, nil
	}
	return "", nil
}

func (s *Service) claimExport(ctx context.Context, exportID string) (bool, error) {
	return s.repo.MarkInProgress(ctx, exportID)
}

func (s *Service) processClaimed(ctx context.Context, exportID string) error {
	// Load the export + payload
	var payload ExportPayload
	// fetch from DB via repo; we store payload in crm_exports.payload
	row := s.repo.db.Pool().QueryRow(ctx, `
		SELECT e.id, e.huginn_lead_id, e.muninn_lead_id, e.status, e.attempt_count, e.payload
		FROM crm_exports e WHERE e.id=$1`, exportID)
	var id, huginnLeadID string
	var muninnLeadID *string
	var status string
	var attemptCount int
	var payloadJSON []byte
	if err := row.Scan(&id, &huginnLeadID, &muninnLeadID, &status, &attemptCount, &payloadJSON); err != nil {
		return err
	}
	if status == "SUCCEEDED" {
		return nil // idempotent no-op
	}
	_ = json.Unmarshal(payloadJSON, &payload)

	muninnID, err := s.exporter.ExportQualifiedLead(ctx, &payload)
	if err != nil {
		_ = s.repo.MarkFailed(ctx, exportID, err.Error())
		return err
	}
	if err := s.repo.MarkSucceeded(ctx, exportID, muninnID); err != nil {
		return err
	}
	return nil
}

func trimSlash(s string) string {
	if len(s) > 0 && s[len(s)-1] == '/' {
		return s[:len(s)-1]
	}
	return s
}

func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n] + "..."
}

var _ = time.Now
