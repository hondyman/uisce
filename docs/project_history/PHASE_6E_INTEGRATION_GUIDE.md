# Phase 6e: Advanced Resilience - Integration Guide

---

## 🚀 Quick Start

### 1. Initialize Components

```go
package main

import (
	"github.com/eganpj/semlayer/backend/internal/resilience"
)

// In your main.go
func init() {
	// Create saga coordinator
	sagaCoordinator = resilience.NewSagaCoordinator(eventBus)
	
	// Create idempotency store
	idempotencyStore = resilience.NewIdempotencyStore()
	
	// Create consistency model
	consistencyModel = resilience.NewConsistencyModel()
}
```

### 2. Use in Handlers

#### Saga Pattern (Order Service)

```go
func (h *OrderHandler) CreateOrder(ctx context.Context, req *CreateOrderRequest) (*Order, error) {
	sagaID := generateID()
	
	// Define saga steps
	steps := []resilience.SagaStep{
		{
			Name: "ValidateOrder",
			Action: func(ctx context.Context, data interface{}) (interface{}, error) {
				return h.validateOrder(ctx, req)
			},
			Compensation: func(ctx context.Context, data interface{}) error {
				return nil // No state to roll back
			},
			Timeout: 5 * time.Second,
			MaxRetries: 1,
		},
		{
			Name: "ReserveInventory",
			Action: func(ctx context.Context, data interface{}) (interface{}, error) {
				reservationID, err := h.inventoryService.Reserve(ctx, req.Items)
				return reservationID, err
			},
			Compensation: func(ctx context.Context, data interface{}) error {
				reservationID := data.(string)
				return h.inventoryService.Release(ctx, reservationID)
			},
			Timeout: 10 * time.Second,
			MaxRetries: 2,
		},
		{
			Name: "ProcessPayment",
			Action: func(ctx context.Context, data interface{}) (interface{}, error) {
				transactionID, err := h.paymentService.Charge(ctx, req.Amount, req.PaymentMethod)
				return transactionID, err
			},
			Compensation: func(ctx context.Context, data interface{}) error {
				transactionID := data.(string)
				return h.paymentService.Refund(ctx, transactionID)
			},
			Timeout: 10 * time.Second,
			MaxRetries: 2,
		},
		{
			Name: "CreateShipment",
			Action: func(ctx context.Context, data interface{}) (interface{}, error) {
				shipmentID, err := h.shippingService.CreateShipment(ctx, req.Items, req.Address)
				return shipmentID, err
			},
			Compensation: func(ctx context.Context, data interface{}) error {
				shipmentID := data.(string)
				return h.shippingService.CancelShipment(ctx, shipmentID)
			},
			Timeout: 10 * time.Second,
			MaxRetries: 2,
		},
	}
	
	// Execute saga with automatic compensation on failure
	err := h.sagaCoordinator.ExecuteSaga(ctx, sagaID, steps)
	if err != nil {
		// All compensations already executed
		return nil, fmt.Errorf("order creation failed: %w", err)
	}
	
	// Get saga status
	status, _ := h.sagaCoordinator.GetSagaStatus(sagaID)
	
	return &Order{
		ID: req.ID,
		Status: "created",
		SagaID: sagaID,
	}, nil
}
```

#### Idempotency Pattern (Transfer Service)

```go
func (h *TransferHandler) TransferMoney(ctx context.Context, req *TransferRequest) (*TransferResponse, error) {
	// Get idempotency key from request header or generate
	idempotencyKey := req.IdempotencyKey
	if idempotencyKey == "" {
		idempotencyKey = generateID()
	}
	
	// Execute with idempotency guarantee
	result, err := h.idempotencyStore.ExecuteIdempotently(
		ctx,
		idempotencyKey,
		"TransferMoney",
		req,
		24 * time.Hour, // Store for 24 hours
		func(ctx context.Context) (interface{}, error) {
			// Actual transfer logic
			return h.performTransfer(ctx, req.FromAccount, req.ToAccount, req.Amount)
		},
	)
	
	if err != nil {
		return nil, err
	}
	
	return &TransferResponse{
		TransactionID: result.(string),
		Status: "completed",
	}, nil
}
```

