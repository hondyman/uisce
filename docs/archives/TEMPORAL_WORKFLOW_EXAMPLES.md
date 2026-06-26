# Temporal Workflow Examples for Northwind

## Overview

These are reference implementations of Temporal workflows for Northwind. Copy these into a separate package when integrating Temporal with your system.

## Installation

```bash
go get go.temporal.io/sdk
```

## Code Examples

```go
// ============================================================================
// ORDER PROCESSING WORKFLOW - Example 1
// ============================================================================

type OrderInput struct {
	OrderID    string
	Total      float64
	CustomerID string
}

type OrderResult struct {
	Status string
	Error  string
}

// OrderProcessingWorkflow orchestrates the order processing steps
// This workflow ensures orders go through validation, approval, and notification
func OrderProcessingWorkflow(ctx workflow.Context, input OrderInput) (OrderResult, error) {
	options := workflow.ActivityOptions{
		StartToCloseTimeout: 5 * time.Minute,
		RetryPolicy: &workflow.RetryPolicy{
			InitialInterval:    time.Second,
			BackoffCoefficient: 2,
			MaximumInterval:    time.Minute,
			MaximumAttempts:    3,
		},
	}
	ctx = workflow.WithActivityOptions(ctx, options)

	// Step 1: Validate Order
	var validationResult string
	err := workflow.ExecuteActivity(ctx, ValidateOrderActivity, input).Get(ctx, &validationResult)
	if err != nil {
		return OrderResult{Status: "failed", Error: fmt.Sprintf("Validation failed: %v", err)}, err
	}

	workflow.GetLogger(ctx).Info("Order validated", "order_id", input.OrderID, "result", validationResult)

	// Step 2: Apply Business Logic
	var approvalResult string
	err = workflow.ExecuteActivity(ctx, ApproveOrderActivity, input).Get(ctx, &approvalResult)
	if err != nil {
		return OrderResult{Status: "failed", Error: fmt.Sprintf("Approval failed: %v", err)}, err
	}

	workflow.GetLogger(ctx).Info("Order approved", "order_id", input.OrderID, "approval", approvalResult)

	// Step 3: Notify Shipping Department
	var notificationResult string
	err = workflow.ExecuteActivity(ctx, NotifyShippingActivity, input).Get(ctx, &notificationResult)
	if err != nil {
		workflow.GetLogger(ctx).Warn("Notification failed (non-blocking)", "order_id", input.OrderID)
		// Continue workflow even if notification fails
	}

	workflow.GetLogger(ctx).Info("Order processing completed", "order_id", input.OrderID)

	return OrderResult{Status: "completed", Error: ""}, nil
}

// ValidateOrderActivity validates the order data
func ValidateOrderActivity(ctx context.Context, input OrderInput) (string, error) {
	activity.GetLogger(ctx).Info("Validating order", "order_id", input.OrderID, "total", input.Total)

	if input.Total < 0 {
		return "", fmt.Errorf("invalid order total: %f", input.Total)
	}

	if input.CustomerID == "" {
		return "", fmt.Errorf("customer ID is required")
	}

	return "order_validated", nil
}

// ApproveOrderActivity applies approval logic
func ApproveOrderActivity(ctx context.Context, input OrderInput) (string, error) {
	activity.GetLogger(ctx).Info("Approving order", "order_id", input.OrderID)

	// Business rule: orders >= $1000 auto-approved, otherwise require manual approval
	if input.Total >= 1000 {
		activity.GetLogger(ctx).Info("Order auto-approved (high value)", "order_id", input.OrderID, "total", input.Total)
		return "auto_approved", nil
	}

	activity.GetLogger(ctx).Info("Order requires manual approval (low value)", "order_id", input.OrderID)
	return "pending_approval", nil
}

// NotifyShippingActivity sends order to shipping queue
func NotifyShippingActivity(ctx context.Context, input OrderInput) (string, error) {
	activity.GetLogger(ctx).Info("Notifying shipping department", "order_id", input.OrderID)
	// In reality, this would publish to RabbitMQ or call another service
	return "shipping_notified", nil
}

// ============================================================================
// EMPLOYEE HIRE WORKFLOW - Example 2
// ============================================================================

type EmployeeInput struct {
	EmployeeID string
	FirstName  string
	LastName   string
	Title      string
	Department string
	HireDate   time.Time
}

type EmployeeResult struct {
	Status string
	Error  string
}

// EmployeeHireWorkflow orchestrates the employee onboarding process
// Includes background check, HR record creation, IT provisioning, and welcome
func EmployeeHireWorkflow(ctx workflow.Context, input EmployeeInput) (EmployeeResult, error) {
	options := workflow.ActivityOptions{
		StartToCloseTimeout: 10 * time.Minute,
		RetryPolicy: &workflow.RetryPolicy{
			InitialInterval:    time.Second,
			BackoffCoefficient: 2,
			MaximumInterval:    time.Minute,
			MaximumAttempts:    2,
		},
	}
	ctx = workflow.WithActivityOptions(ctx, options)

	// Step 1: Conduct Background Check
	var bgCheckResult string
	err := workflow.ExecuteActivity(ctx, BackgroundCheckActivity, input).Get(ctx, &bgCheckResult)
	if err != nil {
		return EmployeeResult{Status: "rejected", Error: fmt.Sprintf("Background check failed: %v", err)}, err
	}

	workflow.GetLogger(ctx).Info("Background check completed", "employee_id", input.EmployeeID, "result", bgCheckResult)

	// Step 2: Create HR Record
	var hrResult string
	err = workflow.ExecuteActivity(ctx, CreateHRRecordActivity, input).Get(ctx, &hrResult)
	if err != nil {
		return EmployeeResult{Status: "failed", Error: fmt.Sprintf("HR record creation failed: %v", err)}, err
	}

	workflow.GetLogger(ctx).Info("HR record created", "employee_id", input.EmployeeID)

	// Step 3: Provision IT Equipment (parallel with welcome email)
	var itResult, emailResult string
	
	itFuture := workflow.ExecuteActivity(ctx, ProvisionITEquipmentActivity, input)
	emailFuture := workflow.ExecuteActivity(ctx, SendWelcomeEmailActivity, input)

	if err = itFuture.Get(ctx, &itResult); err != nil {
		workflow.GetLogger(ctx).Warn("IT provisioning failed", "employee_id", input.EmployeeID)
	}

	if err = emailFuture.Get(ctx, &emailResult); err != nil {
		workflow.GetLogger(ctx).Warn("Welcome email failed", "employee_id", input.EmployeeID)
	}

	workflow.GetLogger(ctx).Info("Employee onboarding completed", "employee_id", input.EmployeeID)

	return EmployeeResult{Status: "completed", Error: ""}, nil
}

// BackgroundCheckActivity performs background check
func BackgroundCheckActivity(ctx context.Context, input EmployeeInput) (string, error) {
	activity.GetLogger(ctx).Info("Running background check", "employee_id", input.EmployeeID)
	// Simulate background check (in reality, call external service)
	time.Sleep(2 * time.Second)
	return "check_passed", nil
}

// CreateHRRecordActivity creates employee HR record
func CreateHRRecordActivity(ctx context.Context, input EmployeeInput) (string, error) {
	activity.GetLogger(ctx).Info("Creating HR record", "employee_id", input.EmployeeID, "name", input.FirstName+" "+input.LastName)
	return "hr_record_created", nil
}

// ProvisionITEquipmentActivity provisions IT equipment
func ProvisionITEquipmentActivity(ctx context.Context, input EmployeeInput) (string, error) {
	activity.GetLogger(ctx).Info("Provisioning IT equipment", "employee_id", input.EmployeeID)
	time.Sleep(1 * time.Second)
	return "it_equipment_provisioned", nil
}

// SendWelcomeEmailActivity sends welcome email
func SendWelcomeEmailActivity(ctx context.Context, input EmployeeInput) (string, error) {
	activity.GetLogger(ctx).Info("Sending welcome email", "employee_id", input.EmployeeID)
	return "welcome_email_sent", nil
}

// ============================================================================
// PRODUCT INVENTORY UPDATE WORKFLOW - Example 3
// ============================================================================

type InventoryInput struct {
	ProductID   string
	ProductName string
	StockChange int
	Reason      string
}

type InventoryResult struct {
	Status string
	Error  string
}

// ProductInventoryWorkflow orchestrates inventory updates
// Validates stock levels, updates inventory, checks reordering, sends notifications
func ProductInventoryWorkflow(ctx workflow.Context, input InventoryInput) (InventoryResult, error) {
	options := workflow.ActivityOptions{
		StartToCloseTimeout: 2 * time.Minute,
		RetryPolicy: &workflow.RetryPolicy{
			InitialInterval:    time.Second,
			BackoffCoefficient: 2,
			MaximumInterval:    time.Minute,
			MaximumAttempts:    3,
		},
	}
	ctx = workflow.WithActivityOptions(ctx, options)

	// Step 1: Validate Stock Levels
	var validationResult string
	err := workflow.ExecuteActivity(ctx, ValidateStockLevelsActivity, input).Get(ctx, &validationResult)
	if err != nil {
		return InventoryResult{Status: "failed", Error: fmt.Sprintf("Stock validation failed: %v", err)}, err
	}

	workflow.GetLogger(ctx).Info("Stock validation passed", "product_id", input.ProductID)

	// Step 2: Update Inventory
	var updateResult string
	err = workflow.ExecuteActivity(ctx, UpdateInventoryActivity, input).Get(ctx, &updateResult)
	if err != nil {
		return InventoryResult{Status: "failed", Error: fmt.Sprintf("Inventory update failed: %v", err)}, err
	}

	workflow.GetLogger(ctx).Info("Inventory updated", "product_id", input.ProductID)

	// Step 3: Check if Reordering Required
	var reorderResult string
	err = workflow.ExecuteActivity(ctx, CheckReorderingActivity, input).Get(ctx, &reorderResult)
	if err != nil {
		workflow.GetLogger(ctx).Warn("Reorder check failed", "product_id", input.ProductID)
	}

	// Step 4: Send Notification
	var notificationResult string
	err = workflow.ExecuteActivity(ctx, SendInventoryNotificationActivity, input).Get(ctx, &notificationResult)
	if err != nil {
		workflow.GetLogger(ctx).Warn("Notification failed", "product_id", input.ProductID)
	}

	workflow.GetLogger(ctx).Info("Inventory workflow completed", "product_id", input.ProductID)

	return InventoryResult{Status: "completed", Error: ""}, nil
}

// ValidateStockLevelsActivity validates stock adjustment
func ValidateStockLevelsActivity(ctx context.Context, input InventoryInput) (string, error) {
	activity.GetLogger(ctx).Info("Validating stock levels", "product_id", input.ProductID, "change", input.StockChange)

	if input.StockChange > 999999 || input.StockChange < -999999 {
		return "", fmt.Errorf("invalid stock adjustment: %d", input.StockChange)
	}

	return "stock_valid", nil
}

// UpdateInventoryActivity updates inventory in database
func UpdateInventoryActivity(ctx context.Context, input InventoryInput) (string, error) {
	activity.GetLogger(ctx).Info("Updating inventory", "product_id", input.ProductID, "change", input.StockChange, "reason", input.Reason)
	// In reality, this would update the database
	return "inventory_updated", nil
}

// CheckReorderingActivity checks if reordering is needed
func CheckReorderingActivity(ctx context.Context, input InventoryInput) (string, error) {
	activity.GetLogger(ctx).Info("Checking if reordering needed", "product_id", input.ProductID)

	if input.StockChange < -100 {
		activity.GetLogger(ctx).Info("Reorder needed", "product_id", input.ProductID)
		return "reorder_needed", nil
	}

	return "reorder_not_needed", nil
}

// SendInventoryNotificationActivity sends notification
func SendInventoryNotificationActivity(ctx context.Context, input InventoryInput) (string, error) {
	activity.GetLogger(ctx).Info("Sending inventory notification", "product_id", input.ProductID)
	return "notification_sent", nil
}

// ============================================================================
// WORKER REGISTRATION - How to Use These Workflows
// ============================================================================

/*
import (
	"log"
	"go.temporal.io/sdk/client"
	"go.temporal.io/sdk/worker"
)

func registerWorkflows() {
	c, err := client.Dial(client.Options{
		HostPort: client.DefaultHostPort, // localhost:7233
	})
	if err != nil {
		log.Fatalln("Unable to create Temporal client", err)
	}
	defer c.Close()

	w := worker.New(c, "northwind_queue", worker.Options{})
	
	// Register Order Processing Workflow
	w.RegisterWorkflow(OrderProcessingWorkflow)
	w.RegisterActivity(ValidateOrderActivity)
	w.RegisterActivity(ApproveOrderActivity)
	w.RegisterActivity(NotifyShippingActivity)
	
	// Register Employee Hire Workflow
	w.RegisterWorkflow(EmployeeHireWorkflow)
	w.RegisterActivity(BackgroundCheckActivity)
	w.RegisterActivity(CreateHRRecordActivity)
	w.RegisterActivity(ProvisionITEquipmentActivity)
	w.RegisterActivity(SendWelcomeEmailActivity)
	
	// Register Inventory Workflow
	w.RegisterWorkflow(ProductInventoryWorkflow)
	w.RegisterActivity(ValidateStockLevelsActivity)
	w.RegisterActivity(UpdateInventoryActivity)
	w.RegisterActivity(CheckReorderingActivity)
	w.RegisterActivity(SendInventoryNotificationActivity)

	err = w.Run(worker.InterruptCh())
	if err != nil {
		log.Fatalln("Unable to start Temporal worker", err)
	}
}
*/
