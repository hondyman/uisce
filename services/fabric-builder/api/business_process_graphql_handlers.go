package api

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
)

type BusinessProcessGraphQLHandlers struct {
	db *sql.DB
}

func NewBusinessProcessGraphQLHandlers(db *sql.DB) *BusinessProcessGraphQLHandlers {
	return &BusinessProcessGraphQLHandlers{db: db}
}

func (h *BusinessProcessGraphQLHandlers) ListBusinessProcessesGraphQL(w http.ResponseWriter, r *http.Request) {
	tenantID := r.URL.Query().Get("tenant_id")

	query := `
		SELECT id, tenant_id, process_name, description, entity_type,
		       status, is_active, version_number, created_at, updated_at, created_by
		FROM business_processes
		WHERE ($1 = '' OR tenant_id::text = $1)
		  AND is_active = true
		ORDER BY created_at DESC
	`

	rows, err := h.db.Query(query, tenantID)
	if err != nil {
		http.Error(w, fmt.Sprintf("database query failed: %v", err), http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	type stepResult struct {
		ID            string
		StepOrder     int
		StepType      string
		StepName      string
		Description   *string
		DurationHours float64
		Status        *string
	}

	processesMap := make(map[string]BusinessProcess)

	for rows.Next() {
		var p BusinessProcess
		var updatedAt sql.NullString
		var entityType, status sql.NullString
		var versionNumber int

		err := rows.Scan(&p.ID, &p.TenantID, &p.ProcessName, &p.Description,
			&entityType, &status, &p.IsActive, &versionNumber, &p.CreatedAt, &updatedAt, &p.CreatedBy)
		if err != nil {
			http.Error(w, fmt.Sprintf("scan failed: %v", err), http.StatusInternalServerError)
			return
		}
		if entityType.Valid {
			p.Entity = entityType.String
		}
		if updatedAt.Valid {
			p.UpdatedAt = &updatedAt.String
		}
		p.Version = versionNumber
		processesMap[p.ID] = p
	}

	if len(processesMap) > 0 {
		stepQuery := `
			SELECT id, business_process_id, step_order, step_type, step_name,
			       description, duration_hours, status
			FROM bp_steps
			WHERE business_process_id = ANY($1)
			ORDER BY step_order
		`
		ids := make([]string, 0, len(processesMap))
		for id := range processesMap {
			ids = append(ids, id)
		}

		stepRows, err := h.db.Query(stepQuery, ids)
		if err != nil {
			http.Error(w, fmt.Sprintf("step query failed: %v", err), http.StatusInternalServerError)
			return
		}
		defer stepRows.Close()

		for stepRows.Next() {
			var s BPStep
			var businessProcessID string
			var durationHours sql.NullFloat64
			var desc, status *string

			err := stepRows.Scan(&s.ID, &businessProcessID, &s.StepOrder, &s.StepType,
				&s.StepName, &desc, &durationHours, &status)
			if err != nil {
				http.Error(w, fmt.Sprintf("step scan failed: %v", err), http.StatusInternalServerError)
				return
			}
			s.Description = desc
			s.Status = status
			if durationHours.Valid {
				s.DurationHours = durationHours.Float64
			}
			if p, ok := processesMap[businessProcessID]; ok {
				p.Steps = append(p.Steps, s)
				processesMap[businessProcessID] = p
			}
		}
	}

	processes := make([]BusinessProcess, 0, len(processesMap))
	for _, p := range processesMap {
		processes = append(processes, p)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(processes)
}

func (h *BusinessProcessGraphQLHandlers) CreateBusinessProcessGraphQL(w http.ResponseWriter, r *http.Request) {
	var input BusinessProcess
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}

	if input.ID == "" {
		input.ID = uuid.New().String()
	}

	tx, err := h.db.Begin()
	if err != nil {
		http.Error(w, fmt.Sprintf("transaction failed: %v", err), http.StatusInternalServerError)
		return
	}
	defer tx.Rollback()

	mutation := `
		INSERT INTO business_processes
		(id, tenant_id, process_name, description, entity_type, status, is_active, created_by)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		RETURNING created_at, updated_at, version_number
	`

	var createdAt, updatedAt string
	var versionNumber int
	err = tx.QueryRow(mutation, input.ID, input.TenantID, input.ProcessName,
		input.Description, input.Entity, "active", input.IsActive, input.CreatedBy).
		Scan(&createdAt, &updatedAt, &versionNumber)
	if err != nil {
		http.Error(w, fmt.Sprintf("insert failed: %v", err), http.StatusInternalServerError)
		return
	}

	for i, step := range input.Steps {
		if step.ID == "" {
			step.ID = uuid.New().String()
		}

		stepMutation := `
			INSERT INTO bp_steps
			(id, business_process_id, tenant_id, step_order, step_type, step_name,
			 description, duration_hours)
			VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		`

		_, err = tx.Exec(stepMutation, step.ID, input.ID, input.TenantID,
			i+1, step.StepType, step.StepName, step.Description, step.DurationHours)
		if err != nil {
			fmt.Printf("Warning: failed to create step %d: %v\n", i+1, err)
		}
		input.Steps[i].ID = step.ID
	}

	if err = tx.Commit(); err != nil {
		http.Error(w, fmt.Sprintf("commit failed: %v", err), http.StatusInternalServerError)
		return
	}

	input.CreatedAt = createdAt
	input.UpdatedAt = &updatedAt
	input.Version = versionNumber

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(input)
}

func (h *BusinessProcessGraphQLHandlers) GetBusinessProcessGraphQL(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")

	query := `
		SELECT id, tenant_id, process_name, description, entity_type,
		       status, is_active, version_number, created_at, updated_at, created_by
		FROM business_processes
		WHERE id = $1
	`

	var p BusinessProcess
	var updatedAt sql.NullString
	var entityType, status sql.NullString
	var versionNumber int

	err := h.db.QueryRow(query, id).Scan(&p.ID, &p.TenantID, &p.ProcessName, &p.Description,
		&entityType, &status, &p.IsActive, &versionNumber, &p.CreatedAt, &updatedAt, &p.CreatedBy)
	if err == sql.ErrNoRows {
		http.Error(w, "process not found", http.StatusNotFound)
		return
	} else if err != nil {
		http.Error(w, fmt.Sprintf("query failed: %v", err), http.StatusInternalServerError)
		return
	}
	if entityType.Valid {
		p.Entity = entityType.String
	}
	if updatedAt.Valid {
		p.UpdatedAt = &updatedAt.String
	}
	p.Version = versionNumber

	stepQuery := `
		SELECT id, step_order, step_type, step_name, description, duration_hours, status
		FROM bp_steps
		WHERE business_process_id = $1
		ORDER BY step_order
	`
	rows, err := h.db.Query(stepQuery, id)
	if err != nil {
		http.Error(w, fmt.Sprintf("step query failed: %v", err), http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	for rows.Next() {
		var s BPStep
		var durationHours sql.NullFloat64

		err := rows.Scan(&s.ID, &s.StepOrder, &s.StepType, &s.StepName,
			&s.Description, &durationHours, &s.Status)
		if err != nil {
			http.Error(w, fmt.Sprintf("step scan failed: %v", err), http.StatusInternalServerError)
			return
		}
		if durationHours.Valid {
			s.DurationHours = durationHours.Float64
		}
		p.Steps = append(p.Steps, s)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(p)
}

func (h *BusinessProcessGraphQLHandlers) UpdateBusinessProcessGraphQL(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")

	var updates map[string]interface{}
	if err := json.NewDecoder(r.Body).Decode(&updates); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}

	setClause := ""
	args := make([]interface{}, 0)
	argCount := 1

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
			if setClause != "" {
				setClause += ", "
			}
			setClause += key + " = $" + fmt.Sprintf("%d", argCount)
			args = append(args, value)
			argCount++
		}
	}

	if setClause == "" {
		http.Error(w, "no valid fields to update", http.StatusBadRequest)
		return
	}

	query := fmt.Sprintf(`
		UPDATE business_processes
		SET %s, updated_at = NOW()
		WHERE id = $%d
		RETURNING updated_at
	`, setClause, argCount)

	args = append(args, id)

	var updatedAt string
	err := h.db.QueryRow(query, args...).Scan(&updatedAt)
	if err != nil {
		http.Error(w, fmt.Sprintf("update failed: %v", err), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"id":         id,
		"updated":    true,
		"updated_at": updatedAt,
	})
}

func (h *BusinessProcessGraphQLHandlers) DeleteBusinessProcessGraphQL(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")

	query := `DELETE FROM business_processes WHERE id = $1`
	_, err := h.db.Exec(query, id)
	if err != nil {
		http.Error(w, fmt.Sprintf("delete failed: %v", err), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"id":      id,
		"deleted": true,
	})
}

func (h *BusinessProcessGraphQLHandlers) ExecuteBusinessProcessGraphQL(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")

	var tenantID string
	err := h.db.QueryRow(`SELECT tenant_id FROM business_processes WHERE id = $1`, id).Scan(&tenantID)
	if err == sql.ErrNoRows {
		http.Error(w, "process not found", http.StatusNotFound)
		return
	} else if err != nil {
		http.Error(w, fmt.Sprintf("query failed: %v", err), http.StatusInternalServerError)
		return
	}

	executionID := uuid.New().String()
	mutation := `
		INSERT INTO bp_executions
		(id, tenant_id, business_process_id, entity_id, initiated_by, execution_status, current_step_order)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
		RETURNING initiated_at
	`

	var initiatedAt string
	err = h.db.QueryRow(mutation, executionID, tenantID, id, uuid.New().String(), "system", "running", 1).Scan(&initiatedAt)
	if err != nil {
		http.Error(w, fmt.Sprintf("insert failed: %v", err), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"id":           id,
		"execution_id": executionID,
		"status":       "started",
		"started_at":   initiatedAt,
	})
}

func RegisterBusinessProcessGraphQLRoutes(r chi.Router, db *sql.DB) {
	h := NewBusinessProcessGraphQLHandlers(db)

	r.Route("/api/business-process/v2", func(r chi.Router) {
		r.Get("/", h.ListBusinessProcessesGraphQL)
		r.Post("/", h.CreateBusinessProcessGraphQL)
		r.Get("/{id}", h.GetBusinessProcessGraphQL)
		r.Put("/{id}", h.UpdateBusinessProcessGraphQL)
		r.Delete("/{id}", h.DeleteBusinessProcessGraphQL)
		r.Post("/{id}/execute", h.ExecuteBusinessProcessGraphQL)

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
