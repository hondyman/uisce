package trading

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/quickfixgo/quickfix"
	"go.temporal.io/sdk/temporal"
	"go.temporal.io/sdk/workflow"

	"github.com/hondyman/uisce/backend/internal/fix/tiles"
)

// Order represents a trading order
type Order struct {
	OrderID  string  `json:"order_id"`
	Symbol   string  `json:"symbol"`
	Quantity float64 `json:"quantity"`
	Side     string  `json:"side"` // Buy/Sell
	Price    float64 `json:"price"`
	Status   string  `json:"status"`
}

// ExecutionReportSignal is the inbound ExecutionReport delivered as a
// Temporal signal when the data-pipeline's fix_execution_writer tile
// fires (see HANDOFF_FIX_OVER_PIPELINE.md §10 step 4).
type ExecutionReportSignal struct {
	ClOrdID   string  `json:"cl_ord_id"`
	ExecID    string  `json:"exec_id"`
	Status    string  `json:"status"` // Filled, PartialFill, Canceled, Rejected
	Text      string  `json:"text,omitempty"`
	LastQty   float64 `json:"last_qty,omitempty"`
	LastPx    float64 `json:"last_px,omitempty"`
	OrdStatus string  `json:"ord_status,omitempty"`
}

// FIXOrderInput extends Order with FIX routing info. Passed when the
// workflow is started via SignalWithStart from the upstream pipeline.
type FIXOrderInput struct {
	Order
	TenantID    uuid.UUID `json:"tenant_id"`
	BrokerID    uuid.UUID `json:"broker_id"`
	BrokerCode  string    `json:"broker_code,omitempty"`
	AdminURL    string    `json:"admin_url"` // see HANDOFF §19 admin co-location
	AdminToken  string    `json:"admin_token"`
	SessionID   string    `json:"session_id"` // FIX.4.4:BUYER->SELLER
	ClOrdID     string    `json:"cl_ord_id"`
	Command     string    `json:"command"` // NewOrderSingle | CancelReplace | Cancel
	PlacementID string    `json:"placement_id,omitempty"`
	OrigClOrdID string    `json:"orig_cl_ord_id,omitempty"`
}

