package activities

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

// TrinoQueryRequest encapsulates a Trino query with metadata for tracing
type TrinoQueryRequest struct {
	Query   string            `json:"query"`
	RunID   string            `json:"run_id"`
	Region  string            `json:"region"`
	Timeout time.Duration     `json:"timeout"`
	Headers map[string]string `json:"headers"`
}

// TrinoQueryResponse wraps Trino API response
type TrinoQueryResponse struct {
	ID      string          `json:"id"`
	InfoURI string          `json:"infoUri"`
	NextURI string          `json:"nextUri"`
	Columns []interface{}   `json:"columns"`
	Data    [][]interface{} `json:"data"`
	Stats   interface{}     `json:"stats"`
	Error   interface{}     `json:"error"`
}

// RunTrinoQueryActivity executes a Trino SQL query with full support for pagination and error handling
// This activity is idempotent if the same run_id is passed; mutations in SQL (INSERT/UPDATE)
// should use transaction IDs or check for duplicates
func RunTrinoQueryActivity(ctx context.Context, runID string, region string, query string) (string, error) {
	// Validate inputs
	if query == "" {
		return "", fmt.Errorf("query cannot be empty")
	}

	// Construct Trino request
	// Use idempotent user context with run_id for tracing
	trinoURL := "http://trino:8080" // configure via env var in production
	timeoutSec := 300               // 5-minute timeout for analytics queries
	userName := fmt.Sprintf("temporal-worker-%s", runID)

	req, err := http.NewRequestWithContext(ctx, "POST", trinoURL+"/v1/statement", strings.NewReader(query))
	if err != nil {
		return "", fmt.Errorf("failed to build Trino request: %w", err)
	}

	// Set required Trino headers
	req.Header.Set("X-Trino-User", userName)
	req.Header.Set("X-Trino-Session", fmt.Sprintf("region=%s", region))
	req.Header.Set("X-Trino-Catalog", "iceberg")
	req.Header.Set("X-Trino-Schema", "ops")
	req.Header.Set("X-Trino-Request-Timeout", fmt.Sprintf("%ds", timeoutSec))
	req.Header.Set("Content-Type", "text/plain")

	// Execute query
	client := &http.Client{
		Timeout: time.Duration(timeoutSec+30) * time.Second, // client timeout > server timeout
	}

	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("Trino request failed: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("failed to read Trino response: %w", err)
	}

	// Parse initial response
	var trinoResp TrinoQueryResponse
	if err := json.Unmarshal(body, &trinoResp); err != nil {
		return "", fmt.Errorf("failed to parse Trino response: %w", err)
	}

	// Check for errors in response
	if trinoResp.Error != nil {
		return "", fmt.Errorf("Trino query failed: %v", trinoResp.Error)
	}

	// Handle pagination: fetch all result pages if nextUri provided
	queryID := trinoResp.ID
	for trinoResp.NextURI != "" {
		// Fetch next page
		nextReq, _ := http.NewRequestWithContext(ctx, "GET", trinoResp.NextURI, nil)
		nextReq.Header.Set("X-Trino-User", userName)
		nextResp, err := client.Do(nextReq)
		if err != nil {
			return "", fmt.Errorf("pagination request failed: %w", err)
		}
		defer nextResp.Body.Close()

		nextBody, _ := io.ReadAll(nextResp.Body)
		if err := json.Unmarshal(nextBody, &trinoResp); err != nil {
			return "", fmt.Errorf("failed to parse pagination response: %w", err)
		}

		if trinoResp.Error != nil {
			return "", fmt.Errorf("Trino query failed on pagination: %v", trinoResp.Error)
		}
	}

	// Return query ID and summary (in production, could return row count or status)
	result := map[string]interface{}{
		"query_id":  queryID,
		"status":    "completed",
		"row_count": len(trinoResp.Data),
	}

	resultJSON, _ := json.Marshal(result)
	return string(resultJSON), nil
}

