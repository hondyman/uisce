# Workflow Orchestration Patterns for Temporal

This guide covers key workflow orchestration patterns that leverage Temporal for reliable, fault-tolerant, distributed systems. Each pattern includes conceptual descriptions, use cases, and Go implementation examples using Temporal.

## 1. Saga Pattern

### Overview
The Saga pattern manages distributed transactions by orchestrating a series of local transactions. If any step fails, compensating transactions undo previous steps, ensuring eventual consistency.

### Use Case
- **Order Processing**: Coordinate ReserveInventory → ProcessPayment → ShipOrder
- **Multi-service workflows**: Handle failures across multiple microservices gracefully
- **Long-running transactions**: Break monolithic transactions into manageable steps

### Key Benefits
- Maintains consistency without distributed locks
- Automatic rollback on failure
- Clear audit trail of compensating actions
- Decouples services via event-driven coordination

### Implementation (Go + Temporal)

```go
package workflows

import (
	"github.com/uber-go/tally"
	"go.temporal.io/sdk/workflow"
	"go.temporal.io/sdk/activity"
)

// OrderSagaWorkflow orchestrates order processing with compensation
func OrderSagaWorkflow(ctx workflow.Context, order Order) (Result, error) {
	var result Result
	var reservationID string
	var paymentID string

	// Step 1: Reserve Inventory
	err := workflow.ExecuteActivity(ctx, ReserveInventoryActivity, order).Get(ctx, &reservationID)
	if err != nil {
		return result, err
	}

	// Step 2: Process Payment (with compensation)
	err = workflow.ExecuteActivity(ctx, ProcessPaymentActivity, order, reservationID).Get(ctx, &paymentID)
	if err != nil {
		// Compensate: Release reservation
		workflow.ExecuteActivity(ctx, ReleaseInventoryActivity, reservationID)
		return result, err
	}

	// Step 3: Ship Order (with compensation)
	var shipmentID string
	err = workflow.ExecuteActivity(ctx, ShipOrderActivity, order, paymentID).Get(ctx, &shipmentID)
	if err != nil {
		// Compensate: Refund payment and release reservation
		workflow.ExecuteActivity(ctx, RefundPaymentActivity, paymentID)
		workflow.ExecuteActivity(ctx, ReleaseInventoryActivity, reservationID)
		return result, err
	}

	result.OrderID = order.ID
	result.ShipmentID = shipmentID
	result.Status = "completed"
	return result, nil
}

// Activity: Reserve Inventory
func ReserveInventoryActivity(ctx context.Context, order Order) (string, error) {
	// Call inventory service
	// Return reservation ID or error
	return "reservation-123", nil
}

// Activity: Process Payment
func ProcessPaymentActivity(ctx context.Context, order Order, reservationID string) (string, error) {
	// Call payment service
	// Return payment ID or error
	return "payment-456", nil
}

// Activity: Ship Order
func ShipOrderActivity(ctx context.Context, order Order, paymentID string) (string, error) {
	// Call shipping service
	// Return shipment ID or error
	return "shipment-789", nil
}

// Compensating Activities
func ReleaseInventoryActivity(ctx context.Context, reservationID string) error {
	// Cancel reservation
	return nil
}

func RefundPaymentActivity(ctx context.Context, paymentID string) error {
	// Initiate refund
	return nil
}
```

### Monitoring in Temporal Web UI
- View workflow execution history with all attempted steps
- See compensation activities in order
- Track retry attempts and backoff policy
- Export audit trail for compliance

---

## 2. Chained Workflow Pattern

### Overview
Chain multiple workflows sequentially, with each workflow triggering the next upon completion. Suitable for data pipelines, ETL processes, and sequential batch jobs.

### Use Case
- **Data Pipeline**: IngestDataWorkflow → TransformDataWorkflow → StoreDataWorkflow
- **ETL Processes**: Extract → Load → Transform
- **Batch Processing**: Validate → Process → Report

### Key Benefits
- Clear dependency chain
- Fault isolation per stage
- Progress tracking and resumability
- Easy to add intermediate steps

