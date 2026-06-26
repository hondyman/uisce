package hasura

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"calendar-service/internal/cache"

	graphql "github.com/hasura/go-graphql-client"
)

// Client represents a Hasura GraphQL client
type Client struct {
	*graphql.Client
	url        string
	httpClient *http.Client
	queryCache *cache.QueryCache
}

// NewClient creates a new Hasura GraphQL client
func NewClient(endpoint string, adminSecret string) *Client {
	httpClient := &http.Client{}

	if adminSecret != "" {
		httpClient = &http.Client{
			Transport: &authedTransport{
				wrapped:     http.DefaultTransport,
				adminSecret: adminSecret,
			},
		}
	}

	client := graphql.NewClient(endpoint, httpClient)
	return &Client{
		Client:     client,
		url:        endpoint,
		httpClient: httpClient,
	}
}

// SetQueryCache injects the query cache layer
func (c *Client) SetQueryCache(qc *cache.QueryCache) {
	c.queryCache = qc
}

// QueryRaw executes a raw GraphQL query
func (c *Client) QueryRaw(ctx context.Context, query string, variables map[string]interface{}, response interface{}) error {
	reqBody := map[string]interface{}{
		"query":     query,
		"variables": variables,
	}
	jsonBody, err := json.Marshal(reqBody)
	if err != nil {
		return fmt.Errorf("failed to marshal query: %w", err)
	}

	var cacheKey string
	if c.queryCache != nil && !strings.Contains(strings.ToLower(query), "mutation") {
		hash := sha256.Sum256(jsonBody)
		cacheKey = hex.EncodeToString(hash[:])
		found, err := c.queryCache.Get(ctx, cacheKey, response)
		if found && err == nil {
			return nil
		}
	}

	req, err := http.NewRequestWithContext(ctx, "POST", c.url, bytes.NewBuffer(jsonBody))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	// If using authedTransport, it handles headers. otherwise we might need to add them?
	// The httpClient passed to graphql.NewClient has the transport.
	// So c.httpClient should work.

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("hasura error: status %d", resp.StatusCode)
	}

	// Hasura returns {"data": ... , "errors": ...}
	// We should unmarshal "data" into response?
	// Or just unmarshal the whole thing?
	// The caller expects `response` to be populated matching the query shape.
	// Usually response struct has `json:"data"`? No, usually fields match query.
	// hasura-go unmarshals "data".

	var graphQLResponse struct {
		Data   json.RawMessage `json:"data"`
		Errors []interface{}   `json:"errors"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&graphQLResponse); err != nil {
		return fmt.Errorf("failed to decode response: %w", err)
	}

	if len(graphQLResponse.Errors) > 0 {
		return fmt.Errorf("graphql errors: %v", graphQLResponse.Errors)
	}

	if err := json.Unmarshal(graphQLResponse.Data, response); err != nil {
		return fmt.Errorf("failed to unmarshal data: %w", err)
	}

	if c.queryCache != nil && cacheKey != "" {
		_ = c.queryCache.Set(ctx, cacheKey, response)
	}

	return nil
}

// Mutate executes a raw GraphQL mutation
func (c *Client) Mutate(ctx context.Context, mutation string, variables map[string]interface{}, response interface{}) error {
	if c.queryCache != nil {
		// A generic, coarse way to invalidate cached queries on mutation
		_ = c.queryCache.Invalidate(ctx, "")
	}
	return c.QueryRaw(ctx, mutation, variables, response)
}

type authedTransport struct {
	wrapped     http.RoundTripper
	adminSecret string
}

func (t *authedTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	req.Header.Set("X-Hasura-Admin-Secret", t.adminSecret)
	return t.wrapped.RoundTrip(req)
}
