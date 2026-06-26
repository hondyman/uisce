package api

import (
	"database/sql"
	"encoding/json"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/hondyman/semlayer/libs/jwt-middleware"
)

// ProcessStepType represents a step type in the palette
type ProcessStepType struct {
	ID          string          `json:"id"`
	Key         string          `json:"key"`
	Label       string          `json:"label"`
	Description string          `json:"description,omitempty"`
	IconSVG     string          `json:"icon_svg,omitempty"`
	DefaultData json.RawMessage `json:"default_data"`
	CreatedAt   time.Time       `json:"created_at"`
	UpdatedAt   time.Time       `json:"updated_at"`
}

// ValidationOperator represents an operator for validation rules
type ValidationOperator struct {
	ID          string          `json:"id"`
	Key         string          `json:"key"`
	Label       string          `json:"label"`
	Description string          `json:"description,omitempty"`
	ValueType   string          `json:"value_type"`
	Config      json.RawMessage `json:"config,omitempty"`
	CreatedAt   time.Time       `json:"created_at"`
	UpdatedAt   time.Time       `json:"updated_at"`
}

// WorkflowEvent represents an event that triggers steps
type WorkflowEvent struct {
	ID          string          `json:"id"`
	Key         string          `json:"key"`
	Label       string          `json:"label"`
	Description string          `json:"description,omitempty"`
	EventType   string          `json:"event_type"`
	Config      json.RawMessage `json:"config,omitempty"`
	CreatedAt   time.Time       `json:"created_at"`
	UpdatedAt   time.Time       `json:"updated_at"`
}

// BusinessObjectField represents a field in a business object
type BusinessObjectField struct {
	Name  string `json:"name"`
	Type  string `json:"type"`
	Label string `json:"label"`
}

// BusinessObject represents an entity (client, account, etc.)
type BusinessObject struct {
	ID          string                `json:"id"`
	Name        string                `json:"name"`
	DisplayName string                `json:"display_name"`
	Description string                `json:"description,omitempty"`
	Fields      []BusinessObjectField `json:"fields"`
	Icon        string                `json:"icon,omitempty"`
	Config      json.RawMessage       `json:"config,omitempty"`
	CreatedAt   time.Time             `json:"created_at"`
	UpdatedAt   time.Time             `json:"updated_at"`
}

// ProcessValidationRule represents a validation rule specific to a process node
type ProcessValidationRule struct {
	ID          string          `json:"id"`
	ProcessID   string          `json:"process_id"`
	NodeID      string          `json:"node_id"`
	Field       string          `json:"field"`
	FieldLabel  string          `json:"field_label,omitempty"`
	OperatorKey string          `json:"op"`
	OpLabel     string          `json:"op_label,omitempty"`
	Value       string          `json:"value"`
	Message     string          `json:"message"`
	Severity    string          `json:"severity"`
	OrderIndex  int             `json:"order_index"`
	Enabled     bool            `json:"enabled"`
	Config      json.RawMessage `json:"config,omitempty"`
	CreatedAt   time.Time       `json:"created_at"`
	UpdatedAt   time.Time       `json:"updated_at"`
}

// ProcessNode represents a step node on the canvas
type ProcessNode struct {
	ID   string          `json:"id"`
	Type string          `json:"type"`
	Data json.RawMessage `json:"data"`
	Pos  struct {
		X float64 `json:"x"`
		Y float64 `json:"y"`
	} `json:"position"`
}

// ProcessEdge represents a connection between nodes
type ProcessEdge struct {
	ID     string `json:"id"`
	Source string `json:"source"`
	Target string `json:"target"`
}

// Process represents the overall workflow definition
type Process struct {
	ID           string          `json:"id"`
	Name         string          `json:"name"`
	Description  string          `json:"description,omitempty"`
	Version      int             `json:"version"`
	Nodes        []ProcessNode   `json:"nodes"`
	Edges        []ProcessEdge   `json:"edges"`
	Config       json.RawMessage `json:"config,omitempty"`
	Status       string          `json:"status"`
	CreatedAt    time.Time       `json:"created_at"`
	UpdatedAt    time.Time       `json:"updated_at"`
	CreatedBy    string          `json:"created_by,omitempty"`
	UpdatedBy    string          `json:"updated_by,omitempty"`
	TenantID     string          `json:"tenant_id,omitempty"`
	DatasourceID string          `json:"datasource_id,omitempty"`
}

// BPDesignerHandlers provides HTTP handlers for the Business Process Designer
type BPDesignerHandlers struct {
	DB *sql.DB
}

// NewBPDesignerHandlers creates a new handler
func NewBPDesignerHandlers(database *sql.DB) *BPDesignerHandlers {
	return &BPDesignerHandlers{DB: database}
}

