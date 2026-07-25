package api

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/machinebox/graphql"
)

// HasuraConfig holds configuration for the Hasura client
type HasuraConfig struct {
	Endpoint    string
	AdminSecret string
}

// HasuraClient provides typed GraphQL operations for business process data access
type HasuraClient struct {
	client      *graphql.Client
	adminSecret string
}

// NewHasuraClient creates a new Hasura GraphQL client
func NewHasuraClient(config *HasuraConfig) *HasuraClient {
	httpClient := &http.Client{
		Timeout: 30 * time.Second,
	}

	client := graphql.NewClient(config.Endpoint, graphql.WithHTTPClient(httpClient))

	return &HasuraClient{
		client:      client,
		adminSecret: config.AdminSecret,
	}
}

// NewHasuraClientFromEnv creates a Hasura client from environment variables
func NewHasuraClientFromEnv() *HasuraClient {
	endpoint := os.Getenv("HASURA_GRAPHQL_ENDPOINT")
	if endpoint == "" {
		endpoint = "http://localhost:8080/v1/graphql"
	}

	adminSecret := os.Getenv("HASURA_ADMIN_SECRET")
	if adminSecret == "" {
		adminSecret = "adminsecret"
	}

	return NewHasuraClient(&HasuraConfig{
		Endpoint:    endpoint,
		AdminSecret: adminSecret,
	})
}

// Query executes a GraphQL query with admin secret header
func (c *HasuraClient) Query(ctx context.Context, query string, variables map[string]interface{}, result interface{}) error {
	req := graphql.NewRequest(query)

	// Add admin secret header
	req.Header.Set("X-Hasura-Admin-Secret", c.adminSecret)

	// Add variables if provided
	for key, value := range variables {
		req.Var(key, value)
	}

	return c.client.Run(ctx, req, result)
}

// BusinessProcessGraphQLHandlers contains handlers that use Hasura GraphQL
type BusinessProcessGraphQLHandlers struct {
	hasura *HasuraClient
}

// NewBusinessProcessGraphQLHandlers creates handlers that use Hasura GraphQL
func NewBusinessProcessGraphQLHandlers(hasura *HasuraClient) *BusinessProcessGraphQLHandlers {
	return &BusinessProcessGraphQLHandlers{hasura: hasura}
}