### Implementation (Go + Temporal)

```go
package workflows

import (
	"go.temporal.io/sdk/workflow"
)

// DataPipelineWorkflow chains three data processing workflows
func DataPipelineWorkflow(ctx workflow.Context, dataSource string) (Result, error) {
	var result Result

	// Step 1: Ingest Data
	var ingestOutput IngestOutput
	err := workflow.ExecuteChildWorkflow(
		ctx,
		IngestDataWorkflow,
		dataSource,
	).Get(ctx, &ingestOutput)
	if err != nil {
		return result, err
	}

	// Step 2: Transform Data (depends on ingest output)
	var transformOutput TransformOutput
	err = workflow.ExecuteChildWorkflow(
		ctx,
		TransformDataWorkflow,
		ingestOutput.DataPath,
	).Get(ctx, &transformOutput)
	if err != nil {
		return result, err
	}

	// Step 3: Store Results (depends on transform output)
	var storeOutput StoreOutput
	err = workflow.ExecuteChildWorkflow(
		ctx,
		StoreDataWorkflow,
		transformOutput.TransformedPath,
	).Get(ctx, &storeOutput)
	if err != nil {
		return result, err
	}

	result.FinalPath = storeOutput.StoragePath
	result.RecordsProcessed = transformOutput.RecordCount
	result.Status = "completed"
	return result, nil
}

// Individual workflows
func IngestDataWorkflow(ctx workflow.Context, dataSource string) (IngestOutput, error) {
	var output IngestOutput
	var result string
	err := workflow.ExecuteActivity(ctx, IngestActivity, dataSource).Get(ctx, &result)
	if err != nil {
		return output, err
	}
	output.DataPath = result
	return output, nil
}

func TransformDataWorkflow(ctx workflow.Context, dataPath string) (TransformOutput, error) {
	var output TransformOutput
	err := workflow.ExecuteActivity(ctx, TransformActivity, dataPath).Get(ctx, &output)
	if err != nil {
		return output, err
	}
	return output, nil
}

func StoreDataWorkflow(ctx workflow.Context, transformedPath string) (StoreOutput, error) {
	var output StoreOutput
	err := workflow.ExecuteActivity(ctx, StoreActivity, transformedPath).Get(ctx, &output)
	if err != nil {
		return output, err
	}
	return output, nil
}

// Supporting structs
type IngestOutput struct {
	DataPath string
}

type TransformOutput struct {
	TransformedPath string
	RecordCount     int
}

type StoreOutput struct {
	StoragePath string
}

type Result struct {
	FinalPath       string
	RecordsProcessed int
	Status          string
}
```

### Scheduling with Cron
Chain workflows on a schedule:

```go
// Temporal cron expression
cronSchedule := "@daily" // Execute daily at midnight

workflowOptions := &client.StartWorkflowOptions{
	CronSchedule: cronSchedule,
	// other options
}

client.ExecuteWorkflow(ctx, workflowOptions, DataPipelineWorkflow, "s3://data-source")
```

---

## 3. Fan-Out/Fan-In Pattern

### Overview
Execute multiple parallel workflows or activities, then gather results. Useful for parallel processing, batch analysis, and concurrent data fetching.

### Use Case
- **Parallel Processing**: Split 1000 items into 10 parallel workers
- **Data Aggregation**: Fetch from N services in parallel
- **Batch Analysis**: Analyze multiple datasets concurrently

### Key Benefits
- Horizontal scaling through parallelism
- Reduced total execution time
- Automatic result aggregation
- Built-in error handling per worker

### Implementation (Go + Temporal)

