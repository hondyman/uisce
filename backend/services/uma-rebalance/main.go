package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/hondyman/semlayer/backend/internal/events"
	"github.com/hondyman/semlayer/backend/internal/models"
	"github.com/hondyman/semlayer/backend/internal/workflows"
	temporalclient "github.com/hondyman/semlayer/libs/temporal-client"
	"go.temporal.io/sdk/client"
	"github.com/hondyman/semlayer/libs/jwt-middleware"
)

// ============================================================================
// UMA REBALANCE MICROSERVICE
// Exposes REST API for initiating UMA rebalances via Temporal workflows
// ============================================================================

// HasuraClient defines the interface for Hasura GraphQL operations
type HasuraClient interface {
	Query(query string, variables map[string]interface{}) (map[string]interface{}, error)
	Mutate(mutation string, variables map[string]interface{}) (map[string]interface{}, error)
}

type UMARebalanceService struct {
	db             *sql.DB
	hasura         HasuraClient
	temporalClient client.Client
	abacEngine     interface{} // Your ABAC engine
	eventBus       interface{} // Your event bus (Redpanda/Kafka)
	taskQueue      string
}

// NewUMARebalanceService creates a new service instance
func NewUMARebalanceService(db *sql.DB, tc client.Client, abacEngine interface{}, eventBus interface{}) *UMARebalanceService {
	return &UMARebalanceService{
		db:             db,
		temporalClient: tc,
		abacEngine:     abacEngine,
		eventBus:       eventBus,
		taskQueue:      "uma-rebalance",
	}
}

// NewUMARebalanceServiceWithHasura creates a new service instance with Hasura support
func NewUMARebalanceServiceWithHasura(db *sql.DB, hasura HasuraClient, tc client.Client, abacEngine interface{}, eventBus interface{}) *UMARebalanceService {
	return &UMARebalanceService{
		db:             db,
		hasura:         hasura,
		temporalClient: tc,
		abacEngine:     abacEngine,
		eventBus:       eventBus,
		taskQueue:      "uma-rebalance",
	}
}

// ============================================================================
// REQUEST/RESPONSE TYPES
// ============================================================================

type RequestRebalanceRequest struct {
	UMAAccountID string `json:"uma_account_id" binding:"required"`
	RequestType  string `json:"request_type" binding:"required"` // drift, manual, scheduled
	Reason       string `json:"reason"`
	InitiatedBy  string `json:"initiated_by"`
}

type RequestRebalanceResponse struct {
	RequestID     string `json:"request_id"`
	UMAAccountID  string `json:"uma_account_id"`
	WorkflowID    string `json:"workflow_id"`
	WorkflowRunID string `json:"workflow_run_id"`
	Status        string `json:"status"`
	Message       string `json:"message"`
	Timestamp     string `json:"timestamp"`
}

type ApproveRebalancePlanRequest struct {
	PlanID     string `json:"plan_id" binding:"required"`
	ApprovedBy string `json:"approved_by" binding:"required"`
	Reason     string `json:"reason,omitempty"`
}

type RebalanceStatusResponse struct {
	RequestID    string                 `json:"request_id"`
	UMAAccountID string                 `json:"uma_account_id"`
	WorkflowID   string                 `json:"workflow_id"`
	Status       string                 `json:"status"`
	CurrentPhase string                 `json:"current_phase"`
	Progress     map[string]interface{} `json:"progress"`
	Message      string                 `json:"message"`
	Timestamp    string                 `json:"timestamp"`
}

// ============================================================================
// HTTP HANDLERS
// ============================================================================

