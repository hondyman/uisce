package trade

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
	"go.temporal.io/sdk/workflow"
)

// Signal names
const (
	SignalApprovePreCompliance = "approve_pre_compliance"
	SignalOrderFulfillment     = "order_fulfillment"
)

// Activity names
const (
	ActivityRunPreCompliance = "RunPreCompliance"
	ActivityCreateTradeOrder = "CreateTradeOrder"
	ActivityPostTradeAudit   = "PostTradeAudit"
)

// WorkflowResult represents the final output of the workflow
type WorkflowResult struct {
	TradeID string
	Status  string
}

// ComplianceResult is the result of the pre-compliance check
type ComplianceResult struct {
	Passed  bool
	Reasons []string
}

// OrderResponse is the result of creating an order
type OrderResponse struct {
	OrderID string
	Status  string
}

// OrderFill represents an order fill event
type OrderFill struct {
	Filled    bool
	FillPrice float64
	Quantity  float64
}

// TradeWorkflow is the main orchestration workflow
func TradeWorkflow(ctx workflow.Context, input TradeInput) (*WorkflowResult, error) {
	ao := workflow.ActivityOptions{
		StartToCloseTimeout: time.Minute * 5,
	}
	ctx = workflow.WithActivityOptions(ctx, ao)

	logger := workflow.GetLogger(ctx)
	logger.Info("Starting Trade Workflow", "TenantID", input.TenantID)

	// 1. Pre-Trade Compliance
	var preCompResult ComplianceResult
	err := workflow.ExecuteActivity(ctx, ActivityRunPreCompliance, input).Get(ctx, &preCompResult)
	if err != nil {
		return nil, err
	}

	if !preCompResult.Passed {
		logger.Info("Compliance failed, waiting for approval signal")
		// Await (human or AI) approval override as a Signal
		var approved bool
		signalChan := workflow.GetSignalChannel(ctx, SignalApprovePreCompliance)

		// Wait for signal or timeout (e.g., 24 hours)
		selector := workflow.NewSelector(ctx)
		selector.AddReceive(signalChan, func(c workflow.ReceiveChannel, more bool) {
			c.Receive(ctx, &approved)
		})

		// Add a timeout for approval
		timerFuture := workflow.NewTimer(ctx, time.Hour*24)
		selector.AddFuture(timerFuture, func(f workflow.Future) {
			logger.Info("Approval timed out")
		})

		selector.Select(ctx)

		if !approved {
			return nil, errors.New("compliance failed and not approved")
		}
		logger.Info("Compliance override approved")
	}

	// 2. Create Trade Order
	var orderResp OrderResponse
	err = workflow.ExecuteActivity(ctx, ActivityCreateTradeOrder, input).Get(ctx, &orderResp)
	if err != nil {
		return nil, err
	}
	logger.Info("Trade Order Created", "OrderID", orderResp.OrderID)

	// 3. Await fulfillment (could be API, broker, or signal from external system)
	var orderFill OrderFill
	fillSignalChan := workflow.GetSignalChannel(ctx, SignalOrderFulfillment)

	logger.Info("Waiting for Order Fulfillment")
	fillSelector := workflow.NewSelector(ctx)
	fillSelector.AddReceive(fillSignalChan, func(c workflow.ReceiveChannel, more bool) {
		c.Receive(ctx, &orderFill)
	})

	// Wait for fill (e.g., 1 hour for market order, longer for limit)
	// For simplicity, using a long timeout
	fillTimer := workflow.NewTimer(ctx, time.Hour*48)
	fillSelector.AddFuture(fillTimer, func(f workflow.Future) {
		logger.Info("Order fulfillment timed out")
	})

	fillSelector.Select(ctx)

	if !orderFill.Filled {
		return nil, errors.New("order not filled within timeout")
	}
	logger.Info("Order Filled", "Price", orderFill.FillPrice)

	// 4. Post-Trade Audit
	err = workflow.ExecuteActivity(ctx, ActivityPostTradeAudit, orderFill).Get(ctx, nil)
	if err != nil {
		return nil, err
	}

	return &WorkflowResult{
		TradeID: orderResp.OrderID,
		Status:  "Completed",
	}, nil
}

// Activities (Stubs for now, would integrate with real services)

type Activities struct{}

func (a *Activities) RunPreCompliance(ctx context.Context, input TradeInput) (*ComplianceResult, error) {
	// Logic to check compliance rules from DB
	// For demo, pass if "qty" < 1000
	// We'd parse input.Data to get qty
	return &ComplianceResult{Passed: true}, nil // Default pass for now
}

func (a *Activities) CreateTradeOrder(ctx context.Context, input TradeInput) (*OrderResponse, error) {
	// Logic to send order to broker/exchange
	return &OrderResponse{OrderID: "ORD-" + uuid.New().String(), Status: "New"}, nil
}

func (a *Activities) PostTradeAudit(ctx context.Context, fill OrderFill) error {
	// Logic to record audit trail
	return nil
}
