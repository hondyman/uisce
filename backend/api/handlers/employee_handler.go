package handlers

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	"github.com/hondyman/semlayer/libs/jwt-middleware"
)

// ============================================================================
// Employee Handler for Dynamic UI Generator
// ============================================================================

type EmployeeHandler struct {
	db *sqlx.DB
}

func NewEmployeeHandler(db *sqlx.DB) *EmployeeHandler {
	return &EmployeeHandler{
		db: db,
	}
}

// ============================================================================
// Request/Response Types
// ============================================================================

type SaveEmployeeRequest struct {
	EmployeeID string  `json:"employee_id" binding:"required"`
	FirstName  string  `json:"first_name" binding:"required"`
	LastName   string  `json:"last_name" binding:"required"`
	Email      string  `json:"email" binding:"required"`
	Phone      *string `json:"phone"`
	HireDate   *string `json:"hire_date"`
	Department string  `json:"department" binding:"required"`
	Status     string  `json:"status"`
	IsVIP      bool    `json:"is_vip"`
	Salary     float64 `json:"salary" binding:"required"`
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

// ============================================================================
// Request Handlers
// ============================================================================

// SaveEmployee creates a new employee record
// POST /api/employees
func (h *EmployeeHandler) SaveEmployee(c *gin.Context) {
	// Extract tenant scoping headers
	claims := jwtmiddleware.GetGinClaimsFromContext(c)
	if claims == nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error":"unauthorized"})
		return
	}
	tenantID := claims.TenantID
	datasourceID := c.GetHeader("X-Tenant-Datasource-ID")

	if tenantID == "" || datasourceID == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Missing required tenant scoping headers: X-Tenant-ID and X-Tenant-Datasource-ID",
		})
		return
	}

	// Parse request
	var req SaveEmployeeRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Invalid request body: " + err.Error(),
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
		// In production, use proper logging
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
		c.JSON(http.StatusInternalServerError, gin.H{
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

	c.JSON(http.StatusCreated, response)
}

// GetEmployee retrieves an employee by ID
// GET /api/employees/:id
func (h *EmployeeHandler) GetEmployee(c *gin.Context) {
	// Extract tenant scoping headers
	claims := jwtmiddleware.GetGinClaimsFromContext(c)
	if claims == nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error":"unauthorized"})
		return
	}
	tenantID := claims.TenantID
	datasourceID := c.GetHeader("X-Tenant-Datasource-ID")

	if tenantID == "" || datasourceID == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Missing required tenant scoping headers",
		})
		return
	}

	empID := c.Param("id")

	var emp GetEmployeeResponse
	err := h.db.Get(&emp, `
		SELECT id, employee_id, first_name, last_name, email, phone, hire_date,
		       department, status, is_vip, salary, created_at, updated_at,
		       tenant_id, datasource_id
		FROM employees
		WHERE id = ? AND tenant_id = ? AND datasource_id = ?
	`, empID, tenantID, datasourceID)

	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{
			"error": "Employee not found",
		})
		return
	}

	c.JSON(http.StatusOK, emp)
}

// ListEmployees retrieves all employees for the tenant
// GET /api/employees
func (h *EmployeeHandler) ListEmployees(c *gin.Context) {
	// Extract tenant scoping headers
	claims := jwtmiddleware.GetGinClaimsFromContext(c)
	if claims == nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error":"unauthorized"})
		return
	}
	tenantID := claims.TenantID
	datasourceID := c.GetHeader("X-Tenant-Datasource-ID")

	if tenantID == "" || datasourceID == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Missing required tenant scoping headers",
		})
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
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to retrieve employees: " + err.Error(),
		})
		return
	}

	if employees == nil {
		employees = []GetEmployeeResponse{}
	}

	c.JSON(http.StatusOK, gin.H{
		"employees": employees,
		"count":     len(employees),
	})
}

// UpdateEmployee updates an existing employee record
// PUT /api/employees/:id
func (h *EmployeeHandler) UpdateEmployee(c *gin.Context) {
	// Extract tenant scoping headers
	claims := jwtmiddleware.GetGinClaimsFromContext(c)
	if claims == nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error":"unauthorized"})
		return
	}
	tenantID := claims.TenantID
	datasourceID := c.GetHeader("X-Tenant-Datasource-ID")

	if tenantID == "" || datasourceID == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Missing required tenant scoping headers",
		})
		return
	}

	empID := c.Param("id")

	// Verify employee exists and belongs to tenant
	var existingEmp GetEmployeeResponse
	err := h.db.Get(&existingEmp, `
		SELECT id FROM employees
		WHERE id = ? AND tenant_id = ? AND datasource_id = ?
	`, empID, tenantID, datasourceID)

	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{
			"error": "Employee not found",
		})
		return
	}

	// Parse request
	var req SaveEmployeeRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Invalid request body",
		})
		return
	}

	// Update employee
	updateSQL := `
	UPDATE employees
	SET first_name = ?, last_name = ?, email = ?, phone = ?,
		hire_date = ?, department = ?, status = ?, is_vip = ?,
		salary = ?, updated_at = ?
	WHERE id = ? AND tenant_id = ? AND datasource_id = ?
	`

	_, err = h.db.Exec(
		updateSQL,
		req.FirstName, req.LastName, req.Email, req.Phone,
		req.HireDate, req.Department, req.Status, req.IsVIP,
		req.Salary, time.Now(),
		empID, tenantID, datasourceID,
	)

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to update employee: " + err.Error(),
		})
		return
	}

	response := gin.H{
		"message": "Employee updated successfully",
		"id":      empID,
	}

	c.JSON(http.StatusOK, response)
}

// DeleteEmployee deletes an employee record
// DELETE /api/employees/:id
func (h *EmployeeHandler) DeleteEmployee(c *gin.Context) {
	// Extract tenant scoping headers
	claims := jwtmiddleware.GetGinClaimsFromContext(c)
	if claims == nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error":"unauthorized"})
		return
	}
	tenantID := claims.TenantID
	datasourceID := c.GetHeader("X-Tenant-Datasource-ID")

	if tenantID == "" || datasourceID == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Missing required tenant scoping headers",
		})
		return
	}

	empID := c.Param("id")

	result, err := h.db.Exec(`
		DELETE FROM employees
		WHERE id = ? AND tenant_id = ? AND datasource_id = ?
	`, empID, tenantID, datasourceID)

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to delete employee: " + err.Error(),
		})
		return
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil || rowsAffected == 0 {
		c.JSON(http.StatusNotFound, gin.H{
			"error": "Employee not found",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Employee deleted successfully",
		"id":      empID,
	})
}

// CheckEmailUniqueness checks if an email is unique
// POST /api/employees/validate/email
func (h *EmployeeHandler) CheckEmailUniqueness(c *gin.Context) {
	// Extract tenant scoping headers
	claims := jwtmiddleware.GetGinClaimsFromContext(c)
	if claims == nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error":"unauthorized"})
		return
	}
	tenantID := claims.TenantID
	datasourceID := c.GetHeader("X-Tenant-Datasource-ID")

	if tenantID == "" || datasourceID == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Missing required tenant scoping headers",
		})
		return
	}

	var req struct {
		Email string `json:"email" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Invalid request",
		})
		return
	}

	var count int
	err := h.db.Get(&count, `
		SELECT COUNT(*) FROM employees
		WHERE email = ? AND tenant_id = ? AND datasource_id = ?
	`, req.Email, tenantID, datasourceID)

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to validate email",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"email":  req.Email,
		"unique": count == 0,
	})
}
