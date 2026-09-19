package handlers

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	"go.temporal.io/sdk/client"

	fixpkg "github.com/hondyman/uisce/backend/internal/fix"
	"github.com/hondyman/uisce/backend/internal/trading"
)

const fixOrderTaskQueue = "bp_queue"

// OMSFIXCommandHandler is the live HTTP path that starts
// FIXOrderEntryWorkflow. Page Studio buttons call this; they do not
// write fills.
type OMSFIXCommandHandler struct {
	db       *sqlx.DB
	temporal client.Client
}

func NewOMSFIXCommandHandler(db *sqlx.DB, temporal client.Client) *OMSFIXCommandHandler {
	return &OMSFIXCommandHandler{db: db, temporal: temporal}
}

func (h *OMSFIXCommandHandler) RegisterRoutes(r chi.Router) {
	r.Post("/oms/commands/fix-order-entry", h.start)
}

type fixOrderCommandRequest struct {
	OrderID     string  `json:"orderId"`
	Command     string  `json:"command"`
	SessionID   string  `json:"sessionId,omitempty"`
	Symbol      string  `json:"symbol,omitempty"`
	Side        string  `json:"side,omitempty"`
	Quantity    float64 `json:"quantity,omitempty"`
	Price       float64 `json:"price,omitempty"`
	OrigClOrdID string  `json:"origClOrdId,omitempty"`
	BrokerCode  string  `json:"brokerCode,omitempty"`
}

func (h *OMSFIXCommandHandler) start(w http.ResponseWriter, r *http.Request) {
	tenantID, ok := mustTenantID(r)
	if !ok {
		http.Error(w, "tenant_id is required", http.StatusUnauthorized)
		return
	}
	if h.temporal == nil {
		http.Error(w, "temporal is not configured; cannot start FIXOrderEntryWorkflow", http.StatusServiceUnavailable)
		return
	}
	var req fixOrderCommandRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}
	if req.OrderID == "" {
		http.Error(w, "orderId is required", http.StatusBadRequest)
		return
	}
	cmd := strings.TrimPrefix(req.Command, "fix.")
	if cmd == "" {
		cmd = "NewOrderSingle"
	}
	switch cmd {
	case "NewOrderSingle", "CancelReplace", "Cancel":
	default:
		http.Error(w, "command must be NewOrderSingle, CancelReplace, or Cancel", http.StatusBadRequest)
		return
	}

	order, err := h.loadOrder(r.Context(), tenantID, req.OrderID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	if req.Symbol != "" {
		order.Symbol = req.Symbol
	}
	if req.Side != "" {
		order.Side = req.Side
	}
	if req.Quantity > 0 {
		order.Quantity = req.Quantity
	}
	if req.Price > 0 {
		order.Price = req.Price
	}

	sessionID := req.SessionID
	if sessionID == "" {
		sessionID = os.Getenv("FIX_DEMO_SESSION_ID")
	}
	if sessionID == "" {
		sessionID = fixpkg.DemoSessionID()
	}
	adminURL := os.Getenv("FIX_ADMIN_ADDR")
	if adminURL == "" {
		adminURL = "127.0.0.1:8981"
	}
	if !strings.HasPrefix(adminURL, "http") {
		adminURL = "http://" + adminURL
	}

	short := strings.ReplaceAll(order.OrderID, "-", "")
	if len(short) > 8 {
		short = short[:8]
	}
	clOrdID := fmt.Sprintf("NW-%s-%d", short, time.Now().UTC().UnixNano()%1_000_000)

	input := trading.FIXOrderInput{
		Order:       *order,
		TenantID:    tenantID,
		BrokerCode:  req.BrokerCode,
		AdminURL:    adminURL,
		AdminToken:  os.Getenv("FIX_ADMIN_TOKEN"),
		SessionID:   sessionID,
		ClOrdID:     clOrdID,
		Command:     cmd,
		OrigClOrdID: req.OrigClOrdID,
		PlacementID: uuid.NewString(),
	}
	if input.BrokerCode == "" {
		input.BrokerCode = "GSCO"
	}

	wfID := "fix-order-" + tenantID.String() + "-" + clOrdID
	run, err := h.temporal.ExecuteWorkflow(r.Context(), client.StartWorkflowOptions{
		ID:        wfID,
		TaskQueue: fixOrderTaskQueue,
	}, trading.FIXOrderEntryWorkflow, input)
	if err != nil {
		http.Error(w, "failed to start FIXOrderEntryWorkflow: "+err.Error(), http.StatusInternalServerError)
		return
	}
	writeJSON(w, http.StatusAccepted, map[string]string{
		"workflowId": run.GetID(),
		"runId":      run.GetRunID(),
		"clOrdId":    clOrdID,
		"command":    cmd,
		"status":     "started",
		"sessionId":  sessionID,
	})
}

func (h *OMSFIXCommandHandler) loadOrder(ctx context.Context, tenantID uuid.UUID, orderID string) (*trading.Order, error) {
	// Shared fence with MCP start_fix_order_entry (trading.LoadOrder).
	return trading.LoadOrder(ctx, tenantID, orderID)
}
