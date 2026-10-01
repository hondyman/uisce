package validationsdk

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

// Client is a typed HTTP client for Uisce's validation rule engine.
// Safe for concurrent use.
type Client struct {
	baseURL    string // e.g. https://uisce.internal
	httpClient *http.Client
	token      func(context.Context) (string, error) // bearer token provider
	tenantID   string
	retry      RetryPolicy
	breaker    *CircuitBreaker
}

type Option func(*Client)

func WithHTTPClient(hc *http.Client) Option { return func(c *Client) { c.httpClient = hc } }
func WithTokenProvider(f func(context.Context) (string, error)) Option {
	return func(c *Client) { c.token = f }
}
func WithTenantID(tenantID string) Option      { return func(c *Client) { c.tenantID = tenantID } }
func WithRetry(p RetryPolicy) Option           { return func(c *Client) { c.retry = p } }
func WithBreaker(b *CircuitBreaker) Option     { return func(c *Client) { c.breaker = b } }

func New(baseURL string, opts ...Option) *Client {
	c := &Client{
		baseURL:    strings.TrimRight(baseURL, "/"),
		httpClient: &http.Client{Timeout: 60 * time.Second},
		retry:      DefaultRetry(),
		breaker:    DefaultBreaker(),
	}
	for _, o := range opts {
		o(c)
	}
	return c
}

// ValidateRecord evaluates one ad-hoc record. Returns the typed result, or
// *ServerContextRequiredError when rules need context the payload lacks
// (caller decides to supply fields or a record id and re-validate).
func (c *Client) ValidateRecord(ctx context.Context, req EvaluateRecordRequest) (*EvaluateRecordResponse, error) {
	var out EvaluateRecordResponse
	err := c.do(ctx, http.MethodPost, "/api/validation-rule-nodes/evaluate-record", req, &out)
	if err != nil {
		return nil, err
	}
	return &out, nil
}

// ValidateBatch evaluates up to 10,000 records against one frozen rule
// snapshot. The result is index-aligned: Records[i] <-> req.Records[i];
// malformed/context-required records appear in Errors, never abort the batch.
func (c *Client) ValidateBatch(ctx context.Context, req EvaluateBatchRequest) (*EvaluateBatchResponse, error) {
	var out EvaluateBatchResponse
	err := c.do(ctx, http.MethodPost, "/api/validation-rule-nodes/evaluate-batch", req, &out)
	if err != nil {
		return nil, err
	}
	return &out, nil
}

// ValidBlocks is a convenience: true when the record violates at least one
// BLOCK-severity rule (the "would the write be blocked?" predicate).
func (r *EvaluateRecordResponse) ValidBlocks() bool {
	return r.Blocked
}

func (c *Client) do(ctx context.Context, method, path string, body any, out any) error {
	payload, err := json.Marshal(body)
	if err != nil {
		return fmt.Errorf("marshal request: %w", err)
	}
	return withRetry(ctx, c.retry, c.breaker, func() error {
		req, err := http.NewRequestWithContext(ctx, method, c.baseURL+path, bytes.NewReader(payload))
		if err != nil {
			return err
		}
		req.Header.Set("Content-Type", "application/json")
		if c.tenantID != "" {
			req.Header.Set("X-Tenant-ID", c.tenantID)
		}
		if c.token != nil {
			tok, terr := c.token(ctx)
			if terr != nil {
				return terr
			}
			req.Header.Set("Authorization", "Bearer "+tok)
		}
		resp, err := c.httpClient.Do(req)
		if err != nil {
			return err // transport error -> retryable
		}
		defer resp.Body.Close()
		raw, err := io.ReadAll(io.LimitReader(resp.Body, 64<<20))
		if err != nil {
			return err
		}
		if resp.StatusCode < 200 || resp.StatusCode > 299 {
			return &APIError{StatusCode: resp.StatusCode, Body: string(raw)}
		}
		// evaluate-record signals context-required via 200 + {"error": ...}.
		var probe struct {
			Error                string   `json:"error"`
			MissingContextFields []string `json:"missing_context_fields"`
			Hint                 string   `json:"hint"`
		}
		_ = json.Unmarshal(raw, &probe)
		if probe.Error == "ERR_SERVER_CONTEXT_REQUIRED" {
			return &ServerContextRequiredError{MissingFields: probe.MissingContextFields, Hint: probe.Hint}
		}
		return json.Unmarshal(raw, out)
	})
}
