package trading

import (
	"context"
	"fmt"
	"time"

	"go.temporal.io/sdk/workflow"
)

// Order represents a trading order
type Order struct {
	OrderID   string  `json:"order_id"`
	Symbol    string  `json:"symbol"`
	Quantity  float64 `json:"quantity"`
	Side      string  `json:"side"` // Buy/Sell
	Price     float64 `json:"price"`
	Status    string  `json:"status"`
}

// OrderEntryWorkflow handles the lifecycle of an order
func OrderEntryWorkflow(ctx workflow.Context, order Order) (*Order, error) {
	logger := workflow.GetLogger(ctx)
	logger.Info("OrderEntryWorkflow Started", "OrderID", order.OrderID)

	// Step 1: Validate Order (Fast, synchronous-like)
	// In a real UpdateWithStart scenario, this validation might happen in the Update handler
	// or the workflow starts and immediately runs this.
	if order.Quantity <= 0 {
		return nil, fmt.Errorf("invalid quantity")
	}
	order.Status = "Validated"

	// Step 2: Send to Market (FIX)
	// Activity: SendFixNewOrderSingle
	err := workflow.ExecuteActivity(workflow.WithActivityOptions(ctx, workflow.ActivityOptions{
		StartToCloseTimeout: time.Second * 10,
	}), SendFixNewOrderSingle, order).Get(ctx, nil)
	if err != nil {
		return nil, err
	}
	order.Status = "SentToMarket"

	// Step 3: Wait for Execution Report (Fill)
	// We wait for a signal "ExecutionReport"
	var executionReport string
	signalChan := workflow.GetSignalChannel(ctx, "ExecutionReport")
	
	// Wait for fill or timeout
	selector := workflow.NewSelector(ctx)
	selector.AddReceive(signalChan, func(c workflow.ReceiveChannel, more bool) {
		c.Receive(ctx, &executionReport)
		logger.Info("Received Execution Report", "Report", executionReport)
		order.Status = "Filled"
	})
	
	// Timeout if no fill received in reasonable time (e.g. Day Order)
	// For demo, short timeout
	timer := workflow.NewTimer(ctx, time.Minute*5)
	selector.AddFuture(timer, func(f workflow.Future) {
		logger.Info("Order Timed Out")
		order.Status = "TimedOut"
		// Logic to cancel order...
	})

	selector.Select(ctx)

	logger.Info("OrderEntryWorkflow Completed", "FinalStatus", order.Status)
	return &order, nil
}

// --- Activities ---

func SendFixNewOrderSingle(ctx context.Context, order Order) error {
	fmt.Printf("Sending FIX NewOrderSingle: %v\n", order)
	// Mock FIX engine interaction
	return nil
}

// --- SignalWithStart Logic (Conceptual) ---
// This logic usually resides in the Consumer (Worker), not the Workflow definition itself.
// The consumer reads from RabbitMQ and calls client.SignalWithStartWorkflow.
