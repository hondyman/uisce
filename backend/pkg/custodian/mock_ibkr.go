package custodian

import (
	"context"
	"fmt"
	"time"
)

// MockIBKR simulates an Interactive Brokers adapter
type MockIBKR struct{}

func (m MockIBKR) PlaceOrder(ctx context.Context, req OrderRequest) (OrderResponse, error) {
	// Simulate network latency
	time.Sleep(30 * time.Millisecond)
	
	// Always fill for demo purposes
	return OrderResponse{
		OrderID:   "ibkr_" + req.ClientOrderID,
		Status:    "FILLED",
		FilledQty: req.Qty,
	}, nil
}

func (m MockIBKR) CancelOrder(ctx context.Context, orderID string) (OrderResponse, error) {
	return OrderResponse{OrderID: orderID, Status: "CANCELED"}, nil
}

func (m MockIBKR) GetOrderStatus(ctx context.Context, orderID string) (OrderResponse, error) {
	return OrderResponse{OrderID: orderID, Status: "FILLED", FilledQty: 1}, nil
}

// ClientOrderID generates a unique client order ID for a workflow leg
func ClientOrderID(workflowID, proposalID string, leg int) string {
	return fmt.Sprintf("%s:%s:%d", workflowID, proposalID, leg)
}
