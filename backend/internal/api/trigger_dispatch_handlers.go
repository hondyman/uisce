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

// ============================================================================
// PHASE 6A: TRIGGER DISPATCH INTEGRATION EXAMPLE
// ============================================================================
// This file demonstrates how to integrate the trigger dispatch system
// into your API endpoints. Each handler shows the pattern:
//
// 1. Parse request
// 2. DispatchTrigger() before DB commit
// 3. If validation passes -> save to DB
// 4. If validation fails -> return 400 with error
//
// Trigger types covered:
// - Create (INSERT new entity)
// - Save (UPDATE existing entity)
// - Delete (REMOVE entity)
// - FieldChange (single field onChange)
// - StatusChange (state transition validation)
// - SubEntityChange (child entity modification)
// - RelationshipChange (FK/link modification)
// ============================================================================

// OrderWithLineItemsRequest represents an order with nested line items
type OrderWithLineItemsRequest struct {
	CustomerID string  `json:"customer_id"`
	Total      float64 `json:"total"`
	Status     string  `json:"status"`
	Items      []struct {
		SKU      string  `json:"sku"`
		Qty      int     `json:"qty"`
		Price    float64 `json:"price"`
		Discount float64 `json:"discount"`
	} `json:"items"`
}

// TriggerDispatchHandler demonstrates all trigger dispatch patterns
type TriggerDispatchHandler struct {
	db            *sql.DB
	triggerEngine *validation.TriggerValidationEngine
}

// NewTriggerDispatchHandler creates a handler with trigger dispatch
func NewTriggerDispatchHandler(db *sql.DB, triggerEngine *validation.TriggerValidationEngine) *TriggerDispatchHandler {
	if triggerEngine == nil {
		triggerEngine = validation.NewTriggerValidationEngine(db, &validation.SimpleLogger{})
	}
	return &TriggerDispatchHandler{
		db:            db,
		triggerEngine: triggerEngine,
	}
}

// ============================================================================
// 1. CREATE TRIGGER - Insert new entity
// ============================================================================
// POST /api/dispatch/orders
// Demonstrates: DispatchTrigger with TriggerTypeCreate
//
// Flow:
//
//	User submits new order form
//	→ DispatchTrigger("create", "orders", data)
//	→ ValidationEngine fetches all "create" triggers for "orders"
//	→ Evaluates each trigger's rules
//	→ If any rule fails: return 400 error (order not created)
//	→ If all pass: insert into database
func (h *TriggerDispatchHandler) HandleCreateOrderWithDispatch(w http.ResponseWriter, r *http.Request) {
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

	var req OrderWithLineItemsRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}

	orderData := map[string]interface{}{
		"customer_id": req.CustomerID,
		"total":       req.Total,
		"status":      req.Status,
		"items_count": len(req.Items),
	}

	tid, err := uuid.Parse(tenantID)
	if err != nil {
		http.Error(w, "invalid tenant ID", http.StatusBadRequest)
		return
	}

	// ===== STEP 1: DISPATCH CREATE TRIGGER =====
	// This calls h.triggerEngine.DispatchTrigger(ctx, tid, TriggerTypeCreate, "orders", orderData)
	// Which fetches all "create" triggers for "orders" entity and evaluates their rules
	if err := h.triggerEngine.DispatchTrigger(ctx, tid, validation.TriggerTypeCreate, "orders", orderData); err != nil {
		log.Printf("[TriggerDispatch] Create order validation failed: %v", err)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"error": err.Error(),
			"type":  "create_validation_failed",
		})
		return
	}

	// ===== STEP 2: VALIDATION PASSED - SAFE TO SAVE =====
	orderID := uuid.New().String()
	q := `
    INSERT INTO orders (id, tenant_id, customer_id, total, status, created_at)
    VALUES ($1, $2, $3, $4, $5, NOW())
  `
	_, err = h.db.ExecContext(ctx, q, orderID, tenantID, req.CustomerID, req.Total, req.Status)
	if err != nil {
		log.Printf("[TriggerDispatch] DB insert failed: %v", err)
		http.Error(w, "insert failed", http.StatusInternalServerError)
		return
	}

	// TODO: Publish RabbitMQ event: orders.created

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"id":      orderID,
		"status":  "created",
		"message": "Order created successfully (all triggers passed)",
	})
}

