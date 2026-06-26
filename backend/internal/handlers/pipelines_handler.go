package handlers

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/hondyman/semlayer/backend/internal/auth"
	"github.com/hondyman/semlayer/backend/pkg/governance"
	"github.com/hondyman/semlayer/backend/pkg/simulation"
	"github.com/hondyman/semlayer/backend/pkg/workflows"
	"github.com/jmoiron/sqlx"
	"go.temporal.io/sdk/client"
	"github.com/hondyman/semlayer/libs/jwt-middleware"
)

// Pipeline represents a saved Uisce Flow pipeline
type Pipeline struct {
	ID             string          `json:"id" db:"id"`
	TenantID       string          `json:"tenant_id" db:"tenant_id"`
	Name           string          `json:"name" db:"name"`
	Description    string          `json:"description,omitempty" db:"description"`
	BusinessObject string          `json:"business_object,omitempty" db:"business_object"`
	PipelineJSON   json.RawMessage `json:"pipeline_json" db:"pipeline_json"`
	IsActive       bool            `json:"is_active" db:"is_active"`
	CreatedBy      string          `json:"created_by,omitempty" db:"created_by"`
	CreatedAt      time.Time       `json:"created_at" db:"created_at"`
	LastModifiedAt time.Time       `json:"last_modified_at" db:"last_modified_at"`
}

// PipelineExecuteRequest is the payload for executing a pipeline
type PipelineExecuteRequest struct {
	InstanceID string                 `json:"instanceId,omitempty"`
	FormData   map[string]interface{} `json:"formData"`
}

// PipelineExecuteResponse is the result of pipeline execution
type PipelineExecuteResponse struct {
	Status       string                 `json:"status"`
	WorkflowID   string                 `json:"workflowId"`
	RunID        string                 `json:"runId"`
	EnrichedData map[string]interface{} `json:"enrichedData,omitempty"` // For compatibility
}

// PipelineHandler handles pipeline CRUD and execution
type PipelineHandler struct {
	db             *sqlx.DB
	temporalClient client.Client
	governance     *governance.GovernanceEngine
	simulation     *simulation.SimulationRunner
}

// NewPipelineHandler creates a new handler
func NewPipelineHandler(db *sqlx.DB, temporalClient client.Client) *PipelineHandler {
	return &PipelineHandler{
		db:             db,
		temporalClient: temporalClient,
		governance:     governance.NewGovernanceEngine(db),
		simulation:     simulation.NewSimulationRunner(),
	}
}

// RegisterRoutes registers the pipeline routes
func (h *PipelineHandler) RegisterRoutes(r chi.Router) {
	r.Get("/api/v1/pipelines", h.ListPipelines)
	r.Post("/api/v1/pipelines", h.CreatePipeline)
	r.Get("/api/v1/pipelines/{id}", h.GetPipeline)
	r.Put("/api/v1/pipelines/{id}", h.UpdatePipeline)
	r.Delete("/api/v1/pipelines/{id}", h.DeletePipeline)
	r.Post("/api/v1/pipelines/{id}/execute", h.ExecutePipeline)
	r.Post("/api/v1/pipelines/{id}/simulate", h.SimulatePipeline)
	r.Get("/api/v1/pipelines/activities/safe", h.GetSafeActivities)
}

// getContextMetadata extracts trusted user and tenant info from context
func (h *PipelineHandler) getContextMetadata(r *http.Request) (string, string) {
	user, _ := auth.GetUserFromContext(r.Context())
	userID := user.ID
	if userID == "" {
		userID = "system"
	}

	tenantID := user.TenantID
	if tenantID == "" {
		// Fallback for dev/system tokens that might pass it via header (deprecated)
		tenantID = jwtmiddleware.GetClaimsFromContext(r).TenantID
		if tenantID == "" {
			tenantID = "00000000-0000-0000-0000-000000000001" // Default/Global
		}
	}
	return userID, tenantID
}

