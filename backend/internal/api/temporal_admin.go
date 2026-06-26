package api

import (
	"database/sql"
	"encoding/json"
	"log"
	"net/http"
	"strconv"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/hondyman/semlayer/backend/internal/identity"
	"github.com/hondyman/semlayer/backend/internal/logging"
	"github.com/hondyman/semlayer/backend/internal/services"
	"github.com/hondyman/semlayer/backend/internal/temporal"
	"go.temporal.io/sdk/client"
	"github.com/hondyman/semlayer/libs/jwt-middleware"
)

// TemporalAdminHandler wraps the Temporal workflow admin service
type TemporalAdminHandler struct {
	adminService *temporal.WorkflowAdminService
	secMgr       *services.SecurityManager
}

// NewTemporalAdminHandler creates a new handler
func NewTemporalAdminHandler(c client.Client, db *sql.DB, sec *services.SecurityManager, adminClient *temporal.AdminClient) *TemporalAdminHandler {
	// Use "default" namespace; adjust if needed
	return &TemporalAdminHandler{
		adminService: temporal.NewWorkflowAdminService(c, "default", db, adminClient),
		secMgr:       sec,
	}
}

// ============================================================================
// SIGNAL ENDPOINT
// ============================================================================

// HandleSignalWorkflow handles POST /api/temporal/workflows/{id}/signal
func (h *TemporalAdminHandler) HandleSignalWorkflow(w http.ResponseWriter, r *http.Request) {
	// RBAC: require temporal.admin permission
	ctx := r.Context()
	userID, ok := identity.ActorIDFromContext(ctx)
	if !ok || userID == "" {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}
	if h.secMgr == nil || !h.secMgr.HasPermission(userID, "temporal.admin") {
		http.Error(w, "forbidden", http.StatusForbidden)
		return
	}

	workflowID := chi.URLParam(r, "id")
	if workflowID == "" {
		http.Error(w, "workflow_id is required", http.StatusBadRequest)
		return
	}

	var req temporal.SignalWorkflowRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}

	req.WorkflowID = workflowID

	// Prepare audit entry
	inputJSON := json.RawMessage("null")
	if len(req.Input) > 0 {
		if b, err := json.Marshal(req.Input); err == nil {
			inputJSON = b
		}
	}
	audit := temporal.AdminActionAudit{
		ID:         uuid.New().String(),
		TenantID:   jwtmiddleware.GetClaimsFromContext(r).TenantID,
		ActorID:    userID,
		Action:     "signal",
		WorkflowID: workflowID,
		RunID:      req.RunID,
		Reason:     req.Reason,
		Input:      inputJSON,
		Timestamp:  time.Now(),
	}

	resp, err := h.adminService.SignalWorkflow(r.Context(), req)
	if err != nil {
		audit.Status = "failed"
		audit.ErrorMessage = err.Error()
		if lgErr := h.adminService.LogAdminAction(r.Context(), audit); lgErr != nil {
			log.Printf("[TemporalAPI] failed to log admin action: %v", lgErr)
		}

		log.Printf("[TemporalAPI] Error signaling workflow: %v", err)
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
		return
	}

	audit.Status = "success"
	if lgErr := h.adminService.LogAdminAction(r.Context(), audit); lgErr != nil {
		log.Printf("[TemporalAPI] failed to log admin action: %v", lgErr)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}

// ============================================================================
// UPDATE ENDPOINT
// ============================================================================

