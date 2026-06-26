package api

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/hondyman/semlayer/services/ai-trade-reconciliation/backend/internal/models"
)

// HasuraClient defines the interface for Hasura GraphQL operations
type HasuraClient interface {
	Query(query string, variables map[string]interface{}) (map[string]interface{}, error)
	Mutate(mutation string, variables map[string]interface{}) (map[string]interface{}, error)
}

// Handler holds API handlers
type Handler struct {
	db     *sql.DB
	hasura HasuraClient
}

// NewHandler creates a new API handler
func NewHandler(db *sql.DB) *Handler {
	return &Handler{db: db}
}

// NewHandlerWithHasura creates a Hasura-enabled API handler
func NewHandlerWithHasura(db *sql.DB, hasura HasuraClient) *Handler {
	return &Handler{db: db, hasura: hasura}
}

// GetReconciliationResults returns latest reconciliation results
func (h *Handler) GetReconciliationResults(c *gin.Context) {
	if h.hasura != nil {
		h.getReconciliationResultsWithHasura(c)
		return
	}

	limit := 10
	offset := 0

	// Query database
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

func (h *Handler) getReconciliationResultsWithHasura(c *gin.Context) {
	query := `
		query GetReconciliationResults {
			reconciliation_results(
				order_by: [{run_date: desc}]
				limit: 10
			) {
				id
				run_date
				match_rate
				matched_count
				unmatched_count
				discrepancies
				model_version
				status
				error_message
				created_at
				updated_at
			}
		}
	`

	result, err := h.hasura.Query(query, nil)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("Query failed: %v", err)})
		return
	}

	resultsData, ok := result["reconciliation_results"].([]interface{})
	if !ok {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "unexpected response format"})
		return
	}

	var results []models.ReconciliationResult
	for _, item := range resultsData {
		resultMap, ok := item.(map[string]interface{})
		if !ok {
			continue
		}

		r := models.ReconciliationResult{}
		if id, ok := resultMap["id"].(string); ok {
			r.ID, _ = uuid.Parse(id)
		}
		if runDate, ok := resultMap["run_date"].(string); ok {
			if t, err := time.Parse(time.RFC3339, runDate); err == nil {
				r.RunDate = t
			}
		}
		if matchRate, ok := resultMap["match_rate"].(float64); ok {
			r.MatchRate = matchRate
		}
		if matchedCount, ok := resultMap["matched_count"].(float64); ok {
			r.MatchedCount = int(matchedCount)
		}
		if unmatchedCount, ok := resultMap["unmatched_count"].(float64); ok {
			r.UnmatchedCount = int(unmatchedCount)
		}
		if discrepancies, ok := resultMap["discrepancies"].(string); ok {
			r.DiscrepancyJSON = []byte(discrepancies)
		}
		if modelVersion, ok := resultMap["model_version"].(float64); ok {
			r.ModelVersion = int(modelVersion)
		}
		if status, ok := resultMap["status"].(string); ok {
			r.Status = status
		}
		if errorMessage, ok := resultMap["error_message"].(string); ok {
			r.ErrorMessage = &errorMessage
		}
		if createdAt, ok := resultMap["created_at"].(string); ok {
			if t, err := time.Parse(time.RFC3339, createdAt); err == nil {
				r.CreatedAt = t
			}
		}
		if updatedAt, ok := resultMap["updated_at"].(string); ok {
			if t, err := time.Parse(time.RFC3339, updatedAt); err == nil {
				r.UpdatedAt = t
			}
		}

		results = append(results, r)
	}

	c.JSON(http.StatusOK, results)
}