```go
package workflows

import (
	"fmt"
	"go.temporal.io/sdk/workflow"
)

// BulkProcessingWorkflow processes many items in parallel
func BulkProcessingWorkflow(ctx workflow.Context, items []Item) (BulkResult, error) {
	var result BulkResult

	// Parallel processing options
	opts := workflow.ParallelActivityOptions{}
	cctx, cancel := workflow.WithCancel(ctx)
	defer cancel()

	// Create parallel activities for each item
	var futures []workflow.Future
	for _, item := range items {
		f := workflow.ExecuteActivity(cctx, ProcessItemActivity, item)
		futures = append(futures, f)
	}

	// Wait for all to complete (fan-in)
	var itemResults []ItemResult
	var failedItems []string

	for i, f := range futures {
		var itemResult ItemResult
		if err := f.Get(cctx, &itemResult); err != nil {
			failedItems = append(failedItems, items[i].ID)
			continue
		}
		itemResults = append(itemResults, itemResult)
	}

	result.SuccessCount = len(itemResults)
	result.FailureCount = len(failedItems)
	result.Results = itemResults
	result.FailedIDs = failedItems

	return result, nil
}

// Alternative: Chunked parallel processing (process in batches)
func ChunkedBulkProcessingWorkflow(ctx workflow.Context, items []Item, chunkSize int) (BulkResult, error) {
	var result BulkResult

	// Split into chunks
	chunks := make([][]Item, 0)
	for i := 0; i < len(items); i += chunkSize {
		end := i + chunkSize
		if end > len(items) {
			end = len(items)
		}
		chunks = append(chunks, items[i:end])
	}

	// Process each chunk in parallel
	var chunkFutures []workflow.Future
	for _, chunk := range chunks {
		f := workflow.ExecuteActivity(ctx, ProcessChunkActivity, chunk)
		chunkFutures = append(chunkFutures, f)
	}

	// Aggregate results
	var allResults []ItemResult
	for _, f := range chunkFutures {
		var chunkResults []ItemResult
		if err := f.Get(ctx, &chunkResults); err != nil {
			continue
		}
		allResults = append(allResults, chunkResults...)
	}

	result.SuccessCount = len(allResults)
	result.Results = allResults
	return result, nil
}

// Activity: Process single item
func ProcessItemActivity(ctx context.Context, item Item) (ItemResult, error) {
	// Process item
	result := ItemResult{
		ItemID: item.ID,
		Status: "processed",
	}
	return result, nil
}

// Activity: Process chunk of items
func ProcessChunkActivity(ctx context.Context, items []Item) ([]ItemResult, error) {
	var results []ItemResult
	for _, item := range items {
		result := ItemResult{
			ItemID: item.ID,
			Status: "processed",
		}
		results = append(results, result)
	}
	return results, nil
}

// Supporting structs
type Item struct {
	ID   string
	Data interface{}
}

type ItemResult struct {
	ItemID string
	Status string
	Data   interface{}
}

type BulkResult struct {
	SuccessCount int
	FailureCount int
	Results      []ItemResult
	FailedIDs    []string
}
```

---

## 4. Retry and Exponential Backoff Pattern

### Overview
Automatically retry failed activities with exponential backoff, jitter, and maximum retry limits for resilience against transient failures.

### Use Case
- **API Calls**: Retry on temporary network failures
- **Database Operations**: Handle transient connection issues
- **External Service Calls**: Graceful degradation during service unavailability

### Key Benefits
- Handles transient failures automatically
- Prevents overwhelming failing services (backoff)
- Configurable retry policies per activity
- Audit trail of all retry attempts

### Implementation (Go + Temporal)