// GetStepTypes returns all available step types
// GET /api/step-types
func (h *BPDesignerHandlers) GetStepTypes(w http.ResponseWriter, r *http.Request) {
	// In a real app, tenantID would come from middleware/context.
	claims := jwtmiddleware.GetClaimsFromContext(r)
	if claims == nil {
		http.Error(w, `{"error":"unauthorized"}`, http.StatusUnauthorized)
		return
	}
	tenantID := claims.TenantID
	if tenantID == "" {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": "tenant_id required"})
		return
	}

	rows, err := h.DB.QueryContext(r.Context(), `
		SELECT id, key, label, description, icon_svg, default_data, created_at, updated_at
		FROM process_step_types
		WHERE tenant_id IS NULL OR tenant_id = $1
		ORDER BY key
	`, tenantID)
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
		return
	}
	defer rows.Close()

	var stepTypes []ProcessStepType
	for rows.Next() {
		var st ProcessStepType
		if err := rows.Scan(&st.ID, &st.Key, &st.Label, &st.Description, &st.IconSVG, &st.DefaultData, &st.CreatedAt, &st.UpdatedAt); err != nil {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusInternalServerError)
			json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
			return
		}
		stepTypes = append(stepTypes, st)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(stepTypes)
}

// GetValidationOperators returns all available validation operators
// GET /api/validation-operators
func (h *BPDesignerHandlers) GetValidationOperators(w http.ResponseWriter, r *http.Request) {
	claims := jwtmiddleware.GetClaimsFromContext(r)
	if claims == nil {
		http.Error(w, `{"error":"unauthorized"}`, http.StatusUnauthorized)
		return
	}
	tenantID := claims.TenantID
	if tenantID == "" {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": "tenant_id required"})
		return
	}

	rows, err := h.DB.QueryContext(r.Context(), `
		SELECT id, key, label, description, value_type, config, created_at, updated_at
		FROM validation_operators
		WHERE tenant_id IS NULL OR tenant_id = $1
		ORDER BY label
	`, tenantID)
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
		return
	}
	defer rows.Close()

	var operators []ValidationOperator
	for rows.Next() {
		var op ValidationOperator
		if err := rows.Scan(&op.ID, &op.Key, &op.Label, &op.Description, &op.ValueType, &op.Config, &op.CreatedAt, &op.UpdatedAt); err != nil {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusInternalServerError)
			json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
			return
		}
		operators = append(operators, op)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(operators)
}

// GetWorkflowEvents returns all available events
// GET /api/events
func (h *BPDesignerHandlers) GetWorkflowEvents(w http.ResponseWriter, r *http.Request) {
	claims := jwtmiddleware.GetClaimsFromContext(r)
	if claims == nil {
		http.Error(w, `{"error":"unauthorized"}`, http.StatusUnauthorized)
		return
	}
	tenantID := claims.TenantID
	if tenantID == "" {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": "tenant_id required"})
		return
	}

	rows, err := h.DB.QueryContext(r.Context(), `
		SELECT id, key, label, description, event_type, config, created_at, updated_at
		FROM workflow_events
		WHERE tenant_id IS NULL OR tenant_id = $1
		ORDER BY label
	`, tenantID)
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
		return
	}
	defer rows.Close()

	var events []WorkflowEvent
	for rows.Next() {
		var evt WorkflowEvent
		if err := rows.Scan(&evt.ID, &evt.Key, &evt.Label, &evt.Description, &evt.EventType, &evt.Config, &evt.CreatedAt, &evt.UpdatedAt); err != nil {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusInternalServerError)
			json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
			return
		}
		events = append(events, evt)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(events)
}

