package main

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	kafka "github.com/segmentio/kafka-go"
	"github.com/sirupsen/logrus"
	"go.temporal.io/sdk/client"
	"gorm.io/gorm"
)

// WorkflowABACPolicy represents workflow-specific ABAC policies
type WorkflowABACPolicy struct {
	ID               string                 `json:"id" gorm:"primaryKey"`
	TenantID         string                 `json:"tenant_id"`
	DatasourceID     string                 `json:"datasource_id"`
	WorkflowType     string                 `json:"workflow_type"`    // "investment", "compliance", "onboarding", etc.
	Action           string                 `json:"action"`           // "create", "execute", "modify", "delete", "view"
	ResourcePattern  string                 `json:"resource_pattern"` // regex pattern for resource matching
	SubjectRules     map[string]interface{} `json:"subject_rules" gorm:"serializer:json"`
	EnvironmentRules map[string]interface{} `json:"environment_rules" gorm:"serializer:json"`
	RiskLevel        string                 `json:"risk_level"` // "low", "medium", "high", "critical"
	RequiresApproval bool                   `json:"requires_approval"`
	ApprovalRoles    []string               `json:"approval_roles" gorm:"serializer:json"`
	TimeRestrictions map[string]interface{} `json:"time_restrictions" gorm:"serializer:json"`
	Enabled          bool                   `json:"enabled"`
	CreatedAt        time.Time              `json:"created_at"`
	UpdatedAt        time.Time              `json:"updated_at"`
}

// WorkflowABACEvaluationRequest extends the base ABAC request for workflows
type WorkflowABACEvaluationRequest struct {
	Subject        string                 `json:"subject"`
	Action         string                 `json:"action"`
	Resource       string                 `json:"resource"`
	WorkflowType   string                 `json:"workflow_type"`
	RiskAssessment map[string]interface{} `json:"risk_assessment"`
	Context        map[string]interface{} `json:"context"`
	TenantID       string                 `json:"tenant_id"`
	DatasourceID   string                 `json:"datasource_id"`
}

// WorkflowABACEngine handles workflow-specific ABAC evaluations
type WorkflowABACEngine struct {
	db             *gorm.DB
	kafkaWriter    *kafka.Writer
	temporalClient client.Client
	logger         *logrus.Logger
}

func NewWorkflowABACEngine(db *gorm.DB, kafkaWriter *kafka.Writer, temporalClient client.Client) *WorkflowABACEngine {
	logger := logrus.New()
	logger.SetFormatter(&logrus.JSONFormatter{})

	return &WorkflowABACEngine{
		db:             db,
		kafkaWriter:    kafkaWriter,
		temporalClient: temporalClient,
		logger:         logger,
	}
}

// EvaluateWorkflowPolicy evaluates workflow-specific ABAC policies
func (wabe *WorkflowABACEngine) EvaluateWorkflowPolicy(ctx context.Context, req WorkflowABACEvaluationRequest) (bool, string, error) {
	// Query matching policies
	var policies []WorkflowABACPolicy
	query := wabe.db.Where("tenant_id = ? AND datasource_id = ? AND workflow_type = ? AND action = ? AND enabled = true",
		req.TenantID, req.DatasourceID, req.WorkflowType, req.Action)

	if err := query.Find(&policies).Error; err != nil {
		return false, "Database error", fmt.Errorf("failed to query policies: %w", err)
	}

	// Evaluate policies in order (could be sorted by priority)
	for _, policy := range policies {
		if wabe.matchesPolicy(policy, req) {
			// Check time restrictions
			if !wabe.checkTimeRestrictions(policy.TimeRestrictions) {
				continue
			}

			// Check risk level compatibility
			if !wabe.checkRiskCompatibility(policy.RiskLevel, req.RiskAssessment) {
				continue
			}

			// If policy requires approval, check if approval workflow exists
			if policy.RequiresApproval {
				approved, err := wabe.checkApprovalStatus(req, policy.ApprovalRoles)
				if err != nil || !approved {
					wabe.logABACDecision(req, "deny", "Approval required but not granted", policy.ID)
					return false, "Approval required", nil
				}
			}

			wabe.logABACDecision(req, "allow", "Policy matched", policy.ID)
			return true, "Access granted", nil
		}
	}

	wabe.logABACDecision(req, "deny", "No matching policy", "")
	return false, "Access denied", nil
}