// RequestRebalanceHandler initiates a new rebalance workflow
func (s *UMARebalanceService) RequestRebalanceHandler(c *gin.Context) {
	var req RequestRebalanceRequest
	if err := c.BindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request: " + err.Error()})
		return
	}

	// Extract tenant context
	tenantID := c.GetString("tenant_id")
	datasourceID := c.GetString("datasource_id")
	userID := c.GetString("user_id")

	if tenantID == "" || datasourceID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Missing tenant context"})
		return
	}

	log.Printf("📥 Rebalance request: uma=%s, type=%s, user=%s", req.UMAAccountID, req.RequestType, userID)

	// Generate IDs
	requestID := uuid.New().String()
	workflowID := fmt.Sprintf("uma-rebalance-%s", uuid.New().String())

	// Create workflow input
	input := models.UMARebalanceWorkflowInput{
		RequestID:    requestID,
		TenantID:     tenantID,
		DatasourceID: datasourceID,
		UMAAccountID: req.UMAAccountID,
		RequestType:  req.RequestType,
		Reason:       req.Reason,
		InitiatedBy:  userID,
		EventData:    map[string]interface{}{},
	}

	// Save request to database
	err := s.saveRebalanceRequest(context.Background(), requestID, tenantID, datasourceID, req.UMAAccountID, req.RequestType, req.Reason, userID)
	if err != nil {
		log.Printf("❌ Failed to save request: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to save request"})
		return
	}

	// Emit UMA Rebalance Requested Event
	s.emitRebalanceRequestedEvent(tenantID, datasourceID, requestID, req.UMAAccountID, req.RequestType, req.Reason, userID)

	// Start Temporal workflow
	we, err := s.temporalClient.ExecuteWorkflow(
		context.Background(),
		client.StartWorkflowOptions{
			ID:        workflowID,
			TaskQueue: s.taskQueue,
		},
		workflows.UMARebalanceWorkflow,
		input,
	)
	if err != nil {
		log.Printf("❌ Failed to start workflow: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to start workflow"})
		return
	}

	log.Printf("✅ Workflow started: %s (run: %s)", we.GetID(), we.GetRunID())

	response := RequestRebalanceResponse{
		RequestID:     requestID,
		UMAAccountID:  req.UMAAccountID,
		WorkflowID:    we.GetID(),
		WorkflowRunID: we.GetRunID(),
		Status:        "pending",
		Message:       "Rebalance workflow initiated",
		Timestamp:     "2025-10-28T00:00:00Z",
	}

	c.JSON(http.StatusAccepted, response)
}

// GetRebalanceStatusHandler retrieves the status of a rebalance workflow
func (s *UMARebalanceService) GetRebalanceStatusHandler(c *gin.Context) {
	workflowID := c.Param("workflow_id")

	if workflowID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Missing workflow_id"})
		return
	}

	log.Printf("📊 Fetching status for workflow: %s", workflowID)

	// Get workflow execution
	we := s.temporalClient.GetWorkflow(context.Background(), workflowID, "")

	// Get workflow result
	var result map[string]interface{}
	err := we.Get(context.Background(), &result)
	if err != nil {
		// Workflow still running or failed
		result = map[string]interface{}{
			"status": "running",
		}
	}

	response := RebalanceStatusResponse{
		WorkflowID: workflowID,
		Status:     fmt.Sprintf("%v", result["workflow_status"]),
		Progress:   result,
		Timestamp:  "2025-10-28T00:00:00Z",
	}

	c.JSON(http.StatusOK, response)
}

// ApproveRebalancePlanHandler approves a rebalance plan
func (s *UMARebalanceService) ApproveRebalancePlanHandler(c *gin.Context) {
	var req ApproveRebalancePlanRequest
	if err := c.BindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request"})
		return
	}

	log.Printf("✅ Approving plan: %s by %s", req.PlanID, req.ApprovedBy)

	// Send approval signal to workflow
	approval := map[string]interface{}{
		"approved":    true,
		"approved_by": req.ApprovedBy,
		"reason":      req.Reason,
		"timestamp":   "2025-10-28T00:00:00Z",
	}

	approvalJSON, _ := json.Marshal(approval)

	// Find workflow associated with this plan
	_, err := s.getPlanByID(context.Background(), req.PlanID)
	if err != nil {
		log.Printf("❌ Plan not found: %v", err)
		c.JSON(http.StatusNotFound, gin.H{"error": "Plan not found"})
		return
	}

	// Signal all running rebalance workflows (in real implementation, find the specific workflow)
	// This is a placeholder - in production, you'd query for the specific workflow ID
	err = s.temporalClient.SignalWorkflow(context.Background(), "", "", "uma_rebalance_approval", approval)
	if err != nil {
		log.Printf("⚠️  Signal sent (may not have active listeners): %v", err)
	}

	// Update plan status in database
	err = s.approvePlan(context.Background(), req.PlanID, req.ApprovedBy)
	if err != nil {
		log.Printf("❌ Failed to update plan: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update plan"})
		return
	}

	// Emit Plan Approved Event
	s.emitRebalancePlanApprovedEvent(req.PlanID, req.ApprovedBy)

	c.JSON(http.StatusOK, gin.H{
		"message":     "Plan approved",
		"plan_id":     req.PlanID,
		"approved_by": req.ApprovedBy,
		"approval":    string(approvalJSON),
	})
}

