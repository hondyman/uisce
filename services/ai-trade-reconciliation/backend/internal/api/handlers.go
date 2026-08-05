package api

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/hondyman/uisce/services/ai-trade-reconciliation/backend/internal/models"
)

// Handler holds API handlers
type Handler struct {
	db *sql.DB
}

// NewHandler creates a new API handler
func NewHandler(db *sql.DB) *Handler {
	return &Handler{db: db}
}

// GetReconciliationResults returns latest reconciliation results
func (h *Handler) GetReconciliationResults(c *gin.Context) {
	limit := 10
	offset := 0

	rows, err := h.db.Query(`
		SELECT id, run_date, match_rate, matched_count, unmatched_count, discrepancies, model_version, status, error_message, created_at, updated_at
		FROM reconciliation_results
		ORDER BY run_date DESC
		LIMIT $1 OFFSET $2
	`, limit, offset)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("Query failed: %v", err)})
		return
	}
	defer rows.Close()

	var results []models.ReconciliationResult
	for rows.Next() {
		var r models.ReconciliationResult
		if err := rows.Scan(&r.ID, &r.RunDate, &r.MatchRate, &r.MatchedCount, &r.UnmatchedCount,
			&r.DiscrepancyJSON, &r.ModelVersion, &r.Status, &r.ErrorMessage, &r.CreatedAt, &r.UpdatedAt); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("Scan failed: %v", err)})
			return
		}
		results = append(results, r)
	}

	c.JSON(http.StatusOK, results)
}

// GetLatestResult returns the most recent reconciliation result
func (h *Handler) GetLatestResult(c *gin.Context) {
	var r models.ReconciliationResult

	err := h.db.QueryRow(`
		SELECT id, run_date, match_rate, matched_count, unmatched_count, discrepancies, model_version, status, error_message, created_at, updated_at
		FROM reconciliation_results
		ORDER BY run_date DESC
		LIMIT 1
	`).Scan(&r.ID, &r.RunDate, &r.MatchRate, &r.MatchedCount, &r.UnmatchedCount,
		&r.DiscrepancyJSON, &r.ModelVersion, &r.Status, &r.ErrorMessage, &r.CreatedAt, &r.UpdatedAt)

	if err == sql.ErrNoRows {
		c.JSON(http.StatusNotFound, gin.H{"error": "No reconciliation results found"})
		return
	}
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("Query failed: %v", err)})
		return
	}

	c.JSON(http.StatusOK, r)
}

// GetDiscrepancies returns discrepancies for a result
func (h *Handler) GetDiscrepancies(c *gin.Context) {
	resultID := c.Param("result_id")

	rows, err := h.db.Query(`
		SELECT id, result_id, trade_id, confirm_id, discrepancy_type, field, trade_value, confirm_value, severity, suggested_fix, created_at
		FROM discrepancies
		WHERE result_id = $1
		ORDER BY severity DESC, created_at DESC
	`, resultID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("Query failed: %v", err)})
		return
	}
	defer rows.Close()

	var discrepancies []models.Discrepancy
	for rows.Next() {
		var d models.Discrepancy
		if err := rows.Scan(&d.ID, &d.ResultID, &d.TradeID, &d.ConfirmID, &d.DiscrepType, &d.Field,
			&d.TradeValue, &d.ConfirmValue, &d.Severity, &d.SuggestedFix, &d.CreatedAt); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("Scan failed: %v", err)})
			return
		}
		discrepancies = append(discrepancies, d)
	}

	c.JSON(http.StatusOK, discrepancies)
}

// GetOpenTasks returns open reconciliation tasks
func (h *Handler) GetOpenTasks(c *gin.Context) {
	rows, err := h.db.Query(`
		SELECT id, result_id, discrepancy_id, status, assigned_to, priority, notes, resolved_at, created_at, updated_at
		FROM reconciliation_tasks
		WHERE status != 'resolved'
		ORDER BY priority DESC, created_at DESC
	`)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("Query failed: %v", err)})
		return
	}
	defer rows.Close()

	var tasks []models.ReconciliationTask
	for rows.Next() {
		var t models.ReconciliationTask
		if err := rows.Scan(&t.ID, &t.ResultID, &t.DiscrepancyID, &t.Status, &t.AssignedTo,
			&t.Priority, &t.Notes, &t.ResolvedAt, &t.CreatedAt, &t.UpdatedAt); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("Scan failed: %v", err)})
			return
		}
		tasks = append(tasks, t)
	}

	c.JSON(http.StatusOK, tasks)
}