// ============================================================================
// 2. SAVE TRIGGER - Update existing entity
// ============================================================================
// PATCH /api/dispatch/orders/{id}
// Demonstrates: DispatchTrigger with TriggerTypeSave
//
// Flow:
//
//	User edits existing order form
//	→ DispatchTrigger("save", "orders", updatedData)
//	→ Validates all "save" triggers for "orders"
//	→ If any rule fails: return 400 (order not updated)
//	→ If all pass: update database
func (h *TriggerDispatchHandler) HandleUpdateOrderWithDispatch(w http.ResponseWriter, r *http.Request) {
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
		http.Error(w, "order ID required in path", http.StatusBadRequest)
		return
	}

	var req OrderWithLineItemsRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}

	orderData := map[string]interface{}{
		"customer_id": req.CustomerID,
		"total":       req.Total,
		"status":      req.Status,
	}

	tid, err := uuid.Parse(tenantID)
	if err != nil {
		http.Error(w, "invalid tenant ID", http.StatusBadRequest)
		return
	}

	// ===== STEP 1: DISPATCH SAVE TRIGGER =====
	// Validates "save" triggers (these are for any entity update)
	if err := h.triggerEngine.DispatchTrigger(ctx, tid, validation.TriggerTypeSave, "orders", orderData); err != nil {
		log.Printf("[TriggerDispatch] Save order validation failed: %v", err)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"error": err.Error(),
			"type":  "save_validation_failed",
		})
		return
	}

	// ===== STEP 2: VALIDATION PASSED - SAFE TO UPDATE =====
	q := `
    UPDATE orders
    SET customer_id = $1, total = $2, status = $3, updated_at = NOW()
    WHERE id = $4 AND tenant_id = $5
  `
	_, err = h.db.ExecContext(ctx, q, req.CustomerID, req.Total, req.Status, orderID, tenantID)
	if err != nil {
		http.Error(w, "update failed", http.StatusInternalServerError)
		return
	}

	// TODO: Publish RabbitMQ event: orders.updated

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"id":      orderID,
		"status":  "updated",
		"message": "Order updated successfully (all triggers passed)",
	})
}

// ============================================================================
// 3. DELETE TRIGGER - Remove entity
// ============================================================================
// DELETE /api/dispatch/orders/{id}
// Demonstrates: DispatchTrigger with TriggerTypeDelete
//
// Flow:
//
//	User confirms delete order
//	→ DispatchTrigger("delete", "orders", currentOrderData)
//	→ Validates "delete" triggers (e.g., "cannot delete if shipped")
//	→ If any rule fails: return 400 (order not deleted)
//	→ If all pass: delete from database
func (h *TriggerDispatchHandler) HandleDeleteOrderWithDispatch(w http.ResponseWriter, r *http.Request) {
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

	tid, err := uuid.Parse(tenantID)
	if err != nil {
		http.Error(w, "invalid tenant ID", http.StatusBadRequest)
		return
	}

	// Fetch current order data for validation context
	var customerID string
	var total float64
	var status string
	q := `SELECT customer_id, total, status FROM orders WHERE id = $1 AND tenant_id = $2`
	err = h.db.QueryRowContext(ctx, q, orderID, tenantID).Scan(&customerID, &total, &status)
	if err != nil {
		http.Error(w, "order not found", http.StatusNotFound)
		return
	}

	currentOrder := map[string]interface{}{
		"customer_id": customerID,
		"total":       total,
		"status":      status,
	}

	// ===== STEP 1: DISPATCH DELETE TRIGGER =====
	// Validates "delete" triggers (e.g., "cannot delete shipped orders")
	if err := h.triggerEngine.DispatchTrigger(ctx, tid, validation.TriggerTypeDelete, "orders", currentOrder); err != nil {
		log.Printf("[TriggerDispatch] Delete order validation failed: %v", err)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"error": err.Error(),
			"type":  "delete_validation_failed",
		})
		return
	}

	// ===== STEP 2: VALIDATION PASSED - SAFE TO DELETE =====
	_, err = h.db.ExecContext(ctx, "DELETE FROM orders WHERE id = $1 AND tenant_id = $2", orderID, tenantID)
	if err != nil {
		http.Error(w, "delete failed", http.StatusInternalServerError)
		return
	}

	// TODO: Publish RabbitMQ event: orders.deleted

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"status":  "deleted",
		"message": "Order deleted successfully (all triggers passed)",
	})
}