#### Event-Driven Consistency Pattern

```go
func (h *OrderHandler) init() {
	// Subscribe to events from other services
	h.consistencyModel.SubscribeToEventType("OrderCreated", func(ctx context.Context, event *resilience.ConsistencyEvent) error {
		// Update inventory read model
		order := event.Payload.(Order)
		return h.inventoryReadModel.UpdateForOrder(ctx, order.ID, order.Items)
	})
	
	h.consistencyModel.SubscribeToEventType("OrderCreated", func(ctx context.Context, event *resilience.ConsistencyEvent) error {
		// Send notification
		order := event.Payload.(Order)
		return h.notificationService.SendOrderConfirmation(ctx, order)
	})
	
	h.consistencyModel.SubscribeToEventType("PaymentProcessed", func(ctx context.Context, event *resilience.ConsistencyEvent) error {
		// Update order status
		return h.updateOrderStatusReadModel(ctx, event.Payload)
	})
}

func (h *OrderHandler) PublishOrderEvent(ctx context.Context, order *Order) error {
	// Publish event for eventual consistency
	event := h.consistencyModel.PublishEvent(
		ctx,
		generateEventID(),
		order.ID,
		"OrderCreated",
		order,
		3, // max retries
	)
	
	// Optionally wait for processing
	err := h.consistencyModel.WaitForEventProcessing(ctx, event.ID, 30*time.Second)
	return err
}
```

---

## 📊 Metrics Integration

### Export Metrics

```go
func (h *OrderHandler) initMetricsEndpoint() {
	http.HandleFunc("/metrics/saga", func(w http.ResponseWriter, r *http.Request) {
		metrics := h.sagaCoordinator.ExportMetrics()
		w.Header().Set("Content-Type", "text/plain")
		w.Write([]byte(metrics))
	})
	
	http.HandleFunc("/metrics/idempotency", func(w http.ResponseWriter, r *http.Request) {
		metrics := h.idempotencyStore.ExportMetrics()
		w.Header().Set("Content-Type", "text/plain")
		w.Write([]byte(metrics))
	})
	
	http.HandleFunc("/metrics/consistency", func(w http.ResponseWriter, r *http.Request) {
		metrics := h.consistencyModel.ExportMetrics()
		w.Header().Set("Content-Type", "text/plain")
		w.Write([]byte(metrics))
	})
}
```

### Monitor in Grafana

Add to Prometheus scrape config:

```yaml
scrape_configs:
  - job_name: 'saga-metrics'
    static_configs:
      - targets: ['localhost:8080/metrics/saga']
  
  - job_name: 'idempotency-metrics'
    static_configs:
      - targets: ['localhost:8080/metrics/idempotency']
  
  - job_name: 'consistency-metrics'
    static_configs:
      - targets: ['localhost:8080/metrics/consistency']
```

---

## 🔧 Configuration

### Saga Configuration

```go
// Customize saga behavior per operation type
sagaConfigs := map[string]resilience.SagaStep{
	"CreateOrder": {
		Name: "CreateOrder",
		Timeout: 10 * time.Second,
		MaxRetries: 2,
	},
	"ProcessPayment": {
		Name: "ProcessPayment",
		Timeout: 15 * time.Second,
		MaxRetries: 3, // More lenient for payment
	},
	"SendNotification": {
		Name: "SendNotification",
		Timeout: 5 * time.Second,
		MaxRetries: 5, // Very lenient for notifications
	},
}
```

### Idempotency TTL by Operation

```go
// Different TTLs for different operation types
ttlConfig := map[string]time.Duration{
	"TransferMoney": 24 * time.Hour,
	"CreateAccount": 7 * 24 * time.Hour,
	"UpdateProfile": 1 * time.Hour,
	"DeleteAccount": 30 * 24 * time.Hour,
}
```

### Consistency Event Subscriptions