// HandleUpdateWorkflow handles POST /api/temporal/workflows/{id}/update
func (h *TemporalAdminHandler) HandleUpdateWorkflow(w http.ResponseWriter, r *http.Request) {
	// RBAC: require temporal.admin permission
	ctx := r.Context()
	userID, ok := identity.ActorIDFromContext(ctx)
	if !ok || userID == "" {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}
	if h.secMgr == nil || !h.secMgr.HasPermission(userID, "temporal.admin") {
		http.Error(w, "forbidden", http.StatusForbidden)
		return
	}

	workflowID := chi.URLParam(r, "id")
	if workflowID == "" {
		http.Error(w, "workflow_id is required", http.StatusBadRequest)
		return
	}

	var req temporal.UpdateWorkflowRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}

	req.WorkflowID = workflowID

	// Prepare audit entry
	inputJSON := json.RawMessage("null")
	if len(req.Input) > 0 {
		if b, err := json.Marshal(req.Input); err == nil {
			inputJSON = b
		}
	}
	audit := temporal.AdminActionAudit{
		ID:         uuid.New().String(),
		TenantID:   jwtmiddleware.GetClaimsFromContext(r).TenantID,
		ActorID:    userID,
		Action:     "update",
		WorkflowID: workflowID,
		RunID:      req.RunID,
		Reason:     req.Reason,
		Input:      inputJSON,
		Timestamp:  time.Now(),
	}

	resp, err := h.adminService.UpdateWorkflow(r.Context(), req)
	if err != nil {
		audit.Status = "failed"
		audit.ErrorMessage = err.Error()
		if lgErr := h.adminService.LogAdminAction(r.Context(), audit); lgErr != nil {
			log.Printf("[TemporalAPI] failed to log admin action: %v", lgErr)
		}

		log.Printf("[TemporalAPI] Error updating workflow: %v", err)
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
		return
	}

	audit.Status = "success"
	if lgErr := h.adminService.LogAdminAction(r.Context(), audit); lgErr != nil {
		log.Printf("[TemporalAPI] failed to log admin action: %v", lgErr)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}

// ============================================================================
// CANCEL ENDPOINT
// ============================================================================

// HandleCancelWorkflow handles POST /api/temporal/workflows/{id}/cancel
func (h *TemporalAdminHandler) HandleCancelWorkflow(w http.ResponseWriter, r *http.Request) {
	// RBAC: require temporal.admin permission
	ctx := r.Context()
	userID, ok := identity.ActorIDFromContext(ctx)
	if !ok || userID == "" {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}
	if h.secMgr == nil || !h.secMgr.HasPermission(userID, "temporal.admin") {
		http.Error(w, "forbidden", http.StatusForbidden)
		return
	}
	workflowID := chi.URLParam(r, "id")
	if workflowID == "" {
		http.Error(w, "workflow_id is required", http.StatusBadRequest)
		return
	}

	var req temporal.CancelWorkflowRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}

	req.WorkflowID = workflowID

	// Prepare audit entry
	audit := temporal.AdminActionAudit{
		ID:         uuid.New().String(),
		TenantID:   jwtmiddleware.GetClaimsFromContext(r).TenantID,
		ActorID:    userID,
		Action:     "cancel",
		WorkflowID: workflowID,
		RunID:      req.RunID,
		Reason:     req.Reason,
		Timestamp:  time.Now(),
	}

	resp, err := h.adminService.CancelWorkflow(r.Context(), req)
	if err != nil {
		audit.Status = "failed"
		audit.ErrorMessage = err.Error()
		if lgErr := h.adminService.LogAdminAction(r.Context(), audit); lgErr != nil {
			log.Printf("[TemporalAPI] failed to log admin action: %v", lgErr)
		}

		log.Printf("[TemporalAPI] Error canceling workflow: %v", err)
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
		return
	}

	audit.Status = "success"
	if lgErr := h.adminService.LogAdminAction(r.Context(), audit); lgErr != nil {
		log.Printf("[TemporalAPI] failed to log admin action: %v", lgErr)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}

// ============================================================================
// TERMINATE ENDPOINT
// ============================================================================