// GetBusinessObjects returns all business objects with their fields
// GET /api/business-objects
func (h *BPDesignerHandlers) GetBusinessObjects(w http.ResponseWriter, r *http.Request) {
	claims := jwtmiddleware.GetClaimsFromContext(r)
	if claims == nil {
		http.Error(w, `{"error":"unauthorized"}`, http.StatusUnauthorized)
		return
	}
	tenantID := claims.TenantID
	datasourceID := r.Header.Get("X-Tenant-Datasource-ID")

	if tenantID == "" {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": "tenant_id required"})
		return
	}

	query := `
		SELECT id, name, display_name, description, icon, config, created_at, updated_at
		FROM business_objects
		WHERE tenant_id = $1
	`
	args := []interface{}{tenantID}

	if datasourceID != "" {
		query += " AND (datasource_id = $2 OR datasource_id IS NULL)"
		args = append(args, datasourceID)
	}
	query += " ORDER BY display_name"

	rows, err := h.DB.QueryContext(r.Context(), query, args...)
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
		return
	}
	defer rows.Close()

	var objects []BusinessObject
	for rows.Next() {
		var bo BusinessObject
		if err := rows.Scan(&bo.ID, &bo.Name, &bo.DisplayName, &bo.Description, &bo.Icon, &bo.Config, &bo.CreatedAt, &bo.UpdatedAt); err != nil {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusInternalServerError)
			json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
			return
		}

		// Load fields from bo_fields table
		fieldRows, err := h.DB.QueryContext(r.Context(), `
			SELECT field_name, display_label, field_type
			FROM bo_fields
			WHERE bo_id = $1
			ORDER BY display_order
		`, bo.ID)
		if err != nil {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusInternalServerError)
			json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
			return
		}

		var fields []BusinessObjectField
		for fieldRows.Next() {
			var f BusinessObjectField
			if err := fieldRows.Scan(&f.Name, &f.Label, &f.Type); err != nil {
				fieldRows.Close()
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusInternalServerError)
				json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
				return
			}
			fields = append(fields, f)
		}
		fieldRows.Close()

		bo.Fields = fields
		objects = append(objects, bo)
	}

	// Convert array to map with IDs as keys (frontend expects object, not array)
	objectsMap := make(map[string]BusinessObject)
	for _, bo := range objects {
		objectsMap[bo.ID] = bo
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(objectsMap)
}

// CreateProcess creates a new process
// POST /api/processes
func (h *BPDesignerHandlers) CreateProcess(w http.ResponseWriter, r *http.Request) {
	claims := jwtmiddleware.GetClaimsFromContext(r)
	if claims == nil {
		http.Error(w, `{"error":"unauthorized"}`, http.StatusUnauthorized)
		return
	}
	tenantID := claims.TenantID
	datasourceID := r.Header.Get("X-Datasource-ID")
	userID := r.Header.Get("X-User-ID")

	if tenantID == "" || datasourceID == "" {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": "tenant_id and datasource_id required"})
		return
	}

	var p Process
	if err := json.NewDecoder(r.Body).Decode(&p); err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
		return
	}

	id := uuid.New().String()
	nodesJSON, _ := json.Marshal(p.Nodes)
	edgesJSON, _ := json.Marshal(p.Edges)

	err := h.DB.QueryRowContext(r.Context(), `
		INSERT INTO processes (id, name, description, version, nodes, edges, status, created_by, updated_by, tenant_id, datasource_id)
		VALUES ($1, $2, $3, 1, $4, $5, 'draft', $6, $7, $8, $9)
		RETURNING id, name, version, created_at, updated_at
	`, id, p.Name, p.Description, nodesJSON, edgesJSON, userID, userID, tenantID, datasourceID).Scan(
		&p.ID, &p.Name, &p.Version, &p.CreatedAt, &p.UpdatedAt)

	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
		return
	}

	p.Nodes = []ProcessNode{}
	p.Edges = []ProcessEdge{}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(p)
}

// GetProcess retrieves a process by ID
// GET /api/processes/:id
func (h *BPDesignerHandlers) GetProcess(w http.ResponseWriter, r *http.Request) {
	claims := jwtmiddleware.GetClaimsFromContext(r)
	if claims == nil {
		http.Error(w, `{"error":"unauthorized"}`, http.StatusUnauthorized)
		return
	}
	tenantID := claims.TenantID
	processID := chi.URLParam(r, "id")

	if tenantID == "" {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": "tenant_id required"})
		return
	}

	var p Process
	var nodesJSON, edgesJSON json.RawMessage

	err := h.DB.QueryRowContext(r.Context(), `
		SELECT id, name, description, version, nodes, edges, status, created_at, updated_at, created_by, updated_by, tenant_id, datasource_id
		FROM processes
		WHERE id = $1 AND tenant_id = $2
	`, processID, tenantID).Scan(
		&p.ID, &p.Name, &p.Description, &p.Version, &nodesJSON, &edgesJSON, &p.Status, &p.CreatedAt, &p.UpdatedAt, &p.CreatedBy, &p.UpdatedBy, &p.TenantID, &p.DatasourceID)

	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusNotFound)
		json.NewEncoder(w).Encode(map[string]string{"error": "process not found"})
		return
	}

	json.Unmarshal(nodesJSON, &p.Nodes)
	json.Unmarshal(edgesJSON, &p.Edges)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(p)
}

