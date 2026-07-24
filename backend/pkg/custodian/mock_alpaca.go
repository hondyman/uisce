package custodian

import (
	"context"
	"math/rand"
	"time"
)

// MockAlpaca simulates an Alpaca adapter with random failures
type MockAlpaca struct{}

func (m MockAlpaca) PlaceOrder(ctx context.Context, req OrderRequest) (OrderResponse, error) {
	time.Sleep(50 * time.Millisecond)
	
	// Random partial/fail for demo
	r := rand.Float64()
	switch {
	case r < 0.1:
		return OrderResponse{OrderID: "alp_" + req.ClientOrderID, Status: "REJECTED"}, nil
	case r < 0.4:
		return OrderResponse{OrderID: "alp_" + req.ClientOrderID, Status: "PARTIAL", FilledQty: req.Qty * 0.5}, nil
	default:
		return OrderResponse{OrderID: "alp_" + req.ClientOrderID, Status: "FILLED", FilledQty: req.Qty}, nil
	}
}

func (m MockAlpaca) CancelOrder(ctx context.Context, orderID string) (OrderResponse, error) {
	return OrderResponse{OrderID: orderID, Status: "CANCELED"}, nil
}

func (m MockAlpaca) GetOrderStatus(ctx context.Context, orderID string) (OrderResponse, error) {
	return OrderResponse{OrderID: orderID, Status: "FILLED", FilledQty: 1}, nil
}
