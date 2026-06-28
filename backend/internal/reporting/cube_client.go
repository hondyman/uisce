package reporting

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

// CubeClient handles communication with the Cube.js API
type CubeClient struct {
	baseURL    string
	httpClient *http.Client
}

// NewCubeClient creates a new Cube.js client
func NewCubeClient(baseURL string) *CubeClient {
	return &CubeClient{
		baseURL: baseURL,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// CubeQuery represents a query to Cube.js
type CubeQuery struct {
	Measures       []string            `json:"measures,omitempty"`
	Dimensions     []string            `json:"dimensions,omitempty"`
	Filters        []CubeFilter        `json:"filters,omitempty"`
	TimeDimensions []CubeTimeDimension `json:"timeDimensions,omitempty"`
	Order          [][]string          `json:"order,omitempty"`
	Limit          int                 `json:"limit,omitempty"`
	Offset         int                 `json:"offset,omitempty"`
	Segments       []string            `json:"segments,omitempty"`
}

// CubeFilter represents a filter in a Cube.js query
type CubeFilter struct {
	Member    string   `json:"member,omitempty"`
	Dimension string   `json:"dimension,omitempty"` // Deprecated, use member
	Operator  string   `json:"operator"`
	Values    []string `json:"values,omitempty"`
}

// CubeTimeDimension represents a time dimension in a Cube.js query
type CubeTimeDimension struct {
	Dimension   string   `json:"dimension"`
	Granularity string   `json:"granularity,omitempty"`
	DateRange   []string `json:"dateRange,omitempty"`
}

// CubeResult represents the result from a Cube.js query
type CubeResult struct {
	Data       []map[string]interface{} `json:"data"`
	Annotation *CubeAnnotation          `json:"annotation,omitempty"`
	Query      *CubeQuery               `json:"query,omitempty"`
	RefreshKey string                   `json:"refreshKeyValues,omitempty"`
}

// CubeAnnotation provides metadata about the query results
type CubeAnnotation struct {
	Measures       map[string]CubeMemberMeta `json:"measures"`
	Dimensions     map[string]CubeMemberMeta `json:"dimensions"`
	TimeDimensions map[string]CubeMemberMeta `json:"timeDimensions"`
}

// CubeMemberMeta provides metadata about a measure or dimension
type CubeMemberMeta struct {
	Title        string   `json:"title"`
	ShortTitle   string   `json:"shortTitle"`
	Type         string   `json:"type"`
	Format       string   `json:"format,omitempty"`
	DrillMembers []string `json:"drillMembers,omitempty"`
}

// ExecuteQuery executes a query against Cube.js
func (c *CubeClient) ExecuteQuery(ctx context.Context, query *CubeQuery, tenantID, datasourceID uuid.UUID) (*CubeResult, error) {
	// Add tenant filtering automatically for security
	query.Filters = append(query.Filters, CubeFilter{
		Member:   "tenant_id",
		Operator: "equals",
		Values:   []string{tenantID.String()},
	})

	// Build request
	queryJSON, err := json.Marshal(map[string]interface{}{"query": query})
	if err != nil {
		return nil, fmt.Errorf("failed to marshal query: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", c.baseURL+"/cubejs-api/v1/load", bytes.NewBuffer(queryJSON))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Tenant-ID", tenantID.String())
	req.Header.Set("X-Datasource-ID", datasourceID.String())

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to execute query: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("cube.js returned status %d: %s", resp.StatusCode, string(body))
	}

	var result CubeResult
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return &result, nil
}

// BuildQueryFromBinding converts a DataBinding to a CubeQuery
func BuildQueryFromBinding(binding *DataBinding, parameters map[string]interface{}) (*CubeQuery, error) {
	query := &CubeQuery{
		Limit: binding.Limit,
	}

	// Add measures with cube prefix
	for _, m := range binding.Measures {
		query.Measures = append(query.Measures, fmt.Sprintf("%s.%s", binding.Cube, m))
	}

	// Add dimensions with cube prefix
	for _, d := range binding.Dimensions {
		query.Dimensions = append(query.Dimensions, fmt.Sprintf("%s.%s", binding.Cube, d))
	}

	// Convert filters
	for _, f := range binding.Filters {
		filter := CubeFilter{
			Member:   fmt.Sprintf("%s.%s", binding.Cube, f.Dimension),
			Operator: f.Operator,
		}

		// Resolve parameter reference
		if f.Parameter != "" {
			if paramValue, ok := parameters[f.Parameter]; ok {
				filter.Values = []string{fmt.Sprintf("%v", paramValue)}
			}
		} else if f.Value != nil {
			filter.Values = []string{fmt.Sprintf("%v", f.Value)}
		}

		query.Filters = append(query.Filters, filter)
	}

	// Add time dimension if specified
	if binding.TimeDimension != nil {
		td := CubeTimeDimension{
			Dimension:   fmt.Sprintf("%s.%s", binding.Cube, binding.TimeDimension.Dimension),
			Granularity: binding.TimeDimension.Granularity,
		}

		// Resolve date range from parameter
		if binding.TimeDimension.DateRange != nil {
			if binding.TimeDimension.DateRange.Parameter != "" {
				if paramValue, ok := parameters[binding.TimeDimension.DateRange.Parameter]; ok {
					// Handle different date range formats
					switch v := paramValue.(type) {
					case string:
						td.DateRange = []string{v}
					case []string:
						td.DateRange = v
					case map[string]interface{}:
						if start, ok := v["start"].(string); ok {
							if end, ok := v["end"].(string); ok {
								td.DateRange = []string{start, end}
							}
						}
					}
				}
			} else if binding.TimeDimension.DateRange.Value != "" {
				td.DateRange = []string{binding.TimeDimension.DateRange.Value}
			}
		}

		query.TimeDimensions = append(query.TimeDimensions, td)
	}

	// Add ordering
	if len(binding.Order) > 0 {
		for field, direction := range binding.Order {
			query.Order = append(query.Order, []string{
				fmt.Sprintf("%s.%s", binding.Cube, field),
				direction,
			})
		}
	}

	return query, nil
}

// GetCubeMeta retrieves metadata about available cubes
func (c *CubeClient) GetCubeMeta(ctx context.Context, tenantID, datasourceID uuid.UUID) (map[string]interface{}, error) {
	req, err := http.NewRequestWithContext(ctx, "GET", c.baseURL+"/cubejs-api/v1/meta", nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("X-Tenant-ID", tenantID.String())
	req.Header.Set("X-Datasource-ID", datasourceID.String())

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to get meta: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("cube.js returned status %d: %s", resp.StatusCode, string(body))
	}

	var result map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return result, nil
}

// ValidateCubeExists checks if a cube exists
func (c *CubeClient) ValidateCubeExists(ctx context.Context, cubeName string, tenantID, datasourceID uuid.UUID) (bool, error) {
	meta, err := c.GetCubeMeta(ctx, tenantID, datasourceID)
	if err != nil {
		return false, err
	}

	cubes, ok := meta["cubes"].([]interface{})
	if !ok {
		return false, nil
	}

	for _, cube := range cubes {
		if cubeMap, ok := cube.(map[string]interface{}); ok {
			if name, ok := cubeMap["name"].(string); ok && name == cubeName {
				return true, nil
			}
		}
	}

	return false, nil
}
