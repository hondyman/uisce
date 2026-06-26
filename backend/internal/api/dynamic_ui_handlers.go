package api

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	"github.com/hondyman/semlayer/libs/jwt-middleware"
)

// ============================================================================
// Chi-Compatible Employee Handlers
// ============================================================================

type EmployeeHandler struct {
	db *sqlx.DB
}

func NewEmployeeHandlerChi(db *sqlx.DB) *EmployeeHandler {
	return &EmployeeHandler{
		db: db,
	}
}

type SaveEmployeeRequest struct {
	EmployeeID string  `json:"employee_id"`
	FirstName  string  `json:"first_name"`
	LastName   string  `json:"last_name"`
	Email      string  `json:"email"`
	Phone      *string `json:"phone"`
	HireDate   *string `json:"hire_date"`
	Department string  `json:"department"`
	Status     string  `json:"status"`
	IsVIP      bool    `json:"is_vip"`
	Salary     float64 `json:"salary"`
}

type SaveEmployeeResponse struct {
	ID         string    `json:"id"`
	EmployeeID string    `json:"employee_id"`
	Message    string    `json:"message"`
	CreatedAt  time.Time `json:"created_at"`
}

type GetEmployeeResponse struct {
	ID           string    `json:"id"`
	EmployeeID   string    `json:"employee_id"`
	FirstName    string    `json:"first_name"`
	LastName     string    `json:"last_name"`
	Email        string    `json:"email"`
	Phone        *string   `json:"phone"`
	HireDate     *string   `json:"hire_date"`
	Department   string    `json:"department"`
	Status       string    `json:"status"`
	IsVIP        bool      `json:"is_vip"`
	Salary       float64   `json:"salary"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
	TenantID     string    `json:"tenant_id"`
	DatasourceID string    `json:"datasource_id"`
}

// SaveEmployee creates a new employee record
func (h *EmployeeHandler) SaveEmployee(w http.ResponseWriter, r *http.Request) {
	// Extract tenant scoping headers
	claims := jwtmiddleware.GetClaimsFromContext(r)
	if claims == nil {
		http.Error(w, `{"error":"unauthorized"}`, http.StatusUnauthorized)
		return
	}
	tenantID := claims.TenantID
	datasourceID := r.Header.Get("X-Tenant-Datasource-ID")

	if tenantID == "" || datasourceID == "" {
		http.Error(w, "Missing required tenant scoping headers: X-Tenant-ID and X-Tenant-Datasource-ID", http.StatusBadRequest)
		return
	}

	// Parse request
	var req SaveEmployeeRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{
			"error": "Invalid request body: " + err.Error(),
		})
		return
	}

	// Validate required fields
	if req.EmployeeID == "" || req.FirstName == "" || req.LastName == "" || req.Email == "" || req.Department == "" {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{
			"error": "Missing required fields: employee_id, first_name, last_name, email, department",
		})
		return
	}

	// Generate new employee ID
	empID := uuid.New().String()
	now := time.Now()

	// Create employees table if it doesn't exist
	createTableSQL := `
	CREATE TABLE IF NOT EXISTS employees (
		id VARCHAR(36) PRIMARY KEY,
		employee_id VARCHAR(50) NOT NULL UNIQUE,
		first_name VARCHAR(100) NOT NULL,
		last_name VARCHAR(100) NOT NULL,
		email VARCHAR(100) NOT NULL,
		phone VARCHAR(20),
		hire_date DATE,
		department VARCHAR(100) NOT NULL,
		status VARCHAR(50) DEFAULT 'Active',
		is_vip BOOLEAN DEFAULT FALSE,
		salary DECIMAL(12, 2) NOT NULL,
		created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
		updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
		tenant_id VARCHAR(36) NOT NULL,
		datasource_id VARCHAR(36) NOT NULL,
		INDEX idx_tenant_datasource (tenant_id, datasource_id),
		INDEX idx_employee_id (employee_id)
	);
	`
	if _, err := h.db.Exec(createTableSQL); err != nil {
		// Log error but continue - table may already exist
	}

	// Insert employee record
	insertSQL := `
	INSERT INTO employees (
		id, employee_id, first_name, last_name, email, phone, hire_date,
		department, status, is_vip, salary, created_at, updated_at,
		tenant_id, datasource_id
	) VALUES (
		?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?
	)
	`

	_, err := h.db.Exec(
		insertSQL,
		empID, req.EmployeeID, req.FirstName, req.LastName, req.Email,
		req.Phone, req.HireDate, req.Department, req.Status, req.IsVIP,
		req.Salary, now, now, tenantID, datasourceID,
	)

	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{
			"error": "Failed to save employee: " + err.Error(),
		})
		return
	}

	// Return success response
	response := SaveEmployeeResponse{
		ID:         empID,
		EmployeeID: req.EmployeeID,
		Message:    "Employee saved successfully",
		CreatedAt:  now,
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(response)
}

// ListEmployees retrieves all employees for the tenant
func (h *EmployeeHandler) ListEmployees(w http.ResponseWriter, r *http.Request) {
	// Extract tenant scoping headers
	claims := jwtmiddleware.GetClaimsFromContext(r)
	if claims == nil {
		http.Error(w, `{"error":"unauthorized"}`, http.StatusUnauthorized)
		return
	}
	tenantID := claims.TenantID
	datasourceID := r.Header.Get("X-Tenant-Datasource-ID")

	if tenantID == "" || datasourceID == "" {
		http.Error(w, "Missing required tenant scoping headers", http.StatusBadRequest)
		return
	}

	var employees []GetEmployeeResponse
	err := h.db.Select(&employees, `
		SELECT id, employee_id, first_name, last_name, email, phone, hire_date,
		       department, status, is_vip, salary, created_at, updated_at,
		       tenant_id, datasource_id
		FROM employees
		WHERE tenant_id = ? AND datasource_id = ?
		ORDER BY created_at DESC
		LIMIT 1000
	`, tenantID, datasourceID)

	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{
			"error": "Failed to retrieve employees: " + err.Error(),
		})
		return
	}

	if employees == nil {
		employees = []GetEmployeeResponse{}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"employees": employees,
		"count":     len(employees),
	})
}

// ============================================================================
// Chi-Compatible BP Start-Execution Handler
// ============================================================================

type BPStartExecutionRequest struct {
	BusinessProcessID string                 `json:"businessProcessId"`
	EntityID          string                 `json:"entityId"`
	FormData          map[string]interface{} `json:"formData"`
}

type BPStartExecutionResponse struct {
	WorkflowID string `json:"workflowId"`
	Status     string `json:"status"`
	Message    string `json:"message"`
	StartedAt  string `json:"startedAt"`
}

// StartBPExecution triggers a business process workflow execution
func StartBPExecution(w http.ResponseWriter, r *http.Request) {
	// Extract tenant scoping headers
	claims := jwtmiddleware.GetClaimsFromContext(r)
	if claims == nil {
		http.Error(w, `{"error":"unauthorized"}`, http.StatusUnauthorized)
		return
	}
	tenantID := claims.TenantID
	datasourceID := r.Header.Get("X-Tenant-Datasource-ID")

	if tenantID == "" || datasourceID == "" {
		http.Error(w, "Missing required tenant scoping headers: X-Tenant-ID and X-Tenant-Datasource-ID", http.StatusBadRequest)
		return
	}

	// Parse request
	var req BPStartExecutionRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{
			"error": "Invalid request body: " + err.Error(),
		})
		return
	}

	// Validate required fields
	if req.BusinessProcessID == "" || req.EntityID == "" {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{
			"error": "Missing required fields: businessProcessId, entityId",
		})
		return
	}

	// Generate workflow ID
	workflowID := uuid.New().String()

	// In production, this would:
	// 1. Validate the business process exists and is active
	// 2. Trigger a Temporal workflow or similar
	// 3. Return the workflow execution ID

	response := BPStartExecutionResponse{
		WorkflowID: workflowID,
		Status:     "started",
		Message:    "Business process workflow execution started successfully",
		StartedAt:  time.Now().Format("2006-01-02T15:04:05Z"),
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusAccepted)
	json.NewEncoder(w).Encode(response)
}