// ============================================================================
// 4. FIELD CHANGE TRIGGER - Single field modification (onChange)
// ============================================================================
// POST /api/dispatch/orders/{id}/field-change
// Demonstrates: DispatchFieldChange
//
// Flow:
//
//	User changes order total in form: 100 → -50
//	→ DispatchFieldChange("orders", "total", 100, -50, recordData)
//	→ Validates "field_change" triggers + field-format rules
//	→ If any rule fails: return 400 (show error to user, don't save)
//	→ If all pass: allow field update
//
// Real-world example:
//
//	Form shows: "Order Total: 100"
//	User types: -50
//	onChange event fires → POST /api/dispatch/orders/123/field-change
//	Trigger checks: "Total must be positive" → FAIL
//	UI shows error: "Order total must be greater than 0"
func (h *TriggerDispatchHandler) HandleFieldChangeDispatch(w http.ResponseWriter, r *http.Request) {
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

	var req struct {
		FieldName string      `json:"field_name"`
		OldValue  interface{} `json:"old_value"`
		NewValue  interface{} `json:"new_value"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}

	tid, err := uuid.Parse(tenantID)
	if err != nil {
		http.Error(w, "invalid tenant ID", http.StatusBadRequest)
		return
	}

	// Fetch current order as context
	var customerID string
	var total float64
	var status string
	q := `SELECT customer_id, total, status FROM orders WHERE id = $1 AND tenant_id = $2`
	err = h.db.QueryRowContext(ctx, q, orderID, tenantID).Scan(&customerID, &total, &status)
	if err != nil {
		http.Error(w, "order not found", http.StatusNotFound)
		return
	}

	record := map[string]interface{}{
		"id":          orderID,
		"customer_id": customerID,
		"total":       total,
		"status":      status,
		req.FieldName: req.NewValue, // Updated value
	}

	// ===== STEP 1: DISPATCH FIELD CHANGE TRIGGER =====
	// This is perfect for onChange in forms - quick validation without saving
	if err := h.triggerEngine.DispatchFieldChange(ctx, tid, "orders", req.FieldName,
		req.OldValue, req.NewValue, record); err != nil {
		log.Printf("[TriggerDispatch] Field change validation failed: %v", err)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"error":      err.Error(),
			"type":       "field_change_validation_failed",
			"field":      req.FieldName,
			"old_value":  req.OldValue,
			"new_value":  req.NewValue,
			"suggestion": "Please correct the field value and try again",
		})
		return
	}

	// ===== STEP 2: VALIDATION PASSED - FIELD CHANGE OK =====
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"status":    "valid",
		"field":     req.FieldName,
		"message":   "Field change validation passed",
		"new_value": req.NewValue,
	})
}

// ============================================================================
// 5. STATUS CHANGE TRIGGER - State transition validation
// ============================================================================
// POST /api/dispatch/orders/{id}/status-change
// Demonstrates: DispatchStatusChange
//
// Flow:
//
//	Order transitions: pending → approved → completed
//	→ DispatchStatusChange("orders", "status", "pending", "approved", record)
//	→ Validates status_change triggers (e.g., "only managers can approve")
//	→ If any rule fails: return 400 (transition not allowed)
//	→ If all pass: update order status
//
// Real-world example:
//
//	Order is in "pending" status
//	Manager clicks "Approve" button
//	→ POST /api/dispatch/orders/123/status-change
//	Trigger checks: "Manager role required", "Total < 10k" rules
//	PASS: Order status → "approved"
//	FAIL: Show error "Cannot approve orders over $10,000"
func (h *TriggerDispatchHandler) HandleStatusChangeDispatch(w http.ResponseWriter, r *http.Request) {
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

	var req struct {
		NewStatus string `json:"new_status"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}

	tid, err := uuid.Parse(tenantID)
	if err != nil {
		http.Error(w, "invalid tenant ID", http.StatusBadRequest)
		return
	}

	// Fetch current order
	var customerID string
	var total float64
	var currentStatus string
	q := `SELECT customer_id, total, status FROM orders WHERE id = $1 AND tenant_id = $2`
	err = h.db.QueryRowContext(ctx, q, orderID, tenantID).Scan(&customerID, &total, &currentStatus)
	if err != nil {
		http.Error(w, "order not found", http.StatusNotFound)
		return
	}

	record := map[string]interface{}{
		"id":          orderID,
		"customer_id": customerID,
		"total":       total,
		"status":      req.NewStatus, // New status
	}

	// ===== STEP 1: DISPATCH STATUS CHANGE TRIGGER =====
	if err := h.triggerEngine.DispatchStatusChange(ctx, tid, "orders", "status",
		currentStatus, req.NewStatus, record); err != nil {
		log.Printf("[TriggerDispatch] Status change validation failed: %v", err)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"error":      err.Error(),
			"type":       "status_change_validation_failed",
			"old_status": currentStatus,
			"new_status": req.NewStatus,
			"suggestion": "Review order requirements before transitioning status",
		})
		return
	}

	// ===== STEP 2: VALIDATION PASSED - SAFE TO TRANSITION STATUS =====
	_, err = h.db.ExecContext(ctx, "UPDATE orders SET status = $1 WHERE id = $2 AND tenant_id = $3",
		req.NewStatus, orderID, tenantID)
	if err != nil {
		http.Error(w, "status update failed", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"status":     "updated",
		"old_status": currentStatus,
		"new_status": req.NewStatus,
		"message":    "Status transitioned successfully",
	})
}