// RejectRebalancePlanHandler rejects a rebalance plan
func (s *UMARebalanceService) RejectRebalancePlanHandler(c *gin.Context) {
	planID := c.Param("plan_id")
	var req struct {
		RejectedBy string `json:"rejected_by" binding:"required"`
		Reason     string `json:"reason" binding:"required"`
	}

	if err := c.BindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request"})
		return
	}

	log.Printf("❌ Rejecting plan: %s by %s", planID, req.RejectedBy)

	// Update plan status
	err := s.rejectPlan(context.Background(), planID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update plan"})
		return
	}

	// Send rejection signal
	rejection := map[string]interface{}{
		"approved":    false,
		"rejected_by": req.RejectedBy,
		"reason":      req.Reason,
	}

	err = s.temporalClient.SignalWorkflow(context.Background(), "", "", "uma_rebalance_approval", rejection)
	if err != nil {
		log.Printf("⚠️  Rejection signal sent: %v", err)
	}

	c.JSON(http.StatusOK, gin.H{
		"message":     "Plan rejected",
		"plan_id":     planID,
		"rejected_by": req.RejectedBy,
		"reason":      req.Reason,
	})
}

// ============================================================================
// EVENT EMISSION
// ============================================================================

func (s *UMARebalanceService) emitRebalanceRequestedEvent(tenantID, datasourceID, requestID, umaAccountID, requestType, reason, userID string) {
	// TODO: Integrate with your event bus
	now := time.Now()
	event := events.UMARebalanceRequestedEvent{
		EventID:      uuid.New().String(),
		EventType:    events.RebalanceRequested,
		TenantID:     tenantID,
		DatasourceID: datasourceID,
		RequestID:    requestID,
		UMAAccountID: umaAccountID,
		RequestType:  requestType,
		Reason:       reason,
		InitiatedBy:  userID,
		Timestamp:    now,
		UserID:       &userID,
	}

	eventJSON, _ := json.Marshal(event)
	log.Printf("📢 Emitting event: %s -> %s", event.EventType, string(eventJSON))

	// TODO: a.eventBus.Emit("uma.rebalance.requested", event)
}

func (s *UMARebalanceService) emitRebalancePlanApprovedEvent(planID, approvedBy string) {
	now := time.Now()
	event := events.UMARebalancePlanApprovedEvent{
		EventID:    uuid.New().String(),
		EventType:  events.RebalancePlanApproved,
		PlanID:     planID,
		ApprovedBy: approvedBy,
		Timestamp:  now,
	}

	eventJSON, _ := json.Marshal(event)
	log.Printf("📢 Emitting event: %s -> %s", event.EventType, string(eventJSON))

	// TODO: a.eventBus.Emit("uma.rebalance.plan.approved", event)
}

// ============================================================================
// MAIN / SETUP
// ============================================================================

func main() {
	// Initialize database
	db, err := sql.Open("postgres", os.Getenv("DATABASE_URL"))
	if err != nil {
		log.Fatalf("❌ Failed to connect to database: %v", err)
	}
	defer db.Close()

	// Initialize Temporal client using centralized helper (env-driven + retries)
	tc, err := temporalclient.NewClientWithRetry()
	if err != nil {
		log.Fatalf("❌ Failed to connect to Temporal: %v", err)
	}
	defer tc.Close()

	// Initialize service
	service := NewUMARebalanceService(db, tc, nil, nil)

	// Setup router
	r := gin.Default()

	// Middleware for tenant context
	r.Use(tenantContextMiddleware())

	// Routes
	r.POST("/uma/rebalance/request", service.RequestRebalanceHandler)
	r.GET("/uma/rebalance/:workflow_id/status", service.GetRebalanceStatusHandler)
	r.POST("/uma/rebalance/plan/:plan_id/approve", service.ApproveRebalancePlanHandler)
	r.POST("/uma/rebalance/plan/:plan_id/reject", service.RejectRebalancePlanHandler)

	// Health check
	r.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})

	log.Printf("🚀 UMA Rebalance Service starting on port 8087...")
	r.Run(":8087")
}