// UpdateTask updates a reconciliation task
func (h *Handler) UpdateTask(c *gin.Context) {
	taskID := c.Param("task_id")

	var req struct {
		Status   string `json:"status"`
		Notes    string `json:"notes"`
		Priority string `json:"priority"`
	}

	if err := c.BindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": fmt.Sprintf("Invalid request: %v", err)})
		return
	}

	now := time.Now()
	var resolvedAt *time.Time
	if req.Status == "resolved" {
		resolvedAt = &now
	}

	_, err := h.db.Exec(`
		UPDATE reconciliation_tasks
		SET status = $1, notes = $2, priority = $3, resolved_at = $4, updated_at = $5
		WHERE id = $6
	`, req.Status, req.Notes, req.Priority, resolvedAt, now, taskID)

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("Update failed: %v", err)})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Task updated"})
}

// GetRules returns all reconciliation rules
func (h *Handler) GetRules(c *gin.Context) {
	rows, err := h.db.Query(`
		SELECT id, name, description, rule_type, enabled, rule_expr, version, created_at, updated_at
		FROM reconciliation_rules
		ORDER BY rule_type, created_at DESC
	`)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("Query failed: %v", err)})
		return
	}
	defer rows.Close()

	var rules []models.ReconciliationRule
	for rows.Next() {
		var r models.ReconciliationRule
		if err := rows.Scan(&r.ID, &r.Name, &r.Description, &r.RuleType, &r.Enabled, &r.RuleExpr, &r.Version, &r.CreatedAt, &r.UpdatedAt); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("Scan failed: %v", err)})
			return
		}
		rules = append(rules, r)
	}

	c.JSON(http.StatusOK, rules)
}

// CreateRule creates a new reconciliation rule
func (h *Handler) CreateRule(c *gin.Context) {
	var rule models.ReconciliationRule
	if err := c.BindJSON(&rule); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": fmt.Sprintf("Invalid request: %v", err)})
		return
	}

	rule.ID = uuid.New()
	rule.CreatedAt = time.Now()
	rule.UpdatedAt = time.Now()
	rule.Version = 1

	_, err := h.db.Exec(`
		INSERT INTO reconciliation_rules
			(id, name, description, rule_type, enabled, rule_expr, version, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
	`, rule.ID, rule.Name, rule.Description, rule.RuleType, rule.Enabled, rule.RuleExpr, rule.Version, rule.CreatedAt, rule.UpdatedAt)

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("Insert failed: %v", err)})
		return
	}

	c.JSON(http.StatusCreated, rule)
}

// GenerateReport generates a PDF reconciliation report
func (h *Handler) GenerateReport(c *gin.Context) {
	resultID := c.Param("result_id")

	var r models.ReconciliationResult
	err := h.db.QueryRow(`
		SELECT id, run_date, match_rate, matched_count, unmatched_count, discrepancies, model_version, status, error_message, created_at, updated_at
		FROM reconciliation_results
		WHERE id = $1
	`, resultID).Scan(&r.ID, &r.RunDate, &r.MatchRate, &r.MatchedCount, &r.UnmatchedCount,
		&r.DiscrepancyJSON, &r.ModelVersion, &r.Status, &r.ErrorMessage, &r.CreatedAt, &r.UpdatedAt)

	if err == sql.ErrNoRows {
		c.JSON(http.StatusNotFound, gin.H{"error": "Result not found"})
		return
	}
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("Query failed: %v", err)})
		return
	}

	// TODO: Generate PDF report
	c.JSON(http.StatusOK, gin.H{"message": "Report generation placeholder"})
}

// RegisterRoutes registers all API routes
func RegisterRoutes(router *gin.Engine, handler *Handler) {
	api := router.Group("/api/reconciliation")
	{
		api.GET("/results", handler.GetReconciliationResults)
		api.GET("/results/latest", handler.GetLatestResult)
		api.GET("/results/:result_id/discrepancies", handler.GetDiscrepancies)
		api.GET("/results/:result_id/report", handler.GenerateReport)

		api.GET("/tasks", handler.GetOpenTasks)
		api.PUT("/tasks/:task_id", handler.UpdateTask)

		api.GET("/rules", handler.GetRules)
		api.POST("/rules", handler.CreateRule)
	}
}