// ============================================================================
// 6. SUB-ENTITY CHANGE TRIGGER - Child entity modification
// ============================================================================
// POST /api/dispatch/orders/{order_id}/line-items
// Demonstrates: DispatchSubEntityChange
//
// Flow:
//
//	User adds line item to order
//	→ DispatchSubEntityChange("orders", orderID, "order_items", lineItemData)
//	→ Validates sub_entity_change triggers
//	→ If any rule fails: return 400 (item not added)
//	→ If all pass: insert line item into database
func (h *TriggerDispatchHandler) HandleAddLineItemWithDispatch(w http.ResponseWriter, r *http.Request) {
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

	var req struct {
		SKU      string  `json:"sku"`
		Qty      int     `json:"qty"`
		Price    float64 `json:"price"`
		Discount float64 `json:"discount"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}

	tid, err := uuid.Parse(tenantID)
	if err != nil {
		http.Error(w, "invalid tenant ID", http.StatusBadRequest)
		return
	}

	orderIDUUID, err := uuid.Parse(orderID)
	if err != nil {
		http.Error(w, "invalid order ID", http.StatusBadRequest)
		return
	}

	lineItemData := map[string]interface{}{
		"sku":      req.SKU,
		"qty":      req.Qty,
		"price":    req.Price,
		"discount": req.Discount,
	}

	// ===== STEP 1: DISPATCH SUB-ENTITY CHANGE TRIGGER =====
	if err := h.triggerEngine.DispatchSubEntityChange(ctx, tid, "orders", orderIDUUID, "order_items", lineItemData); err != nil {
		log.Printf("[TriggerDispatch] Line item validation failed: %v", err)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"error": err.Error(),
			"type":  "line_item_validation_failed",
		})
		return
	}

	// ===== STEP 2: VALIDATION PASSED - SAFE TO ADD LINE ITEM =====
	itemID := uuid.New().String()
	q := `
    INSERT INTO order_items (id, order_id, tenant_id, sku, qty, price, discount, created_at)
    VALUES ($1, $2, $3, $4, $5, $6, $7, NOW())
  `
	_, err = h.db.ExecContext(ctx, q, itemID, orderID, tenantID, req.SKU, req.Qty, req.Price, req.Discount)
	if err != nil {
		http.Error(w, "insert failed", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"id":      itemID,
		"status":  "added",
		"message": "Line item added successfully",
	})
}

// ============================================================================
// REGISTRATION HELPER
// ============================================================================
// Example: Wire into chi router
//
// func RegisterTriggerDispatchRoutes(r chi.Router, db *sql.DB, triggerEngine *validation.TriggerValidationEngine) {
//     handler := NewTriggerDispatchHandler(db, triggerEngine)
//     r.Route("/dispatch", func(r chi.Router) {
//         r.Route("/orders", func(r chi.Router) {
//             r.Post("/", handler.HandleCreateOrderWithDispatch)
//             r.Patch("/{id}", handler.HandleUpdateOrderWithDispatch)
//             r.Delete("/{id}", handler.HandleDeleteOrderWithDispatch)
//             r.Post("/{id}/field-change", handler.HandleFieldChangeDispatch)
//             r.Post("/{id}/status-change", handler.HandleStatusChangeDispatch)
//             r.Post("/{id}/line-items", handler.HandleAddLineItemWithDispatch)
//         })
//     })
// }
//