// tenantContextMiddleware extracts tenant context from headers or localStorage
func tenantContextMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		claims := jwtmiddleware.GetGinClaimsFromContext(c)
	if claims == nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error":"unauthorized"})
		return
	}
	tenantID := claims.TenantID
		datasourceID := c.GetHeader("X-Tenant-Datasource-ID")
		userID := c.GetHeader("X-User-ID")

		// Fallback to query params (for testing)
		if tenantID == "" {
			tenantID = c.Query("tenant_id")
		}
		if datasourceID == "" {
			datasourceID = c.Query("datasource_id")
		}

		c.Set("tenant_id", tenantID)
		c.Set("datasource_id", datasourceID)
		c.Set("user_id", userID)

		c.Next()
	}
}

// ============================================================================
// HASURA HELPER METHODS
// ============================================================================

// saveRebalanceRequest saves a rebalance request using Hasura or SQL fallback
func (s *UMARebalanceService) saveRebalanceRequest(ctx context.Context, requestID, tenantID, datasourceID, umaAccountID, requestType, reason, initiatedBy string) error {
	if s.hasura != nil {
		err := s.saveRebalanceRequestWithHasura(ctx, requestID, tenantID, datasourceID, umaAccountID, requestType, reason, initiatedBy)
		if err == nil {
			return nil
		}
		log.Printf("Hasura mutation failed, falling back to SQL: %v\n", err)
	}

	// SQL fallback
	// TODO: Hasura-first pattern already implemented via saveRebalanceRequestWithHasura()
	// Primary implementation uses Hasura GraphQL, this SQL is fallback only
	// See saveRebalanceRequestWithHasura() for the Hasura mutation:
	// mutation CreateRebalanceRequest($object: uma_rebalance_requests_insert_input!)
	saveQuery := `
		INSERT INTO uma_rebalance_requests (id, tenant_id, datasource_id, uma_account_id, request_type, reason, initiated_by, status, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, NOW(), NOW())
	`
	_, err := s.db.ExecContext(ctx, saveQuery, requestID, tenantID, datasourceID, umaAccountID, requestType, reason, initiatedBy, "pending")
	return err
}

// saveRebalanceRequestWithHasura saves a rebalance request using Hasura GraphQL
func (s *UMARebalanceService) saveRebalanceRequestWithHasura(ctx context.Context, requestID, tenantID, datasourceID, umaAccountID, requestType, reason, initiatedBy string) error {
	mutation := `
		mutation CreateRebalanceRequest($object: uma_rebalance_requests_insert_input!) {
			insert_uma_rebalance_requests_one(object: $object) {
				id
			}
		}
	`

	variables := map[string]interface{}{
		"object": map[string]interface{}{
			"id":             requestID,
			"tenant_id":      tenantID,
			"datasource_id":  datasourceID,
			"uma_account_id": umaAccountID,
			"request_type":   requestType,
			"reason":         reason,
			"initiated_by":   initiatedBy,
			"status":         "pending",
		},
	}

	_, err := s.hasura.Mutate(mutation, variables)
	return err
}

// getPlanByID retrieves a plan ID using Hasura or SQL fallback
func (s *UMARebalanceService) getPlanByID(ctx context.Context, planID string) (string, error) {
	if s.hasura != nil {
		id, err := s.getPlanByIDWithHasura(ctx, planID)
		if err == nil {
			return id, nil
		}
		log.Printf("Hasura query failed, falling back to SQL: %v\n", err)
	}

	// SQL fallback
	query := `
		SELECT uma_rebalance_plans.id
		FROM uma_rebalance_plans
		WHERE id = $1
		LIMIT 1
	`
	row := s.db.QueryRowContext(ctx, query, planID)

	var id string
	err := row.Scan(&id)
	return id, err
}

