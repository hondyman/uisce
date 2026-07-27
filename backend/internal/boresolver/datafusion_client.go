package boresolver

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

// DataFusionClient executes SQL queries against the Apache DataFusion REST API.
type DataFusionClient struct {
	EndpointURL string
	HTTPClient  *http.Client
}

// NewDataFusionClient creates a new DataFusion HTTP client.
func NewDataFusionClient(endpointURL string) *DataFusionClient {
	if endpointURL == "" {
		endpointURL = "http://100.84.50.65:8555"
	}
	return &DataFusionClient{
		EndpointURL: endpointURL,
		HTTPClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// DataFusionQueryRequest is the payload for the DataFusion /api/v1/query endpoint.
type DataFusionQueryRequest struct {
	TenantID string `json:"tenant_id,omitempty"`
	Query    string `json:"query"`
}

// DataFusionQueryResponse is the response from the DataFusion /api/v1/query endpoint.
type DataFusionQueryResponse struct {
	ExecutionTimeMS float64         `json:"execution_time_ms"`
	Records         [][]interface{} `json:"records"`
	RowCount        int             `json:"row_count"`
	Error           string          `json:"error,omitempty"`
}

// ExecuteQuery executes a compiled SQL string directly on the DataFusion REST engine.
func (c *DataFusionClient) ExecuteQuery(ctx context.Context, tenantID, compiledSQL string) (*DataFusionQueryResponse, error) {
	payload := DataFusionQueryRequest{
		TenantID: tenantID,
		Query:    compiledSQL,
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal DataFusion request payload: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.EndpointURL+"/api/v1/query", bytes.NewBuffer(body))
	if err != nil {
		return nil, fmt.Errorf("failed to construct DataFusion HTTP request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("datafusion execution failed: endpoint unreachable at %s: %w", c.EndpointURL, err)
	}
	defer resp.Body.Close()

	respBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read DataFusion response payload: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("datafusion query error (HTTP %d): %s", resp.StatusCode, string(respBytes))
	}

	var queryResp DataFusionQueryResponse
	if err := json.Unmarshal(respBytes, &queryResp); err != nil {
		return nil, fmt.Errorf("failed to unmarshal DataFusion query response: %w", err)
	}

	if queryResp.Error != "" {
		return nil, fmt.Errorf("datafusion execution error: %s", queryResp.Error)
	}

	return &queryResp, nil
}