// ListPipelines returns all pipelines for a tenant
func (h *PipelineHandler) ListPipelines(w http.ResponseWriter, r *http.Request) {
	_, tenantID := h.getContextMetadata(r)

	boFilter := r.URL.Query().Get("business_object")

	var pipelines []Pipeline
	var err error

	if boFilter != "" {
		err = h.db.SelectContext(r.Context(), &pipelines,
			`SELECT id, tenant_id, name, description, business_object, pipeline_json, 
			        is_active, created_by, created_at, last_modified_at
			 FROM pipelines 
			 WHERE tenant_id = $1 AND business_object = $2 AND is_active = true
			 ORDER BY name`, tenantID, boFilter)
	} else {
		err = h.db.SelectContext(r.Context(), &pipelines,
			`SELECT id, tenant_id, name, description, business_object, pipeline_json, 
			        is_active, created_by, created_at, last_modified_at
			 FROM pipelines 
			 WHERE tenant_id = $1 AND is_active = true
			 ORDER BY name`, tenantID)
	}

	if err != nil {
		http.Error(w, "Failed to list pipelines: "+err.Error(), http.StatusInternalServerError)
		return
	}

	if pipelines == nil {
		pipelines = []Pipeline{}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(pipelines)
}

// CreatePipeline creates a new pipeline
func (h *PipelineHandler) CreatePipeline(w http.ResponseWriter, r *http.Request) {
	userID, tenantID := h.getContextMetadata(r)

	var pipeline Pipeline
	if err := json.NewDecoder(r.Body).Decode(&pipeline); err != nil {
		http.Error(w, "Invalid request body: "+err.Error(), http.StatusBadRequest)
		return
	}

	pipeline.ID = uuid.New().String()
	pipeline.TenantID = tenantID
	pipeline.CreatedBy = userID
	pipeline.IsActive = true

	// Governance Check: Policy-as-Code
	var pipelineMap map[string]interface{}
	if err := json.Unmarshal(pipeline.PipelineJSON, &pipelineMap); err == nil {
		validationParams := map[string]interface{}{
			"nodes": pipelineMap["nodes"],
			"edges": pipelineMap["edges"],
			"meta":  map[string]string{"tenantId": tenantID, "userId": userID},
		}

		result, err := h.governance.ValidatePipeline(r.Context(), tenantID, validationParams)
		if err != nil {
			// Fail open or closed? Closed for governance.
			http.Error(w, "Governance check failed: "+err.Error(), http.StatusInternalServerError)
			return
		}
		if !result.Allowed {
			http.Error(w, fmt.Sprintf("Pipeline Policy Violation: %v", result.Reasons), http.StatusForbidden)
			return
		}
	} else {
		// Invalid JSON should probably fail validation too, or be caught by database constraints?
		// For now, let it pass to DB or fail here.
		http.Error(w, "Invalid Pipeline Definition JSON", http.StatusBadRequest)
		return
	}

	query := `
		INSERT INTO pipelines (id, tenant_id, name, description, business_object, pipeline_json, is_active, created_by)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		RETURNING created_at, last_modified_at
	`

	err := h.db.QueryRowContext(r.Context(), query,
		pipeline.ID, pipeline.TenantID, pipeline.Name, pipeline.Description,
		pipeline.BusinessObject, pipeline.PipelineJSON, pipeline.IsActive, pipeline.CreatedBy,
	).Scan(&pipeline.CreatedAt, &pipeline.LastModifiedAt)

	if err != nil {
		http.Error(w, "Failed to create pipeline: "+err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(pipeline)
}

// GetPipeline returns a single pipeline by ID
func (h *PipelineHandler) GetPipeline(w http.ResponseWriter, r *http.Request) {
	pipelineID := chi.URLParam(r, "id")

	claims := jwtmiddleware.GetClaimsFromContext(r)
	if claims == nil {
		http.Error(w, `{"error":"unauthorized"}`, http.StatusUnauthorized)
		return
	}
	tenantID := claims.TenantID
	if tenantID == "" {
		tenantID = "00000000-0000-0000-0000-000000000001"
	}

	var pipeline Pipeline
	err := h.db.GetContext(r.Context(), &pipeline,
		`SELECT id, tenant_id, name, description, business_object, pipeline_json, 
		        is_active, created_by, created_at, last_modified_at
		 FROM pipelines 
		 WHERE id = $1 AND tenant_id = $2`, pipelineID, tenantID)

	if err == sql.ErrNoRows {
		http.Error(w, "Pipeline not found", http.StatusNotFound)
		return
	}
	if err != nil {
		http.Error(w, "Failed to get pipeline: "+err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(pipeline)
}

// UpdatePipeline updates an existing pipeline
func (h *PipelineHandler) UpdatePipeline(w http.ResponseWriter, r *http.Request) {
	pipelineID := chi.URLParam(r, "id")

	claims := jwtmiddleware.GetClaimsFromContext(r)
	if claims == nil {
		http.Error(w, `{"error":"unauthorized"}`, http.StatusUnauthorized)
		return
	}
	tenantID := claims.TenantID
	if tenantID == "" {
		tenantID = "00000000-0000-0000-0000-000000000001"
	}

	var pipeline Pipeline
	if err := json.NewDecoder(r.Body).Decode(&pipeline); err != nil {
		http.Error(w, "Invalid request body: "+err.Error(), http.StatusBadRequest)
		return
	}

	// Governance Check: Policy-as-Code
	var pipelineMap map[string]interface{}
	if err := json.Unmarshal(pipeline.PipelineJSON, &pipelineMap); err == nil {
		validationParams := map[string]interface{}{
			"nodes": pipelineMap["nodes"],
			"edges": pipelineMap["edges"],
			"meta":  map[string]string{"tenantId": tenantID},
		}

		result, err := h.governance.ValidatePipeline(r.Context(), tenantID, validationParams)
		if err != nil {
			http.Error(w, "Governance check failed: "+err.Error(), http.StatusInternalServerError)
			return
		}
		if !result.Allowed {
			http.Error(w, fmt.Sprintf("Pipeline Policy Violation: %v", result.Reasons), http.StatusForbidden)
			return
		}
	} else {
		http.Error(w, "Invalid Pipeline Definition JSON", http.StatusBadRequest)
		return
	}

	query := `
		UPDATE pipelines 
		SET name = $1, description = $2, business_object = $3, pipeline_json = $4, last_modified_at = NOW()
		WHERE id = $5 AND tenant_id = $6
		RETURNING id, tenant_id, name, description, business_object, pipeline_json, 
		          is_active, created_by, created_at, last_modified_at
	`

	var updated Pipeline
	err := h.db.QueryRowContext(r.Context(), query,
		pipeline.Name, pipeline.Description, pipeline.BusinessObject, pipeline.PipelineJSON, pipelineID, tenantID,
	).Scan(&updated.ID, &updated.TenantID, &updated.Name, &updated.Description,
		&updated.BusinessObject, &updated.PipelineJSON, &updated.IsActive, &updated.CreatedBy,
		&updated.CreatedAt, &updated.LastModifiedAt)

	if err == sql.ErrNoRows {
		http.Error(w, "Pipeline not found", http.StatusNotFound)
		return
	}
	if err != nil {
		http.Error(w, "Failed to update pipeline: "+err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(updated)
}

// DeletePipeline soft-deletes a pipeline
func (h *PipelineHandler) DeletePipeline(w http.ResponseWriter, r *http.Request) {
	pipelineID := chi.URLParam(r, "id")

	claims := jwtmiddleware.GetClaimsFromContext(r)
	if claims == nil {
		http.Error(w, `{"error":"unauthorized"}`, http.StatusUnauthorized)
		return
	}
	tenantID := claims.TenantID
	if tenantID == "" {
		tenantID = "00000000-0000-0000-0000-000000000001"
	}

	result, err := h.db.ExecContext(r.Context(),
		`UPDATE pipelines SET is_active = false, last_modified_at = NOW() 
		 WHERE id = $1 AND tenant_id = $2`, pipelineID, tenantID)

	if err != nil {
		http.Error(w, "Failed to delete pipeline: "+err.Error(), http.StatusInternalServerError)
		return
	}

	rows, _ := result.RowsAffected()
	if rows == 0 {
		http.Error(w, "Pipeline not found", http.StatusNotFound)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// ExecutePipeline runs a pipeline using the Temporal Interpreter
func (h *PipelineHandler) ExecutePipeline(w http.ResponseWriter, r *http.Request) {
	pipelineID := chi.URLParam(r, "id")

	claims := jwtmiddleware.GetClaimsFromContext(r)
	if claims == nil {
		http.Error(w, `{"error":"unauthorized"}`, http.StatusUnauthorized)
		return
	}
	tenantID := claims.TenantID
	if tenantID == "" {
		tenantID = "00000000-0000-0000-0000-000000000001"
	}

	// 1. Parse request
	var req PipelineExecuteRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body: "+err.Error(), http.StatusBadRequest)
		return
	}

	// 2. Load pipeline definition
	var pipeline Pipeline
	err := h.db.GetContext(r.Context(), &pipeline,
		`SELECT id, tenant_id, name, pipeline_json FROM pipelines 
		 WHERE id = $1 AND tenant_id = $2 AND is_active = true`, pipelineID, tenantID)

	if err == sql.ErrNoRows {
		http.Error(w, "Pipeline not found", http.StatusNotFound)
		return
	}
	if err != nil {
		http.Error(w, "Failed to load pipeline: "+err.Error(), http.StatusInternalServerError)
		return
	}

	// Governance Check (Runtime): Policy-as-Code
	var pipelineMap map[string]interface{}
	if err := json.Unmarshal(pipeline.PipelineJSON, &pipelineMap); err == nil {
		validationParams := map[string]interface{}{
			"nodes":      pipelineMap["nodes"],
			"edges":      pipelineMap["edges"],
			"meta":       map[string]string{"tenantId": tenantID},
			"runtimeCtx": req.FormData, // Pass runtime data for advanced policy checks
		}

		result, err := h.governance.ValidatePipeline(r.Context(), tenantID, validationParams)
		if err != nil {
			http.Error(w, "Governance check failed: "+err.Error(), http.StatusInternalServerError)
			return
		}
		if !result.Allowed {
			http.Error(w, fmt.Sprintf("Pipeline Execution Blocked by Policy: %v", result.Reasons), http.StatusForbidden)
			return
		}
	}

	// 3. Transform React Flow JSON to Interpreter DSL
	dsl, err := transformReactFlowToDSL(pipeline.PipelineJSON)
	if err != nil {
		http.Error(w, "Failed to compile pipeline definition: "+err.Error(), http.StatusInternalServerError)
		return
	}
	dsl.Name = pipeline.Name
	dsl.GlobalState = req.FormData

	// 4. Start Temporal Workflow
	workflowID := fmt.Sprintf("pipeline-%s-%d", pipelineID, time.Now().Unix())
	options := client.StartWorkflowOptions{
		ID:        workflowID,
		TaskQueue: "bp_workflow_queue",
	}

	run, err := h.temporalClient.ExecuteWorkflow(context.Background(), options, workflows.InterpreterWorkflow, dsl)
	if err != nil {
		http.Error(w, "Failed to start workflow: "+err.Error(), http.StatusInternalServerError)
		return
	}

	// 5. Return success with workflow ID
	response := PipelineExecuteResponse{
		Status:       "submitted",
		WorkflowID:   run.GetID(),
		RunID:        run.GetRunID(),
		EnrichedData: req.FormData, // Echo back for immediate UI feedback
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// SimulatePipeline runs a "Dry Run" of the pipeline in a test environment
func (h *PipelineHandler) SimulatePipeline(w http.ResponseWriter, r *http.Request) {
	pipelineID := chi.URLParam(r, "id")

	claims := jwtmiddleware.GetClaimsFromContext(r)
	if claims == nil {
		http.Error(w, `{"error":"unauthorized"}`, http.StatusUnauthorized)
		return
	}
	tenantID := claims.TenantID
	if tenantID == "" {
		tenantID = "00000000-0000-0000-0000-000000000001"
	}

	// 1. Parse request (reuses ExecuteRequest for formData)
	var req PipelineExecuteRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body: "+err.Error(), http.StatusBadRequest)
		return
	}

	// 2. Load pipeline definition
	var pipeline Pipeline
	err := h.db.GetContext(r.Context(), &pipeline,
		`SELECT id, tenant_id, name, pipeline_json FROM pipelines 
		 WHERE id = $1 AND tenant_id = $2 AND is_active = true`, pipelineID, tenantID)

	if err != nil {
		http.Error(w, "Failed to load pipeline: "+err.Error(), http.StatusInternalServerError)
		return
	}

	// 3. Transform DSL
	dsl, err := transformReactFlowToDSL(pipeline.PipelineJSON)
	if err != nil {
		http.Error(w, "Failed to compile pipeline: "+err.Error(), http.StatusInternalServerError)
		return
	}
	dsl.Name = pipeline.Name
	dsl.GlobalState = req.FormData

	// 4. Run Simulation
	result, err := h.simulation.RunSimulation(r.Context(), dsl)
	if err != nil {
		http.Error(w, "Simulation failed: "+err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(result)
}

// GetSafeActivities returns the list of activities safe for client use
func (h *PipelineHandler) GetSafeActivities(w http.ResponseWriter, r *http.Request) {
	activities := workflows.GetClientSafeActivities()
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string][]string{"activities": activities})
}

// ============================================================================
// DSL Transformation
// ============================================================================

type ReactFlowDef struct {
	Nodes []struct {
		ID       string                 `json:"id"`
		Type     string                 `json:"type"`
		Data     map[string]interface{} `json:"data"` // Contains label, config
		Position interface{}            `json:"position"`
	} `json:"nodes"`
	Edges []struct {
		Source string `json:"source"`
		Target string `json:"target"`
		Label  string `json:"label,omitempty"` // For branch conditions?
	} `json:"edges"`
}

func transformReactFlowToDSL(rawJSON json.RawMessage) (workflows.WorkflowDefinition, error) {
	var rf ReactFlowDef
	if err := json.Unmarshal(rawJSON, &rf); err != nil {
		return workflows.WorkflowDefinition{}, err
	}

	if len(rf.Nodes) == 0 {
		return workflows.WorkflowDefinition{}, fmt.Errorf("pipeline has no nodes")
	}

	// Build Nodes map
	nodesMap := make(map[string]workflows.WorkflowNode)
	for _, rfNode := range rf.Nodes {
		label, _ := rfNode.Data["label"].(string)
		filterType := ""
		if val, ok := rfNode.Data["filterType"].(string); ok {
			filterType = val
		} else {
			// Infer from label if missing (legacy)
			filterType = inferTypeFromLabel(label)
		}

		// Extract legacy config if present
		config := make(map[string]interface{})
		if c, ok := rfNode.Data["config"].(map[string]interface{}); ok {
			config = c
		}
		// Also pass top-level data fields
		for k, v := range rfNode.Data {
			if k != "config" && k != "label" {
				config[k] = v
			}
		}
		config["legacyType"] = filterType

		// Determine Node Type for Interpreter
		// Default to ACTIVITY unless we detect branching later
		nodeType := "ACTIVITY"

		nodesMap[rfNode.ID] = workflows.WorkflowNode{
			ID:     rfNode.ID,
			Type:   nodeType,
			Name:   label,
			Config: config,
		}
	}

	// Build Edges / Transitions
	// 1. Group edges by Source
	edgesBySource := make(map[string][]string) // sourceID -> []targetID
	for _, edge := range rf.Edges {
		edgesBySource[edge.Source] = append(edgesBySource[edge.Source], edge.Target)
	}

	// 2. Update Nodes with transition info
	for id, node := range nodesMap {
		targets := edgesBySource[id]

		if len(targets) == 0 {
			// No outgoing edges -> implicitly END or leaf
			// Leave NextNodeID nil
		} else if len(targets) == 1 {
			// Simple sequence
			target := targets[0]
			node.NextNodeID = &target
		} else {
			// Branching
			node.Type = "BRANCH"
			// Convert targets to BranchOptions
			// TODO: We need edge labels or conditions to know WHICH branch is which.
			// React Flow edges might not have condition logic stored on them directly in this simple version.
			// For now, we just add them as options. The interpreter BRANCH logic needs to know conditions.
			// If missing logic, this is a race or parallel behavior.
			// For this MVP, let's assume simple branching is not fully configured in React Flow yet
			// or assume all branches are valid (parallel).
			// The InterpreterWorkflow implementation expects `Branches` with `Condition`.

			for _, t := range targets {
				node.Branches = append(node.Branches, workflows.BranchOption{
					TargetNodeID: t,
					Condition:    "", // Default/Else
				})
			}
		}
		nodesMap[id] = node
	}

	// Find Start Node (node with no incoming edges)
	incomingEdgeCounts := make(map[string]int)
	for _, edge := range rf.Edges {
		incomingEdgeCounts[edge.Target]++
	}

	var startNodeID string
	for _, node := range rf.Nodes {
		if incomingEdgeCounts[node.ID] == 0 {
			startNodeID = node.ID
			break
		}
	}

	// Fallback if loop or all have incoming: pick first
	if startNodeID == "" && len(rf.Nodes) > 0 {
		startNodeID = rf.Nodes[0].ID
	}

	return workflows.WorkflowDefinition{
		Nodes:       nodesMap,
		StartNodeID: startNodeID,
	}, nil
}

func inferTypeFromLabel(label string) string {
	// Simple heuristic for legacy data
	if contains(label, "Validation") {
		return "validate"
	}
	if contains(label, "Approver") {
		return "approve"
	}
	if contains(label, "Notify") {
		return "notify"
	}
	if contains(label, "External") {
		return "integrate"
	}
	return "generic"
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && len(s) > 0 &&
		(s == substr || (len(s) > len(substr) && stringContains(s, substr)))
}

func stringContains(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
