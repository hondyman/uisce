package calcengine

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
	"github.com/hondyman/semlayer/libs/jwt-middleware"
)

// CubeBridge provides a Go client for the Cube.js semantic layer API
// Following the "configure over custom" and "metadata first" patterns
type CubeBridge struct {
	apiURL     string
	httpClient *http.Client
}

// CubeQuery represents a Cube.js query
type CubeQuery struct {
	// Measures to aggregate (e.g., ["Orders.count", "Orders.totalAmount"])
	Measures []string `json:"measures,omitempty"`

	// Dimensions to group by (e.g., ["Orders.status", "Orders.createdAt"])
	Dimensions []string `json:"dimensions,omitempty"`

	// Segments to filter (e.g., ["Orders.paid"])
	Segments []string `json:"segments,omitempty"`

	// Time dimensions with granularity
	TimeDimensions []CubeTimeDimension `json:"timeDimensions,omitempty"`

	// Filters for WHERE clause
	Filters []CubeFilter `json:"filters,omitempty"`

	// Order by clause
	Order []CubeOrder `json:"order,omitempty"`

	// Limit results
	Limit int `json:"limit,omitempty"`

	// Offset for pagination
	Offset int `json:"offset,omitempty"`

	// Timezone for time calculations
	Timezone string `json:"timezone,omitempty"`

	// Security context (tenant isolation)
	TenantID     string `json:"-"` // Set via header
	DatasourceID string `json:"-"` // Set via header
}

// CubeTimeDimension represents a time-based dimension with granularity
type CubeTimeDimension struct {
	Dimension   string   `json:"dimension"`
	Granularity string   `json:"granularity,omitempty"` // second, minute, hour, day, week, month, quarter, year
	DateRange   []string `json:"dateRange,omitempty"`   // e.g., ["2024-01-01", "2024-12-31"] or "last 7 days"
}

// CubeFilter represents a filter condition
type CubeFilter struct {
	Member   string      `json:"member"`           // e.g., "Orders.status"
	Operator string      `json:"operator"`         // equals, notEquals, contains, gt, lt, etc.
	Values   []string    `json:"values,omitempty"` // For equals, notEquals, contains
	Value    interface{} `json:"value,omitempty"`  // For gt, lt, gte, lte
}

// CubeOrder represents sort order
type CubeOrder struct {
	Member string `json:"member"` // e.g., "Orders.count"
	Order  string `json:"order"`  // asc or desc
}

// CubeResponse represents the Cube.js API response
type CubeResponse struct {
	Data           []map[string]interface{} `json:"data"`
	Annotation     CubeAnnotation           `json:"annotation,omitempty"`
	RefreshKeyTime string                   `json:"refreshKeyTime,omitempty"`
	SlowQuery      bool                     `json:"slowQuery,omitempty"`
	TotalCount     int                      `json:"total,omitempty"`
}

// CubeAnnotation contains metadata about the response
type CubeAnnotation struct {
	Measures       map[string]CubeMemberMeta `json:"measures,omitempty"`
	Dimensions     map[string]CubeMemberMeta `json:"dimensions,omitempty"`
	TimeDimensions map[string]CubeMemberMeta `json:"timeDimensions,omitempty"`
}

// CubeMemberMeta contains metadata about a measure or dimension
type CubeMemberMeta struct {
	Title        string   `json:"title"`
	ShortTitle   string   `json:"shortTitle"`
	Type         string   `json:"type"`
	Format       string   `json:"format,omitempty"`
	DrillMembers []string `json:"drillMembers,omitempty"`
}

// NewCubeBridge creates a new Cube.js bridge client
func NewCubeBridge(apiURL string) *CubeBridge {
	return &CubeBridge{
		apiURL: apiURL,
		httpClient: &http.Client{
			Timeout: 60 * time.Second,
		},
	}
}

// Query executes a Cube.js query
func (b *CubeBridge) Query(ctx context.Context, query *CubeQuery) (*CalcResult, error) {
	resp, err := b.executeQuery(ctx, query)
	if err != nil {
		return nil, err
	}

	return b.convertToCalcResult(resp, query)
}