```go
package workflows

import (
	"go.temporal.io/sdk/activity"
	"go.temporal.io/sdk/workflow"
	"time"
)

// WorkflowWithRetry demonstrates retry logic
func WorkflowWithRetry(ctx workflow.Context, request APIRequest) (Response, error) {
	var response Response

	// Define retry policy
	retryPolicy := &temporal.RetryPolicy{
		InitialInterval:    time.Second,
		BackoffCoefficient: 2.0,
		MaximumInterval:    time.Minute,
		MaximumAttempts:    5,
		NonRetryableErrors: []string{"InvalidRequest", "Unauthorized"},
	}

	// Create activity options with retry policy
	activityOptions := workflow.ActivityOptions{
		ScheduleToCloseTimeout: time.Minute * 5,
		RetryPolicy:            retryPolicy,
	}
	ctx = workflow.WithActivityOptions(ctx, activityOptions)

	// Execute activity with automatic retry
	err := workflow.ExecuteActivity(
		ctx,
		CallExternalAPIActivity,
		request,
	).Get(ctx, &response)

	if err != nil {
		return response, err
	}

	return response, nil
}

// Activity with context for retry information
func CallExternalAPIActivity(ctx context.Context, request APIRequest) (Response, error) {
	// Get retry information
	info := activity.GetInfo(ctx)
	attempt := info.Attempt

	// Log attempt (visible in Temporal Web UI)
	activity.RecordHeartbeat(ctx, fmt.Sprintf("Attempt %d", attempt))

	// Call external API
	resp, err := callAPIWithTimeout(request, 10*time.Second)
	if err != nil {
		// Return error to trigger retry (unless in NonRetryableErrors)
		return Response{}, err
	}

	return resp, nil
}

type APIRequest struct {
	URL string
	Headers map[string]string
}

type Response struct {
	StatusCode int
	Body       string
}

// Helper to call API with timeout
func callAPIWithTimeout(req APIRequest, timeout time.Duration) (Response, error) {
	// Actual implementation would use http.Client
	return Response{StatusCode: 200}, nil
}
```

### Retry Policy Configuration

```yaml
# Example in config
activities:
  external_api_call:
    timeout: 5m
    retry:
      initial_interval: 1s
      backoff_coefficient: 2.0
      maximum_interval: 1m
      maximum_attempts: 5
      non_retryable_errors:
        - "InvalidRequest"
        - "Unauthorized"
        - "Forbidden"
```

---

## 5. Event-Sourcing Pattern

### Overview
Use Temporal signals to update a long-running workflow based on external events. Ideal for order processing, approval workflows, and state machines.

### Use Case
- **Order Workflow**: Listen for OrderApproved, OrderShipped, OrderCancelled signals
- **Approval Workflows**: Collect multiple approval signals before proceeding
- **State Machines**: Transition states based on received events

### Key Benefits
- Decoupled event producers from workflow logic
- Long-running stateful workflows
- Clear event audit trail
- Easy to extend with new event types

### Implementation (Go + Temporal)

```go
package workflows

import (
	"errors"
	"go.temporal.io/sdk/workflow"
	"time"
)

// OrderWorkflow demonstrates event-sourcing pattern
func OrderWorkflow(ctx workflow.Context, orderID string) (OrderStatus, error) {
	status := OrderStatus{
		OrderID: orderID,
		State:   "pending",
		Events:  []OrderEvent{},
	}

	// Define signals
	approvalSignal := workflow.GetSignalChannel(ctx, "order_approved")
	shipSignal := workflow.GetSignalChannel(ctx, "order_shipped")
	cancelSignal := workflow.GetSignalChannel(ctx, "order_cancelled")

	// Set timeout for entire workflow
	ctx, cancel := workflow.WithTimeout(ctx, time.Hour*24)
	defer cancel()

	for {
		selector := workflow.NewSelector(ctx)

		// Handle approval signal
		selector.AddReceive(approvalSignal, func(c workflow.ReceiveChannel, more bool) {
			var event OrderEvent
			c.Receive(ctx, &event)
			status.State = "approved"
			status.Events = append(status.Events, event)
			
			// Trigger shipping
			workflow.ExecuteActivity(ctx, ShipOrderActivity, orderID).Get(ctx, nil)
		})

		// Handle shipped signal
		selector.AddReceive(shipSignal, func(c workflow.ReceiveChannel, more bool) {
			var event OrderEvent
			c.Receive(ctx, &event)
			status.State = "shipped"
			status.Events = append(status.Events, event)
		})

		// Handle cancel signal
		selector.AddReceive(cancelSignal, func(c workflow.ReceiveChannel, more bool) {
			var event OrderEvent
			c.Receive(ctx, &event)
			status.State = "cancelled"
			status.Events = append(status.Events, event)
			
			// Trigger refund
			workflow.ExecuteActivity(ctx, RefundOrderActivity, orderID).Get(ctx, nil)
		})

		// Check for final state
		selector.AddDefault(func() {
			// Periodic check or exit condition
			if status.State == "shipped" || status.State == "cancelled" {
				selector.Done()
			}
		})

		selector.Select(ctx)

		// Exit loop on final states
		if status.State == "shipped" || status.State == "cancelled" {
			break
		}
	}

	return status, nil
}

// Client-side: Send signals to workflow
func SendOrderSignal(client client.Client, ctx context.Context, orderID string, signalName string, event OrderEvent) error {
	workflowID := fmt.Sprintf("order-%s", orderID)
	return client.SignalWorkflow(ctx, workflowID, "", signalName, event)
}

// Supporting types
type OrderEvent struct {
	Type      string    `json:"type"`
	Timestamp time.Time `json:"timestamp"`
	Data      map[string]interface{} `json:"data"`
}

type OrderStatus struct {
	OrderID string
	State   string
	Events  []OrderEvent
}
```