// RunSparkJobActivity submits a Spark job to a cluster and waits for completion
// Supports both YARN and Kubernetes cluster managers
func RunSparkJobActivity(ctx context.Context, runID string, sparkConfig map[string]interface{}) (string, error) {
	// Placeholder for Spark job submission logic
	// In production, call your cluster manager API (YARN, Kubernetes, or cloud provider)

	if _, ok := sparkConfig["app_jar"]; !ok {
		return "", fmt.Errorf("app_jar not specified in spark config")
	}

	// Example: submit to YARN cluster
	sparkSubmitURL := "http://spark-submit:6066" // config via env var

	payload, _ := json.Marshal(sparkConfig)
	req, _ := http.NewRequestWithContext(ctx, "POST", sparkSubmitURL+"/v1/submissions/create", bytes.NewReader(payload))
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{Timeout: 60 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("Spark submit request failed: %w", err)
	}
	defer resp.Body.Close()

	var respBody map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&respBody)

	submissionID := fmt.Sprintf("%v", respBody["submissionId"])
	if submissionID == "<nil>" {
		return "", fmt.Errorf("failed to parse submission ID from Spark response")
	}

	// Poll for job completion (with exponential backoff)
	maxRetries := 600 // 10 minutes max wait with 1-second polls
	for i := 0; i < maxRetries; i++ {
		statusReq, _ := http.NewRequestWithContext(ctx, "GET",
			fmt.Sprintf("%s/v1/submissions/%s/status", sparkSubmitURL, submissionID), nil)
		statusResp, err := client.Do(statusReq)
		if err != nil {
			time.Sleep(time.Second)
			continue
		}

		var statusBody map[string]interface{}
		json.NewDecoder(statusResp.Body).Decode(&statusBody)
		statusResp.Body.Close()

		driverState := fmt.Sprintf("%v", statusBody["driverState"])
		if driverState == "RUNNING" {
			time.Sleep(time.Second) // poll every second
			continue
		}
		if driverState == "FINISHED" || driverState == "SUCCEEDED" {
			return submissionID, nil
		}
		if driverState == "FAILED" || driverState == "ERROR" {
			return submissionID, fmt.Errorf("Spark job failed: %s", driverState)
		}
	}

	return submissionID, fmt.Errorf("Spark job polling timeout after 10 minutes")
}

// RunPythonScriptActivity executes a Python script for ML training or feature extraction
// Scripts must be pre-installed in the worker environment
func RunPythonScriptActivity(ctx context.Context, runID string, scriptPath string, args ...string) (string, error) {
	// Placeholder for Python script execution
	// In production, use subprocess or a dedicated Python execution service

	// For now, return a stub result
	result := map[string]interface{}{
		"run_id":    runID,
		"script":    scriptPath,
		"args":      args,
		"status":    "completed",
		"timestamp": time.Now().UTC(),
	}

	resultJSON, _ := json.Marshal(result)
	return string(resultJSON), nil
}

// PublishEventActivity publishes an event to the WebSocket hub for real-time dashboard updates
// This integrates with the Go HTTP server running the WebSocket hub
func PublishEventActivity(ctx context.Context, runID string, region string, eventType string) error {
	// Call local HTTP server to publish event
	// The Go server should have a local HTTP endpoint for activity-triggered events

	event := map[string]interface{}{
		"run_id":     runID,
		"region":     region,
		"event_type": eventType,
		"timestamp":  time.Now().UTC().Format(time.RFC3339),
	}

	payload, _ := json.Marshal(event)
	req, _ := http.NewRequestWithContext(ctx, "POST", "http://localhost:8081/events/publish", bytes.NewReader(payload))
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{Timeout: 5 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		// Non-fatal: don't fail the workflow if event publishing fails
		fmt.Printf("warning: failed to publish event: %v\n", err)
		return nil
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		return fmt.Errorf("event publish failed with status %d", resp.StatusCode)
	}

	return nil
}