// ListBusinessProcessesGraphQL returns all business processes using GraphQL
func (h *BusinessProcessGraphQLHandlers) ListBusinessProcessesGraphQL(w http.ResponseWriter, r *http.Request) {
	tenantID := r.URL.Query().Get("tenant_id")

	query := `
		query ListBusinessProcesses($tenantId: uuid) {
			business_processes(
				where: {
					tenant_id: { _eq: $tenantId }
					is_active: { _eq: true }
				}
				order_by: { created_at: desc }
			) {
				id
				tenant_id
				process_name
				description
				entity_type
				status
				is_active
				version_number
				created_at
				updated_at
				created_by
				steps(order_by: { step_order: asc }) {
					id
					step_order
					step_type
					step_name
					description
					duration_hours
					status
				}
			}
		}
	`

	var variables map[string]interface{}
	if tenantID != "" {
		variables = map[string]interface{}{
			"tenantId": tenantID,
		}
	}

	var result struct {
		BusinessProcesses []BusinessProcessGraphQL `json:"business_processes"`
	}

	if err := h.hasura.Query(r.Context(), query, variables, &result); err != nil {
		http.Error(w, fmt.Sprintf("GraphQL query failed: %v", err), http.StatusInternalServerError)
		return
	}

	// Convert to API response format
	processes := make([]BusinessProcess, 0, len(result.BusinessProcesses))
	for _, bp := range result.BusinessProcesses {
		process := BusinessProcess{
			ID:          bp.ID,
			TenantID:    bp.TenantID,
			ProcessName: bp.ProcessName,
			Description: bp.Description,
			Entity:      bp.EntityType,
			IsActive:    bp.IsActive,
			Version:     bp.VersionNumber,
			CreatedAt:   bp.CreatedAt,
			UpdatedAt:   bp.UpdatedAt,
			CreatedBy:   bp.CreatedBy,
		}

		// Convert steps
		for _, s := range bp.Steps {
			step := BPStep{
				ID:            s.ID,
				StepOrder:     s.StepOrder,
				StepType:      s.StepType,
				StepName:      s.StepName,
				Description:   s.Description,
				DurationHours: s.DurationHours,
				Status:        s.Status,
			}
			process.Steps = append(process.Steps, step)
		}

		processes = append(processes, process)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(processes)
}

// BusinessProcessGraphQL represents the GraphQL response structure
type BusinessProcessGraphQL struct {
	ID            string          `json:"id"`
	TenantID      string          `json:"tenant_id"`
	ProcessName   string          `json:"process_name"`
	Description   string          `json:"description"`
	EntityType    string          `json:"entity_type"`
	Status        string          `json:"status"`
	IsActive      bool            `json:"is_active"`
	VersionNumber int             `json:"version_number"`
	CreatedAt     string          `json:"created_at"`
	UpdatedAt     *string         `json:"updated_at"`
	CreatedBy     string          `json:"created_by"`
	Steps         []BPStepGraphQL `json:"steps"`
}

// BPStepGraphQL represents a step in the GraphQL response
type BPStepGraphQL struct {
	ID            string  `json:"id"`
	StepOrder     int     `json:"step_order"`
	StepType      string  `json:"step_type"`
	StepName      string  `json:"step_name"`
	Description   *string `json:"description"`
	DurationHours float64 `json:"duration_hours"`
	Status        *string `json:"status"`
}

// CreateBusinessProcessGraphQL creates a new business process using GraphQL
func (h *BusinessProcessGraphQLHandlers) CreateBusinessProcessGraphQL(w http.ResponseWriter, r *http.Request) {
	var input BusinessProcess
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}

	// Generate ID if not provided
	if input.ID == "" {
		input.ID = uuid.New().String()
	}

	mutation := `
		mutation CreateBusinessProcess(
			$id: uuid!
			$tenant_id: uuid!
			$process_name: String!
			$description: String
			$entity_type: String!
			$is_active: Boolean
			$created_by: String!
		) {
			insert_business_processes_one(
				object: {
					id: $id
					tenant_id: $tenant_id
					process_name: $process_name
					description: $description
					entity_type: $entity_type
					is_active: $is_active
					created_by: $created_by
				}
			) {
				id
				tenant_id
				process_name
				description
				entity_type
				status
				is_active
				version_number
				created_at
				updated_at
				created_by
			}
		}
	`

	variables := map[string]interface{}{
		"id":           input.ID,
		"tenant_id":    input.TenantID,
		"process_name": input.ProcessName,
		"description":  input.Description,
		"entity_type":  input.Entity,
		"is_active":    input.IsActive,
		"created_by":   input.CreatedBy,
	}

	var result struct {
		InsertBusinessProcessesOne BusinessProcessGraphQL `json:"insert_business_processes_one"`
	}

	if err := h.hasura.Query(r.Context(), mutation, variables, &result); err != nil {
		http.Error(w, fmt.Sprintf("GraphQL mutation failed: %v", err), http.StatusInternalServerError)
		return
	}

	// Insert steps if provided
	if len(input.Steps) > 0 {
		for i, step := range input.Steps {
			if step.ID == "" {
				step.ID = uuid.New().String()
			}

			stepMutation := `
				mutation CreateBPStep(
					$id: uuid!
					$business_process_id: uuid!
					$step_order: Int!
					$step_type: String!
					$step_name: String!
					$description: String
					$duration_hours: Int
				) {
					insert_bp_steps_one(
						object: {
							id: $id
							business_process_id: $business_process_id
							step_order: $step_order
							step_type: $step_type
							step_name: $step_name
							description: $description
							duration_hours: $duration_hours
						}
					) {
						id
					}
				}
			`

			stepVars := map[string]interface{}{
				"id":                  step.ID,
				"business_process_id": input.ID,
				"step_order":          i + 1,
				"step_type":           step.StepType,
				"step_name":           step.StepName,
				"description":         step.Description,
				"duration_hours":      int(step.DurationHours),
			}

			var stepResult struct {
				InsertBPStepsOne struct {
					ID string `json:"id"`
				} `json:"insert_bp_steps_one"`
			}

			if err := h.hasura.Query(r.Context(), stepMutation, stepVars, &stepResult); err != nil {
				// Log error but continue - process was created
				fmt.Printf("Warning: failed to create step %d: %v\n", i+1, err)
			}
			input.Steps[i].ID = step.ID
		}
	}

	// Build response
	response := BusinessProcess{
		ID:          result.InsertBusinessProcessesOne.ID,
		TenantID:    result.InsertBusinessProcessesOne.TenantID,
		ProcessName: result.InsertBusinessProcessesOne.ProcessName,
		Description: result.InsertBusinessProcessesOne.Description,
		Entity:      result.InsertBusinessProcessesOne.EntityType,
		IsActive:    result.InsertBusinessProcessesOne.IsActive,
		Version:     result.InsertBusinessProcessesOne.VersionNumber,
		CreatedAt:   result.InsertBusinessProcessesOne.CreatedAt,
		CreatedBy:   result.InsertBusinessProcessesOne.CreatedBy,
		Steps:       input.Steps,
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(response)
}

// GetBusinessProcessGraphQL returns a specific business process using GraphQL
func (h *BusinessProcessGraphQLHandlers) GetBusinessProcessGraphQL(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")

	query := `
		query GetBusinessProcess($id: uuid!) {
			business_processes_by_pk(id: $id) {
				id
				tenant_id
				process_name
				description
				entity_type
				status
				is_active
				version_number
				created_at
				updated_at
				created_by
				steps(order_by: { step_order: asc }) {
					id
					step_order
					step_type
					step_name
					description
					duration_hours
					status
				}
			}
		}
	`

	variables := map[string]interface{}{
		"id": id,
	}

	var result struct {
		BusinessProcessesByPK *BusinessProcessGraphQL `json:"business_processes_by_pk"`
	}

	if err := h.hasura.Query(r.Context(), query, variables, &result); err != nil {
		http.Error(w, fmt.Sprintf("GraphQL query failed: %v", err), http.StatusInternalServerError)
		return
	}

	if result.BusinessProcessesByPK == nil {
		http.Error(w, "process not found", http.StatusNotFound)
		return
	}

	bp := result.BusinessProcessesByPK
	response := BusinessProcess{
		ID:          bp.ID,
		TenantID:    bp.TenantID,
		ProcessName: bp.ProcessName,
		Description: bp.Description,
		Entity:      bp.EntityType,
		IsActive:    bp.IsActive,
		Version:     bp.VersionNumber,
		CreatedAt:   bp.CreatedAt,
		UpdatedAt:   bp.UpdatedAt,
		CreatedBy:   bp.CreatedBy,
	}

	for _, s := range bp.Steps {
		step := BPStep{
			ID:            s.ID,
			StepOrder:     s.StepOrder,
			StepType:      s.StepType,
			StepName:      s.StepName,
			Description:   s.Description,
			DurationHours: s.DurationHours,
			Status:        s.Status,
		}
		response.Steps = append(response.Steps, step)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// UpdateBusinessProcessGraphQL updates a business process using GraphQL
func (h *BusinessProcessGraphQLHandlers) UpdateBusinessProcessGraphQL(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")

	var updates map[string]interface{}
	if err := json.NewDecoder(r.Body).Decode(&updates); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}

	mutation := `
		mutation UpdateBusinessProcess($id: uuid!, $updates: business_processes_set_input!) {
			update_business_processes_by_pk(
				pk_columns: { id: $id }
				_set: $updates
			) {
				id
				updated_at
			}
		}
	`

	// Build the _set input - only include fields that are valid for update
	setInput := make(map[string]interface{})
	allowedFields := map[string]bool{
		"process_name":         true,
		"description":          true,
		"entity_type":          true,
		"status":               true,
		"is_active":            true,
		"total_duration_hours": true,
	}

	for key, value := range updates {
		if allowedFields[key] {
			setInput[key] = value
		}
	}

	variables := map[string]interface{}{
		"id":      id,
		"updates": setInput,
	}

	var result struct {
		UpdateBusinessProcessesByPK struct {
			ID        string `json:"id"`
			UpdatedAt string `json:"updated_at"`
		} `json:"update_business_processes_by_pk"`
	}

	if err := h.hasura.Query(r.Context(), mutation, variables, &result); err != nil {
		http.Error(w, fmt.Sprintf("GraphQL mutation failed: %v", err), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"id":         result.UpdateBusinessProcessesByPK.ID,
		"updated":    true,
		"updated_at": result.UpdateBusinessProcessesByPK.UpdatedAt,
	})
}

// DeleteBusinessProcessGraphQL deletes a business process using GraphQL
func (h *BusinessProcessGraphQLHandlers) DeleteBusinessProcessGraphQL(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")

	mutation := `
		mutation DeleteBusinessProcess($id: uuid!) {
			delete_business_processes_by_pk(id: $id) {
				id
			}
		}
	`

	variables := map[string]interface{}{
		"id": id,
	}

	var result struct {
		DeleteBusinessProcessesByPK *struct {
			ID string `json:"id"`
		} `json:"delete_business_processes_by_pk"`
	}

	if err := h.hasura.Query(r.Context(), mutation, variables, &result); err != nil {
		http.Error(w, fmt.Sprintf("GraphQL mutation failed: %v", err), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"id":      id,
		"deleted": result.DeleteBusinessProcessesByPK != nil,
	})
}

// ExecuteBusinessProcessGraphQL starts execution of a business process using GraphQL
func (h *BusinessProcessGraphQLHandlers) ExecuteBusinessProcessGraphQL(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")

	// First, get the business process to get tenant_id
	query := `
		query GetProcessForExecution($id: uuid!) {
			business_processes_by_pk(id: $id) {
				id
				tenant_id
			}
		}
	`

	var processResult struct {
		BusinessProcessesByPK *struct {
			ID       string `json:"id"`
			TenantID string `json:"tenant_id"`
		} `json:"business_processes_by_pk"`
	}

	if err := h.hasura.Query(r.Context(), query, map[string]interface{}{"id": id}, &processResult); err != nil {
		http.Error(w, fmt.Sprintf("GraphQL query failed: %v", err), http.StatusInternalServerError)
		return
	}

	if processResult.BusinessProcessesByPK == nil {
		http.Error(w, "process not found", http.StatusNotFound)
		return
	}

	// Create execution record
	executionID := uuid.New().String()
	mutation := `
		mutation CreateBPExecution(
			$id: uuid!
			$tenant_id: uuid!
			$business_process_id: uuid!
			$entity_id: uuid!
			$initiated_by: String!
			$execution_status: String!
			$current_step_order: Int!
		) {
			insert_bp_executions_one(
				object: {
					id: $id
					tenant_id: $tenant_id
					business_process_id: $business_process_id
					entity_id: $entity_id
					initiated_by: $initiated_by
					execution_status: $execution_status
					current_step_order: $current_step_order
				}
			) {
				id
				initiated_at
			}
		}
	`

	variables := map[string]interface{}{
		"id":                  executionID,
		"tenant_id":           processResult.BusinessProcessesByPK.TenantID,
		"business_process_id": id,
		"entity_id":           uuid.New().String(), // Placeholder entity
		"initiated_by":        "system",
		"execution_status":    "running",
		"current_step_order":  1,
	}

	var execResult struct {
		InsertBPExecutionsOne struct {
			ID          string `json:"id"`
			InitiatedAt string `json:"initiated_at"`
		} `json:"insert_bp_executions_one"`
	}

	if err := h.hasura.Query(r.Context(), mutation, variables, &execResult); err != nil {
		http.Error(w, fmt.Sprintf("GraphQL mutation failed: %v", err), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"id":           id,
		"execution_id": executionID,
		"status":       "started",
		"started_at":   execResult.InsertBPExecutionsOne.InitiatedAt,
	})
}

// RegisterBusinessProcessGraphQLRoutes registers routes that use GraphQL instead of SQL
func RegisterBusinessProcessGraphQLRoutes(r chi.Router, hasura *HasuraClient) {
	h := NewBusinessProcessGraphQLHandlers(hasura)

	r.Route("/api/business-process/v2", func(r chi.Router) {
		r.Get("/", h.ListBusinessProcessesGraphQL)
		r.Post("/", h.CreateBusinessProcessGraphQL)
		r.Get("/{id}", h.GetBusinessProcessGraphQL)
		r.Put("/{id}", h.UpdateBusinessProcessGraphQL)
		r.Delete("/{id}", h.DeleteBusinessProcessGraphQL)
		r.Post("/{id}/execute", h.ExecuteBusinessProcessGraphQL)

		// Static configuration endpoints (no GraphQL needed)
		r.Get("/step-types", func(w http.ResponseWriter, r *http.Request) {
			stepTypes := []ProcessStepType{
				{
					ID:          "data_entry",
					Key:         "data_entry",
					Label:       "Data Entry",
					Description: "Manual data entry step",
					DefaultData: json.RawMessage(`{"required": true}`),
				},
				{
					ID:          "validation",
					Key:         "validation",
					Label:       "Validation",
					Description: "Data validation step",
					DefaultData: json.RawMessage(`{"rules": []}`),
				},
				{
					ID:          "approval",
					Key:         "approval",
					Label:       "Approval",
					Description: "Approval workflow step",
					DefaultData: json.RawMessage(`{"approvers": []}`),
				},
			}
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(stepTypes)
		})

		r.Get("/validation-operators", func(w http.ResponseWriter, r *http.Request) {
			operators := []ValidationOperator{
				{
					ID:          "equals",
					Key:         "equals",
					Label:       "Equals",
					Description: "Value must equal specified value",
					ValueType:   "string",
				},
				{
					ID:          "greater_than",
					Key:         "greater_than",
					Label:       "Greater Than",
					Description: "Value must be greater than specified value",
					ValueType:   "number",
				},
			}
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(operators)
		})

		r.Get("/workflow-events", func(w http.ResponseWriter, r *http.Request) {
			events := []WorkflowEvent{
				{
					ID:          "record_created",
					Key:         "record_created",
					Label:       "Record Created",
					Description: "Triggered when a new record is created",
					EventType:   "data",
				},
				{
					ID:          "record_updated",
					Key:         "record_updated",
					Label:       "Record Updated",
					Description: "Triggered when a record is updated",
					EventType:   "data",
				},
			}
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(events)
		})
	})
}
