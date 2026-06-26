package api

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/hondyman/semlayer/backend/internal/business_process"
	"go.temporal.io/sdk/client"
)

// ============================================================================
// PHASE 6B: BUSINESS PROCESS API ENDPOINTS
// ============================================================================
// REST API for BP management:
// - POST   /api/bp                    → Create new BP
// - GET    /api/bp                    → List all BPs
// - GET    /api/bp/:id                → Get BP details
// - POST   /api/bp/:id/start          → Start BP execution
// - GET    /api/bp/instance/:id       → Get instance status
// - POST   /api/bp/instance/:id/approve → Approve pending step
//
// All endpoints require tenant_id and datasource_id in query params.
// ============================================================================

// CreateBPRequest is the request body for creating a BP
type CreateBPRequest struct {
	ProcessName string         `json:"process_name"`
	Description string         `json:"description"`
	Steps       []CreateBPStep `json:"steps"`
}

// CreateBPStep represents a step when creating a BP
type CreateBPStep struct {
	StepOrder     int             `json:"step_order"`
	StepType      string          `json:"step_type"`
	StepName      string          `json:"step_name"`
	DurationHours int             `json:"duration_hours"`
	AssigneeRole  string          `json:"assignee_role"`
	AssigneeUser  string          `json:"assignee_user"`
	TriggerIDs    []string        `json:"trigger_ids"`
	ConditionJSON json.RawMessage `json:"condition_json"`
	ActionConfig  json.RawMessage `json:"action_config"`
}