// HandleTerminateWorkflow handles POST /api/temporal/workflows/{id}/terminate
func (h *TemporalAdminHandler) HandleTerminateWorkflow(w http.ResponseWriter, r *http.Request) {
	// RBAC: require temporal.admin permission
	userID := r.Header.Get("X-User-ID")
	if h.secMgr == nil || !h.secMgr.HasPermission(userID, "temporal.admin") {
		http.Error(w, "forbidden", http.StatusForbidden)
		return
	}
	workflowID := chi.URLParam(r, "id")
	if workflowID == "" {
		http.Error(w, "workflow_id is required", http.StatusBadRequest)
		return
	}

	var req temporal.TerminateWorkflowRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}

	req.WorkflowID = workflowID

	// Prepare audit entry
	audit := temporal.AdminActionAudit{
		ID:         uuid.New().String(),
		TenantID:   jwtmiddleware.GetClaimsFromContext(r).TenantID,
		ActorID:    userID,
		Action:     "terminate",
		WorkflowID: workflowID,
		RunID:      req.RunID,
		Reason:     req.Reason,
		Timestamp:  time.Now(),
	}

	resp, err := h.adminService.TerminateWorkflow(r.Context(), req)
	if err != nil {
		audit.Status = "failed"
		audit.ErrorMessage = err.Error()
		if lgErr := h.adminService.LogAdminAction(r.Context(), audit); lgErr != nil {
			log.Printf("[TemporalAPI] failed to log admin action: %v", lgErr)
		}

		log.Printf("[TemporalAPI] Error terminating workflow: %v", err)
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
		return
	}

	audit.Status = "success"
	if lgErr := h.adminService.LogAdminAction(r.Context(), audit); lgErr != nil {
		log.Printf("[TemporalAPI] failed to log admin action: %v", lgErr)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}

// ============================================================================
// RESET ENDPOINT
// ============================================================================

// HandleResetWorkflow handles POST /api/temporal/workflows/{id}/reset
func (h *TemporalAdminHandler) HandleResetWorkflow(w http.ResponseWriter, r *http.Request) {
	// RBAC: require temporal.admin permission
	userID := r.Header.Get("X-User-ID")
	if h.secMgr == nil || !h.secMgr.HasPermission(userID, "temporal.admin") {
		http.Error(w, "forbidden", http.StatusForbidden)
		return
	}
	workflowID := chi.URLParam(r, "id")
	if workflowID == "" {
		http.Error(w, "workflow_id is required", http.StatusBadRequest)
		return
	}

	var req temporal.ResetWorkflowRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}

	req.WorkflowID = workflowID

	// Prepare audit entry
	audit := temporal.AdminActionAudit{
		ID:         uuid.New().String(),
		TenantID:   jwtmiddleware.GetClaimsFromContext(r).TenantID,
		ActorID:    userID,
		Action:     "reset",
		WorkflowID: workflowID,
		RunID:      req.RunID,
		Reason:     req.Reason,
		Timestamp:  time.Now(),
	}

	resp, err := h.adminService.ResetWorkflow(r.Context(), req)
	if err != nil {
		audit.Status = "failed"
		audit.ErrorMessage = err.Error()
		if lgErr := h.adminService.LogAdminAction(r.Context(), audit); lgErr != nil {
			log.Printf("[TemporalAPI] failed to log admin action: %v", lgErr)
		}

		log.Printf("[TemporalAPI] Error resetting workflow: %v", err)
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
		return
	}

	audit.Status = "success"
	if lgErr := h.adminService.LogAdminAction(r.Context(), audit); lgErr != nil {
		log.Printf("[TemporalAPI] failed to log admin action: %v", lgErr)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}

// ============================================================================
// SEARCH ATTRIBUTES ENDPOINT
// ============================================================================

// HandleSearchAttributeDefinitions handles GET /api/temporal/search-attributes
func (h *TemporalAdminHandler) HandleSearchAttributeDefinitions(w http.ResponseWriter, r *http.Request) {
	searchAttrInit := temporal.NewSearchAttributeInitializer(nil, "default")
	defs := searchAttrInit.GetSearchAttributeDefinitions()

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"status":            "success",
		"search_attributes": defs,
	})
}

// HandleCLISetupScript handles GET /api/temporal/setup-cli-script
// Returns shell script for registering Search Attributes via CLI
func (h *TemporalAdminHandler) HandleCLISetupScript(w http.ResponseWriter, r *http.Request) {
	script := temporal.GenerateCLISetupScript()

	w.Header().Set("Content-Type", "text/plain")
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(script))
}

