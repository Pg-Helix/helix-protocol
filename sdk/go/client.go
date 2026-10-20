// Package sdk is the minimal Go client for the PgHelix Phase 1 REST contract
// (openapi/openapi.yaml). It wraps net/http directly: no HTTP framework,
// since context deadlines/cancellation, typed errors, pagination and
// idempotency are all a few lines of stdlib plus the generated types.
package sdk

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strconv"

	protocol "github.com/Pg-Helix/helix-protocol/gen/go"
)

// APIError is returned for any non-2xx response. Code matches errors/codes.md
// (request-level codes and RFC-0003 publish rejection codes); callers switch
// on Code, not on Message, which may change.
type APIError struct {
	Code    string
	Message string
	TraceID string
}

func (e *APIError) Error() string {
	return fmt.Sprintf("%s: %s (trace %s)", e.Code, e.Message, e.TraceID)
}

// Client is the PgHelix REST client. The zero value is not usable; use NewClient.
type Client struct {
	baseURL    string
	httpClient *http.Client
}

// Option configures a Client.
type Option func(*Client)

// WithHTTPClient overrides the default http.Client (e.g. for custom timeouts
// or transport-level TLS config). The caller still controls per-call
// deadlines via the context passed to each method.
func WithHTTPClient(hc *http.Client) Option {
	return func(c *Client) { c.httpClient = hc }
}

// NewClient builds a Client against baseURL (e.g. "http://localhost:8080/v1").
func NewClient(baseURL string, opts ...Option) *Client {
	c := &Client{baseURL: baseURL, httpClient: http.DefaultClient}
	for _, opt := range opts {
		opt(c)
	}
	return c
}

func (c *Client) do(ctx context.Context, method, path string, query url.Values, headers map[string]string, body, out any) error {
	var reqBody *bytes.Reader
	if body != nil {
		b, err := json.Marshal(body)
		if err != nil {
			return fmt.Errorf("encode request: %w", err)
		}
		reqBody = bytes.NewReader(b)
	} else {
		reqBody = bytes.NewReader(nil)
	}

	u := c.baseURL + path
	if len(query) > 0 {
		u += "?" + query.Encode()
	}

	req, err := http.NewRequestWithContext(ctx, method, u, reqBody)
	if err != nil {
		return fmt.Errorf("build request: %w", err)
	}
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	for k, v := range headers {
		req.Header.Set(k, v)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		// Context cancellation/deadline surfaces here via ctx.Err(), unwrapped.
		return fmt.Errorf("do request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 300 {
		var apiErr protocol.ErrorResponse
		if decErr := json.NewDecoder(resp.Body).Decode(&apiErr); decErr != nil {
			return &APIError{Code: "internal", Message: fmt.Sprintf("http %d, undecodable error body", resp.StatusCode)}
		}
		return &APIError{Code: apiErr.Code, Message: apiErr.Message, TraceID: apiErr.TraceId}
	}

	if out == nil {
		return nil
	}
	if err := json.NewDecoder(resp.Body).Decode(out); err != nil {
		return fmt.Errorf("decode response: %w", err)
	}
	return nil
}

// CollectionPage is one cursor-paginated page of collections.
type CollectionPage struct {
	Collections []protocol.Collection `json:"collections"`
	NextCursor  string                `json:"next_cursor,omitempty"`
}

// ListCollections returns one page. Pass the previous page's NextCursor to
// continue; an empty NextCursor means there is no further page.
func (c *Client) ListCollections(ctx context.Context, cursor string, limit int) (*CollectionPage, error) {
	q := url.Values{}
	if cursor != "" {
		q.Set("cursor", cursor)
	}
	if limit > 0 {
		q.Set("limit", strconv.Itoa(limit))
	}
	var page CollectionPage
	if err := c.do(ctx, http.MethodGet, "/collections", q, nil, nil, &page); err != nil {
		return nil, err
	}
	return &page, nil
}

// CreateCollection creates a collection. idempotencyKey must be stable across
// retries of the same logical request (e.g. a UUID generated once by the
// caller and reused on retry) — the server treats repeated keys as the same
// operation rather than creating duplicates.
func (c *Client) CreateCollection(ctx context.Context, tenant, name, idempotencyKey string) (*protocol.Collection, error) {
	body := protocol.CreateCollectionJSONBody{Tenant: tenant, Name: name}
	headers := map[string]string{"Idempotency-Key": idempotencyKey}
	var out protocol.Collection
	if err := c.do(ctx, http.MethodPost, "/collections", nil, headers, body, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

// GetIngestStatus reports the RFC-0004 chunk-lifecycle status for a collection.
func (c *Client) GetIngestStatus(ctx context.Context, collectionID string) (*protocol.IngestStatus, error) {
	var out protocol.IngestStatus
	path := "/collections/" + url.PathEscape(collectionID) + "/ingest-status"
	if err := c.do(ctx, http.MethodGet, path, nil, nil, nil, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

// SearchResponse wraps one hybrid-retrieval result set.
type SearchResponse struct {
	Results []protocol.SearchResult `json:"results"`
}

// Search runs ACL-filtered hybrid (FTS + dense) retrieval (RFC-0004).
func (c *Client) Search(ctx context.Context, req protocol.SearchJSONBody) (*SearchResponse, error) {
	var out SearchResponse
	if err := c.do(ctx, http.MethodPost, "/search", nil, nil, req, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

// GetAnswer returns a cited answer. Citations are references re-checked
// against current ACLs at render time, not a copy of the underlying content.
func (c *Client) GetAnswer(ctx context.Context, req protocol.GetAnswerJSONBody) (*protocol.AnswerResponse, error) {
	var out protocol.AnswerResponse
	if err := c.do(ctx, http.MethodPost, "/answers", nil, nil, req, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

// ModelPage is one cursor-paginated page of the model registry.
type ModelPage struct {
	Models     []protocol.Model `json:"models"`
	NextCursor string           `json:"next_cursor,omitempty"`
}

// ListModels returns one page of the model registry (RFC-0003 target_model identities).
func (c *Client) ListModels(ctx context.Context, cursor string, limit int) (*ModelPage, error) {
	q := url.Values{}
	if cursor != "" {
		q.Set("cursor", cursor)
	}
	if limit > 0 {
		q.Set("limit", strconv.Itoa(limit))
	}
	var page ModelPage
	if err := c.do(ctx, http.MethodGet, "/models", q, nil, nil, &page); err != nil {
		return nil, err
	}
	return &page, nil
}
