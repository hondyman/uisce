package api

import (
	"database/sql"
	"encoding/json"
	"log"
	"net/http"
	"sync"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/gorilla/websocket"
	"github.com/jmoiron/sqlx"
)

// ProcessMonitorHandlers handles live process monitoring and WebSocket connections
type ProcessMonitorHandlers struct {
	db        *sqlx.DB
	upgrader  websocket.Upgrader
	clients   map[*websocket.Conn]ClientInfo
	mu        sync.RWMutex
	broadcast chan ProcessEvent
}

// ClientInfo stores metadata about connected WebSocket clients
type ClientInfo struct {
	TenantID     string
	DatasourceID string
	Filters      map[string]string
}

// ProcessEvent represents a real-time process event
type ProcessEvent struct {
	Type         string                 `json:"type"` // step_started, step_completed, step_failed, instance_created
	WorkflowID   string                 `json:"workflow_id"`
	WorkflowType string                 `json:"workflow_type"`
	StepName     string                 `json:"step_name,omitempty"`
	Status       string                 `json:"status"` // running, completed, failed
	Timestamp    time.Time              `json:"timestamp"`
	TenantID     string                 `json:"tenant_id"`
	DatasourceID string                 `json:"datasource_id"`
	Metadata     map[string]interface{} `json:"metadata,omitempty"`
}

// ProcessInstance represents a live process instance
type ProcessInstance struct {
	WorkflowID       string                 `json:"workflow_id" db:"workflow_id"`
	WorkflowType     string                 `json:"workflow_type" db:"workflow_type"`
	Status           string                 `json:"status" db:"status"`
	CurrentStep      sql.NullString         `json:"current_step" db:"current_step"`
	StartedAt        time.Time              `json:"started_at" db:"started_at"`
	LastActivityAt   time.Time              `json:"last_activity_at" db:"last_activity_at"`
	StepsCompleted   int                    `json:"steps_completed" db:"steps_completed"`
	StepsTotal       int                    `json:"steps_total" db:"steps_total"`
	SLADeadline      *time.Time             `json:"sla_deadline" db:"sla_deadline"`
	TimeRemaining    *float64               `json:"time_remaining,omitempty"` // minutes
	HealthScore      int                    `json:"health_score" db:"health_score"`
	TenantID         string                 `json:"tenant_id" db:"tenant_id"`
	DatasourceID     string                 `json:"datasource_id" db:"datasource_id"`
	Owner            sql.NullString         `json:"owner" db:"owner"`
	Metadata         map[string]interface{} `json:"metadata" db:"metadata"`
	ExecutionHistory []StepExecution        `json:"execution_history,omitempty"`
}

// StepExecution represents a single step execution in history
type StepExecution struct {
	StepName    string                 `json:"step_name" db:"step_name"`
	Status      string                 `json:"status" db:"status"`
	StartedAt   time.Time              `json:"started_at" db:"started_at"`
	CompletedAt *time.Time             `json:"completed_at" db:"completed_at"`
	Duration    *float64               `json:"duration" db:"duration"` // seconds
	ErrorMsg    sql.NullString         `json:"error_msg" db:"error_msg"`
	Metadata    map[string]interface{} `json:"metadata" db:"metadata"`
}

// InterventionRequest represents a manual intervention action
type InterventionRequest struct {
	Action      string                 `json:"action"` // skip_step, reassign, cancel, retry
	WorkflowID  string                 `json:"workflow_id"`
	StepName    string                 `json:"step_name,omitempty"`
	NewAssignee string                 `json:"new_assignee,omitempty"`
	Reason      string                 `json:"reason"`
	Metadata    map[string]interface{} `json:"metadata,omitempty"`
}

// NewProcessMonitorHandlers creates a new process monitor handler
func NewProcessMonitorHandlers(db *sqlx.DB) *ProcessMonitorHandlers {
	h := &ProcessMonitorHandlers{
		db: db,
		upgrader: websocket.Upgrader{
			ReadBufferSize:  1024,
			WriteBufferSize: 1024,
			CheckOrigin: func(r *http.Request) bool {
				// TODO: Implement proper origin checking in production
				return true
			},
		},
		clients:   make(map[*websocket.Conn]ClientInfo),
		broadcast: make(chan ProcessEvent, 100),
	}

	// Start broadcast goroutine
	go h.handleBroadcasts()

	return h
}