// QueryMetric executes a query for a specific metric using semantic layer conventions
func (b *CubeBridge) QueryMetric(ctx context.Context, metric string, inputs map[string]interface{}) (*CalcResult, error) {
	// Build query from inputs
	query := &CubeQuery{
		TenantID:     getStringInput(inputs, "tenant_id"),
		DatasourceID: getStringInput(inputs, "datasource_id"),
	}

	// Add the metric as a measure
	query.Measures = []string{metric}

	// Add dimensions if provided
	if dims, ok := inputs["dimensions"].([]string); ok {
		query.Dimensions = dims
	}

	// Add time dimension if date range provided
	if startDate := getTimeInput(inputs, "start_date"); !startDate.IsZero() {
		endDate := getTimeInput(inputs, "end_date")
		if endDate.IsZero() {
			endDate = time.Now()
		}

		granularity := getStringInput(inputs, "granularity")
		if granularity == "" {
			granularity = "day"
		}

		// Find time dimension from cube name
		cubeName := extractCubeName(metric)
		timeDim := cubeName + ".createdAt" // Default convention
		if td := getStringInput(inputs, "time_dimension"); td != "" {
			timeDim = td
		}

		query.TimeDimensions = []CubeTimeDimension{
			{
				Dimension:   timeDim,
				Granularity: granularity,
				DateRange:   []string{startDate.Format("2006-01-02"), endDate.Format("2006-01-02")},
			},
		}
	}

	// Add filters from inputs
	for key, val := range inputs {
		if key == "tenant_id" || key == "datasource_id" || key == "dimensions" ||
			key == "start_date" || key == "end_date" || key == "granularity" ||
			key == "time_dimension" || key == "data_tier" {
			continue
		}

		// Convert input to filter
		if strVal, ok := val.(string); ok {
			query.Filters = append(query.Filters, CubeFilter{
				Member:   key,
				Operator: "equals",
				Values:   []string{strVal},
			})
		}
	}

	// Add limit if provided
	if limit, ok := inputs["limit"].(int); ok {
		query.Limit = limit
	}

	return b.Query(ctx, query)
}

// QueryWithRLS executes a query with automatic tenant/datasource isolation
func (b *CubeBridge) QueryWithRLS(ctx context.Context, tenantID, datasourceID string, query *CubeQuery) (*CalcResult, error) {
	query.TenantID = tenantID
	query.DatasourceID = datasourceID
	return b.Query(ctx, query)
}

