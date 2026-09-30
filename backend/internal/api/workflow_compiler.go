package api

import (
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/hondyman/uisce/backend/internal/bp"
	"github.com/hondyman/uisce/backend/internal/handlers"
	"github.com/hondyman/uisce/backend/internal/tenant"
	"github.com/hondyman/uisce/libs/jwt-middleware"
)

// WorkflowCompilerHandler handles HTTP endpoints for compiling and executing workflows
type WorkflowCompilerHandler struct {
	db           *sql.DB
	compiler     *bp.WorkflowCompiler
	securityDeps handlers.SecurityContextDeps
}

// NewWorkflowCompilerHandler creates a new WorkflowCompilerHandler instance
func NewWorkflowCompilerHandler(db *sql.DB, securityDeps handlers.SecurityContextDeps) *WorkflowCompilerHandler {
	subsStore := bp.NewTriggerSubscriptionStore(db)
	compiler := bp.NewWorkflowCompiler(db, subsStore)
	return &WorkflowCompilerHandler{
		db:           db,
		compiler:     compiler,
		securityDeps: securityDeps,
	}
}

// RegisterRoutes registers the compile and execute endpoints on the router
func (h *WorkflowCompilerHandler) RegisterRoutes(r chi.Router) {
	r.Route("/bp", func(r chi.Router) {
		r.Post("/compile", h.HandleCompile)
		r.Post("/execute", h.HandleExecute)
	})
}

// HandleCompile compiles, extends, and publishes a workflow process definition
// POST /api/bp/compile
func (h *WorkflowCompilerHandler) HandleCompile(w http.ResponseWriter, r *http.Request) {
	var req bp.CompileProcessRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, fmt.Sprintf(`{"error":"invalid request payload: %s"}`, err.Error()), http.StatusBadRequest)
		return
	}

	// Resolve tenant ID from JWT claims or request header
	tenantID := req.TenantID
	if claims := jwtmiddleware.GetClaimsFromContext(r); claims != nil && claims.TenantID != "" {
		tenantID = claims.TenantID
	}
	if tenantID == "" {
		tenantID = r.Header.Get("X-Tenant-ID")
	}
	if tenantID == "" {
		http.Error(w, `{"error":"tenant_id is required"}`, http.StatusBadRequest)
		return
	}
	req.TenantID = tenantID

	if req.ProcessID == "" {
		http.Error(w, `{"error":"process_id is required"}`, http.StatusBadRequest)
		return
	}

	// Delegate to extension compiler
	persisted, err := h.compiler.CompileAndPublish(r.Context(), req)
	if err != nil {
		if errors.Is(err, bp.ErrExtensionAnchorNotFound) {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusUnprocessableEntity)
			json.NewEncoder(w).Encode(map[string]interface{}{
				"error": err.Error(),
				"code":  "EXTENSION_ANCHOR_NOT_FOUND",
			})
			return
		}
		if errors.Is(err, bp.ErrBaseDefinitionNotFound) {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusNotFound)
			json.NewEncoder(w).Encode(map[string]interface{}{
				"error": err.Error(),
				"code":  "BASE_DEFINITION_NOT_FOUND",
			})
			return
		}
		if errors.Is(err, bp.ErrInvalidSourceType) {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusBadRequest)
			json.NewEncoder(w).Encode(map[string]interface{}{
				"error": err.Error(),
				"code":  "INVALID_SOURCE_TYPE",
			})
			return
		}

		http.Error(w, fmt.Sprintf(`{"error":"compile failed: %s"}`, err.Error()), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(persisted)
}

// ExecuteWorkflowRequest is the request payload for pre-flight and execution dispatch
type ExecuteWorkflowRequest struct {
	ProcessID   string                 `json:"process_id"`
	TenantID    string                 `json:"tenant_id,omitempty"`
	TriggerName string                 `json:"trigger_name,omitempty"`
	Entity      string                 `json:"entity,omitempty"`
	EntityID    string                 `json:"entity_id,omitempty"`
	EventData   map[string]interface{} `json:"event_data,omitempty"`
}

// HandleExecute runs pre-flight RLS validation and dispatches workflow execution
// POST /api/bp/execute
func (h *WorkflowCompilerHandler) HandleExecute(w http.ResponseWriter, r *http.Request) {
	var req ExecuteWorkflowRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, fmt.Sprintf(`{"error":"invalid request payload: %s"}`, err.Error()), http.StatusBadRequest)
		return
	}

	tenantID := req.TenantID
	if claims := jwtmiddleware.GetClaimsFromContext(r); claims != nil && claims.TenantID != "" {
		tenantID = claims.TenantID
	}
	if tenantID == "" {
		tenantID = r.Header.Get("X-Tenant-ID")
	}
	if tenantID == "" {
		http.Error(w, `{"error":"tenant_id is required"}`, http.StatusBadRequest)
		return
	}

	// Pre-flight check: Verify process definition exists for this tenant under active RLS
	tx, err := h.db.BeginTx(r.Context(), &sql.TxOptions{ReadOnly: true})
	if err != nil {
		http.Error(w, fmt.Sprintf(`{"error":"begin preflight tx failed: %s"}`, err.Error()), http.StatusInternalServerError)
		return
	}
	defer tx.Rollback() //nolint:errcheck

	if err := tenant.SetRLSContext(r.Context(), tx, tenantID); err != nil {
		http.Error(w, fmt.Sprintf(`{"error":"set rls context failed: %s"}`, err.Error()), http.StatusInternalServerError)
		return
	}

	var defID uuid.UUID
	var version int
	err = tx.QueryRowContext(r.Context(), `
		SELECT id, version
		FROM public.bp_process_definition
		WHERE process_id = $1
		ORDER BY version DESC
		LIMIT 1
	`, req.ProcessID).Scan(&defID, &version)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusPreconditionFailed)
			json.NewEncoder(w).Encode(map[string]interface{}{
				"error":      "Process definition not found for tenant under active security policy",
				"code":       "PRECONDITION_FAILED",
				"process_id": req.ProcessID,
			})
			return
		}
		http.Error(w, fmt.Sprintf(`{"error":"query process definition failed: %s"}`, err.Error()), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"status":     "DISPATCHED",
		"process_id": req.ProcessID,
		"version":    version,
		"def_id":     defID,
		"tenant_id":  tenantID,
	})
}
