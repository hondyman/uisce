package api

import (
	"database/sql"
	"encoding/json"
	"log"
	"net/http"

	"github.com/google/uuid"
	"github.com/hondyman/semlayer/backend/internal/validation"
	"github.com/hondyman/semlayer/libs/jwt-middleware"
)

// OrdersHandler demonstrates how to integrate trigger validation into a real endpoint
type OrdersHandler struct {
	db            *sql.DB
	triggerEngine *validation.TriggerValidationEngine
}

// NewOrdersHandler creates a handler with trigger validation
func NewOrdersHandler(db *sql.DB, triggerEngine *validation.TriggerValidationEngine) *OrdersHandler {
	if triggerEngine == nil {
		triggerEngine = validation.NewTriggerValidationEngine(db, &validation.SimpleLogger{})
	}
	return &OrdersHandler{
		db:            db,
		triggerEngine: triggerEngine,
	}
}

// CreateOrderRequest represents a client request to create an order
type CreateOrderRequest struct {
	CustomerID string  `json:"customer_id"`
	Total      float64 `json:"total"`
	Items      []struct {
		SKU   string  `json:"sku"`
		Qty   int     `json:"qty"`
		Price float64 `json:"price"`
	} `json:"items"`
}

// HandleCreateOrder is an example CREATE endpoint that uses trigger validation
// POST /api/orders
// This demonstrates the exact pattern:
// 1. Parse request
// 2. Run trigger validation for "create" + "orders" entity
// 3. If pass -> persist to DB
// 4. If fail -> return 400 with error message
func (h *OrdersHandler) HandleCreateOrder(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	claims := jwtmiddleware.GetClaimsFromContext(r)
	if claims == nil {
		http.Error(w, `{"error":"unauthorized"}`, http.StatusUnauthorized)
		return
	}
	tenantID := claims.TenantID
	if tenantID == "" {
		http.Error(w, "X-Tenant-ID header required", http.StatusBadRequest)
		return
	}

	var req CreateOrderRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}

	// Convert to generic map for validation engine (this matches your DB schema)
	orderData := map[string]interface{}{
		"customer_id": req.CustomerID,
		"total":       req.Total,
		"items":       req.Items,
	}

	tid, err := uuid.Parse(tenantID)
	if err != nil {
		http.Error(w, "invalid tenant ID", http.StatusBadRequest)
		return
	}

	// ===== KEY STEP: Trigger Validation =====
	// Fetch and evaluate all "create" triggers for "orders" entity
	if err := h.triggerEngine.TriggerValidate(ctx, tid, "create", "orders", "", orderData); err != nil {
		// Validation failed -> return friendly error to client
		log.Printf("[OrdersHandler] Create order validation failed: %v", err)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
		return
	}

	// ===== All validations passed -> safe to persist =====
	orderID := uuid.New().String()

	// Insert into DB (simplified; your schema may differ)
	q := `
    INSERT INTO orders (id, tenant_id, customer_id, total, created_at)
    VALUES ($1, $2, $3, $4, NOW())
  `
	_, err = h.db.ExecContext(ctx, q, orderID, tenantID, req.CustomerID, req.Total)
	if err != nil {
		log.Printf("[OrdersHandler] DB insert failed: %v", err)
		http.Error(w, "insert failed", http.StatusInternalServerError)
		return
	}

	// TODO: Publish event to RabbitMQ: orders.created

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"id":     orderID,
		"status": "created",
	})
}

