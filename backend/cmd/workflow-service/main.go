package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	kafka "github.com/segmentio/kafka-go"
)

// ============================================================================
// MODELS
// ============================================================================

// WorkflowRule represents a rule fetched from Hasura
type WorkflowRule struct {
	ID              string          `json:"id"`
	WorkflowName    string          `json:"workflow_name"`
	StepName        string          `json:"step_name"`
	StepOrder       int             `json:"step_order"`
	ConditionJSON   json.RawMessage `json:"condition_json"`
	ActionOnSuccess string          `json:"action_on_success"`
	ActionOnFailure string          `json:"action_on_failure"`
	ErrorMessage    string          `json:"error_message"`
	TimeoutSeconds  int             `json:"timeout_seconds"`
	RetryCount      int             `json:"retry_count"`
}

// WorkflowRequest from the core application (API call to trigger workflow)
type WorkflowRequest struct {
	TenantID     uuid.UUID              `json:"tenant_id" binding:"required"`
	WorkflowName string                 `json:"workflow_name" binding:"required"`
	StepName     string                 `json:"step_name" binding:"required"`
	BOType       string                 `json:"bo_type" binding:"required"` // e.g., "orders"
	BOID         uuid.UUID              `json:"bo_id" binding:"required"`
	FormData     map[string]interface{} `json:"form_data"`
	UserID       uuid.UUID              `json:"user_id" binding:"required"`
}

// WorkflowResponse for the API response
type WorkflowResponse struct {
	Status     string    `json:"status"`
	WorkflowID string    `json:"workflow_id,omitempty"`
	HistoryID  uuid.UUID `json:"history_id,omitempty"`
	Error      string    `json:"error,omitempty"`
	Message    string    `json:"message,omitempty"`
	NextAction string    `json:"next_action,omitempty"`
}

// WorkflowEvent for RabbitMQ routing
type WorkflowEvent struct {
	TenantID     uuid.UUID              `json:"tenant_id"`
	WorkflowName string                 `json:"workflow_name"`
	StepName     string                 `json:"step_name"`
	BOType       string                 `json:"bo_type"`
	BOID         uuid.UUID              `json:"bo_id"`
	Result       string                 `json:"result"` // "success" or "failure"
	Message      string                 `json:"message,omitempty"`
	HistoryID    uuid.UUID              `json:"history_id"`
	FormData     map[string]interface{} `json:"form_data"`
	UserID       uuid.UUID              `json:"user_id"`
	Timestamp    time.Time              `json:"timestamp"`
}

// ConditionEvaluator evaluates JSONB conditions
type ConditionEvaluator struct {
	Condition map[string]interface{}
	Data      map[string]interface{}
}

// ============================================================================
// GLOBAL CLIENTS
// ============================================================================

var (
	hasuraURL   string
	hasuraToken string
	kafkaWriter *kafka.Writer
)

// ============================================================================
// INITIALIZATION
// ============================================================================

func init() {
	hasuraURL = os.Getenv("HASURA_URL")
	if hasuraURL == "" {
		hasuraURL = "http://localhost:8080"
	}
	hasuraToken = os.Getenv("HASURA_ADMIN_SECRET")

	kafkaBrokers := os.Getenv("KAFKA_BROKERS")
	if kafkaBrokers == "" {
		kafkaBrokers = "localhost:9092"
	}
	brokers := strings.Split(kafkaBrokers, ",")

	kafkaWriter = &kafka.Writer{
		Addr:     kafka.TCP(brokers...),
		Balancer: &kafka.LeastBytes{},
	}

	log.Println("✓ Workflow Service initialized")
	log.Printf("  Hasura: %s\n", hasuraURL)
	log.Printf("  Kafka brokers: %s\n", kafkaBrokers)
}

// ============================================================================
// MAIN
// ============================================================================

func main() {
	defer func() {
		if kafkaWriter != nil {
			kafkaWriter.Close()
		}
	}()

	r := gin.Default()

	// Health check
	r.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})

	// Trigger workflow
	r.POST("/workflow/trigger", triggerWorkflow)

	// Get workflow history
	r.GET("/workflow/history/:tenant_id/:bo_type/:bo_id", getWorkflowHistory)

	// Get available workflows
	r.GET("/workflow/templates/:tenant_id", getAvailableWorkflows)

	port := os.Getenv("WORKFLOW_SERVICE_PORT")
	if port == "" {
		port = "8082"
	}

	log.Printf("Workflow Service listening on :%s\n", port)
	if err := r.Run(":" + port); err != nil {
		log.Fatalf("Server error: %v", err)
	}
}