// GetLatestResult returns the most recent reconciliation result
func (h *Handler) GetLatestResult(c *gin.Context) {
	if h.hasura != nil {
		h.getLatestResultWithHasura(c)
		return
	}

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

func (h *Handler) getLatestResultWithHasura(c *gin.Context) {
	query := `
		query GetLatestResult {
			reconciliation_results(
				order_by: [{run_date: desc}]
				limit: 1
			) {
				id
				run_date
				match_rate
				matched_count
				unmatched_count
				discrepancies
				model_version
				status
				error_message
				created_at
				updated_at
			}
		}
	`

	result, err := h.hasura.Query(query, nil)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("Query failed: %v", err)})
		return
	}

	resultsData, ok := result["reconciliation_results"].([]interface{})
	if !ok || len(resultsData) == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": "No reconciliation results found"})
		return
	}

	resultMap, ok := resultsData[0].(map[string]interface{})
	if !ok {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "unexpected response format"})
		return
	}

	r := models.ReconciliationResult{}
	if id, ok := resultMap["id"].(string); ok {
		r.ID, _ = uuid.Parse(id)
	}
	if runDate, ok := resultMap["run_date"].(string); ok {
		if t, err := time.Parse(time.RFC3339, runDate); err == nil {
			r.RunDate = t
		}
	}
	if matchRate, ok := resultMap["match_rate"].(float64); ok {
		r.MatchRate = matchRate
	}
	if matchedCount, ok := resultMap["matched_count"].(float64); ok {
		r.MatchedCount = int(matchedCount)
	}
	if unmatchedCount, ok := resultMap["unmatched_count"].(float64); ok {
		r.UnmatchedCount = int(unmatchedCount)
	}
	if discrepancies, ok := resultMap["discrepancies"].(string); ok {
		r.DiscrepancyJSON = []byte(discrepancies)
	}
	if modelVersion, ok := resultMap["model_version"].(float64); ok {
		r.ModelVersion = int(modelVersion)
	}
	if status, ok := resultMap["status"].(string); ok {
		r.Status = status
	}
	if errorMessage, ok := resultMap["error_message"].(string); ok {
		r.ErrorMessage = &errorMessage
	}
	if createdAt, ok := resultMap["created_at"].(string); ok {
		if t, err := time.Parse(time.RFC3339, createdAt); err == nil {
			r.CreatedAt = t
		}
	}
	if updatedAt, ok := resultMap["updated_at"].(string); ok {
		if t, err := time.Parse(time.RFC3339, updatedAt); err == nil {
			r.UpdatedAt = t
		}
	}

	c.JSON(http.StatusOK, r)
}

// GetDiscrepancies returns discrepancies for a result
func (h *Handler) GetDiscrepancies(c *gin.Context) {
	if h.hasura != nil {
		h.getDiscrepanciesWithHasura(c)
		return
	}

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

func (h *Handler) getDiscrepanciesWithHasura(c *gin.Context) {
	resultID := c.Param("result_id")

	query := `
		query GetDiscrepancies($result_id: uuid!) {
			discrepancies(
				where: {result_id: {_eq: $result_id}}
				order_by: [{severity: desc}, {created_at: desc}]
			) {
				id
				result_id
				trade_id
				confirm_id
				discrepancy_type
				field
				trade_value
				confirm_value
				severity
				suggested_fix
				created_at
			}
		}
	`

	result, err := h.hasura.Query(query, map[string]interface{}{"result_id": resultID})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("Query failed: %v", err)})
		return
	}

	discrepanciesData, ok := result["discrepancies"].([]interface{})
	if !ok {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "unexpected response format"})
		return
	}

	var discrepancies []models.Discrepancy
	for _, item := range discrepanciesData {
		discMap, ok := item.(map[string]interface{})
		if !ok {
			continue
		}

		d := models.Discrepancy{}
		if id, ok := discMap["id"].(string); ok {
			d.ID, _ = uuid.Parse(id)
		}
		if resultID, ok := discMap["result_id"].(string); ok {
			d.ResultID, _ = uuid.Parse(resultID)
		}
		if tradeID, ok := discMap["trade_id"].(string); ok {
			d.TradeID = &tradeID
		}
		if confirmID, ok := discMap["confirm_id"].(string); ok {
			d.ConfirmID = &confirmID
		}
		if discType, ok := discMap["discrepancy_type"].(string); ok {
			d.DiscrepType = discType
		}
		if field, ok := discMap["field"].(string); ok {
			d.Field = &field
		}
		if tradeValue, ok := discMap["trade_value"].(string); ok {
			d.TradeValue = json.RawMessage(tradeValue)
		}
		if confirmValue, ok := discMap["confirm_value"].(string); ok {
			d.ConfirmValue = json.RawMessage(confirmValue)
		}
		if severity, ok := discMap["severity"].(string); ok {
			d.Severity = severity
		}
		if suggestedFix, ok := discMap["suggested_fix"].(string); ok {
			d.SuggestedFix = &suggestedFix
		}
		if createdAt, ok := discMap["created_at"].(string); ok {
			if t, err := time.Parse(time.RFC3339, createdAt); err == nil {
				d.CreatedAt = t
			}
		}

		discrepancies = append(discrepancies, d)
	}

	c.JSON(http.StatusOK, discrepancies)
}

