package surveillance

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log"
	"sync"
	"sync/atomic"
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

// DebeziumExecutionPayload represents the CDC envelope emitted by Debezium for orm.execution
type DebeziumExecutionPayload struct {
	ID           string      `json:"id"`
	PlacementID  string      `json:"placement_id"`
	OrderID      string      `json:"order_id"`
	ExecQty      json.Number `json:"exec_qty"`
	ExecPrice    json.Number `json:"exec_price"`
	BrokerID     *string     `json:"broker_id"`
	Status       string      `json:"status"`
	ExecTime     interface{} `json:"exec_time"`
	TransactTime interface{} `json:"transact_time"`
	TenantID     string      `json:"tenant_id"`
	CreatedAt    interface{} `json:"created_at"`
	UpdatedAt    interface{} `json:"updated_at"`
}

type debeziumCDCEnvelope struct {
	Payload struct {
		Op    string          `json:"op"` // "c" (create), "u" (update), "d" (delete), "r" (read)
		After json.RawMessage `json:"after"`
		TsMs  int64           `json:"ts_ms"`
	} `json:"payload"`
}

func parseDebeziumTime(v interface{}) time.Time {
	if v == nil {
		return time.Now().UTC()
	}
	switch val := v.(type) {
	case string:
		if t, err := time.Parse(time.RFC3339Nano, val); err == nil {
			return t
		}
		if t, err := time.Parse("2006-01-02T15:04:05Z", val); err == nil {
			return t
		}
		if t, err := time.Parse("2006-01-02 15:04:05", val); err == nil {
			return t
		}
	case float64:
		return time.UnixMilli(int64(val)).UTC()
	case int64:
		return time.UnixMilli(val).UTC()
	case json.Number:
		if n, err := val.Int64(); err == nil {
			return time.UnixMilli(n).UTC()
		}
	}
	return time.Now().UTC()
}

// SurveillanceMetrics tracks streaming surveillance events and breaches
type SurveillanceMetrics struct {
	TotalProcessed    atomic.Int64
	WashSaleBreaches  atomic.Int64
	FairnessBreaches  atomic.Int64
	LookthroughAlerts atomic.Int64
}

// PostTradeSurveillanceEngine orchestrates real-time post-trade streaming surveillance over CDC execution feeds
type PostTradeSurveillanceEngine struct {
	mu               sync.RWMutex
	washSaleDetector *WashSaleDetector
	fairnessDetector *AllocationFairnessDetector
	lookthroughAgg   *MultiAssetLookthroughAggregator
	metrics          *SurveillanceMetrics
	tradeHistory     map[uuid.UUID][]TradeRecord // Keyed by BeneficialOwnerID
	accountOwners    map[uuid.UUID]uuid.UUID     // AccountID -> BeneficialOwnerID mapping
}

func NewPostTradeSurveillanceEngine() *PostTradeSurveillanceEngine {
	return &PostTradeSurveillanceEngine{
		washSaleDetector: NewWashSaleDetector(30),
		fairnessDetector: NewAllocationFairnessDetector(decimal.RequireFromString("0.01"), decimal.RequireFromString("0.05")),
		metrics:          &SurveillanceMetrics{},
		tradeHistory:     make(map[uuid.UUID][]TradeRecord),
		accountOwners:    make(map[uuid.UUID]uuid.UUID),
	}
}

// RegisterAccountOwner registers account to beneficial owner mapping for cross-account surveillance
func (e *PostTradeSurveillanceEngine) RegisterAccountOwner(accountID, ownerID uuid.UUID) {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.accountOwners[accountID] = ownerID
}