// ============================================================================
// HANDLERS
// ============================================================================

// triggerWorkflow handles POST /workflow/trigger
func triggerWorkflow(c *gin.Context) {
	var req WorkflowRequest

	if err := c.BindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, WorkflowResponse{
			Status: "error",
			Error:  fmt.Sprintf("Invalid request: %v", err),
		})
		return
	}

	log.Printf("→ Workflow triggered: %s/%s (tenant: %s, bo: %s/%s)\n",
		req.WorkflowName, req.StepName, req.TenantID, req.BOType, req.BOID)

	// Fetch rules from Hasura
	rules, err := fetchWorkflowRules(c.Request.Context(), req.TenantID, req.WorkflowName, req.StepName)
	if err != nil {
		c.JSON(http.StatusInternalServerError, WorkflowResponse{
			Status: "error",
			Error:  fmt.Sprintf("Failed to fetch workflow rules: %v", err),
		})
		return
	}

	if len(rules) == 0 {
		c.JSON(http.StatusNotFound, WorkflowResponse{
			Status: "error",
			Error:  fmt.Sprintf("No rules found for workflow %s/step %s", req.WorkflowName, req.StepName),
		})
		return
	}

	rule := rules[0]

	// Evaluate condition
	evaluator := NewConditionEvaluator(rule.ConditionJSON, req.FormData)
	conditionMet, evalErr := evaluator.Evaluate()

	historyID := uuid.New()

	if evalErr != nil || !conditionMet {
		// Condition failed
		errorMsg := rule.ErrorMessage
		if evalErr != nil {
			errorMsg = fmt.Sprintf("Condition evaluation error: %v", evalErr)
		}

		log.Printf("✗ Condition failed: %s\n", errorMsg)

		// Record failure
		if err := recordWorkflowHistory(c.Request.Context(), req.TenantID, req, rule.StepName, "failure", errorMsg, historyID); err != nil {
			log.Printf("Failed to record history: %v\n", err)
		}

		// Route failure event
		if rule.ActionOnFailure != "" {
			go routeEvent(req, rule.ActionOnFailure, "failure", errorMsg, historyID)
		}

		c.JSON(http.StatusBadRequest, WorkflowResponse{
			Status:    "failed",
			HistoryID: historyID,
			Error:     errorMsg,
			Message:   "Workflow step condition not satisfied",
		})
		return
	}

	// Condition passed
	log.Printf("✓ Condition passed: %s\n", rule.StepName)

	// Record success
	if err := recordWorkflowHistory(c.Request.Context(), req.TenantID, req, rule.StepName, "success", "", historyID); err != nil {
		log.Printf("Failed to record history: %v\n", err)
	}

	// Route success event
	if rule.ActionOnSuccess != "" {
		go routeEvent(req, rule.ActionOnSuccess, "success", "", historyID)
	}

	c.JSON(http.StatusOK, WorkflowResponse{
		Status:     "success",
		HistoryID:  historyID,
		Message:    fmt.Sprintf("Workflow step %s completed successfully", rule.StepName),
		NextAction: rule.ActionOnSuccess,
	})
}

// getWorkflowHistory fetches execution history for a business object
func getWorkflowHistory(c *gin.Context) {
	tenantID := c.Param("tenant_id")
	boType := c.Param("bo_type")
	boID := c.Param("bo_id")

	query := `
		query GetWorkflowHistory($tenantID: uuid!, $boType: String!, $boID: uuid!) {
			workflow_history(
				where: {tenant_id: {_eq: $tenantID}, bo_type: {_eq: $boType}, bo_id: {_eq: $boID}}
				order_by: {created_at: desc}
				limit: 50
			) {
				id
				workflow_name
				step_name
				status
				details
				created_at
				user_id
			}
		}
	`

	variables := map[string]interface{}{
		"tenantID": tenantID,
		"boType":   boType,
		"boID":     boID,
	}

	data, err := hasuraGraphQLQuery(c.Request.Context(), query, variables)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": fmt.Sprintf("Failed to fetch history: %v", err),
		})
		return
	}

	var resp struct {
		WorkflowHistory []map[string]interface{} `json:"workflow_history"`
	}

	if err := json.Unmarshal(data, &resp); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": fmt.Sprintf("Failed to parse history: %v", err),
		})
		return
	}

	c.JSON(http.StatusOK, resp.WorkflowHistory)
}