```go
// Configure event subscribers
eventSubscribers := map[string][]resilience.EventHandler{
	"OrderCreated": {
		inventoryUpdateHandler,
		notificationHandler,
		analyticsHandler,
	},
	"PaymentProcessed": {
		orderStatusHandler,
		ledgerHandler,
	},
	"ShipmentDispatched": {
		customerNotificationHandler,
		trackingHandler,
	},
}

for eventType, handlers := range eventSubscribers {
	for _, handler := range handlers {
		consistencyModel.SubscribeToEventType(eventType, handler)
	}
}
```

---

## 🧪 Testing

### Test Saga Compensation

```go
func TestOrderSagaCompensation(t *testing.T) {
	coordinator := resilience.NewSagaCoordinator(nil)
	
	compensationExecuted := false
	
	steps := []resilience.SagaStep{
		{
			Name: "Step1",
			Action: func(ctx context.Context, data interface{}) (interface{}, error) {
				return "step1-result", nil
			},
			Compensation: func(ctx context.Context, data interface{}) error {
				compensationExecuted = true
				return nil
			},
		},
		{
			Name: "Step2",
			Action: func(ctx context.Context, data interface{}) (interface{}, error) {
				return nil, fmt.Errorf("step2 failed")
			},
		},
	}
	
	err := coordinator.ExecuteSaga(context.Background(), "saga-1", steps)
	
	assert.Error(t, err)
	assert.True(t, compensationExecuted, "Compensation should be executed")
}
```

### Test Idempotency

```go
func TestIdempotentOperations(t *testing.T) {
	store := resilience.NewIdempotencyStore()
	callCount := 0
	
	operation := func(ctx context.Context) (interface{}, error) {
		callCount++
		return "result", nil
	}
	
	// First call
	result1, err1 := store.ExecuteIdempotently(
		context.Background(),
		"op-1",
		"Test",
		nil,
		1*time.Hour,
		operation,
	)
	
	// Second call (duplicate)
	result2, err2 := store.ExecuteIdempotently(
		context.Background(),
		"op-1",
		"Test",
		nil,
		1*time.Hour,
		operation,
	)
	
	assert.Equal(t, 1, callCount, "Operation should only be called once")
	assert.Equal(t, result1, result2)
	assert.NoError(t, err1)
	assert.NoError(t, err2)
}
```

### Test Eventual Consistency

```go
func TestEventualConsistency(t *testing.T) {
	model := resilience.NewConsistencyModel()
	processed := false
	
	model.SubscribeToEventType("TestEvent", func(ctx context.Context, event *resilience.ConsistencyEvent) error {
		processed = true
		return nil
	})
	
	eventID := "event-1"
	model.PublishEvent(context.Background(), eventID, "agg-1", "TestEvent", nil, 1)
	
	// Wait for processing
	err := model.WaitForEventProcessing(context.Background(), eventID, 5*time.Second)
	
	assert.NoError(t, err)
	assert.True(t, processed)
}
```

---

## 📋 Deployment Checklist

Before deploying Phase 6e:

- [ ] All components compile successfully
- [ ] Saga coordinator tested with multi-step workflows
- [ ] Idempotency store tested with duplicate requests
- [ ] Consistency model tested with event processing
- [ ] Metrics exported to Prometheus
- [ ] Grafana dashboards configured
- [ ] Alert rules set up for failed sagas
- [ ] Event subscribers registered
- [ ] TTL cleanup configured
- [ ] Compensation logic tested
- [ ] Error scenarios tested (service timeouts, failures)
- [ ] Load test completed
- [ ] Documentation reviewed
- [ ] On-call training completed

---

## 🚀 Next Steps

1. **Integrate with existing services:**
   - Order service (saga pattern)
   - Payment service (idempotency)
   - Notification service (consistency events)

2. **Add monitoring:**
   - Prometheus scraping
   - Grafana dashboards
   - Alert rules

3. **Set up event subscribers:**
   - Inventory updates
   - Notifications
   - Analytics

4. **Production deployment:**
   - Staging test
   - Gradual rollout
   - Monitor metrics

---

**Phase 6e Status: ✅ COMPLETE AND READY FOR INTEGRATION**