// getPlanByIDWithHasura retrieves a plan ID using Hasura GraphQL
func (s *UMARebalanceService) getPlanByIDWithHasura(ctx context.Context, planID string) (string, error) {
	query := `
		query GetPlan($id: String!) {
			uma_rebalance_plans_by_pk(id: $id) {
				id
			}
		}
	`

	variables := map[string]interface{}{
		"id": planID,
	}

	result, err := s.hasura.Query(query, variables)
	if err != nil {
		return "", err
	}

	planData, ok := result["uma_rebalance_plans_by_pk"].(map[string]interface{})
	if !ok || planData == nil {
		return "", fmt.Errorf("plan not found")
	}

	if id, ok := planData["id"].(string); ok {
		return id, nil
	}

	return "", fmt.Errorf("invalid plan data")
}

// approvePlan approves a rebalance plan using Hasura or SQL fallback
func (s *UMARebalanceService) approvePlan(ctx context.Context, planID, approvedBy string) error {
	if s.hasura != nil {
		err := s.approvePlanWithHasura(ctx, planID, approvedBy)
		if err == nil {
			return nil
		}
		log.Printf("Hasura mutation failed, falling back to SQL: %v\n", err)
	}

	// SQL fallback
	// TODO: Hasura-first pattern already implemented via approvePlanWithHasura()
	// Primary implementation uses Hasura GraphQL, this SQL is fallback only
	// See approvePlanWithHasura() for the Hasura mutation:
	// mutation ApprovePlan($id: String!, $approved_by: String!)
	updateQuery := `
		UPDATE uma_rebalance_plans
		SET status = 'approved', approved_at = NOW(), approved_by = $1, updated_at = NOW()
		WHERE id = $2
	`
	_, err := s.db.ExecContext(ctx, updateQuery, approvedBy, planID)
	return err
}

// approvePlanWithHasura approves a rebalance plan using Hasura GraphQL
func (s *UMARebalanceService) approvePlanWithHasura(ctx context.Context, planID, approvedBy string) error {
	mutation := `
		mutation ApprovePlan($id: String!, $approved_by: String!) {
			update_uma_rebalance_plans(
where: {id: {_eq: $id}},
_set: {
status: "approved",
approved_by: $approved_by
}
) {
				affected_rows
			}
		}
	`

	variables := map[string]interface{}{
		"id":          planID,
		"approved_by": approvedBy,
	}

	_, err := s.hasura.Mutate(mutation, variables)
	return err
}

// rejectPlan rejects a rebalance plan using Hasura or SQL fallback
func (s *UMARebalanceService) rejectPlan(ctx context.Context, planID string) error {
	if s.hasura != nil {
		err := s.rejectPlanWithHasura(ctx, planID)
		if err == nil {
			return nil
		}
		log.Printf("Hasura mutation failed, falling back to SQL: %v\n", err)
	}

	// SQL fallback
	// TODO: Hasura-first pattern already implemented via rejectPlanWithHasura()
	// Primary implementation uses Hasura GraphQL, this SQL is fallback only
	// See rejectPlanWithHasura() for the Hasura mutation:
	// mutation RejectPlan($id: String!)
	updateQuery := `
		UPDATE uma_rebalance_plans
		SET status = 'rejected', updated_at = NOW()
		WHERE id = $1
	`
	_, err := s.db.ExecContext(ctx, updateQuery, planID)
	return err
}

// rejectPlanWithHasura rejects a rebalance plan using Hasura GraphQL
func (s *UMARebalanceService) rejectPlanWithHasura(ctx context.Context, planID string) error {
	mutation := `
		mutation RejectPlan($id: String!) {
			update_uma_rebalance_plans(
where: {id: {_eq: $id}},
_set: {status: "rejected"}
) {
				affected_rows
			}
		}
	`

	variables := map[string]interface{}{
		"id": planID,
	}

	_, err := s.hasura.Mutate(mutation, variables)
	return err
}
