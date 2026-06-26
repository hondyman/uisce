package api

import (
	"database/sql"
	"encoding/json"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
)

// BusinessProcess represents a complete business process workflow
type BusinessProcess struct {
	ID           string   `json:"id"`
	TenantID     string   `json:"tenant_id"`
	DatasourceID string   `json:"datasource_id"`
	ProcessName  string   `json:"processName"`
	Entity       string   `json:"entity"`
	Description  string   `json:"description"`
	Steps        []BPStep `json:"steps"`
	IsActive     bool     `json:"isActive"`
	CreatedBy    string   `json:"createdBy"`
	CreatedAt    string   `json:"createdAt"`
	UpdatedAt    *string  `json:"updatedAt,omitempty"`
	Version      int      `json:"version"`
	Tags         []string `json:"tags"`
}

// BPStep represents a single step in a business process
type BPStep struct {
	ID                       string           `json:"id"`
	StepOrder                int              `json:"stepOrder"`
	StepType                 string           `json:"stepType"`
	StepName                 string           `json:"stepName"`
	DurationHours            float64          `json:"durationHours"`
	AssigneeRole             *string          `json:"assigneeRole,omitempty"`
	AssigneeUser             *string          `json:"assigneeUser,omitempty"`
	ValidationRules          []string         `json:"validationRules,omitempty"`
	NotificationTemplate     *string          `json:"notificationTemplate,omitempty"`
	ConditionLogic           *ConditionBranch `json:"conditionLogic,omitempty"`
	Description              *string          `json:"description,omitempty"`
	Status                   *string          `json:"status,omitempty"`
	EscalationThresholdHours *float64         `json:"escalationThresholdHours,omitempty"`
}

// ConditionBranch represents conditional branching logic
type ConditionBranch struct {
	Condition   string `json:"condition"`
	TrueStepID  string `json:"trueStepId"`
	FalseStepID string `json:"falseStepId"`
}

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

// ListBusinessProcessesHandler returns all business processes with database query
func ListBusinessProcessesHandler(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		tenantID := r.URL.Query().Get("tenant_id")
		datasourceID := r.URL.Query().Get("datasource_id")

		query := `
SELECT id, tenant_id, process_name, description, process_type, 
       is_active, version, created_at, updated_at, created_by
FROM business_processes
WHERE ($1 = '' OR tenant_id::text = $1)
  AND is_active = true
ORDER BY created_at DESC
`

		rows, err := db.Query(query, tenantID)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		defer rows.Close()

		var processes []BusinessProcess
		for rows.Next() {
			var p BusinessProcess
			var updatedAt sql.NullString
			err := rows.Scan(&p.ID, &p.TenantID, &p.ProcessName, &p.Description,
				&p.Entity, &p.IsActive, &p.Version, &p.CreatedAt, &updatedAt, &p.CreatedBy)
			if err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
			if updatedAt.Valid {
				p.UpdatedAt = &updatedAt.String
			}
			p.DatasourceID = datasourceID
			processes = append(processes, p)
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(processes)
	}
}

// CreateBusinessProcessHandler creates a new business process with database insertion
func CreateBusinessProcessHandler(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var process BusinessProcess
		if err := json.NewDecoder(r.Body).Decode(&process); err != nil {
			http.Error(w, "invalid request body", http.StatusBadRequest)
			return
		}

		// Begin transaction
		tx, err := db.Begin()
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		defer tx.Rollback()

		// Generate ID if not provided
		if process.ID == "" {
			process.ID = uuid.New().String()
		}

		// Insert business process
		query := `
INSERT INTO business_processes 
(id, tenant_id, process_name, description, process_type, is_active, created_by)
VALUES ($1, $2, $3, $4, $5, $6, $7)
RETURNING created_at, updated_at
`
		err = tx.QueryRow(query, process.ID, process.TenantID, process.ProcessName,
			process.Description, process.Entity, process.IsActive, process.CreatedBy).
			Scan(&process.CreatedAt, &process.UpdatedAt)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		// Insert steps if provided
		for i, step := range process.Steps {
			if step.ID == "" {
				step.ID = uuid.New().String()
			}

			stepQuery := `
INSERT INTO bp_steps
(id, process_id, tenant_id, step_order, step_type, step_name, 
 duration_hours, assignee_role, assignee_user, description)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
`
			_, err = tx.Exec(stepQuery, step.ID, process.ID, process.TenantID,
				i+1, step.StepType, step.StepName, step.DurationHours,
				step.AssigneeRole, step.AssigneeUser, step.Description)
			if err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
			process.Steps[i].ID = step.ID
		}

		// Commit transaction
		if err = tx.Commit(); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(process)
	}
}