// GetOpenTasks returns open reconciliation tasks
func (h *Handler) GetOpenTasks(c *gin.Context) {
	if h.hasura != nil {
		h.getOpenTasksWithHasura(c)
		return
	}

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

func (h *Handler) getOpenTasksWithHasura(c *gin.Context) {
	query := `
		query GetOpenTasks {
			reconciliation_tasks(
				where: {status: {_neq: "resolved"}}
				order_by: [{priority: desc}, {created_at: desc}]
			) {
				id
				result_id
				discrepancy_id
				status
				assigned_to
				priority
				notes
				resolved_at
				created_at
				updated_at
			}
		}
	`

	result, err := h.hasura.Query(query, nil)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("Query failed: %v", err)})
		return
	}

	tasksData, ok := result["reconciliation_tasks"].([]interface{})
	if !ok {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "unexpected response format"})
		return
	}

	var tasks []models.ReconciliationTask
	for _, item := range tasksData {
		taskMap, ok := item.(map[string]interface{})
		if !ok {
			continue
		}

		task := models.ReconciliationTask{}
		if id, ok := taskMap["id"].(string); ok {
			task.ID, _ = uuid.Parse(id)
		}
		if resultID, ok := taskMap["result_id"].(string); ok {
			task.ResultID, _ = uuid.Parse(resultID)
		}
		if discrepancyID, ok := taskMap["discrepancy_id"].(string); ok {
			task.DiscrepancyID, _ = uuid.Parse(discrepancyID)
		}
		if status, ok := taskMap["status"].(string); ok {
			task.Status = status
		}
		if assignedTo, ok := taskMap["assigned_to"].(string); ok {
			if assignedToUUID, err := uuid.Parse(assignedTo); err == nil {
				task.AssignedTo = &assignedToUUID
			}
		}
		if priority, ok := taskMap["priority"].(string); ok {
			task.Priority = priority
		}
		if notes, ok := taskMap["notes"].(string); ok {
			task.Notes = notes
		}
		if resolvedAt, ok := taskMap["resolved_at"].(string); ok {
			if parsedTime, err := time.Parse(time.RFC3339, resolvedAt); err == nil {
				task.ResolvedAt = &parsedTime
			}
		}
		if createdAt, ok := taskMap["created_at"].(string); ok {
			if parsedTime, err := time.Parse(time.RFC3339, createdAt); err == nil {
				task.CreatedAt = parsedTime
			}
		}
		if updatedAt, ok := taskMap["updated_at"].(string); ok {
			if parsedTime, err := time.Parse(time.RFC3339, updatedAt); err == nil {
				task.UpdatedAt = parsedTime
			}
		}

		tasks = append(tasks, task)
	}

	c.JSON(http.StatusOK, tasks)
}

// UpdateTask updates a reconciliation task
func (h *Handler) UpdateTask(c *gin.Context) {
	if h.hasura != nil {
		h.updateTaskWithHasura(c)
		return
	}

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

func (h *Handler) updateTaskWithHasura(c *gin.Context) {
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
	var resolvedAt *string
	if req.Status == "resolved" {
		resolvedAtStr := now.Format(time.RFC3339)
		resolvedAt = &resolvedAtStr
	}

	mutation := `
		mutation UpdateTask($id: uuid!, $_set: reconciliation_tasks_set_input!) {
			update_reconciliation_tasks(
				where: {id: {_eq: $id}},
				_set: $_set
			) {
				affected_rows
				returning {
					id
					updated_at
				}
			}
		}
	`

	setObject := map[string]interface{}{
		"status":     req.Status,
		"notes":      req.Notes,
		"priority":   req.Priority,
		"updated_at": now.Format(time.RFC3339),
	}
	if resolvedAt != nil {
		setObject["resolved_at"] = *resolvedAt
	}

	result, err := h.hasura.Mutate(mutation, map[string]interface{}{
		"id":   taskID,
		"_set": setObject,
	})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("Update failed: %v", err)})
		return
	}

	updateData, ok := result["update_reconciliation_tasks"].(map[string]interface{})
	if !ok {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "unexpected response format"})
		return
	}

	affectedRows, _ := updateData["affected_rows"].(float64)
	if affectedRows == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": "Task not found"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Task updated"})
}

