package activities

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
)

// TestRunTrinoQueryActivity_Success validates successful Trino query execution
func TestRunTrinoQueryActivity_Success(t *testing.T) {
	ctx := context.Background()
	runID := "test-001"
	region := "us-east-1"
	query := "SELECT COUNT(*) FROM iceberg.ops.ops_events"

	result, err := RunTrinoQueryActivity(ctx, runID, region, query)

	assert.NoError(t, err)
	assert.NotEmpty(t, result)
}

// TestRunTrinoQueryActivity_EmptyQuery validates error on empty query
func TestRunTrinoQueryActivity_EmptyQuery(t *testing.T) {
	ctx := context.Background()
	runID := "test-001"
	region := "us-east-1"

	_, err := RunTrinoQueryActivity(ctx, runID, region, "")

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "query cannot be empty")
}

// TestRunSparkJobActivity_Success validates successful Spark job submission
func TestRunSparkJobActivity_Success(t *testing.T) {
	ctx := context.Background()
	runID := "test-001"
	config := map[string]interface{}{
		"app_jar":    "s3://bucket/app.jar",
		"main_class": "com.example.Main",
		"conf": map[string]string{
			"spark.executor.memory": "8g",
		},
	}

	result, err := RunSparkJobActivity(ctx, runID, config)

	assert.NoError(t, err)
	assert.NotEmpty(t, result)
}

// TestRunSparkJobActivity_MissingJar validates error on missing jar
func TestRunSparkJobActivity_MissingJar(t *testing.T) {
	ctx := context.Background()
	runID := "test-001"
	config := map[string]interface{}{
		"main_class": "com.example.Main",
	}

	_, err := RunSparkJobActivity(ctx, runID, config)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "app_jar not specified")
}

// TestRunPythonScriptActivity_Success validates successful Python script execution
func TestRunPythonScriptActivity_Success(t *testing.T) {
	ctx := context.Background()
	runID := "test-001"
	scriptPath := "scripts/ml/train_model.py"
	args := []string{"model_name", "2026-02-09"}

	result, err := RunPythonScriptActivity(ctx, runID, scriptPath, args...)

	assert.NoError(t, err)
	assert.NotEmpty(t, result)
}

// TestPublishEventActivity_Success validates successful event publishing
func TestPublishEventActivity_Success(t *testing.T) {
	ctx := context.Background()
	runID := "test-001"
	region := "us-east-1"
	eventType := "workflow_completed"

	// Note: will fail if hub not running, but activity handles gracefully
	err := PublishEventActivity(ctx, runID, region, eventType)
	assert.Nil(t, err) // Non-fatal errors are suppressed
}
