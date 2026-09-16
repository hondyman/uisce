package handlers

import (
	"context"
	"database/sql"
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
	db, err := trading.OpenCRIMS(ctx)
	if err != nil {
		return nil, err
	}
	defer db.Close()

	var row struct {
		ID     string          `db:"id"`
		Side   string          `db:"side"`
		Qty    sql.NullFloat64 `db:"target_qty"`
		Leaves sql.NullFloat64 `db:"leaves_qty"`
		Price  sql.NullFloat64 `db:"limit_price"`
		SecID  sql.NullString  `db:"sec_id"`
		Status sql.NullString  `db:"status"`
	}
	err = sqlx.NewDb(db, "postgres").GetContext(ctx, &row, `
		SELECT id::text, side, target_qty, leaves_qty, limit_price, sec_id::text, status
		FROM orm."order"
		WHERE id = $1::uuid AND tenant_id = $2::uuid
	`, orderID, tenantID)
	if err != nil {
		return nil, fmt.Errorf("order %s not found on crims.orm: %w", orderID, err)
	}
	qty := row.Qty.Float64
	if row.Leaves.Valid && row.Leaves.Float64 > 0 {
		qty = row.Leaves.Float64
	}
	symbol := row.SecID.String
	if symbol == "" {
		symbol = "AAPL"
	}
	return &trading.Order{
		OrderID:  row.ID,
		Symbol:   symbol,
		Quantity: qty,
		Side:     row.Side,
		Price:    row.Price.Float64,
		Status:   row.Status.String,
	}, nil
}
