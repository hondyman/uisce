package api

import (
	"database/sql"
	"encoding/json"
	"net/http"
	"strconv"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/hondyman/uisce/backend/internal/temporal"
	"github.com/hondyman/uisce/libs/jwt-middleware"
	"go.temporal.io/sdk/client"
)

// BPRunHandler handles run history queries for business users and operators
type BPRunHandler struct {
	tracker *temporal.BPRunTracker
	tc      client.Client
}

// NewBPRunHandler creates a new handler
func NewBPRunHandler(db *sql.DB, tc client.Client) *BPRunHandler {
	return &BPRunHandler{
		tracker: temporal.NewBPRunTracker(db, tc),
		tc:      tc,
	}
}

// RegisterRoutes registers the BP run history routes
func (h *BPRunHandler) RegisterRoutes(r chi.Router) {
	r.Get("/api/bp/runs", h.HandleListRuns)
	r.Get("/api/bp/runs/{id}", h.HandleGetRun)
	r.Post("/api/bp/runs/reconcile", h.HandleReconcileRuns)
}

// HandleListRuns queries workflow execution history filtered by entity, status, etc.
func (h *BPRunHandler) HandleListRuns(w http.ResponseWriter, r *http.Request) {
	tenantID := jwtmiddleware.GetTenantIDFromContext(r)
	if tenantID == "" {
		tenantID = r.Header.Get("X-Tenant-ID")
	}
	if tenantID == "" {
		http.Error(w, `{"error":"missing tenant context"}`, http.StatusUnauthorized)
		return
	}

	q := r.URL.Query()
	entity := q.Get("entity")
	entityID := q.Get("entity_id")
	status := q.Get("status")
	processID := q.Get("process_id")

	limit, _ := strconv.Atoi(q.Get("limit"))
	offset, _ := strconv.Atoi(q.Get("offset"))

	// Auto-reconcile stale runs if requested or on broad queries
	if q.Get("reconcile") == "true" {
		_, _ = h.tracker.ReconcileStaleRuns(r.Context(), tenantID, 5*time.Minute)
	}

	runs, err := h.tracker.ListRuns(r.Context(), temporal.WorkflowRunFilter{
		TenantID:  tenantID,
		Entity:    entity,
		EntityID:  entityID,
		Status:    status,
		ProcessID: processID,
		Limit:     limit,
		Offset:    offset,
	})
	if err != nil {
		http.Error(w, `{"error":"`+err.Error()+`"}`, http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]interface{}{
		"runs":  runs,
		"count": len(runs),
	})
}

// HandleGetRun returns a single run record with full payloads
func (h *BPRunHandler) HandleGetRun(w http.ResponseWriter, r *http.Request) {
	tenantID := jwtmiddleware.GetTenantIDFromContext(r)
	if tenantID == "" {
		tenantID = r.Header.Get("X-Tenant-ID")
	}
	if tenantID == "" {
		http.Error(w, `{"error":"missing tenant context"}`, http.StatusUnauthorized)
		return
	}

	id := chi.URLParam(r, "id")
	if id == "" {
		http.Error(w, `{"error":"missing run id"}`, http.StatusBadRequest)
		return
	}

	run, err := h.tracker.GetRun(r.Context(), tenantID, id)
	if err != nil {
		http.Error(w, `{"error":"`+err.Error()+`"}`, http.StatusInternalServerError)
		return
	}
	if run == nil {
		http.Error(w, `{"error":"run not found"}`, http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(run)
}

// HandleReconcileRuns manually triggers reconciliation against Temporal Visibility API
func (h *BPRunHandler) HandleReconcileRuns(w http.ResponseWriter, r *http.Request) {
	tenantID := jwtmiddleware.GetTenantIDFromContext(r)
	if tenantID == "" {
		tenantID = r.Header.Get("X-Tenant-ID")
	}

	count, err := h.tracker.ReconcileStaleRuns(r.Context(), tenantID, 2*time.Minute)
	if err != nil {
		http.Error(w, `{"error":"`+err.Error()+`"}`, http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]interface{}{
		"reconciled_count": count,
		"status":           "success",
	})
}