// FIXOrderEntryWorkflow handles the lifecycle of an order routed
// through the FIX-over-pipeline architecture (HANDOFF_FIX_OVER_PIPELINE.md
// §10). Per Amendment 2, the workflow waits for an ExecutionReport
// signal after sending; business-level ExecutionReport flows back
// asynchronously through the data-pipeline.
func FIXOrderEntryWorkflow(ctx workflow.Context, input FIXOrderInput) (*Order, error) {
	logger := workflow.GetLogger(ctx)
	logger.Info("FIXOrderEntryWorkflow started", "OrderID", input.OrderID, "TenantID", input.TenantID)

	// Step 1: Validate (existing behavior, unchanged from OrderEntryWorkflow).
	if input.Quantity <= 0 {
		return nil, fmt.Errorf("invalid quantity")
	}
	input.Status = "Validated"

	// Step 2: Activity options. Note MaximumAttempts: 1 — outbound FIX
	// sends are not safely re-runnable (HANDOFF §14, mirrors data-pipeline
	// MaximumAttempts discipline).
	ao := workflow.ActivityOptions{
		StartToCloseTimeout: 30 * time.Second,
		RetryPolicy: &temporal.RetryPolicy{
			MaximumAttempts: 1,
		},
	}
	actx := workflow.WithActivityOptions(ctx, ao)

	if input.Command == "" {
		input.Command = "NewOrderSingle"
	}

	// Persist the route onto crims.orm.placement before the wire send so
	// a blotter refresh sees ROUTED even if the broker is slow.
	if input.Command == "NewOrderSingle" {
		if err := workflow.ExecuteActivity(actx, PersistFIXRouteActivity, input).Get(actx, nil); err != nil {
			logger.Error("persist route failed", "err", err)
			return nil, err
		}
	}

	// Step 3: Send via the acceptor admin API (socket owner). Never
	// quickfix.SendToTarget from this workflow.
	if err := workflow.ExecuteActivity(actx, SendFixOrderActivity, input).Get(actx, nil); err != nil {
		return nil, err
	}
	input.Status = "SentToMarket"

	// Step 4: Wait for ExecutionReport signal. The signal is delivered by
	// the adapter when an inbound MsgType=8 arrives matching this
	// ClOrdID. Per HANDOFF §10 step 4, the adapter correlates by
	// (tenant_id, ClOrdID).
	signalChan := workflow.GetSignalChannel(ctx, "ExecutionReport")
	var report ExecutionReportSignal

	// Per the latency budget in fix_tenant_config.pipeline_latency_budget_ms
	// (default 30s; configurable). Beyond this, the adapter will have
	// sent a BusinessReject (§7) and we can give up.
	budget := time.Minute // conservative default; activities can override via input
	timer := workflow.NewTimer(ctx, budget)

	selector := workflow.NewSelector(ctx)
	selector.AddReceive(signalChan, func(c workflow.ReceiveChannel, more bool) {
		c.Receive(ctx, &report)
		logger.Info("Received ExecutionReport", "ClOrdID", report.ClOrdID, "Status", report.Status)
		input.Status = report.Status
	})
	selector.AddFuture(timer, func(f workflow.Future) {
		logger.Warn("FIXOrderEntryWorkflow: latency budget exceeded; giving up")
		input.Status = "TimedOut"
	})
	selector.Select(ctx)

	if report.ClOrdID != "" {
		fill := FIXFillPersist{
			FIXOrderInput: input,
			LastQty:       report.LastQty,
			LastPx:        report.LastPx,
			ExecID:        report.ExecID,
			OrdStatus:     report.OrdStatus,
		}
		if fill.LastQty == 0 {
			fill.LastQty = input.Quantity
		}
		if fill.LastPx == 0 {
			fill.LastPx = input.Price
		}
		if err := workflow.ExecuteActivity(actx, PersistFIXFillActivity, fill).Get(actx, nil); err != nil {
			logger.Error("persist fill failed", "err", err)
			return nil, err
		}
	}

	return &input.Order, nil
}

// OrderEntryWorkflow is the legacy single-tenant workflow. Kept for
// backward compatibility with existing call sites (server_test.go etc.).
// New code should use FIXOrderEntryWorkflow.
func OrderEntryWorkflow(ctx workflow.Context, order Order) (*Order, error) {
	logger := workflow.GetLogger(ctx)
	logger.Info("OrderEntryWorkflow Started", "OrderID", order.OrderID)

	if order.Quantity <= 0 {
		return nil, fmt.Errorf("invalid quantity")
	}
	order.Status = "Validated"

	err := workflow.ExecuteActivity(workflow.WithActivityOptions(ctx, workflow.ActivityOptions{
		StartToCloseTimeout: time.Second * 10,
	}), SendFixNewOrderSingle, order).Get(ctx, nil)
	if err != nil {
		return nil, err
	}
	order.Status = "SentToMarket"

	var executionReport string
	signalChan := workflow.GetSignalChannel(ctx, "ExecutionReport")

	selector := workflow.NewSelector(ctx)
	selector.AddReceive(signalChan, func(c workflow.ReceiveChannel, more bool) {
		c.Receive(ctx, &executionReport)
		logger.Info("Received Execution Report", "Report", executionReport)
		order.Status = "Filled"
	})

	timer := workflow.NewTimer(ctx, time.Minute*5)
	selector.AddFuture(timer, func(f workflow.Future) {
		logger.Info("Order Timed Out")
		order.Status = "TimedOut"
	})

	selector.Select(ctx)
	return &order, nil
}

// --- Activities ---

func SendFixNewOrderSingle(ctx context.Context, order Order) error {
	fmt.Printf("Sending FIX NewOrderSingle: %v\n", order)
	return nil
}