// UpdateProcessNodes updates nodes and edges for a process
// PATCH /api/processes/:id
func (h *BPDesignerHandlers) UpdateProcessNodes(w http.ResponseWriter, r *http.Request) {
	claims := jwtmiddleware.GetClaimsFromContext(r)
	if claims == nil {
		http.Error(w, `{"error":"unauthorized"}`, http.StatusUnauthorized)
		return
	}
	tenantID := claims.TenantID
	datasourceID := r.Header.Get("X-Datasource-ID")
	userID := r.Header.Get("X-User-ID")
	processID := chi.URLParam(r, "id")

	if tenantID == "" {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": "tenant_id required"})
		return
	}

	var payload struct {
		Nodes []ProcessNode `json:"nodes"`
		Edges []ProcessEdge `json:"edges"`
	}

	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
		return
	}

	nodesJSON, _ := json.Marshal(payload.Nodes)
	edgesJSON, _ := json.Marshal(payload.Edges)

	_, err := h.DB.ExecContext(r.Context(), `
		UPDATE processes
		SET nodes = $1, edges = $2, updated_by = $3, updated_at = now()
		WHERE id = $4 AND tenant_id = $5 AND datasource_id = $6
	`, nodesJSON, edgesJSON, userID, processID, tenantID, datasourceID)

	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"status": "updated"})
}

// SaveValidationRules saves rules for a validation step
// POST /api/processes/:id/nodes/:nodeId/rules
func (h *BPDesignerHandlers) SaveValidationRules(w http.ResponseWriter, r *http.Request) {
	claims := jwtmiddleware.GetClaimsFromContext(r)
	if claims == nil {
		http.Error(w, `{"error":"unauthorized"}`, http.StatusUnauthorized)
		return
	}
	tenantID := claims.TenantID
	processID := chi.URLParam(r, "id")
	nodeID := chi.URLParam(r, "nodeId")

	if tenantID == "" {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": "tenant_id required"})
		return
	}

	var rules []ProcessValidationRule
	if err := json.NewDecoder(r.Body).Decode(&rules); err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
		return
	}

	// Delete existing rules for this node
	h.DB.ExecContext(r.Context(), `DELETE FROM validation_rules WHERE process_id = $1 AND node_id = $2 AND tenant_id = $3`,
		processID, nodeID, tenantID)

	// Insert new rules
	for i, rule := range rules {
		id := uuid.New().String()
		_, err := h.DB.ExecContext(r.Context(), `
			INSERT INTO validation_rules (id, process_id, node_id, field, operator_key, value, message, severity, order_index, enabled, tenant_id)
			VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, true, $10)
		`, id, processID, nodeID, rule.Field, rule.OperatorKey, rule.Value, rule.Message, "warning", i, tenantID)

		if err != nil {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusInternalServerError)
			json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
			return
		}
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(map[string]any{"status": "rules saved", "count": len(rules)})
}

// GetValidationRules retrieves rules for a validation step
// GET /api/processes/:id/nodes/:nodeId/rules
func (h *BPDesignerHandlers) GetValidationRules(w http.ResponseWriter, r *http.Request) {
	claims := jwtmiddleware.GetClaimsFromContext(r)
	if claims == nil {
		http.Error(w, `{"error":"unauthorized"}`, http.StatusUnauthorized)
		return
	}
	tenantID := claims.TenantID
	processID := chi.URLParam(r, "id")
	nodeID := chi.URLParam(r, "nodeId")

	if tenantID == "" {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": "tenant_id required"})
		return
	}

	rows, err := h.DB.QueryContext(r.Context(), `
		SELECT id, process_id, node_id, field, operator_key, value, message, severity, order_index, enabled, created_at, updated_at
		FROM validation_rules
		WHERE process_id = $1 AND node_id = $2 AND tenant_id = $3
		ORDER BY order_index
	`, processID, nodeID, tenantID)
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
		return
	}
	defer rows.Close()

	var rules []ProcessValidationRule
	for rows.Next() {
		var r ProcessValidationRule
		if err := rows.Scan(&r.ID, &r.ProcessID, &r.NodeID, &r.Field, &r.OperatorKey, &r.Value, &r.Message, &r.Severity, &r.OrderIndex, &r.Enabled, &r.CreatedAt, &r.UpdatedAt); err != nil {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusInternalServerError)
			json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
			return
		}
		rules = append(rules, r)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(rules)
}

// RegisterRoutes registers all BP Designer routes
func (h *BPDesignerHandlers) RegisterRoutes(r chi.Router) {
	// Configuration endpoints
	r.Get("/api/step-types", h.GetStepTypes)
	r.Get("/api/validation-operators", h.GetValidationOperators)
	r.Get("/api/events", h.GetWorkflowEvents)
	r.Get("/api/business-objects", h.GetBusinessObjects)

	// Process CRUD
	r.Route("/api/processes", func(r chi.Router) {
		r.Post("/", h.CreateProcess)
		r.Route("/{id}", func(r chi.Router) {
			r.Get("/", h.GetProcess)
			r.Patch("/", h.UpdateProcessNodes)
			r.Route("/nodes/{nodeId}/rules", func(r chi.Router) {
				r.Post("/", h.SaveValidationRules)
				r.Get("/", h.GetValidationRules)
			})
		})
	})
}
