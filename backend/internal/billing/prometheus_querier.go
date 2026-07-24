package billing

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"time"
)

// HTTPPrometheusQuerier implements PrometheusQuerier by calling the
// Prometheus HTTP API directly. It re-uses the same env-var-driven
// endpoint (PROMETHEUS_URL) as the existing metrics proxy.
type HTTPPrometheusQuerier struct {
	baseURL string
	client  *http.Client
}

// NewHTTPPrometheusQuerier creates a querier that targets the
// PROMETHEUS_URL environment variable or the Docker-default
// "http://prometheus:9090".
func NewHTTPPrometheusQuerier() *HTTPPrometheusQuerier {
	base := os.Getenv("PROMETHEUS_URL")
	if base == "" {
		base = "http://prometheus:9090"
	}
	return &HTTPPrometheusQuerier{
		baseURL: base,
		client:  &http.Client{Timeout: 5 * time.Second},
	}
}

type promResponse struct {
	Status string `json:"status"`
	Data   struct {
		ResultType string `json:"resultType"`
		Result     []struct {
			Metric map[string]string `json:"metric"`
			Value  [2]interface{}    `json:"value"`
		} `json:"result"`
	} `json:"data"`
}

// InstantQuery executes a PromQL instant query.
func (q *HTTPPrometheusQuerier) InstantQuery(ctx context.Context, query string) ([]QueryResult, error) {
	u, err := url.Parse(q.baseURL)
	if err != nil {
		return nil, fmt.Errorf("invalid prometheus URL: %w", err)
	}
	u.Path = "/api/v1/query"
	params := url.Values{}
	params.Set("query", query)
	u.RawQuery = params.Encode()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u.String(), nil)
	if err != nil {
		return nil, err
	}

	resp, err := q.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("prometheus query failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("prometheus returned %d: %s", resp.StatusCode, string(body))
	}

	var pr promResponse
	if err := json.NewDecoder(resp.Body).Decode(&pr); err != nil {
		return nil, fmt.Errorf("decode prometheus response: %w", err)
	}
	if pr.Status != "success" {
		return nil, fmt.Errorf("prometheus status: %s", pr.Status)
	}

	var results []QueryResult
	for _, r := range pr.Data.Result {
		val := 0.0
		switch v := r.Value[1].(type) {
		case string:
			val, _ = strconv.ParseFloat(v, 64)
		case float64:
			val = v
		}
		results = append(results, QueryResult{
			Labels: r.Metric,
			Value:  val,
		})
	}
	return results, nil
}