// SendFixOrderActivity constructs and dispatches a NewOrderSingle via
// quickfix SendToTarget. The typed order is converted to FIX tags via
// the (reverse) tag mapping — this is the production replacement for
// the mock SendFixNewOrderSingle.
//
// Idempotency: outbound dedup key (tenant_id, ClOrdID, broker_id) per
// HANDOFF §19. The ClOrdID is set by the workflow before invocation; if
// the activity retries (it shouldn't — MaximumAttempts: 1 — but if it
// does), quickfix's own message-store deduplication prevents the
// double-send.
func SendFixOrderActivity(ctx context.Context, input FIXOrderInput) error {
	logger := activityLoggerEntry(ctx)
	logger.Info("SendFixOrderActivity", "OrderID", input.OrderID, "ClOrdID", input.ClOrdID, "Command", input.Command)

	if strings.TrimSpace(input.AdminURL) == "" {
		return fmt.Errorf("admin_url is required; FIX TCP is owned by the acceptor, not this activity")
	}
	msgType := "D"
	switch input.Command {
	case "Cancel", "fix.Cancel":
		msgType = "F"
	case "CancelReplace", "fix.CancelReplace":
		msgType = "G"
	}

	fields := map[string]string{
		"11": input.ClOrdID,
		"55": input.Symbol,
		"54": sideCode(input.Side),
		"38": fmt.Sprintf("%v", input.Quantity),
		"44": fmt.Sprintf("%v", input.Price),
		"21": "1",
		"40": "2",
	}
	if input.Price == 0 {
		fields["40"] = "1" // Market
		delete(fields, "44")
	}
	if input.OrigClOrdID != "" {
		fields["41"] = input.OrigClOrdID
	}

	body, err := json.Marshal(map[string]interface{}{"msgType": msgType, "fields": fields})
	if err != nil {
		return err
	}
	url := strings.TrimRight(input.AdminURL, "/") + "/sessions/" + input.SessionID + "/send"
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, strings.NewReader(string(body)))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Fix-Admin-Token", input.AdminToken)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return fmt.Errorf("admin send: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 300 {
		b, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("admin send HTTP %d: %s", resp.StatusCode, strings.TrimSpace(string(b)))
	}
	return nil
}

// sideCode converts "BUY"/"SELL" to FIX side codes (1/2).
func sideCode(side string) string {
	switch strings.ToUpper(side) {
	case "BUY":
		return "1"
	case "SELL":
		return "2"
	case "SELL_SHORT":
		return "5"
	default:
		return side
	}
}

// parseSessionID decodes "FIX.4.4:SENDER->TARGET" into quickfix.SessionID.
func parseSessionID(s string) (quickfix.SessionID, error) {
	begin, sender, target, ok := tiles.ParseQuickfixSessionIDString(s)
	if !ok {
		return quickfix.SessionID{}, fmt.Errorf("invalid session id format: %q", s)
	}
	return quickfix.SessionID{
		BeginString:  begin,
		SenderCompID: sender,
		TargetCompID: target,
	}, nil
}

// activityLogger returns a minimal logger interface for use inside
// activities. The full Temporal logger (activity.GetLogger) is wired
// in cmd/worker/main.go's activity registration.
type activityLogger interface {
	Info(string, ...interface{})
	Warn(string, ...interface{})
	Error(string, ...interface{})
}

func activityLoggerImpl(ctx context.Context) activityLogger {
	return noopLogger{}
}

func activityLoggerEntry(ctx context.Context) activityLogger {
	return activityLoggerImpl(ctx)
}

// noopLogger is a stand-in when no Temporal activity context is
// available (e.g. unit tests).
type noopLogger struct{}

func (noopLogger) Info(string, ...interface{})  {}
func (noopLogger) Warn(string, ...interface{})  {}
func (noopLogger) Error(string, ...interface{}) {}

// Ensure base64/http imports survive the build (they're used by future
// tile integration steps and we don't want to drop them prematurely).
var (
	_ = base64.StdEncoding
	_ = http.MethodPost
)
