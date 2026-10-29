package documents

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"

	"github.com/tafakorpardazan/huginn/internal/config"
	"github.com/tafakorpardazan/huginn/internal/storage"
)

// Extractor extracts text from source documents via Apache Tika.
type Extractor struct {
	baseURL string
	client  *http.Client
	store   *storage.Store
}

// ExtractionResult is the normalized text output.
type ExtractionResult struct {
	Text string `json:"text"`
}

func NewExtractor(cfg config.TikaConfig, store *storage.Store) *Extractor {
	return &Extractor{
		baseURL: strings.TrimRight(cfg.BaseURL, "/"),
		client:  &http.Client{},
		store:   store,
	}
}

// Extract fetches an object from MinIO and runs it through Tika.
func (e *Extractor) Extract(ctx context.Context, objectKey string) (*ExtractionResult, error) {
	reader, _, err := e.store.Get(ctx, objectKey)
	if err != nil {
		return nil, fmt.Errorf("read object: %w", err)
	}
	defer reader.Close()

	req, err := http.NewRequestWithContext(ctx, http.MethodPut, e.baseURL+"/tika", reader)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Accept", "text/plain")

	resp, err := e.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("tika request: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 1024))
		return nil, fmt.Errorf("tika status %d: %s", resp.StatusCode, truncate(string(body), 300))
	}
	body, err := io.ReadAll(io.LimitReader(resp.Body, 8<<20))
	if err != nil {
		return nil, err
	}
	return &ExtractionResult{Text: string(body)}, nil
}

// ParseContentType wraps Tika's content-type detection (metadata endpoint).
func (e *Extractor) ParseContentType(ctx context.Context, r io.Reader) (string, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodPut, e.baseURL+"/detect/stream", r)
	if err != nil {
		return "", err
	}
	resp, err := e.client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(io.LimitReader(resp.Body, 256))
	return strings.TrimSpace(string(body)), nil
}

// PresignedURL builds a temporary URL for the stored document (used for OCR configs).
func (e *Extractor) PresignedURL(ctx context.Context, objectKey string) (string, error) {
	return e.store.PresignedGetURL(ctx, objectKey)
}

var _ = url.Values{}

func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n] + "..."
}