// getAvailableWorkflows fetches workflow templates for a tenant
func getAvailableWorkflows(c *gin.Context) {
	tenantID := c.Param("tenant_id")

	query := `
		query GetWorkflowTemplates($tenantID: uuid!) {
			workflow_templates(
				where: {_or: [{tenant_id: {_is_null: true}}, {tenant_id: {_eq: $tenantID}}]}
				order_by: {workflow_name: asc}
			) {
				id
				workflow_name
				description
				bo_type
				trigger_event
			}
		}
	`

	variables := map[string]interface{}{
		"tenantID": tenantID,
	}

	data, err := hasuraGraphQLQuery(c.Request.Context(), query, variables)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": fmt.Sprintf("Failed to fetch templates: %v", err),
		})
		return
	}

	var resp struct {
		WorkflowTemplates []map[string]interface{} `json:"workflow_templates"`
	}

	if err := json.Unmarshal(data, &resp); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": fmt.Sprintf("Failed to parse templates: %v", err),
		})
		return
	}

	c.JSON(http.StatusOK, resp.WorkflowTemplates)
}

// ============================================================================
// HELPER FUNCTIONS
// ============================================================================

// hasuraGraphQLQuery makes a GraphQL query to Hasura and returns the result
func hasuraGraphQLQuery(ctx context.Context, query string, variables map[string]interface{}) (json.RawMessage, error) {
	payload := map[string]interface{}{
		"query":     query,
		"variables": variables,
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequestWithContext(ctx, "POST", hasuraURL+"/v1/graphql", strings.NewReader(string(body)))
	if err != nil {
		return nil, err
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("x-hasura-admin-secret", hasuraToken)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var result struct {
		Data   json.RawMessage `json:"data"`
		Errors []interface{}   `json:"errors"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}

	if len(result.Errors) > 0 {
		return nil, fmt.Errorf("GraphQL error: %v", result.Errors[0])
	}

	return result.Data, nil
}

// fetchWorkflowRules retrieves rules from Hasura
func fetchWorkflowRules(ctx context.Context, tenantID uuid.UUID, workflowName, stepName string) ([]WorkflowRule, error) {
	query := `
		query FetchWorkflowRules($tenantID: uuid!, $workflowName: String!, $stepName: String!) {
			workflow_rules(
				where: {
					tenant_id: {_eq: $tenantID}
					workflow_name: {_eq: $workflowName}
					step_name: {_eq: $stepName}
					is_active: {_eq: true}
				}
			) {
				id
				workflow_name
				step_name
				step_order
				condition_json
				action_on_success
				action_on_failure
				error_message
				timeout_seconds
				retry_count
			}
		}
	`

	variables := map[string]interface{}{
		"tenantID":     tenantID.String(),
		"workflowName": workflowName,
		"stepName":     stepName,
	}

	data, err := hasuraGraphQLQuery(ctx, query, variables)
	if err != nil {
		return nil, err
	}

	var resp struct {
		WorkflowRules []WorkflowRule `json:"workflow_rules"`
	}

	if err := json.Unmarshal(data, &resp); err != nil {
		return nil, err
	}

	return resp.WorkflowRules, nil
}

// recordWorkflowHistory inserts a history record in Hasura
func recordWorkflowHistory(ctx context.Context, tenantID uuid.UUID, req WorkflowRequest, stepName, status, message string, historyID uuid.UUID) error {
	query := `
		mutation RecordWorkflowHistory($object: workflow_history_insert_input!) {
			insert_workflow_history_one(object: $object) {
				id
			}
		}
	`

	details := map[string]interface{}{
		"form_data": req.FormData,
	}
	if message != "" {
		details["error"] = message
	}

	detailsJSON, _ := json.Marshal(details)

	object := map[string]interface{}{
		"id":            historyID.String(),
		"tenant_id":     tenantID.String(),
		"workflow_name": req.WorkflowName,
		"step_name":     stepName,
		"bo_type":       req.BOType,
		"bo_id":         req.BOID.String(),
		"status":        status,
		"details":       json.RawMessage(detailsJSON),
		"user_id":       req.UserID.String(),
	}

	variables := map[string]interface{}{
		"object": object,
	}

	_, err := hasuraGraphQLQuery(ctx, query, variables)
	return err
}

// routeEvent publishes an event to Kafka based on the action
func routeEvent(req WorkflowRequest, action, result, message string, historyID uuid.UUID) {
	parts := strings.Split(action, ":")
	if len(parts) < 2 {
		log.Printf("Invalid action format: %s\n", action)
		return
	}

	actionType := parts[0]
	target := parts[1]

	switch actionType {
	case "route":
		// Route to RabbitMQ queue
		event := WorkflowEvent{
			TenantID:     req.TenantID,
			WorkflowName: req.WorkflowName,
			StepName:     "pending",
			BOType:       req.BOType,
			BOID:         req.BOID,
			Result:       result,
			Message:      message,
			HistoryID:    historyID,
			FormData:     req.FormData,
			UserID:       req.UserID,
			Timestamp:    time.Now(),
		}

		body, _ := json.Marshal(event)

		if kafkaWriter == nil {
			log.Printf("Kafka writer not initialized; cannot route event to %s", target)
			return
		}

		msg := kafka.Message{Topic: target, Key: []byte(historyID.String()), Value: body, Time: time.Now()}
		if err := kafkaWriter.WriteMessages(context.Background(), msg); err != nil {
			log.Printf("Failed to publish to topic %s: %v\n", target, err)
		} else {
			log.Printf("✓ Event routed to topic: %s\n", target)
		}

	case "notify":
		// Could implement notification service here
		log.Printf("✓ Notification queued: %s\n", target)

	default:
		log.Printf("Unknown action type: %s\n", actionType)
	}
}

// ============================================================================
// CONDITION EVALUATOR
// ============================================================================

// NewConditionEvaluator creates a new evaluator
func NewConditionEvaluator(conditionJSON json.RawMessage, data map[string]interface{}) *ConditionEvaluator {
	var condition map[string]interface{}
	json.Unmarshal(conditionJSON, &condition)

	return &ConditionEvaluator{
		Condition: condition,
		Data:      data,
	}
}

// Evaluate evaluates the condition against form data
func (ce *ConditionEvaluator) Evaluate() (bool, error) {
	if _, ok := ce.Condition["and"]; ok {
		return ce.evaluateAnd()
	}
	if _, ok := ce.Condition["or"]; ok {
		return ce.evaluateOr()
	}

	// Single condition
	return ce.evaluateCondition(ce.Condition)
}

func (ce *ConditionEvaluator) evaluateAnd() (bool, error) {
	conditions := ce.Condition["and"].([]interface{})
	for _, c := range conditions {
		cond := c.(map[string]interface{})
		result, err := ce.evaluateCondition(cond)
		if err != nil {
			return false, err
		}
		if !result {
			return false, nil
		}
	}
	return true, nil
}

func (ce *ConditionEvaluator) evaluateOr() (bool, error) {
	conditions := ce.Condition["or"].([]interface{})
	for _, c := range conditions {
		cond := c.(map[string]interface{})
		result, err := ce.evaluateCondition(cond)
		if err != nil {
			return false, err
		}
		if result {
			return true, nil
		}
	}
	return false, nil
}

func (ce *ConditionEvaluator) evaluateCondition(cond map[string]interface{}) (bool, error) {
	field := cond["field"].(string)
	operator := cond["operator"].(string)
	target := cond["value"]

	val, ok := ce.Data[field]
	if !ok {
		// If field not provided, treat as null
		return operator == "is_null" || operator == "not_provided", nil
	}

	switch operator {
	case "=":
		return val == target, nil
	case "!=":
		return val != target, nil
	case ">":
		return compareNumeric(val, target, func(a, b float64) bool { return a > b })
	case ">=":
		return compareNumeric(val, target, func(a, b float64) bool { return a >= b })
	case "<":
		return compareNumeric(val, target, func(a, b float64) bool { return a < b })
	case "<=":
		return compareNumeric(val, target, func(a, b float64) bool { return a <= b })
	case "contains":
		return strings.Contains(fmt.Sprintf("%v", val), fmt.Sprintf("%v", target)), nil
	case "not_null":
		return val != nil, nil
	case "is_null":
		return val == nil, nil
	default:
		return false, fmt.Errorf("unknown operator: %s", operator)
	}
}

func compareNumeric(val, target interface{}, compare func(float64, float64) bool) (bool, error) {
	var valNum, targetNum float64

	switch v := val.(type) {
	case float64:
		valNum = v
	case int:
		valNum = float64(v)
	default:
		return false, fmt.Errorf("cannot compare non-numeric value: %v", val)
	}

	switch t := target.(type) {
	case float64:
		targetNum = t
	case int:
		targetNum = float64(t)
	default:
		return false, fmt.Errorf("cannot compare non-numeric target: %v", target)
	}

	return compare(valNum, targetNum), nil
}