### Signal Examples

```go
// Example: Send approval signal
event := OrderEvent{
	Type:      "approved",
	Timestamp: time.Now(),
	Data: map[string]interface{}{
		"approvedBy": "admin-user",
	},
}
SendOrderSignal(client, ctx, "order-123", "order_approved", event)

// Example: Send shipment signal
shipEvent := OrderEvent{
	Type:      "shipped",
	Timestamp: time.Now(),
	Data: map[string]interface{}{
		"trackingNumber": "TRACK-789",
		"carrier":        "FedEx",
	},
}
SendOrderSignal(client, ctx, "order-123", "order_shipped", shipEvent)
```

---

## 6. Query Pattern

### Overview
Query the state of long-running workflows without stopping them, enabling real-time status monitoring and dashboards.

### Use Case
- **Progress Tracking**: Query workflow progress during execution
- **Status Dashboards**: Real-time status updates for dashboards
- **Admin Tools**: Administrators check workflow state

### Implementation (Go + Temporal)

```go
package workflows

import "go.temporal.io/sdk/workflow"

// WorkflowWithQuery demonstrates query capability
func WorkflowWithQuery(ctx workflow.Context, dataSize int) (Result, error) {
	status := QueryableStatus{
		TotalItems:    dataSize,
		ProcessedItems: 0,
		Status:        "running",
	}

	// Register query handler
	err := workflow.SetQueryHandler(ctx, "get_status", func() (QueryableStatus, error) {
		return status, nil
	})
	if err != nil {
		return Result{}, err
	}

	// Long-running process
	for i := 0; i < dataSize; i++ {
		// Do work
		workflow.ExecuteActivity(ctx, ProcessItemActivity, i)

		// Update status
		status.ProcessedItems = i + 1
		status.Percentage = float64(status.ProcessedItems) / float64(status.TotalItems) * 100
	}

	status.Status = "completed"
	return Result{Status: status}, nil
}

type QueryableStatus struct {
	TotalItems     int
	ProcessedItems int
	Status         string
	Percentage     float64
}

type Result struct {
	Status QueryableStatus
}
```

### Client-side Query

```go
// Query workflow without affecting execution
resp, err := client.QueryWorkflow(ctx, workflowID, "", "get_status")
if err != nil {
	log.Fatal(err)
}

var status QueryableStatus
err = resp.Get(&status)
fmt.Printf("Progress: %d/%d (%.1f%%)\n", status.ProcessedItems, status.TotalItems, status.Percentage)
```

---

## Summary: Choosing the Right Pattern

| Pattern | Use Case | Key Feature |
|---------|----------|-------------|
| **Saga** | Distributed transactions | Compensating transactions |
| **Chained** | Sequential pipelines | Clear dependency chain |
| **Fan-Out/Fan-In** | Parallel processing | Concurrent execution |
| **Retry** | Resilient operations | Automatic retry with backoff |
| **Event-Sourcing** | Long-running workflows | Event-driven state updates |
| **Query** | Progress monitoring | Real-time status visibility |

Each pattern can be combined for complex workflows. Temporal's durability guarantee ensures workflows survive failures and continue from the last checkpoint.