// HandleExportHistory handles POST /api/temporal/workflows/{id}/history/export
func (h *TemporalAdminHandler) HandleExportHistory(w http.ResponseWriter, r *http.Request) {
	// RBAC: require temporal.admin permission to export history
	ctx := r.Context()
	userID, ok := identity.ActorIDFromContext(ctx)
	if !ok || userID == "" {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}
	if h.secMgr == nil || !h.secMgr.HasPermission(userID, "temporal.admin") {
		http.Error(w, "forbidden", http.StatusForbidden)
		return
	}
	workflowID := chi.URLParam(r, "id")
	if workflowID == "" {
		http.Error(w, "workflow_id is required", http.StatusBadRequest)
		return
	}

	var req temporal.HistoryExportRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}

	req.WorkflowID = workflowID

	// Audit read/export
	audit := temporal.AdminActionAudit{
		ID:         uuid.New().String(),
		TenantID:   jwtmiddleware.GetClaimsFromContext(r).TenantID,
		ActorID:    userID,
		Action:     "export_history",
		WorkflowID: workflowID,
		RunID:      req.RunID,
		Timestamp:  time.Now(),
	}

	resp, err := h.adminService.ExportHistory(r.Context(), req)
	if err != nil {
		audit.Status = "failed"
		audit.ErrorMessage = err.Error()
		if lgErr := h.adminService.LogAdminAction(r.Context(), audit); lgErr != nil {
			log.Printf("[TemporalAPI] failed to log admin action: %v", lgErr)
		}

		log.Printf("[TemporalAPI] Error exporting history: %v", err)
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
		return
	}

	audit.Status = "success"
	if lgErr := h.adminService.LogAdminAction(r.Context(), audit); lgErr != nil {
		log.Printf("[TemporalAPI] failed to log admin action: %v", lgErr)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}

// HandleExportAudit handles GET /api/temporal/workflows/{id}/audit
func (h *TemporalAdminHandler) HandleExportAudit(w http.ResponseWriter, r *http.Request) {
	// RBAC: require temporal.admin permission to view audit
	ctx := r.Context()
	userID, ok := identity.ActorIDFromContext(ctx)
	if !ok || userID == "" {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}
	if h.secMgr == nil || !h.secMgr.HasPermission(userID, "temporal.admin") {
		http.Error(w, "forbidden", http.StatusForbidden)
		return
	}
	workflowID := chi.URLParam(r, "id")
	if workflowID == "" {
		http.Error(w, "workflow_id is required", http.StatusBadRequest)
		return
	}

	runID := r.URL.Query().Get("run_id")

	// Audit read
	audit := temporal.AdminActionAudit{
		ID:         uuid.New().String(),
		TenantID:   jwtmiddleware.GetClaimsFromContext(r).TenantID,
		ActorID:    userID,
		Action:     "export_audit",
		WorkflowID: workflowID,
		RunID:      runID,
		Timestamp:  time.Now(),
	}

	resp, err := h.adminService.ExportAuditTrail(r.Context(), workflowID, runID)
	if err != nil {
		audit.Status = "failed"
		audit.ErrorMessage = err.Error()
		if lgErr := h.adminService.LogAdminAction(r.Context(), audit); lgErr != nil {
			log.Printf("[TemporalAPI] failed to log admin action: %v", lgErr)
		}

		log.Printf("[TemporalAPI] Error exporting audit trail: %v", err)
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
		return
	}

	audit.Status = "success"
	if lgErr := h.adminService.LogAdminAction(r.Context(), audit); lgErr != nil {
		log.Printf("[TemporalAPI] failed to log admin action: %v", lgErr)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{"status": "success", "audit": resp})
}