// LoadCubeDefinition fetches cube metadata for a specific cube
func (b *CubeBridge) LoadCubeDefinition(ctx context.Context, tenantID, datasourceID, cubeName string) (*CubeDefinition, error) {
	url := fmt.Sprintf("%s/cubejs-api/v1/meta", b.apiURL)

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, err
	}

	b.setSecurityHeaders(req, tenantID, datasourceID)

	resp, err := b.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch cube meta: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("cube meta API error (%d): %s", resp.StatusCode, string(body))
	}

	var metaResp struct {
		Cubes []CubeDefinition `json:"cubes"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&metaResp); err != nil {
		return nil, fmt.Errorf("failed to decode cube meta: %w", err)
	}

	// Find the requested cube
	for _, cube := range metaResp.Cubes {
		if cube.Name == cubeName {
			return &cube, nil
		}
	}

	return nil, fmt.Errorf("cube %s not found", cubeName)
}

// CubeDefinition represents a Cube.js cube metadata
type CubeDefinition struct {
	Name        string          `json:"name"`
	Title       string          `json:"title"`
	Description string          `json:"description,omitempty"`
	Measures    []CubeMeasure   `json:"measures"`
	Dimensions  []CubeDimension `json:"dimensions"`
	Segments    []CubeSegment   `json:"segments"`
}

// CubeMeasure represents a measure in a cube
type CubeMeasure struct {
	Name         string   `json:"name"`
	Title        string   `json:"title"`
	ShortTitle   string   `json:"shortTitle"`
	Type         string   `json:"type"` // number, count, sum, avg, min, max, countDistinct, etc.
	DrillMembers []string `json:"drillMembers,omitempty"`
	Format       string   `json:"format,omitempty"`
}

// CubeDimension represents a dimension in a cube
type CubeDimension struct {
	Name       string `json:"name"`
	Title      string `json:"title"`
	ShortTitle string `json:"shortTitle"`
	Type       string `json:"type"` // string, number, time, boolean, geo
	PrimaryKey bool   `json:"primaryKey,omitempty"`
}

// CubeSegment represents a segment (pre-defined filter) in a cube
type CubeSegment struct {
	Name       string `json:"name"`
	Title      string `json:"title"`
	ShortTitle string `json:"shortTitle"`
}

// executeQuery performs the actual HTTP request to Cube.js
func (b *CubeBridge) executeQuery(ctx context.Context, query *CubeQuery) (*CubeResponse, error) {
	url := fmt.Sprintf("%s/cubejs-api/v1/load", b.apiURL)

	// Marshal query
	queryBody := map[string]interface{}{
		"query": query,
	}
	body, err := json.Marshal(queryBody)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal query: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	b.setSecurityHeaders(req, query.TenantID, query.DatasourceID)

	resp, err := b.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("Cube API request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		respBody, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("Cube API error (%d): %s", resp.StatusCode, string(respBody))
	}

	var cubeResp CubeResponse
	if err := json.NewDecoder(resp.Body).Decode(&cubeResp); err != nil {
		return nil, fmt.Errorf("failed to decode Cube response: %w", err)
	}

	return &cubeResp, nil
}

// setSecurityHeaders adds tenant isolation headers for RLS
func (b *CubeBridge) setSecurityHeaders(req *http.Request, tenantID, datasourceID string) {
	if tenantID != "" {
		req.Header.Set("X-Tenant-ID", tenantID)
	}
	if datasourceID != "" {
		req.Header.Set("X-Tenant-Datasource-ID", datasourceID)
	}
}

// convertToCalcResult converts Cube response to CalcResult
func (b *CubeBridge) convertToCalcResult(resp *CubeResponse, query *CubeQuery) (*CalcResult, error) {
	if len(resp.Data) == 0 {
		return &CalcResult{
			Metric: getMeasureName(query.Measures),
			Value:  0,
		}, nil
	}

	// Extract primary metric value
	var primaryValue float64
	if len(query.Measures) > 0 && len(resp.Data) > 0 {
		if val, ok := resp.Data[0][query.Measures[0]]; ok {
			switch v := val.(type) {
			case float64:
				primaryValue = v
			case int:
				primaryValue = float64(v)
			case int64:
				primaryValue = float64(v)
			}
		}
	}

	// Build breakdown from all data rows
	var breakdown []map[string]interface{}
	for _, row := range resp.Data {
		breakdown = append(breakdown, row)
	}

	// Build sources list
	var sources []string
	for _, measure := range query.Measures {
		sources = append(sources, extractCubeName(measure))
	}
	for _, dim := range query.Dimensions {
		cubeName := extractCubeName(dim)
		if !contains(sources, cubeName) {
			sources = append(sources, cubeName)
		}
	}

	return &CalcResult{
		Metric:    getMeasureName(query.Measures),
		Value:     primaryValue,
		Sources:   sources,
		Breakdown: breakdown,
	}, nil
}

// ExecuteSQL runs a raw SQL query through Cube's SQL interface
func (b *CubeBridge) ExecuteSQL(ctx context.Context, tenantID, datasourceID, sql string) (*CalcResult, error) {
	url := fmt.Sprintf("%s/cubejs-api/v1/sql", b.apiURL)

	body, _ := json.Marshal(map[string]string{"query": sql})

	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewReader(body))
	if err != nil {
		return nil, err
	}

	req.Header.Set("Content-Type", "application/json")
	b.setSecurityHeaders(req, tenantID, datasourceID)

	resp, err := b.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("Cube SQL API request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		respBody, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("Cube SQL API error (%d): %s", resp.StatusCode, string(respBody))
	}

	var sqlResp struct {
		SQL  []string                 `json:"sql"`
		Data []map[string]interface{} `json:"data"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&sqlResp); err != nil {
		return nil, err
	}

	return &CalcResult{
		Metric:    "SQL",
		Sources:   sqlResp.SQL,
		Breakdown: sqlResp.Data,
	}, nil
}

// Helper functions

func getStringInput(inputs map[string]interface{}, key string) string {
	if val, ok := inputs[key].(string); ok {
		return val
	}
	return ""
}

func getTimeInput(inputs map[string]interface{}, key string) time.Time {
	if val, ok := inputs[key].(time.Time); ok {
		return val
	}
	return time.Time{}
}

func extractCubeName(member string) string {
	for i, c := range member {
		if c == '.' {
			return member[:i]
		}
	}
	return member
}

func getMeasureName(measures []string) string {
	if len(measures) == 0 {
		return "Unknown"
	}
	return measures[0]
}

func contains(slice []string, item string) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}