// matchesPolicy checks if a request matches a policy's rules
func (wabe *WorkflowABACEngine) matchesPolicy(policy WorkflowABACPolicy, req WorkflowABACEvaluationRequest) bool {
	// Check subject rules
	if !wabe.evaluateRules(policy.SubjectRules, map[string]interface{}{
		"subject": req.Subject,
		"context": req.Context,
	}) {
		return false
	}

	// Check environment rules
	if !wabe.evaluateRules(policy.EnvironmentRules, map[string]interface{}{
		"tenant_id":     req.TenantID,
		"datasource_id": req.DatasourceID,
		"timestamp":     time.Now(),
		"context":       req.Context,
	}) {
		return false
	}

	// Check resource pattern (simple string matching for now)
	if policy.ResourcePattern != "" && policy.ResourcePattern != "*" {
		// In production, implement regex matching
		if policy.ResourcePattern != req.Resource {
			return false
		}
	}

	return true
}

// evaluateRules evaluates JSON-based rules (simplified implementation)
func (wabe *WorkflowABACEngine) evaluateRules(rules, context map[string]interface{}) bool {
	if len(rules) == 0 {
		return true
	}

	// Simple rule evaluation - in production, implement full rule engine
	for key, expected := range rules {
		if actual, exists := context[key]; exists {
			if actual != expected {
				return false
			}
		} else {
			return false
		}
	}

	return true
}

// checkTimeRestrictions validates time-based access controls
func (wabe *WorkflowABACEngine) checkTimeRestrictions(restrictions map[string]interface{}) bool {
	if len(restrictions) == 0 {
		return true
	}

	now := time.Now()

	// Check business hours
	if businessHours, ok := restrictions["business_hours"].(bool); ok && businessHours {
		hour := now.Hour()
		weekday := now.Weekday()
		if weekday < time.Monday || weekday > time.Friday || hour < 9 || hour > 17 {
			return false
		}
	}

	// Check time windows
	if startTime, ok := restrictions["start_time"].(string); ok {
		if endTime, ok := restrictions["end_time"].(string); ok {
			start, err1 := time.Parse("15:04", startTime)
			end, err2 := time.Parse("15:04", endTime)
			if err1 == nil && err2 == nil {
				nowTime := time.Date(0, 1, 1, now.Hour(), now.Minute(), 0, 0, time.UTC)
				if nowTime.Before(start) || nowTime.After(end) {
					return false
				}
			}
		}
	}

	return true
}

// checkRiskCompatibility validates risk level compatibility
func (wabe *WorkflowABACEngine) checkRiskCompatibility(policyRisk string, assessment map[string]interface{}) bool {
	if policyRisk == "" {
		return true
	}

	// Map risk levels to numeric values
	riskLevels := map[string]int{
		"low":      1,
		"medium":   2,
		"high":     3,
		"critical": 4,
	}

	policyLevel, exists := riskLevels[policyRisk]
	if !exists {
		return false
	}

	// Check assessment risk level
	if assessmentRisk, ok := assessment["level"].(string); ok {
		assessmentLevel, exists := riskLevels[assessmentRisk]
		if exists && assessmentLevel > policyLevel {
			return false // Assessment risk too high for policy
		}
	}

	return true
}

// checkApprovalStatus checks if required approvals have been granted
func (wabe *WorkflowABACEngine) checkApprovalStatus(req WorkflowABACEvaluationRequest, requiredRoles []string) (bool, error) {
	// Query approval status from database or Temporal workflow
	// This is a simplified implementation

	// In production, this would check:
	// 1. Active approval workflows in Temporal
	// 2. Approval status in database
	// 3. User's role against required approval roles

	// For now, check if user has any of the required roles
	userRoles := req.Context["user_roles"]
	if roles, ok := userRoles.([]interface{}); ok {
		for _, requiredRole := range requiredRoles {
			for _, userRole := range roles {
				if roleStr, ok := userRole.(string); ok && roleStr == requiredRole {
					return true, nil
				}
			}
		}
	}

	return false, nil
}

