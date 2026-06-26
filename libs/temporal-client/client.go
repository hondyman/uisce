package temporalclient

import (
	"context"
	"fmt"
	"time"

	"go.temporal.io/sdk/client"
	"go.temporal.io/sdk/workflow"
)

// WorkflowOptions represents workflow execution options
type WorkflowOptions struct {
	TaskQueue  string
	WorkflowID string
	Timeout    int
}

// WorkflowExecutionInfo represents workflow execution information
type WorkflowExecutionInfo struct {
	Execution interface{}
	Type      interface{}
	StartTime interface{}
	CloseTime interface{}
	Status    interface{}
}

// Client wraps Temporal client with convenience methods
type Client struct {
	client client.Client
}

// NewClient creates a new temporal client wrapper
func NewClient(c client.Client) *Client {
	return &Client{client: c}
}

// ExecuteWorkflow executes a workflow with the given options
func (c *Client) ExecuteWorkflow(ctx context.Context, options WorkflowOptions, workflowName string, args ...interface{}) (client.WorkflowRun, error) {
	workflowOptions := client.StartWorkflowOptions{
		TaskQueue: options.TaskQueue,
	}

	if options.WorkflowID != "" {
		workflowOptions.ID = options.WorkflowID
	}

	if options.Timeout > 0 {
		workflowOptions.WorkflowExecutionTimeout = time.Duration(options.Timeout) * time.Second
	}

	return c.client.ExecuteWorkflow(ctx, workflowOptions, workflowName, args...)
}

// GetWorkflowResult gets the result of a completed workflow
func (c *Client) GetWorkflowResult(ctx context.Context, workflowID string, result interface{}) error {
	run := c.client.GetWorkflow(ctx, workflowID, "")
	return run.Get(ctx, result)
}

// CancelWorkflow cancels a running workflow
func (c *Client) CancelWorkflow(ctx context.Context, workflowID string) error {
	return c.client.CancelWorkflow(ctx, workflowID, "")
}

// TerminateWorkflow terminates a running workflow
func (c *Client) TerminateWorkflow(ctx context.Context, workflowID, reason string) error {
	return c.client.TerminateWorkflow(ctx, workflowID, "", reason)
}

// SignalWorkflow sends a signal to a running workflow
func (c *Client) SignalWorkflow(ctx context.Context, workflowID, signalName string, args ...interface{}) error {
	return c.client.SignalWorkflow(ctx, workflowID, "", signalName, nil)
}

// QueryWorkflow queries a running workflow
func (c *Client) QueryWorkflow(ctx context.Context, workflowID, queryType string, args ...interface{}) (interface{}, error) {
	var result interface{}
	_, err := c.client.QueryWorkflow(ctx, workflowID, "", queryType, &result)
	if err != nil {
		return nil, err
	}
	return result, nil
}

// ListWorkflows lists workflows matching the query
func (c *Client) ListWorkflows(ctx context.Context, query string) ([]WorkflowExecutionInfo, error) {
	// For now, return empty slice - full implementation would require workflow service client
	return []WorkflowExecutionInfo{}, nil
}

// GetWorkflowHistory gets the history of a workflow
func (c *Client) GetWorkflowHistory(ctx context.Context, workflowID string) ([]interface{}, error) {
	// Simplified implementation - returns empty slice for now
	return []interface{}{}, nil
}

// GetWorkflowInfo gets detailed information about a workflow execution
func (c *Client) GetWorkflowInfo(ctx context.Context, workflowID string) (*WorkflowExecutionInfo, error) {
	resp, err := c.client.DescribeWorkflowExecution(ctx, workflowID, "")
	if err != nil {
		return nil, fmt.Errorf("failed to describe workflow: %w", err)
	}
	return &WorkflowExecutionInfo{
		Execution: resp.WorkflowExecutionInfo.Execution,
		Type:      resp.WorkflowExecutionInfo.Type,
		StartTime: resp.WorkflowExecutionInfo.StartTime,
		CloseTime: resp.WorkflowExecutionInfo.CloseTime,
		Status:    resp.WorkflowExecutionInfo.Status,
	}, nil
}

// WaitForWorkflowCompletion waits for a workflow to complete and returns the result
func (c *Client) WaitForWorkflowCompletion(ctx context.Context, run client.WorkflowRun) (interface{}, error) {
	var result interface{}
	err := run.Get(ctx, &result)
	if err != nil {
		return nil, fmt.Errorf("workflow execution failed: %w", err)
	}
	return result, nil
}

// ScheduleWorkflow schedules a workflow to run at a future time
func (c *Client) ScheduleWorkflow(ctx context.Context, scheduleID, workflowType string, startTime time.Time, options WorkflowOptions, args ...interface{}) error {
	// Simplified implementation - scheduling not fully implemented yet
	return fmt.Errorf("schedule workflow not yet implemented")
}

// GetWorkflowStatus gets the current status of a workflow
func (c *Client) GetWorkflowStatus(ctx context.Context, workflowID string) (interface{}, error) {
	info, err := c.GetWorkflowInfo(ctx, workflowID)
	if err != nil {
		return nil, err
	}
	return info.Status, nil
}

// CreateActivityOptions creates activity options with common defaults
func CreateActivityOptions(timeout time.Duration) workflow.ActivityOptions {
	return workflow.ActivityOptions{
		StartToCloseTimeout: timeout,
	}
}

// CreateWorkflowOptions creates workflow options with common defaults
func CreateWorkflowOptions(taskQueue string, timeout time.Duration) WorkflowOptions {
	return WorkflowOptions{
		TaskQueue: taskQueue,
		Timeout:   int(timeout.Seconds()),
	}
}

// CreateChildWorkflowOptions creates options for child workflow execution
func CreateChildWorkflowOptions(taskQueue string, timeout time.Duration) workflow.ChildWorkflowOptions {
	return workflow.ChildWorkflowOptions{
		WorkflowExecutionTimeout: timeout,
		TaskQueue:                taskQueue,
	}
}

// ExecuteChildWorkflow executes a child workflow
func (c *Client) ExecuteChildWorkflow(ctx workflow.Context, options workflow.ChildWorkflowOptions, workflowName string, args ...interface{}) error {
	future := workflow.ExecuteChildWorkflow(ctx, options, args...)
	return future.Get(ctx, nil)
}