// GetBusinessProcessHandler returns a specific business process with steps
func GetBusinessProcessHandler(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id := chi.URLParam(r, "id")

		// Get process
		query := `
SELECT id, tenant_id, process_name, description, process_type, 
       is_active, version, created_at, updated_at, created_by
FROM business_processes
WHERE id = $1
`
		var process BusinessProcess
		var updatedAt sql.NullString
		err := db.QueryRow(query, id).Scan(&process.ID, &process.TenantID,
			&process.ProcessName, &process.Description, &process.Entity,
			&process.IsActive, &process.Version, &process.CreatedAt, &updatedAt, &process.CreatedBy)
		if err == sql.ErrNoRows {
			http.Error(w, "process not found", http.StatusNotFound)
			return
		} else if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		if updatedAt.Valid {
			process.UpdatedAt = &updatedAt.String
		}

		// Get steps
		stepQuery := `
SELECT id, step_order, step_type, step_name, duration_hours,
       assignee_role, assignee_user, description
FROM bp_steps
WHERE process_id = $1
ORDER BY step_order
`
		rows, err := db.Query(stepQuery, id)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		defer rows.Close()

		for rows.Next() {
			var step BPStep
			var durationHours sql.NullFloat64
			err := rows.Scan(&step.ID, &step.StepOrder, &step.StepType, &step.StepName,
				&durationHours, &step.AssigneeRole, &step.AssigneeUser, &step.Description)
			if err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
			if durationHours.Valid {
				step.DurationHours = durationHours.Float64
			}
			process.Steps = append(process.Steps, step)
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(process)
	}
}

// UpdateBusinessProcessHandler updates a business process
func UpdateBusinessProcessHandler(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id := chi.URLParam(r, "id")

		var updates map[string]interface{}
		if err := json.NewDecoder(r.Body).Decode(&updates); err != nil {
			http.Error(w, "invalid request body", http.StatusBadRequest)
			return
		}

		query := `
UPDATE business_processes
SET updated_at = NOW()
WHERE id = $1
RETURNING updated_at
`
		var updatedAt string
		err := db.QueryRow(query, id).Scan(&updatedAt)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"id":         id,
			"updated":    true,
			"updated_at": updatedAt,
		})
	}
}

// DeleteBusinessProcessHandler deletes a business process
func DeleteBusinessProcessHandler(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id := chi.URLParam(r, "id")

		query := `DELETE FROM business_processes WHERE id = $1`
		_, err := db.Exec(query, id)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"id":      id,
			"deleted": true,
		})
	}
}

// ExecuteBusinessProcessHandler starts execution of a business process
func ExecuteBusinessProcessHandler(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id := chi.URLParam(r, "id")

		// Create instance
		instanceID := uuid.New().String()
		query := `
INSERT INTO bp_instances
(id, tenant_id, process_id, entity_id, entity_type, current_step, status, started_at)
SELECT $1, tenant_id, $2, $3, $4, 1, 'in_progress', NOW()
FROM business_processes WHERE id = $2
RETURNING started_at
`
		var startedAt string
		err := db.QueryRow(query, instanceID, id, "entity-1", "default").Scan(&startedAt)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"id":           id,
			"execution_id": instanceID,
			"status":       "started",
			"started_at":   startedAt,
		})
	}
}

// GetBusinessProcessStatusHandler returns the execution status of a business process
func GetBusinessProcessStatusHandler(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id := chi.URLParam(r, "id")

		query := `
SELECT bi.status, bi.current_step, bs.step_name
FROM bp_instances bi
LEFT JOIN bp_steps bs ON bs.process_id = bi.process_id AND bs.step_order = bi.current_step
WHERE bi.process_id = $1
ORDER BY bi.created_at DESC
LIMIT 1
`
		var status string
		var currentStep int
		var stepName sql.NullString
		err := db.QueryRow(query, id).Scan(&status, &currentStep, &stepName)
		if err == sql.ErrNoRows {
			http.Error(w, "no execution found", http.StatusNotFound)
			return
		} else if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		currentStepName := ""
		if stepName.Valid {
			currentStepName = stepName.String
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"id":           id,
			"status":       status,
			"current_step": currentStepName,
			"progress":     currentStep * 25,
		})
	}
}

// GetStepTypesHandler returns available step types for the designer
func GetStepTypesHandler(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
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
	}
}

// GetValidationOperatorsHandler returns available validation operators
func GetValidationOperatorsHandler(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
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
	}
}

// GetWorkflowEventsHandler returns available workflow trigger events
func GetWorkflowEventsHandler(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
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
	}
}
