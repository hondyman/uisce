package main

import (
	"context"
	"fmt"
	"log"
	"time"

	temporalclient "github.com/hondyman/semlayer/libs/temporal-client"
	"go.temporal.io/sdk/worker"
	"go.temporal.io/sdk/workflow"
)

// ProcessWorkflowSuggestion is the Temporal workflow that processes AI suggestions
func ProcessWorkflowSuggestion(ctx workflow.Context, suggestion WorkflowSuggestion) error {
	logger := workflow.GetLogger(ctx)

	logger.Info("Processing AI workflow suggestion", "description", suggestion.Description)

	// Activity to validate the workflow suggestion
	var validationResult map[string]interface{}
	err := workflow.ExecuteActivity(
		workflow.WithActivityOptions(ctx, workflow.ActivityOptions{
			StartToCloseTimeout: time.Minute * 5,
		}),
		ValidateWorkflowSuggestion,
		suggestion,
	).Get(ctx, &validationResult)

	if err != nil {
		logger.Error("Failed to validate workflow suggestion", "error", err)
		return err
	}

	logger.Info("Workflow suggestion validated", "result", validationResult)

	// Activity to store the workflow suggestion
	var storageResult string
	err = workflow.ExecuteActivity(
		workflow.WithActivityOptions(ctx, workflow.ActivityOptions{
			StartToCloseTimeout: time.Minute * 2,
		}),
		StoreWorkflowSuggestion,
		suggestion,
	).Get(ctx, &storageResult)

	if err != nil {
		logger.Error("Failed to store workflow suggestion", "error", err)
		return err
	}

	logger.Info("Workflow suggestion stored", "result", storageResult)

	// Activity to notify stakeholders
	err = workflow.ExecuteActivity(
		workflow.WithActivityOptions(ctx, workflow.ActivityOptions{
			StartToCloseTimeout: time.Minute * 1,
		}),
		NotifyWorkflowStakeholders,
		suggestion,
	).Get(ctx, nil)

	if err != nil {
		logger.Error("Failed to notify stakeholders", "error", err)
		// Don't fail the workflow for notification errors
	}

	return nil
}

// ValidateWorkflowSuggestion activity validates the AI-generated workflow
func ValidateWorkflowSuggestion(ctx context.Context, suggestion WorkflowSuggestion) (map[string]interface{}, error) {
	log.Printf("Validating workflow suggestion: %s", suggestion.Description)

	// Basic validation
	result := map[string]interface{}{
		"valid":       true,
		"errors":      []string{},
		"warnings":    []string{},
		"score":       0.85, // Mock validation score
		"validatedAt": time.Now(),
	}

	// Check for required elements
	if len(suggestion.Elements) == 0 {
		result["valid"] = false
		result["errors"] = append(result["errors"].([]string), "No workflow elements found")
	}

	// Check for start/end nodes
	hasStart := false
	hasEnd := false
	for _, element := range suggestion.Elements {
		if element.Type == "input" {
			hasStart = true
		}
		if element.Type == "output" {
			hasEnd = true
		}
	}

	if !hasStart {
		result["warnings"] = append(result["warnings"].([]string), "Missing start node")
	}
	if !hasEnd {
		result["warnings"] = append(result["warnings"].([]string), "Missing end node")
	}

	if suggestion.YAML == "" {
		result["warnings"] = append(result["warnings"].([]string), "No YAML definition provided")
	}

	return result, nil
}

// StoreWorkflowSuggestion activity stores the validated suggestion
func StoreWorkflowSuggestion(ctx context.Context, suggestion WorkflowSuggestion) (string, error) {
	log.Printf("Storing workflow suggestion: %s", suggestion.Description)

	// In a real implementation, this would store to a database
	// For now, we'll simulate storage
	suggestionId := fmt.Sprintf("wf-suggestion-%d", time.Now().Unix())

	// Add storage metadata
	if suggestion.Metadata == nil {
		suggestion.Metadata = make(map[string]interface{})
	}
	suggestion.Metadata["stored_at"] = time.Now()
	suggestion.Metadata["suggestion_id"] = suggestionId

	log.Printf("Workflow suggestion stored with ID: %s", suggestionId)
	return suggestionId, nil
}

// NotifyWorkflowStakeholders activity sends notifications
func NotifyWorkflowStakeholders(ctx context.Context, suggestion WorkflowSuggestion) error {
	log.Printf("Notifying stakeholders about workflow suggestion: %s", suggestion.Description)

	// In a real implementation, this would send emails, Slack messages, etc.
	// For now, we'll just log the notification
	log.Printf("Notification sent: New AI-generated workflow suggestion available for review")

	return nil
}

func runWorker() {
	// Create Temporal client with retry helper from libs
	tc, err := temporalclient.NewClientWithRetry()
	if err != nil {
		log.Fatalf("Failed to create Temporal client: %v", err)
	}
	defer tc.Close()

	// Create worker
	w := worker.New(tc, "ai-builder", worker.Options{})

	// Register workflow and activities
	w.RegisterWorkflow(ProcessWorkflowSuggestion)
	w.RegisterActivity(ValidateWorkflowSuggestion)
	w.RegisterActivity(StoreWorkflowSuggestion)
	w.RegisterActivity(NotifyWorkflowStakeholders)

	// Start worker
	err = w.Start()
	if err != nil {
		log.Fatalf("Failed to start worker: %v", err)
	}

	log.Println("AI Builder Temporal worker started")
	select {}
}