// GetRules returns all reconciliation rules
func (h *Handler) GetRules(c *gin.Context) {
	if h.hasura != nil {
		h.getRulesWithHasura(c)
		return
	}

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

func (h *Handler) getRulesWithHasura(c *gin.Context) {
	query := `
		query GetRules {
			reconciliation_rules(
				order_by: [{rule_type: asc}, {created_at: desc}]
			) {
				id
				name
				description
				rule_type
				enabled
				rule_expr
				version
				created_at
				updated_at
			}
		}
	`

	result, err := h.hasura.Query(query, nil)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("Query failed: %v", err)})
		return
	}

	rulesData, ok := result["reconciliation_rules"].([]interface{})
	if !ok {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "unexpected response format"})
		return
	}

	var rules []models.ReconciliationRule
	for _, item := range rulesData {
		ruleMap, ok := item.(map[string]interface{})
		if !ok {
			continue
		}

		r := models.ReconciliationRule{}
		if id, ok := ruleMap["id"].(string); ok {
			r.ID, _ = uuid.Parse(id)
		}
		if name, ok := ruleMap["name"].(string); ok {
			r.Name = name
		}
		if description, ok := ruleMap["description"].(string); ok {
			r.Description = description
		}
		if ruleType, ok := ruleMap["rule_type"].(string); ok {
			r.RuleType = ruleType
		}
		if enabled, ok := ruleMap["enabled"].(bool); ok {
			r.Enabled = enabled
		}
		if ruleExpr, ok := ruleMap["rule_expr"].(string); ok {
			r.RuleExpr = ruleExpr
		}
		if version, ok := ruleMap["version"].(float64); ok {
			r.Version = int(version)
		}
		if createdAt, ok := ruleMap["created_at"].(string); ok {
			if t, err := time.Parse(time.RFC3339, createdAt); err == nil {
				r.CreatedAt = t
			}
		}
		if updatedAt, ok := ruleMap["updated_at"].(string); ok {
			if t, err := time.Parse(time.RFC3339, updatedAt); err == nil {
				r.UpdatedAt = t
			}
		}

		rules = append(rules, r)
	}

	c.JSON(http.StatusOK, rules)
}

// CreateRule creates a new reconciliation rule
func (h *Handler) CreateRule(c *gin.Context) {
	if h.hasura != nil {
		h.createRuleWithHasura(c)
		return
	}

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

func (h *Handler) createRuleWithHasura(c *gin.Context) {
	var rule models.ReconciliationRule
	if err := c.BindJSON(&rule); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": fmt.Sprintf("Invalid request: %v", err)})
		return
	}

	rule.ID = uuid.New()
	rule.CreatedAt = time.Now()
	rule.UpdatedAt = time.Now()
	rule.Version = 1

	mutation := `
		mutation CreateRule($object: reconciliation_rules_insert_input!) {
			insert_reconciliation_rules_one(object: $object) {
				id
				name
				description
				rule_type
				enabled
				rule_expr
				version
				created_at
				updated_at
			}
		}
	`

	object := map[string]interface{}{
		"id":          rule.ID.String(),
		"name":        rule.Name,
		"description": rule.Description,
		"rule_type":   rule.RuleType,
		"enabled":     rule.Enabled,
		"rule_expr":   rule.RuleExpr,
		"version":     rule.Version,
		"created_at":  rule.CreatedAt.Format(time.RFC3339),
		"updated_at":  rule.UpdatedAt.Format(time.RFC3339),
	}

	result, err := h.hasura.Mutate(mutation, map[string]interface{}{"object": object})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("Insert failed: %v", err)})
		return
	}

	ruleData, ok := result["insert_reconciliation_rules_one"].(map[string]interface{})
	if !ok {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "unexpected response format"})
		return
	}

	if id, ok := ruleData["id"].(string); ok {
		rule.ID, _ = uuid.Parse(id)
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