// BPResponse is the response for BP details
type BPResponse struct {
	ID          string    `json:"id"`
	ProcessName string    `json:"process_name"`
	Description string    `json:"description"`
	IsActive    bool      `json:"is_active"`
	Version     int       `json:"version"`
	StepCount   int       `json:"step_count"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

// StartBPRequest is the request body for starting a BP execution
type StartBPRequest struct {
	EntityID   string                 `json:"entity_id"`
	EntityType string                 `json:"entity_type"`
	Data       map[string]interface{} `json:"data"`
}

// BPInstanceResponse is the response for BP instance status
type BPInstanceResponse struct {
	InstanceID         string                 `json:"instance_id"`
	ProcessID          string                 `json:"process_id"`
	ProcessName        string                 `json:"process_name"`
	EntityID           string                 `json:"entity_id"`
	EntityType         string                 `json:"entity_type"`
	CurrentStep        int                    `json:"current_step"`
	Status             string                 `json:"status"`
	InstanceData       map[string]interface{} `json:"instance_data"`
	StartedAt          time.Time              `json:"started_at"`
	CurrentStepStartAt time.Time              `json:"current_step_started_at"`
	CurrentStepDueAt   time.Time              `json:"current_step_due_at"`
	TemporalWorkflowID string                 `json:"temporal_workflow_id,omitempty"`
	CreatedAt          time.Time              `json:"created_at"`
}

// ApproveStepRequest is the request body for approving a pending step
type ApproveStepRequest struct {
	Decision string `json:"decision"`
	Comment  string `json:"comment"`
	Reason   string `json:"reason"`
}

// ============================================================================
// REGISTRATION
// ============================================================================
// Register routes in your chi router:
//
//	r.Post("/api/bp", APICreateBusinessProcess(server))
//	r.Get("/api/bp", APIListBusinessProcesses(server))
//	r.Get("/api/bp/{id}", APIGetBusinessProcess(server))
//	r.Get("/api/bp/{id}/audit", APIGetBusinessProcessAuditTrail(server))
//	r.Post("/api/bp/{id}/start", APIStartBusinessProcessExecution(server))
//	r.Get("/api/bp/instance/{id}", APIGetBusinessProcessInstanceStatus(server))
//	r.Post("/api/bp/instance/{id}/approve", APIApproveBusinessProcessStep(server))
// Register routes in your chi router:
//
//	r.Post("/api/bp", APICreateBusinessProcess(server))
//	r.Get("/api/bp", APIListBusinessProcesses(server))
//	r.Get("/api/bp/{id}", APIGetBusinessProcess(server))
//	r.Get("/api/bp/{id}/audit", APIGetBusinessProcessAuditTrail(server))
//	r.Post("/api/bp/{id}/start", APIStartBusinessProcessExecution(server))
//	r.Get("/api/bp/instance/{id}", APIGetBusinessProcessInstanceStatus(server))
//	r.Post("/api/bp/instance/{id}/approve", APIApproveBusinessProcessStep(server))
func APICreateBusinessProcess(s *Server) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		log.Println("[APICreateBusinessProcess] Creating new BP")

		// Check permissions - requires design role
		if err := checkBusinessProcessPermission(r, "design"); err != nil {
			log.Printf("[APICreateBusinessProcess] Permission denied: %v", err)
			http.Error(w, err.Error(), http.StatusForbidden)
			return
		}

		tenantID := r.URL.Query().Get("tenant_id")
		datasourceID := r.URL.Query().Get("datasource_id")
		if tenantID == "" || datasourceID == "" {
			http.Error(w, "tenant_id and datasource_id required", http.StatusBadRequest)
			return
		}

		// Parse the request as a ProcessTemplate (from internal/business_process/types.go)
		// We define a local struct or use map for flexibility if we don't want to import the internal package here directly to avoid cycles if any.
		// But ideally we should import it. For now, let's use a generic map or struct matching the JSON.
		var templateReq struct {
			Name        string          `json:"name"`
			Description string          `json:"description"`
			Steps       json.RawMessage `json:"steps"`
			Transitions json.RawMessage `json:"transitions"`
			Audit       json.RawMessage `json:"audit"`
		}

		// We also want to store the raw JSON
		bodyBytes, err := io.ReadAll(r.Body)
		if err != nil {
			http.Error(w, "failed to read body", http.StatusBadRequest)
			return
		}

		// Restore body for decoding
		r.Body = io.NopCloser(bytes.NewBuffer(bodyBytes))
		
		err = json.NewDecoder(r.Body).Decode(&templateReq)
		if err != nil {
			http.Error(w, fmt.Sprintf("invalid request: %v", err), http.StatusBadRequest)
			return
		}

		if templateReq.Name == "" {
			http.Error(w, "name required", http.StatusBadRequest)
			return
		}

		bpID := uuid.New().String()
		q := `
			INSERT INTO business_processes (id, tenant_id, datasource_id, process_name, description, is_active, version, template_json, created_at, updated_at)
			VALUES ($1, $2, $3, $4, $5, true, 1, $6, NOW(), NOW())
		`

		_, err = s.DB.ExecContext(r.Context(), q, bpID, tenantID, datasourceID, templateReq.Name, templateReq.Description, bodyBytes)
		if err != nil {
			log.Printf("[APICreateBusinessProcess] Error: %v", err)
			http.Error(w, fmt.Sprintf("failed to create BP: %v", err), http.StatusInternalServerError)
			return
		}

		// We skip inserting into bp_steps for now as we are moving to template_json source of truth.
		// Or we could map the graph to steps if needed for legacy compatibility.

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{"id": bpID, "message": "BP created"})
		log.Printf("[APICreateBusinessProcess] BP %s created", bpID)

		// Log audit entry
		logBusinessProcessAudit(s, r, tenantID, bpID, "created", map[string]interface{}{
			"processName": templateReq.Name,
		})
	}
}

// ============================================================================
// API: List Business Processes
// ============================================================================
// GET /api/bp
func APIListBusinessProcesses(s *Server) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		log.Println("[APIListBusinessProcesses] Listing BPs")

		// Check permissions - requires view role
		if err := checkBusinessProcessPermission(r, "view"); err != nil {
			log.Printf("[APIListBusinessProcesses] Permission denied: %v", err)
			http.Error(w, err.Error(), http.StatusForbidden)
			return
		}

		tenantID := r.URL.Query().Get("tenant_id")
		datasourceID := r.URL.Query().Get("datasource_id")
		if tenantID == "" || datasourceID == "" {
			http.Error(w, "tenant_id and datasource_id required", http.StatusBadRequest)
			return
		}

		q := `
			SELECT p.id, p.process_name, p.description, p.is_active, p.version, p.created_at, p.updated_at,
			       COUNT(s.id) as step_count
			FROM business_processes p
			LEFT JOIN bp_steps s ON s.process_id = p.id
			WHERE p.tenant_id = $1 AND p.datasource_id = $2
			GROUP BY p.id, p.process_name, p.description, p.is_active, p.version, p.created_at, p.updated_at
			ORDER BY p.created_at DESC
		`

		rows, err := s.DB.QueryContext(r.Context(), q, tenantID, datasourceID)
		if err != nil {
			http.Error(w, fmt.Sprintf("query failed: %v", err), http.StatusInternalServerError)
			return
		}
		defer rows.Close()

		var bps []BPResponse
		for rows.Next() {
			var bp BPResponse
			err := rows.Scan(&bp.ID, &bp.ProcessName, &bp.Description, &bp.IsActive, &bp.Version, &bp.CreatedAt, &bp.UpdatedAt, &bp.StepCount)
			if err != nil {
				log.Printf("[APIListBusinessProcesses] Scan error: %v", err)
				continue
			}
			bps = append(bps, bp)
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{"count": len(bps), "bps": bps})
		log.Printf("[APIListBusinessProcesses] Returned %d BPs", len(bps))
	}
}

// ============================================================================
// API: Get Business Process
// ============================================================================
// GET /api/bp/:id
func APIGetBusinessProcess(s *Server) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id := r.URL.Query().Get("id")
		if id == "" {
			id = r.PathValue("id")
		}
		log.Printf("[APIGetBusinessProcess] Getting BP %s", id)

		// Check permissions - requires view role
		if err := checkBusinessProcessPermission(r, "view"); err != nil {
			log.Printf("[APIGetBusinessProcess] Permission denied: %v", err)
			http.Error(w, err.Error(), http.StatusForbidden)
			return
		}

		tenantID := r.URL.Query().Get("tenant_id")
		datasourceID := r.URL.Query().Get("datasource_id")
		if tenantID == "" || datasourceID == "" {
			http.Error(w, "tenant_id and datasource_id required", http.StatusBadRequest)
			return
		}

		var bp BPResponse
		bpQ := `
			SELECT id, process_name, description, is_active, version, created_at, updated_at
			FROM business_processes
			WHERE id = $1 AND tenant_id = $2 AND datasource_id = $3
		`

		err := s.DB.QueryRowContext(r.Context(), bpQ, id, tenantID, datasourceID).Scan(
			&bp.ID, &bp.ProcessName, &bp.Description, &bp.IsActive, &bp.Version, &bp.CreatedAt, &bp.UpdatedAt,
		)
		if err != nil {
			http.Error(w, "BP not found", http.StatusNotFound)
			return
		}

		stepsQ := `SELECT COUNT(*) FROM bp_steps WHERE process_id = $1`
		_ = s.DB.QueryRowContext(r.Context(), stepsQ, id).Scan(&bp.StepCount)

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(bp)
	}
}

// ============================================================================
// API: Start Business Process Execution
// ============================================================================
// POST /api/bp/:id/start
func APIStartBusinessProcessExecution(s *Server) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		bpID := r.URL.Query().Get("id")
		if bpID == "" {
			bpID = r.PathValue("id")
		}
		log.Printf("[APIStartBusinessProcessExecution] Starting BP %s", bpID)

		// Check permissions - requires execute role
		if err := checkBusinessProcessPermission(r, "execute"); err != nil {
			log.Printf("[APIStartBusinessProcessExecution] Permission denied: %v", err)
			http.Error(w, err.Error(), http.StatusForbidden)
			return
		}

		tenantID := r.URL.Query().Get("tenant_id")
		datasourceID := r.URL.Query().Get("datasource_id")
		if tenantID == "" || datasourceID == "" {
			http.Error(w, "tenant_id and datasource_id required", http.StatusBadRequest)
			return
		}

		var req StartBPRequest
		err := json.NewDecoder(r.Body).Decode(&req)
		if err != nil {
			http.Error(w, fmt.Sprintf("invalid request: %v", err), http.StatusBadRequest)
			return
		}

		if req.EntityID == "" || req.EntityType == "" {
			http.Error(w, "entity_id and entity_type required", http.StatusBadRequest)
			return
		}

		if req.EntityID == "" || req.EntityType == "" {
			http.Error(w, "entity_id and entity_type required", http.StatusBadRequest)
			return
		}

		instanceID := uuid.New().String()
		dataJSON, _ := json.Marshal(req.Data)

		// Create Business Object wrapper
		obj := business_process.GenericBusinessObject{
			ID:       req.EntityID,
			Type:     req.EntityType,
			TenantID: tenantID,
			Data:     req.Data,
		}

		// Start Temporal Workflow
		workflowOptions := client.StartWorkflowOptions{
			ID:        "bp-" + instanceID,
			TaskQueue: "business-process-queue",
		}

		// Use DynamicWorkflowParams for Hot Reload support
		params := business_process.DynamicWorkflowParams{
			WorkflowDefinitionID: bpID, // Assuming ID passed is the Workflow Definition ID
			BusinessObject:       obj,
		}

		we, err := s.TemporalClient.ExecuteWorkflow(r.Context(), workflowOptions, business_process.DynamicProcessWorkflow, params)
		if err != nil {
			log.Printf("[APIStartBusinessProcessExecution] Failed to start workflow: %v", err)
			http.Error(w, fmt.Sprintf("Failed to start workflow: %v", err), http.StatusInternalServerError)
			return
		}

		q := `
			INSERT INTO bp_instances 
			(id, tenant_id, datasource_id, process_id, entity_id, entity_type, current_step, status, instance_data, temporal_workflow_id, started_at, created_at)
			VALUES ($1, $2, $3, $4, $5, $6, 1, 'pending', $7::jsonb, $8, NOW(), NOW())
		`

		_, err = s.DB.ExecContext(r.Context(), q, instanceID, tenantID, datasourceID, bpID, req.EntityID, req.EntityType, dataJSON, we.GetID())
		if err != nil {
			log.Printf("[APIStartBusinessProcessExecution] Error inserting instance: %v", err)
			// Note: Workflow is already started, we might want to cancel it if DB insert fails, or rely on reconciliation.
			http.Error(w, fmt.Sprintf("failed to create instance record: %v", err), http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{
			"instance_id":          instanceID,
			"process_id":           bpID,
			"status":               "started",
			"temporal_workflow_id": we.GetID(),
			"temporal_run_id":      we.GetRunID(),
		})
		log.Printf("[APIStartBusinessProcessExecution] BP execution %s started (WorkflowID: %s)", instanceID, we.GetID())

		// Log audit entry
		logBusinessProcessAudit(s, r, tenantID, bpID, "execution_started", map[string]interface{}{
			"instanceId":         instanceID,
			"entityId":           req.EntityID,
			"entityType":         req.EntityType,
			"temporalWorkflowId": we.GetID(),
		})
	}
}

// ============================================================================
// API: Get Business Process Instance Status
// ============================================================================
// GET /api/bp/instance/:id
func APIGetBusinessProcessInstanceStatus(s *Server) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		instanceID := r.URL.Query().Get("id")
		if instanceID == "" {
			instanceID = r.PathValue("id")
		}
		log.Printf("[APIGetBusinessProcessInstanceStatus] Getting instance %s", instanceID)

		// Check permissions - requires view role
		if err := checkBusinessProcessPermission(r, "view"); err != nil {
			log.Printf("[APIGetBusinessProcessInstanceStatus] Permission denied: %v", err)
			http.Error(w, err.Error(), http.StatusForbidden)
			return
		}

		tenantID := r.URL.Query().Get("tenant_id")
		datasourceID := r.URL.Query().Get("datasource_id")
		if tenantID == "" || datasourceID == "" {
			http.Error(w, "tenant_id and datasource_id required", http.StatusBadRequest)
			return
		}

		var response BPInstanceResponse
		var instanceDataJSON string

		q := `
			SELECT i.id, i.process_id, p.process_name, i.entity_id, i.entity_type, i.current_step, i.status,
			       i.instance_data, i.started_at, i.current_step_started_at, i.current_step_due_at, 
			       i.temporal_workflow_id, i.created_at
			FROM bp_instances i
			JOIN business_processes p ON p.id = i.process_id
			WHERE i.id = $1 AND i.tenant_id = $2 AND i.datasource_id = $3
		`

		err := s.DB.QueryRowContext(r.Context(), q, instanceID, tenantID, datasourceID).Scan(
			&response.InstanceID, &response.ProcessID, &response.ProcessName, &response.EntityID,
			&response.EntityType, &response.CurrentStep, &response.Status, &instanceDataJSON,
			&response.StartedAt, &response.CurrentStepStartAt, &response.CurrentStepDueAt,
			&response.TemporalWorkflowID, &response.CreatedAt,
		)
		if err != nil {
			http.Error(w, "instance not found", http.StatusNotFound)
			return
		}

		if instanceDataJSON != "" {
			_ = json.Unmarshal([]byte(instanceDataJSON), &response.InstanceData)
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
		log.Printf("[APIGetBusinessProcessInstanceStatus] Status: %s", response.Status)
	}
}

// ============================================================================
// API: Approve Business Process Step
// ============================================================================
// POST /api/bp/instance/:id/approve
// POST /api/bp/instance/:id/approve
func APIApproveBusinessProcessStep(s *Server) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		instanceID := r.URL.Query().Get("id")
		if instanceID == "" {
			instanceID = r.PathValue("id")
		}
		log.Printf("[APIApproveBusinessProcessStep] Approving step for instance %s", instanceID)

		// Check permissions - requires execute role
		if err := checkBusinessProcessPermission(r, "execute"); err != nil {
			log.Printf("[APIApproveBusinessProcessStep] Permission denied: %v", err)
			http.Error(w, err.Error(), http.StatusForbidden)
			return
		}

		tenantID := r.URL.Query().Get("tenant_id")
		if tenantID == "" {
			http.Error(w, "tenant_id required", http.StatusBadRequest)
			return
		}

		var req ApproveStepRequest
		err := json.NewDecoder(r.Body).Decode(&req)
		if err != nil {
			http.Error(w, fmt.Sprintf("invalid request: %v", err), http.StatusBadRequest)
			return
		}

		if req.Decision == "" {
			http.Error(w, "decision required", http.StatusBadRequest)
			return
		}

		// Get Temporal Workflow ID
		var temporalWorkflowID string
		err = s.DB.QueryRowContext(r.Context(), "SELECT temporal_workflow_id FROM bp_instances WHERE id = $1", instanceID).Scan(&temporalWorkflowID)
		if err != nil {
			log.Printf("[APIApproveBusinessProcessStep] Instance not found or no workflow ID: %v", err)
			http.Error(w, "Instance not found", http.StatusNotFound)
			return
		}

		if temporalWorkflowID == "" {
			http.Error(w, "Workflow not started for this instance", http.StatusBadRequest)
			return
		}

		// Signal Workflow
		// Signal name must match what the workflow expects: "ApprovalSignal"
		err = s.TemporalClient.SignalWorkflow(r.Context(), temporalWorkflowID, "", "ApprovalSignal", req.Decision)
		if err != nil {
			log.Printf("[APIApproveBusinessProcessStep] Failed to signal workflow: %v", err)
			http.Error(w, fmt.Sprintf("Failed to signal workflow: %v", err), http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{"instance_id": instanceID, "decision": req.Decision, "status": "signaled"})
		log.Printf("[APIApproveBusinessProcessStep] Signal sent to workflow %s", temporalWorkflowID)

		// Log audit entry
		logBusinessProcessAudit(s, r, tenantID, instanceID, "step_approved", map[string]interface{}{
			"instanceId":         instanceID,
			"decision":           req.Decision,
			"comment":            req.Comment,
			"temporalWorkflowId": temporalWorkflowID,
		})
	}
}

// ============================================================================
// API: Get Business Process Audit Trail
// ============================================================================
// GET /api/bp/:id/audit
func APIGetBusinessProcessAuditTrail(s *Server) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		processID := r.URL.Query().Get("id")
		if processID == "" {
			processID = r.PathValue("id")
		}
		log.Printf("[APIGetBusinessProcessAuditTrail] Getting audit trail for BP %s", processID)

		// Check permissions - requires view role
		if err := checkBusinessProcessPermission(r, "view"); err != nil {
			log.Printf("[APIGetBusinessProcessAuditTrail] Permission denied: %v", err)
			http.Error(w, err.Error(), http.StatusForbidden)
			return
		}

		tenantID := r.URL.Query().Get("tenant_id")
		datasourceID := r.URL.Query().Get("datasource_id")
		if tenantID == "" || datasourceID == "" {
			http.Error(w, "tenant_id and datasource_id required", http.StatusBadRequest)
			return
		}

		limitStr := r.URL.Query().Get("limit")
		limit := 50 // default
		if limitStr != "" {
			if parsedLimit, err := strconv.Atoi(limitStr); err == nil && parsedLimit > 0 && parsedLimit <= 100 {
				limit = parsedLimit
			}
		}

		q := `
			SELECT id, tenant_id, business_process_id, action_type, actor_email, actor_role, 
			       action_details, timestamp, ip_address
			FROM bp_audit_trail
			WHERE tenant_id = $1 AND business_process_id = $2
			ORDER BY timestamp DESC
			LIMIT $3
		`

		rows, err := s.DB.QueryContext(r.Context(), q, tenantID, processID, limit)
		if err != nil {
			log.Printf("[APIGetBusinessProcessAuditTrail] Error: %v", err)
			http.Error(w, fmt.Sprintf("failed to get audit trail: %v", err), http.StatusInternalServerError)
			return
		}
		defer rows.Close()

		var entries []map[string]interface{}
		for rows.Next() {
			var id, tenantID, businessProcessID, actionType, actorEmail, timestamp string
			var actorRole, ipAddress *string
			var actionDetails []byte

			err := rows.Scan(&id, &tenantID, &businessProcessID, &actionType, &actorEmail, &actorRole, &actionDetails, &timestamp, &ipAddress)
			if err != nil {
				log.Printf("[APIGetBusinessProcessAuditTrail] Scan error: %v", err)
				continue
			}

			entry := map[string]interface{}{
				"id":                id,
				"tenantId":          tenantID,
				"businessProcessId": businessProcessID,
				"actionType":        actionType,
				"actorEmail":        actorEmail,
				"timestamp":         timestamp,
			}

			if actorRole != nil {
				entry["actorRole"] = *actorRole
			}
			if ipAddress != nil {
				entry["ipAddress"] = *ipAddress
			}
			if len(actionDetails) > 0 {
				var details map[string]interface{}
				if json.Unmarshal(actionDetails, &details) == nil {
					entry["actionDetails"] = details
				}
			}

			entries = append(entries, entry)
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"processId": processID,
			"entries":   entries,
			"limit":     limit,
		})
		log.Printf("[APIGetBusinessProcessAuditTrail] Returned %d audit entries for BP %s", len(entries), processID)
	}
}

// ============================================================================
// ABAC HELPER FUNCTIONS
// ============================================================================

// hasBusinessProcessDesignerRole checks if user can design/modify business processes
func hasBusinessProcessDesignerRole(role string) bool {
	designerRoles := []string{
		"ProcessDesigner",
		"BusinessAnalyst",
		"WorkflowDesigner",
		"Admin",
	}
	for _, r := range designerRoles {
		if strings.EqualFold(r, role) {
			return true
		}
	}
	return false
}

// hasBusinessProcessExecutorRole checks if user can execute/start business processes
func hasBusinessProcessExecutorRole(role string) bool {
	executorRoles := []string{
		"ProcessDesigner",
		"BusinessAnalyst",
		"WorkflowDesigner",
		"OperationsManager",
		"ComplianceOfficer",
		"Advisor",
		"Admin",
	}
	for _, r := range executorRoles {
		if strings.EqualFold(r, role) {
			return true
		}
	}
	return false
}

// hasBusinessProcessViewerRole checks if user can view business processes
func hasBusinessProcessViewerRole(role string) bool {
	viewerRoles := []string{
		"ProcessDesigner",
		"BusinessAnalyst",
		"WorkflowDesigner",
		"OperationsManager",
		"ComplianceOfficer",
		"Advisor",
		"Client",
		"Admin",
	}
	for _, r := range viewerRoles {
		if strings.EqualFold(r, role) {
			return true
		}
	}
	return false
}

// checkBusinessProcessPermission validates user permissions for BP operations
func checkBusinessProcessPermission(r *http.Request, requiredPermission string) error {
	userRole := r.Header.Get("X-User-Role")
	if userRole == "" {
		return fmt.Errorf("missing X-User-Role header")
	}

	switch requiredPermission {
	case "design":
		if !hasBusinessProcessDesignerRole(userRole) {
			return fmt.Errorf("insufficient permissions: requires ProcessDesigner, BusinessAnalyst, WorkflowDesigner, or Admin role")
		}
	case "execute":
		if !hasBusinessProcessExecutorRole(userRole) {
			return fmt.Errorf("insufficient permissions: requires execution role")
		}
	case "view":
		if !hasBusinessProcessViewerRole(userRole) {
			return fmt.Errorf("insufficient permissions: requires viewer role")
		}
	default:
		return fmt.Errorf("unknown permission type: %s", requiredPermission)
	}

	return nil
}

// logBusinessProcessAudit logs an audit entry for BP operations
func logBusinessProcessAudit(s *Server, r *http.Request, tenantID, processID, actionType string, details map[string]interface{}) {
	userEmail := r.Header.Get("X-User-Email")
	userRole := r.Header.Get("X-User-Role")
	ipAddress := r.Header.Get("X-Forwarded-For")
	if ipAddress == "" {
		ipAddress = r.RemoteAddr
	}

	// Convert details to JSON
	detailsJSON, _ := json.Marshal(details)

	q := `
		INSERT INTO bp_audit_trail 
		(id, tenant_id, business_process_id, action_type, actor_email, actor_role, action_details, timestamp, ip_address)
		VALUES ($1, $2, $3, $4, $5, $6, $7, NOW(), $8)
	`

	processUUID, _ := uuid.Parse(processID)
	_, _ = s.DB.ExecContext(r.Context(), q, uuid.New(), tenantID, processUUID, actionType, userEmail, userRole, detailsJSON, ipAddress)
}
