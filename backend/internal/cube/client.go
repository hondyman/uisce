package cube

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/google/uuid"
)

// Client wraps Cube.js API interactions with tenant-aware headers
type Client struct {
	baseURL    string
	apiSecret  string
	httpClient *http.Client
}

// NewClient creates a Cube.js client with tenant isolation support
func NewClient(baseURL, apiSecret string) *Client {
	return &Client{
		baseURL:   baseURL,
		apiSecret: apiSecret,
		httpClient: &http.Client{
			Timeout: 60 * time.Second,
		},
	}
}

// Query represents a Cube.js query with tenant context
type Query struct {
	Measures       []string          `json:"measures,omitempty"`
	Dimensions     []string          `json:"dimensions,omitempty"`
	Filters        []Filter          `json:"filters,omitempty"`
	TimeDimensions []TimeDimension   `json:"timeDimensions,omitempty"`
	Order          map[string]string `json:"order,omitempty"`
	Limit          int               `json:"limit,omitempty"`
	Offset         int               `json:"offset,omitempty"`
	Segments       []string          `json:"segments,omitempty"`
	Timezone       string            `json:"timezone,omitempty"`
	RenewQuery     bool              `json:"renewQuery,omitempty"`
}

// Filter represents a Cube.js filter
type Filter struct {
	Member   string   `json:"member"`
	Operator string   `json:"operator"`
	Values   []string `json:"values,omitempty"`
}

// TimeDimension represents a time dimension with granularity
type TimeDimension struct {
	Dimension   string   `json:"dimension"`
	DateRange   []string `json:"dateRange,omitempty"`
	Granularity string   `json:"granularity,omitempty"`
}

// QueryResult represents the Cube.js query response
type QueryResult struct {
	Data       []map[string]interface{} `json:"data"`
	Annotation *Annotation              `json:"annotation,omitempty"`
	Query      *Query                   `json:"query,omitempty"`
}

// Annotation provides metadata about query results
type Annotation struct {
	Measures       map[string]MemberMeta `json:"measures"`
	Dimensions     map[string]MemberMeta `json:"dimensions"`
	TimeDimensions map[string]MemberMeta `json:"timeDimensions"`
}

// MemberMeta describes a measure or dimension
type MemberMeta struct {
	Title        string   `json:"title"`
	ShortTitle   string   `json:"shortTitle"`
	Type         string   `json:"type"`
	Format       string   `json:"format,omitempty"`
	DrillMembers []string `json:"drillMembers,omitempty"`
}

// TenantContext carries tenant isolation headers
type TenantContext struct {
	TenantID     uuid.UUID
	DatasourceID uuid.UUID
	UserID       string
}

// ExecuteQuery runs a query against Cube.js with tenant isolation
func (c *Client) ExecuteQuery(ctx context.Context, query *Query, tenantCtx TenantContext) (*QueryResult, error) {
	// Prepare request
	body, err := json.Marshal(map[string]interface{}{
		"query": query,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to marshal query: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", fmt.Sprintf("%s/cubejs-api/v1/load", c.baseURL), bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	// Add tenant headers (mandatory for security context)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", c.apiSecret)
	req.Header.Set("X-Tenant-ID", tenantCtx.TenantID.String())
	req.Header.Set("X-Tenant-Datasource-ID", tenantCtx.DatasourceID.String())
	if tenantCtx.UserID != "" {
		req.Header.Set("X-User-ID", tenantCtx.UserID)
	}

	// Execute request
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		bodyBytes, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("cube.js error (status %d): %s", resp.StatusCode, string(bodyBytes))
	}

	// Parse response
	var result struct {
		Data       []map[string]interface{} `json:"data"`
		Annotation *Annotation              `json:"annotation"`
		Query      *Query                   `json:"query"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return &QueryResult{
		Data:       result.Data,
		Annotation: result.Annotation,
		Query:      result.Query,
	}, nil
}

// GetMeta retrieves metadata about available cubes for a tenant
func (c *Client) GetMeta(ctx context.Context, tenantCtx TenantContext) (map[string]interface{}, error) {
	req, err := http.NewRequestWithContext(ctx, "GET", fmt.Sprintf("%s/cubejs-api/v1/meta", c.baseURL), nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create meta request: %w", err)
	}

	req.Header.Set("Authorization", c.apiSecret)
	req.Header.Set("X-Tenant-ID", tenantCtx.TenantID.String())
	req.Header.Set("X-Tenant-Datasource-ID", tenantCtx.DatasourceID.String())

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("meta request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		bodyBytes, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("meta error (status %d): %s", resp.StatusCode, string(bodyBytes))
	}

	var meta map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&meta); err != nil {
		return nil, fmt.Errorf("failed to decode meta: %w", err)
	}

	return meta, nil
}

// PreAggregationStatus checks the status of pre-aggregations
func (c *Client) PreAggregationStatus(ctx context.Context, tenantCtx TenantContext) ([]map[string]interface{}, error) {
	req, err := http.NewRequestWithContext(ctx, "GET", fmt.Sprintf("%s/cubejs-api/v1/pre-aggregations", c.baseURL), nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create pre-agg request: %w", err)
	}

	req.Header.Set("Authorization", c.apiSecret)
	req.Header.Set("X-Tenant-ID", tenantCtx.TenantID.String())
	req.Header.Set("X-Tenant-Datasource-ID", tenantCtx.DatasourceID.String())

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("pre-agg request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		bodyBytes, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("pre-agg error (status %d): %s", resp.StatusCode, string(bodyBytes))
	}

	var preAggs []map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&preAggs); err != nil {
		return nil, fmt.Errorf("failed to decode pre-agg status: %w", err)
	}

	return preAggs, nil
}

// Dry run validates a query without executing it
func (c *Client) DryRun(ctx context.Context, query *Query, tenantCtx TenantContext) (map[string]interface{}, error) {
	body, err := json.Marshal(map[string]interface{}{
		"query": query,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to marshal query: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", fmt.Sprintf("%s/cubejs-api/v1/dry-run", c.baseURL), bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("failed to create dry-run request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", c.apiSecret)
	req.Header.Set("X-Tenant-ID", tenantCtx.TenantID.String())
	req.Header.Set("X-Tenant-Datasource-ID", tenantCtx.DatasourceID.String())

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("dry-run failed: %w", err)
	}
	defer resp.Body.Close()

	var result map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode dry-run: %w", err)
	}

	return result, nil
}

// BuildQuery is a helper for constructing common query patterns
func BuildQuery(cube string, measures, dimensions []string, filters []Filter) *Query {
	return &Query{
		Measures:   measures,
		Dimensions: dimensions,
		Filters:    filters,
		Timezone:   "UTC",
	}
}

// BuildTimeDimension creates a time dimension with date range
func BuildTimeDimension(dimension, granularity string, dateRange []string) TimeDimension {
	return TimeDimension{
		Dimension:   dimension,
		Granularity: granularity,
		DateRange:   dateRange,
	}
}
