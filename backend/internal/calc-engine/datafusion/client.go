package datafusion

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

type Config struct {
	Host string
	Port int
}

type Client struct {
	host       string
	port       int
	baseURL    string
	httpClient *http.Client
	timeout    time.Duration
}

func NewClient(cfg Config) (*Client, error) {
	host := cfg.Host
	if host == "" {
		host = "localhost"
	}
	port := cfg.Port
	if port == 0 {
		port = 8554
	}

	return &Client{
		host:     host,
		port:     port,
		baseURL:  fmt.Sprintf("http://%s:%d", host, port),
		timeout:  5 * time.Minute,
		httpClient: &http.Client{
			Timeout: 5 * time.Minute,
		},
	}, nil
}

type QueryRequest struct {
	Query string `json:"query"`
}

type QueryResponse struct {
	Schema          []SchemaField `json:"schema"`
	Records         [][]any      `json:"records"`
	ExecutionTimeMs float64      `json:"execution_time_ms"`
	RowCount        int          `json:"row_count"`
	Error           string       `json:"error,omitempty"`
}

type SchemaField struct {
	Name string `json:"name"`
	Type string `json:"type"`
}

func (c *Client) QueryToMap(ctx context.Context, query string) ([]map[string]interface{}, error) {
	reqBody := QueryRequest{Query: query}
	body, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal query request: %w", err)
	}

	url := c.baseURL + "/api/v1/query"
	httpReq, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("datafusion query failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		respBody, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("datafusion HTTP error %d: %s", resp.StatusCode, string(respBody))
	}

	var result QueryResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	if result.Error != "" {
		return nil, fmt.Errorf("datafusion query error: %s", result.Error)
	}

	maps := make([]map[string]interface{}, 0, len(result.Records))
	for _, record := range result.Records {
		m := make(map[string]interface{})
		for i, field := range result.Schema {
			if i < len(record) {
				m[field.Name] = record[i]
			}
		}
		maps = append(maps, m)
	}

	return maps, nil
}

func (c *Client) Ping(ctx context.Context) error {
	resp, err := c.httpClient.Get(c.baseURL + "/health")
	if err != nil {
		return fmt.Errorf("datafusion ping failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("datafusion health check returned %d", resp.StatusCode)
	}
	return nil
}

func (c *Client) Close() error {
	return nil
}