// HandleStackTrace handles GET /api/temporal/workflows/{id}/stack
func (h *TemporalAdminHandler) HandleStackTrace(w http.ResponseWriter, r *http.Request) {
	// RBAC: require temporal.admin permission to get stack trace
	ctx := r.Context()
	userID, ok := identity.ActorIDFromContext(ctx)
	if !ok || userID == "" {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}
	if h.secMgr == nil || !h.secMgr.HasPermission(userID, "temporal.admin") {
		http.Error(w, "forbidden", http.StatusForbidden)
		return
	}
	workflowID := chi.URLParam(r, "id")
	if workflowID == "" {
		http.Error(w, "workflow_id is required", http.StatusBadRequest)
		return
	}

	runID := r.URL.Query().Get("run_id")
	// Audit read
	audit := temporal.AdminActionAudit{
		ID:         uuid.New().String(),
		TenantID:   jwtmiddleware.GetClaimsFromContext(r).TenantID,
		ActorID:    userID,
		Action:     "stack_trace",
		WorkflowID: workflowID,
		RunID:      runID,
		Timestamp:  time.Now(),
	}

	resp, err := h.adminService.StackTrace(r.Context(), workflowID, runID)
	if err != nil {
		audit.Status = "failed"
		audit.ErrorMessage = err.Error()
		if lgErr := h.adminService.LogAdminAction(r.Context(), audit); lgErr != nil {
			log.Printf("[TemporalAPI] failed to log admin action: %v", lgErr)
		}

		log.Printf("[TemporalAPI] Error querying stack trace: %v", err)
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
		return
	}

	audit.Status = "success"
	if lgErr := h.adminService.LogAdminAction(r.Context(), audit); lgErr != nil {
		log.Printf("[TemporalAPI] failed to log admin action: %v", lgErr)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{"status": "success", "stack": resp})
}

// HandleListExecutions handles GET /api/temporal/executions
// Returns a small set of recent executions taken from the temporal_workflows projection
// table. This endpoint intentionally returns a compact JSON array suitable for UI lists.
func (h *TemporalAdminHandler) HandleListExecutions(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// RBAC: require temporal.admin permission for admin actions
	userID, ok := identity.ActorIDFromContext(ctx)
	if !ok || userID == "" {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}
	if h.secMgr == nil || !h.secMgr.HasPermission(userID, "temporal.admin") {
		http.Error(w, "forbidden", http.StatusForbidden)
		return
	}

	// allow optional ?limit= param
	limit := 200
	if q := r.URL.Query().Get("limit"); q != "" {
		if v, err := strconv.Atoi(q); err == nil && v > 0 {
			limit = v
		}
	}

	executions, err := h.adminService.ListExecutions(ctx, limit)
	if err != nil {
		logging.GetLogger().Sugar().Warnf("temporal: failed to list executions: %v", err)
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(executions)
}

// HandleSaveWorkflow handles POST /api/temporal/workflows
// Persist a designer workflow representation into the temporal_workflows projection
// Requires temporal.admin permission.
func (h *TemporalAdminHandler) HandleSaveWorkflow(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// RBAC: require temporal.admin permission
	userID, ok := identity.ActorIDFromContext(ctx)
	if !ok || userID == "" {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}
	if h.secMgr == nil || !h.secMgr.HasPermission(userID, "temporal.admin") {
		http.Error(w, "forbidden", http.StatusForbidden)
		return
	}

	// Read request body as flexible JSON
	var payload map[string]interface{}
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}

	claims := jwtmiddleware.GetClaimsFromContext(r)
	if claims == nil {
		http.Error(w, `{"error":"unauthorized"}`, http.StatusUnauthorized)
		return
	}
	tenantID := claims.TenantID

	// Prepare an audit entry for this create action
	audit := temporal.AdminActionAudit{
		ID:         uuid.New().String(),
		TenantID:   tenantID,
		ActorID:    userID,
		Action:     "create",
		WorkflowID: "",
		Timestamp:  time.Now(),
	}

	id, err := h.adminService.SaveWorkflow(ctx, tenantID, payload)
	if err != nil {
		audit.Status = "failed"
		audit.ErrorMessage = err.Error()
		if lgErr := h.adminService.LogAdminAction(ctx, audit); lgErr != nil {
			log.Printf("[TemporalAPI] failed to log admin action: %v", lgErr)
		}

		log.Printf("[TemporalAPI] Error saving workflow: %v", err)
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
		return
	}

	audit.Status = "success"
	audit.WorkflowID = id
	if lgErr := h.adminService.LogAdminAction(ctx, audit); lgErr != nil {
		log.Printf("[TemporalAPI] failed to log admin action: %v", lgErr)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{"status": "success", "id": id})
}

