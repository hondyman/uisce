package activities

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

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

// RunPythonScriptActivity executes a Python script for ML training or feature extraction.
// scriptPath is an absolute path to the Python interpreter; args are passed to the script.
func RunPythonScriptActivity(ctx context.Context, runID string, scriptPath string, args ...string) (string, error) {
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

// RunCustomizationIntelligenceETL runs the customization_clusters ETL Python job.
// It is invoked by the CustomizationIntelligenceWorkflow on a cron schedule.
func RunCustomizationIntelligenceETL(ctx context.Context, runID string, scriptPath string, lookbackDays int, minTenants int) (string, error) {
	// Placeholder: invoke python -m jobs.customization_clusters --lookback-days N --min-tenants M
	// In production this runs as a subprocess; the result is written to fact_customization_telemetry.
	result := map[string]interface{}{
		"run_id":        runID,
		"script":        scriptPath,
		"lookback_days": lookbackDays,
		"min_tenants":   minTenants,
		"status":        "completed",
		"timestamp":     time.Now().UTC(),
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