// RegisterRoutes registers all process monitoring routes
func (h *ProcessMonitorHandlers) RegisterRoutes(r chi.Router) {
	r.Route("/api/process-monitor", func(r chi.Router) {
		// WebSocket endpoint
		r.Get("/ws", h.HandleWebSocket)

		// REST endpoints
		r.Get("/active-instances", h.GetActiveInstances)
		r.Get("/instance/{workflowID}", h.GetInstanceDetails)
		r.Get("/instance/{workflowID}/history", h.GetExecutionHistory)
		r.Post("/intervene", h.HandleIntervention)
		r.Get("/stats", h.GetMonitoringStats)
	})
}

// HandleWebSocket handles WebSocket connections for real-time updates
func (h *ProcessMonitorHandlers) HandleWebSocket(w http.ResponseWriter, r *http.Request) {
	// Extract tenant/datasource from query params
	tenantID := r.URL.Query().Get("tenant_id")
	datasourceID := r.URL.Query().Get("datasource_id")

	if tenantID == "" || datasourceID == "" {
		http.Error(w, "Missing tenant_id or datasource_id", http.StatusBadRequest)
		return
	}

	// Upgrade connection
	conn, err := h.upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("WebSocket upgrade error: %v", err)
		return
	}

	// Register client
	h.mu.Lock()
	h.clients[conn] = ClientInfo{
		TenantID:     tenantID,
		DatasourceID: datasourceID,
		Filters:      make(map[string]string),
	}
	h.mu.Unlock()

	log.Printf("WebSocket client connected: tenant=%s, datasource=%s", tenantID, datasourceID)

	// Send initial connection confirmation
	conn.WriteJSON(map[string]string{
		"type":    "connected",
		"message": "Successfully connected to process monitor",
	})

	// Handle client messages and cleanup
	defer func() {
		h.mu.Lock()
		delete(h.clients, conn)
		h.mu.Unlock()
		conn.Close()
		log.Printf("WebSocket client disconnected: tenant=%s", tenantID)
	}()

	// Read loop for client messages (filters, heartbeat, etc.)
	for {
		var msg map[string]interface{}
		err := conn.ReadJSON(&msg)
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				log.Printf("WebSocket error: %v", err)
			}
			break
		}

		// Handle client messages (e.g., filter updates)
		if msgType, ok := msg["type"].(string); ok {
			if msgType == "update_filters" {
				h.mu.Lock()
				if info, exists := h.clients[conn]; exists {
					if filters, ok := msg["filters"].(map[string]interface{}); ok {
						info.Filters = make(map[string]string)
						for k, v := range filters {
							if strVal, ok := v.(string); ok {
								info.Filters[k] = strVal
							}
						}
						h.clients[conn] = info
					}
				}
				h.mu.Unlock()
			}
		}
	}
}

// handleBroadcasts processes events and sends to relevant WebSocket clients
func (h *ProcessMonitorHandlers) handleBroadcasts() {
	for event := range h.broadcast {
		h.mu.RLock()
		for conn, info := range h.clients {
			// Filter events by tenant/datasource
			if info.TenantID != event.TenantID || info.DatasourceID != event.DatasourceID {
				continue
			}

			// Apply additional filters if set
			if len(info.Filters) > 0 {
				if workflowType, ok := info.Filters["workflow_type"]; ok {
					if workflowType != "" && workflowType != event.WorkflowType {
						continue
					}
				}
				if status, ok := info.Filters["status"]; ok {
					if status != "" && status != event.Status {
						continue
					}
				}
			}

			// Send event to client (non-blocking)
			go func(c *websocket.Conn, e ProcessEvent) {
				if err := c.WriteJSON(e); err != nil {
					log.Printf("Error sending to WebSocket client: %v", err)
				}
			}(conn, event)
		}
		h.mu.RUnlock()
	}
}