// HandleUpdateOrder is an example UPDATE endpoint
// PATCH /api/orders/{id}
// This runs "save" triggers (not "update", but "save" is the Workday equivalent)
func (h *OrdersHandler) HandleUpdateOrder(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	claims := jwtmiddleware.GetClaimsFromContext(r)
	if claims == nil {
		http.Error(w, `{"error":"unauthorized"}`, http.StatusUnauthorized)
		return
	}
	tenantID := claims.TenantID
	if tenantID == "" {
		http.Error(w, "X-Tenant-ID header required", http.StatusBadRequest)
		return
	}

	orderID := r.PathValue("id") // or chi.URLParam if using chi
	if orderID == "" {
		http.Error(w, "order ID required", http.StatusBadRequest)
		return
	}

	var req CreateOrderRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}

	orderData := map[string]interface{}{
		"customer_id": req.CustomerID,
		"total":       req.Total,
		"items":       req.Items,
	}

	tid, err := uuid.Parse(tenantID)
	if err != nil {
		http.Error(w, "invalid tenant ID", http.StatusBadRequest)
		return
	}

	// ===== KEY STEP: Trigger Validation for SAVE (not UPDATE) =====
	// Workday uses "save" as the primary trigger; "update" is a sub-case
	if err := h.triggerEngine.TriggerValidate(ctx, tid, "save", "orders", "", orderData); err != nil {
		log.Printf("[OrdersHandler] Save order validation failed: %v", err)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
		return
	}

	// Update DB
	q := `
    UPDATE orders
    SET customer_id = $1, total = $2, updated_at = NOW()
    WHERE id = $3 AND tenant_id = $4
  `
	_, err = h.db.ExecContext(ctx, q, req.CustomerID, req.Total, orderID, tenantID)
	if err != nil {
		http.Error(w, "update failed", http.StatusInternalServerError)
		return
	}

	// TODO: Publish event to RabbitMQ: orders.updated

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"id":     orderID,
		"status": "updated",
	})
}

// HandleDeleteOrder is an example DELETE endpoint
// DELETE /api/orders/{id}
func (h *OrdersHandler) HandleDeleteOrder(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	claims := jwtmiddleware.GetClaimsFromContext(r)
	if claims == nil {
		http.Error(w, `{"error":"unauthorized"}`, http.StatusUnauthorized)
		return
	}
	tenantID := claims.TenantID
	if tenantID == "" {
		http.Error(w, "X-Tenant-ID header required", http.StatusBadRequest)
		return
	}

	orderID := r.PathValue("id")
	if orderID == "" {
		http.Error(w, "order ID required", http.StatusBadRequest)
		return
	}

	// Fetch current order data for validation context
	var currentOrder map[string]interface{}
	q := `SELECT id, customer_id, total FROM orders WHERE id = $1 AND tenant_id = $2`
	err := h.db.QueryRowContext(ctx, q, orderID, tenantID).Scan(&currentOrder)
	if err != nil {
		http.Error(w, "order not found", http.StatusNotFound)
		return
	}

	tid, _ := uuid.Parse(tenantID)

	// ===== KEY STEP: Trigger Validation for DELETE =====
	// Run "delete" triggers to ensure order is safe to delete
	if err := h.triggerEngine.TriggerValidate(ctx, tid, "delete", "orders", "", currentOrder); err != nil {
		log.Printf("[OrdersHandler] Delete order validation failed: %v", err)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
		return
	}

	// Delete from DB
	_, err = h.db.ExecContext(ctx, "DELETE FROM orders WHERE id = $1 AND tenant_id = $2", orderID, tenantID)
	if err != nil {
		http.Error(w, "delete failed", http.StatusInternalServerError)
		return
	}

	// TODO: Publish event to RabbitMQ: orders.deleted

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"status": "deleted"})
}

// HandleFieldChangeValidation is a lightweight endpoint for client-side onChange validation
// Called from UI form when a field changes (e.g., phone input)
// POST /api/validate/field
func (h *OrdersHandler) HandleFieldChangeValidation(w http.ResponseWriter, r *http.Request) {
	// This is already implemented in validation_triggers_handlers.go
	// Just a reference to show the pattern
}

// Example: Wire into chi router
/*
func RegisterOrdersRoutes(r chi.Router, db *sql.DB, triggerEngine *validation.TriggerValidationEngine) {
    handler := NewOrdersHandler(db, triggerEngine)
    r.Route("/orders", func(r chi.Router) {
        r.Post("/", handler.HandleCreateOrder)
        r.Patch("/{id}", handler.HandleUpdateOrder)
        r.Delete("/{id}", handler.HandleDeleteOrder)
    })
}
*/