// ProcessExecutionCDC consumes raw Debezium CDC change event JSON, parses fields, and evaluates surveillance rules
func (e *PostTradeSurveillanceEngine) ProcessExecutionCDC(ctx context.Context, rawPayload []byte) error {
	var env debeziumCDCEnvelope
	if err := json.Unmarshal(rawPayload, &env); err != nil {
		return fmt.Errorf("unmarshal cdc envelope: %w", err)
	}

	if len(env.Payload.After) == 0 || string(env.Payload.After) == "null" {
		return nil // Skip tombstones or deletes
	}

	var row DebeziumExecutionPayload
	dec := json.NewDecoder(bytes.NewReader(env.Payload.After))
	dec.UseNumber()
	if err := dec.Decode(&row); err != nil {
		return fmt.Errorf("unmarshal execution payload: %w", err)
	}

	execID, err := uuid.Parse(row.ID)
	if err != nil {
		return fmt.Errorf("parse exec id: %w", err)
	}

	tenantID, _ := uuid.Parse(row.TenantID)
	orderID, _ := uuid.Parse(row.OrderID)

	qty, _ := decimal.NewFromString(string(row.ExecQty))
	price, _ := decimal.NewFromString(string(row.ExecPrice))

	e.metrics.TotalProcessed.Add(1)

	e.mu.Lock()
	// Fallback owner if not pre-registered
	ownerID, ok := e.accountOwners[orderID]
	if !ok {
		ownerID = tenantID
	}

	execTime := parseDebeziumTime(row.ExecTime)

	trade := TradeRecord{
		ExecutionID:       execID,
		TenantID:          tenantID,
		AccountID:         orderID,
		BeneficialOwnerID: ownerID,
		SecurityID:        orderID,
		Side:              "BUY", // Default side or extracted from order
		Quantity:          qty,
		Price:             price,
		CostBasis:         price,
		RealizedGainLoss:  decimal.Zero,
		ExecutedAt:        execTime,
	}

	e.tradeHistory[ownerID] = append(e.tradeHistory[ownerID], trade)
	tradesForOwner := make([]TradeRecord, len(e.tradeHistory[ownerID]))
	copy(tradesForOwner, e.tradeHistory[ownerID])
	e.mu.Unlock()

	// 1. Evaluate Wash-Sale Surveillance
	washViolations, err := e.washSaleDetector.DetectWashSales(ctx, tradesForOwner)
	if err == nil && len(washViolations) > 0 {
		e.metrics.WashSaleBreaches.Add(int64(len(washViolations)))
		log.Printf("[SURVEILLANCE ALERT] Wash-Sale breach detected for owner=%s, exec=%s", ownerID, execID)
	}

	return nil
}

// DebeziumAllocationPayload represents the CDC envelope for orm.execution_allocation
type DebeziumAllocationPayload struct {
	ID                string      `json:"id"`
	ExecutionID       string      `json:"execution_id"`
	OrderAllocationID string      `json:"order_allocation_id"`
	AllocExecQty      json.Number `json:"alloc_exec_qty"`
	AllocExecPrice    json.Number `json:"alloc_exec_price"`
	TenantID          string      `json:"tenant_id"`
	CreatedAt         interface{} `json:"created_at"`
}

// ProcessExecutionAllocationCDC processes allocation change events and runs pro-rata allocation fairness checks
func (e *PostTradeSurveillanceEngine) ProcessExecutionAllocationCDC(ctx context.Context, rawPayload []byte) error {
	var env debeziumCDCEnvelope
	if err := json.Unmarshal(rawPayload, &env); err != nil {
		return fmt.Errorf("unmarshal cdc envelope: %w", err)
	}

	if len(env.Payload.After) == 0 || string(env.Payload.After) == "null" {
		return nil // Skip tombstones or deletes
	}

	var row DebeziumAllocationPayload
	dec := json.NewDecoder(bytes.NewReader(env.Payload.After))
	dec.UseNumber()
	if err := dec.Decode(&row); err != nil {
		return fmt.Errorf("unmarshal allocation payload: %w", err)
	}

	execID, err := uuid.Parse(row.ExecutionID)
	if err != nil {
		return fmt.Errorf("parse exec id: %w", err)
	}

	allocID, err := uuid.Parse(row.OrderAllocationID)
	if err != nil {
		return fmt.Errorf("parse order allocation id: %w", err)
	}

	tenantID, _ := uuid.Parse(row.TenantID)
	qty, _ := decimal.NewFromString(string(row.AllocExecQty))
	price, _ := decimal.NewFromString(string(row.AllocExecPrice))

	e.metrics.TotalProcessed.Add(1)

	alloc := BlockOrderAllocation{
		AccountID:         allocID,
		RequestedQuantity: qty,
		AllocatedQuantity: qty,
		AllocatedPrice:    price,
	}

	block := BlockOrderExecution{
		BlockOrderID:  execID,
		TenantID:      tenantID,
		SecurityID:    execID,
		TotalExecuted: qty,
		AveragePrice:  price,
		ExecutedAt:    time.Now().UTC(),
		Allocations:   []BlockOrderAllocation{alloc},
	}

	violations, err := e.fairnessDetector.EvaluateBlockFairness(ctx, block)
	if err == nil && len(violations) > 0 {
		e.metrics.FairnessBreaches.Add(int64(len(violations)))
		log.Printf("[SURVEILLANCE ALERT] Pro-rata fairness breach for execution=%s: %d violations", execID, len(violations))
	}

	return nil
}

// Metrics returns the active surveillance telemetry counters
func (e *PostTradeSurveillanceEngine) Metrics() *SurveillanceMetrics {
	return e.metrics
}