// logABACDecision logs ABAC evaluation decisions to RabbitMQ for compliance
func (wabe *WorkflowABACEngine) logABACDecision(req WorkflowABACEvaluationRequest, decision, reason, policyID string) {
	event := map[string]interface{}{
		"event_type":    "workflow_abac_decision",
		"subject":       req.Subject,
		"action":        req.Action,
		"resource":      req.Resource,
		"workflow_type": req.WorkflowType,
		"decision":      decision,
		"reason":        reason,
		"policy_id":     policyID,
		"tenant_id":     req.TenantID,
		"datasource_id": req.DatasourceID,
		"context":       req.Context,
		"timestamp":     time.Now(),
	}

	eventJSON, err := json.Marshal(event)
	if err != nil {
		wabe.logger.Errorf("Failed to marshal ABAC event: %v", err)
		return
	}

	msg := kafka.Message{
		Topic: "abac.audit",
		Key:   []byte(req.TenantID),
		Value: eventJSON,
		Time:  time.Now(),
	}

	if wabe.kafkaWriter != nil {
		if err := wabe.kafkaWriter.WriteMessages(context.Background(), msg); err != nil {
			wabe.logger.Errorf("Failed to publish ABAC audit event to Kafka: %v", err)
		}
	} else {
		wabe.logger.Warn("Kafka writer not initialized; ABAC audit event not published")
	}
}

// InitializeDefaultWorkflowPolicies creates default workflow ABAC policies
func (wabe *WorkflowABACEngine) InitializeDefaultWorkflowPolicies(tenantID, datasourceID string) error {
	defaultPolicies := []WorkflowABACPolicy{
		{
			ID:               uuid.New().String(),
			TenantID:         tenantID,
			DatasourceID:     datasourceID,
			WorkflowType:     "investment",
			Action:           "create",
			ResourcePattern:  "*",
			SubjectRules:     map[string]interface{}{"role": []string{"advisor", "portfolio_manager"}},
			EnvironmentRules: map[string]interface{}{"business_hours": true},
			RiskLevel:        "medium",
			RequiresApproval: false,
			TimeRestrictions: map[string]interface{}{"business_hours": true},
			Enabled:          true,
			CreatedAt:        time.Now(),
			UpdatedAt:        time.Now(),
		},
		{
			ID:               uuid.New().String(),
			TenantID:         tenantID,
			DatasourceID:     datasourceID,
			WorkflowType:     "compliance",
			Action:           "execute",
			ResourcePattern:  "*",
			SubjectRules:     map[string]interface{}{"role": []string{"compliance_officer", "senior_compliance"}},
			EnvironmentRules: map[string]interface{}{},
			RiskLevel:        "high",
			RequiresApproval: true,
			ApprovalRoles:    []string{"senior_compliance", "chief_compliance_officer"},
			TimeRestrictions: map[string]interface{}{},
			Enabled:          true,
			CreatedAt:        time.Now(),
			UpdatedAt:        time.Now(),
		},
		{
			ID:               uuid.New().String(),
			TenantID:         tenantID,
			DatasourceID:     datasourceID,
			WorkflowType:     "onboarding",
			Action:           "modify",
			ResourcePattern:  "*",
			SubjectRules:     map[string]interface{}{"role": []string{"client_services", "relationship_manager"}},
			EnvironmentRules: map[string]interface{}{},
			RiskLevel:        "low",
			RequiresApproval: false,
			TimeRestrictions: map[string]interface{}{},
			Enabled:          true,
			CreatedAt:        time.Now(),
			UpdatedAt:        time.Now(),
		},
	}

	for _, policy := range defaultPolicies {
		if err := wabe.db.Create(&policy).Error; err != nil {
			return fmt.Errorf("failed to create default policy: %w", err)
		}
	}

	wabe.logger.Infof("Initialized %d default workflow ABAC policies", len(defaultPolicies))
	return nil
}

// HTTP API for workflow ABAC evaluation
func (wabe *WorkflowABACEngine) EvaluateWorkflowPolicyHTTP(c *gin.Context) {
	var req WorkflowABACEvaluationRequest
	if err := c.BindJSON(&req); err != nil {
		c.JSON(400, gin.H{"error": err.Error()})
		return
	}

	allowed, reason, err := wabe.EvaluateWorkflowPolicy(c.Request.Context(), req)
	if err != nil {
		wabe.logger.Errorf("Workflow ABAC evaluation error: %v", err)
		c.JSON(500, gin.H{"error": "Evaluation failed"})
		return
	}

	c.JSON(200, gin.H{
		"allowed": allowed,
		"reason":  reason,
	})
}
