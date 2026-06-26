package custodian

import "context"

// OrderRequest represents a request to place an order
type OrderRequest struct {
	ClientOrderID string
	Symbol        string
	Side          string // BUY/SELL
	Qty           float64
	Type          string // MARKET/LIMIT
	LimitPrice    float64
}

// OrderResponse represents the response from a custodian
type OrderResponse struct {
	OrderID   string
	Status    string // NEW, FILLED, PARTIAL, REJECTED, CANCELED
	FilledQty float64
}

// Adapter defines the interface for interacting with a custodian
type Adapter interface {
	PlaceOrder(ctx context.Context, req OrderRequest) (OrderResponse, error)
	CancelOrder(ctx context.Context, orderID string) (OrderResponse, error)
	GetOrderStatus(ctx context.Context, orderID string) (OrderResponse, error)
}