// HandleDescribeTaskQueue handles GET /api/temporal/taskqueue/describe
func (h *TemporalAdminHandler) HandleDescribeTaskQueue(w http.ResponseWriter, r *http.Request) {
	// RBAC: require temporal.admin permission to view task queue info
	ctx := r.Context()
	userID, _ := identity.ActorIDFromContext(ctx)
	if userID == "" {
		userID = r.Header.Get("X-User-ID") // fallback
	}
	if h.secMgr == nil || !h.secMgr.HasPermission(userID, "temporal.admin") {
		http.Error(w, "forbidden", http.StatusForbidden)
		return
	}

	queue := r.URL.Query().Get("queue")
	if queue == "" {
		http.Error(w, "queue parameter required", http.StatusBadRequest)
		return
	}

	// Audit read
	audit := temporal.AdminActionAudit{
		ID:         uuid.New().String(),
		TenantID:   jwtmiddleware.GetClaimsFromContext(r).TenantID,
		ActorID:    userID,
		Action:     "describe_taskqueue",
		WorkflowID: "",
		Timestamp:  time.Now(),
	}

	resp, err := h.adminService.DescribeTaskQueue(r.Context(), queue)
	if err != nil {
		audit.Status = "failed"
		audit.ErrorMessage = err.Error()
		if lgErr := h.adminService.LogAdminAction(r.Context(), audit); lgErr != nil {
			log.Printf("[TemporalAPI] failed to log admin action: %v", lgErr)
		}

		log.Printf("[TemporalAPI] Error describing task queue: %v", err)
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
		return
	}

	audit.Status = "success"
	if lgErr := h.adminService.LogAdminAction(r.Context(), audit); lgErr != nil {
		log.Printf("[TemporalAPI] failed to log admin action: %v", lgErr)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{"status": "success", "queue": resp})
}

// ============================================================================
// ROUTER REGISTRATION
// ============================================================================

// RegisterTemporalAdminRoutes registers all Temporal admin endpoints
// Call this from api.go in the r.Route("/api", ...) block
func RegisterTemporalAdminRoutes(r chi.Router, temporalClient client.Client, db *sql.DB, secMgr *services.SecurityManager, adminClient *temporal.AdminClient) {
	handler := NewTemporalAdminHandler(temporalClient, db, secMgr, adminClient)

	r.Route("/temporal", func(r chi.Router) {
		// create/save workflow (designer)
		r.Post("/workflows", handler.HandleSaveWorkflow)

		// Workflow control endpoints
		r.Route("/workflows/{id}", func(r chi.Router) {
			r.Post("/signal", handler.HandleSignalWorkflow)
			r.Post("/update", handler.HandleUpdateWorkflow)
			r.Post("/cancel", handler.HandleCancelWorkflow)
			r.Post("/terminate", handler.HandleTerminateWorkflow)
			r.Post("/reset", handler.HandleResetWorkflow)
		})

		// Search Attributes endpoints
		r.Get("/search-attributes", handler.HandleSearchAttributeDefinitions)
		r.Get("/setup-cli-script", handler.HandleCLISetupScript)

		// Lightweight listing of recent executions (projections table)
		r.Get("/executions", handler.HandleListExecutions)
		// History export / audit endpoints
		r.Post("/workflows/{id}/history/export", handler.HandleExportHistory)
		r.Get("/workflows/{id}/audit", handler.HandleExportAudit)
		r.Get("/workflows/{id}/stack", handler.HandleStackTrace)
		// Task queue info
		r.Get("/taskqueue/describe", handler.HandleDescribeTaskQueue)
	})
}