// BroadcastEvent broadcasts an event to all connected clients
func (h *ProcessMonitorHandlers) BroadcastEvent(event ProcessEvent) {
	select {
	case h.broadcast <- event:
	default:
		log.Printf("Broadcast channel full, dropping event: %s", event.Type)
	}
}

// GetActiveInstances returns all currently running process instances
func (h *ProcessMonitorHandlers) GetActiveInstances(w http.ResponseWriter, r *http.Request) {
	tenantID := r.URL.Query().Get("tenant_id")
	datasourceID := r.URL.Query().Get("datasource_id")
	workflowType := r.URL.Query().Get("workflow_type")
	status := r.URL.Query().Get("status")

	if tenantID == "" || datasourceID == "" {
		http.Error(w, "Missing tenant_id or datasource_id", http.StatusBadRequest)
		return
	}

	query := `
		WITH latest_steps AS (
			SELECT DISTINCT ON (workflow_id)
				workflow_id,
				step_name,
				status,
				started_at,
				completed_at
			FROM process_execution_metrics
			WHERE tenant_id = $1
				AND datasource_id = $2
			ORDER BY workflow_id, started_at DESC
		),
		workflow_stats AS (
			SELECT
				workflow_id,
				workflow_type,
				COUNT(*) as steps_total,
				COUNT(*) FILTER (WHERE status = 'completed') as steps_completed,
				MIN(started_at) as started_at,
				MAX(started_at) as last_activity_at
			FROM process_execution_metrics
			WHERE tenant_id = $1
				AND datasource_id = $2
			GROUP BY workflow_id, workflow_type
		)
		SELECT
			ws.workflow_id,
			ws.workflow_type,
			CASE
				WHEN ls.status = 'failed' THEN 'failed'
				WHEN ws.steps_completed = ws.steps_total THEN 'completed'
				ELSE 'running'
			END as status,
			ls.step_name as current_step,
			ws.started_at,
			ws.last_activity_at,
			ws.steps_completed,
			ws.steps_total,
			NULL::TIMESTAMP as sla_deadline,
			CASE
				WHEN ws.steps_completed = ws.steps_total THEN 100
				WHEN ls.status = 'failed' THEN 0
				ELSE ROUND(((ws.steps_completed::FLOAT / NULLIF(ws.steps_total, 0)) * 100))::INT
			END as health_score,
			$1 as tenant_id,
			$2 as datasource_id,
			NULL::VARCHAR as owner,
			'{}'::JSONB as metadata
		FROM workflow_stats ws
		LEFT JOIN latest_steps ls ON ws.workflow_id = ls.workflow_id
		WHERE 1=1
	`

	args := []interface{}{tenantID, datasourceID}
	argCount := 3

	if workflowType != "" {
		query += " AND ws.workflow_type = $" + string(rune(argCount))
		args = append(args, workflowType)
		argCount++
	}

	if status != "" {
		query += ` AND (
			CASE
				WHEN ls.status = 'failed' THEN 'failed'
				WHEN ws.steps_completed = ws.steps_total THEN 'completed'
				ELSE 'running'
			END = $` + string(rune(argCount)) + `
		)`
		args = append(args, status)
		argCount++
	}

	// Only show processes with activity in last 24 hours for "active" view
	query += " AND ws.last_activity_at > NOW() - INTERVAL '24 hours'"
	query += " ORDER BY ws.last_activity_at DESC LIMIT 100"

	var instances []ProcessInstance
	err := h.db.Select(&instances, query, args...)
	if err != nil {
		log.Printf("Error querying active instances: %v", err)
		http.Error(w, "Failed to fetch active instances", http.StatusInternalServerError)
		return
	}

	// Calculate time remaining for SLA
	now := time.Now()
	for i := range instances {
		if instances[i].SLADeadline != nil {
			remaining := instances[i].SLADeadline.Sub(now).Minutes()
			instances[i].TimeRemaining = &remaining
		}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(instances)
}

// GetInstanceDetails returns detailed information about a specific process instance
func (h *ProcessMonitorHandlers) GetInstanceDetails(w http.ResponseWriter, r *http.Request) {
	workflowID := chi.URLParam(r, "workflowID")
	tenantID := r.URL.Query().Get("tenant_id")
	datasourceID := r.URL.Query().Get("datasource_id")

	if tenantID == "" || datasourceID == "" {
		http.Error(w, "Missing tenant_id or datasource_id", http.StatusBadRequest)
		return
	}

	var instance ProcessInstance

	// Get instance summary
	query := `
		WITH latest_step AS (
			SELECT DISTINCT ON (workflow_id)
				workflow_id,
				step_name,
				status
			FROM process_execution_metrics
			WHERE workflow_id = $1
				AND tenant_id = $2
				AND datasource_id = $3
			ORDER BY workflow_id, started_at DESC
		),
		workflow_stats AS (
			SELECT
				workflow_id,
				workflow_type,
				COUNT(*) as steps_total,
				COUNT(*) FILTER (WHERE status = 'completed') as steps_completed,
				MIN(started_at) as started_at,
				MAX(started_at) as last_activity_at
			FROM process_execution_metrics
			WHERE workflow_id = $1
				AND tenant_id = $2
				AND datasource_id = $3
			GROUP BY workflow_id, workflow_type
		)
		SELECT
			ws.workflow_id,
			ws.workflow_type,
			CASE
				WHEN ls.status = 'failed' THEN 'failed'
				WHEN ws.steps_completed = ws.steps_total THEN 'completed'
				ELSE 'running'
			END as status,
			ls.step_name as current_step,
			ws.started_at,
			ws.last_activity_at,
			ws.steps_completed,
			ws.steps_total,
			NULL::TIMESTAMP as sla_deadline,
			CASE
				WHEN ws.steps_completed = ws.steps_total THEN 100
				WHEN ls.status = 'failed' THEN 0
				ELSE ROUND(((ws.steps_completed::FLOAT / NULLIF(ws.steps_total, 0)) * 100))::INT
			END as health_score,
			$2 as tenant_id,
			$3 as datasource_id,
			NULL::VARCHAR as owner,
			'{}'::JSONB as metadata
		FROM workflow_stats ws
		LEFT JOIN latest_step ls ON ws.workflow_id = ls.workflow_id
	`

	err := h.db.Get(&instance, query, workflowID, tenantID, datasourceID)
	if err != nil {
		if err == sql.ErrNoRows {
			http.Error(w, "Instance not found", http.StatusNotFound)
			return
		}
		log.Printf("Error querying instance details: %v", err)
		http.Error(w, "Failed to fetch instance details", http.StatusInternalServerError)
		return
	}

	// Get execution history
	historyQuery := `
		SELECT
			step_name,
			status,
			started_at,
			completed_at,
			EXTRACT(EPOCH FROM (completed_at - started_at)) as duration,
			metadata
		FROM process_execution_metrics
		WHERE workflow_id = $1
			AND tenant_id = $2
			AND datasource_id = $3
		ORDER BY started_at ASC
	`

	err = h.db.Select(&instance.ExecutionHistory, historyQuery, workflowID, tenantID, datasourceID)
	if err != nil && err != sql.ErrNoRows {
		log.Printf("Error querying execution history: %v", err)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(instance)
}

// GetExecutionHistory returns full execution history for a workflow
func (h *ProcessMonitorHandlers) GetExecutionHistory(w http.ResponseWriter, r *http.Request) {
	workflowID := chi.URLParam(r, "workflowID")
	tenantID := r.URL.Query().Get("tenant_id")
	datasourceID := r.URL.Query().Get("datasource_id")

	if tenantID == "" || datasourceID == "" {
		http.Error(w, "Missing tenant_id or datasource_id", http.StatusBadRequest)
		return
	}

	query := `
		SELECT
			step_name,
			status,
			started_at,
			completed_at,
			EXTRACT(EPOCH FROM (completed_at - started_at)) as duration,
			metadata
		FROM process_execution_metrics
		WHERE workflow_id = $1
			AND tenant_id = $2
			AND datasource_id = $3
		ORDER BY started_at ASC
	`

	var history []StepExecution
	err := h.db.Select(&history, query, workflowID, tenantID, datasourceID)
	if err != nil {
		log.Printf("Error querying execution history: %v", err)
		http.Error(w, "Failed to fetch execution history", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(history)
}

// HandleIntervention handles manual intervention requests
func (h *ProcessMonitorHandlers) HandleIntervention(w http.ResponseWriter, r *http.Request) {
	tenantID := r.URL.Query().Get("tenant_id")
	datasourceID := r.URL.Query().Get("datasource_id")

	if tenantID == "" || datasourceID == "" {
		http.Error(w, "Missing tenant_id or datasource_id", http.StatusBadRequest)
		return
	}

	var req InterventionRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// Validate intervention action
	validActions := map[string]bool{
		"skip_step": true,
		"reassign":  true,
		"cancel":    true,
		"retry":     true,
	}

	if !validActions[req.Action] {
		http.Error(w, "Invalid action", http.StatusBadRequest)
		return
	}

	// TODO: Integrate with Temporal to actually perform the intervention
	// For now, log the intervention and broadcast event

	interventionID := uuid.New().String()

	// Log intervention to database
	_, err := h.db.Exec(`
		INSERT INTO process_interventions (
			id, workflow_id, action, step_name, new_assignee, reason, metadata, tenant_id, datasource_id, created_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, NOW())
	`, interventionID, req.WorkflowID, req.Action, req.StepName, req.NewAssignee, req.Reason, req.Metadata, tenantID, datasourceID)

	if err != nil {
		log.Printf("Error logging intervention: %v", err)
		http.Error(w, "Failed to log intervention", http.StatusInternalServerError)
		return
	}

	// Broadcast intervention event
	h.BroadcastEvent(ProcessEvent{
		Type:         "intervention_" + req.Action,
		WorkflowID:   req.WorkflowID,
		StepName:     req.StepName,
		Status:       "intervention_pending",
		Timestamp:    time.Now(),
		TenantID:     tenantID,
		DatasourceID: datasourceID,
		Metadata: map[string]interface{}{
			"intervention_id": interventionID,
			"action":          req.Action,
			"reason":          req.Reason,
		},
	})

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success":         true,
		"intervention_id": interventionID,
		"message":         "Intervention scheduled: " + req.Action,
	})
}

// GetMonitoringStats returns summary statistics for the monitoring dashboard
func (h *ProcessMonitorHandlers) GetMonitoringStats(w http.ResponseWriter, r *http.Request) {
	tenantID := r.URL.Query().Get("tenant_id")
	datasourceID := r.URL.Query().Get("datasource_id")

	if tenantID == "" || datasourceID == "" {
		http.Error(w, "Missing tenant_id or datasource_id", http.StatusBadRequest)
		return
	}

	query := `
		WITH latest_steps AS (
			SELECT DISTINCT ON (workflow_id)
				workflow_id,
				workflow_type,
				status,
				started_at
			FROM process_execution_metrics
			WHERE tenant_id = $1
				AND datasource_id = $2
				AND started_at > NOW() - INTERVAL '24 hours'
			ORDER BY workflow_id, started_at DESC
		)
		SELECT
			COUNT(*) as total_active,
			COUNT(*) FILTER (WHERE status = 'running') as running_count,
			COUNT(*) FILTER (WHERE status = 'completed') as completed_count,
			COUNT(*) FILTER (WHERE status = 'failed') as failed_count,
			COUNT(DISTINCT workflow_type) as workflow_types
		FROM latest_steps
	`

	var stats struct {
		TotalActive    int `json:"total_active" db:"total_active"`
		RunningCount   int `json:"running_count" db:"running_count"`
		CompletedCount int `json:"completed_count" db:"completed_count"`
		FailedCount    int `json:"failed_count" db:"failed_count"`
		WorkflowTypes  int `json:"workflow_types" db:"workflow_types"`
	}

	err := h.db.Get(&stats, query, tenantID, datasourceID)
	if err != nil {
		log.Printf("Error querying monitoring stats: %v", err)
		http.Error(w, "Failed to fetch stats", http.StatusInternalServerError)
		return
	}

	// Add WebSocket client count
	h.mu.RLock()
	connectedClients := len(h.clients)
	h.mu.RUnlock()

	response := map[string]interface{}{
		"stats":             stats,
		"connected_clients": connectedClients,
		"timestamp":         time.Now(),
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}
