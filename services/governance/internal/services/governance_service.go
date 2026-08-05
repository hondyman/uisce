package services

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	abacclient "github.com/hondyman/uisce/libs/abac-client"
	sharedtypes "github.com/hondyman/uisce/libs/shared-types"
	temporalclient "github.com/hondyman/uisce/libs/temporal-client"
)

// GovernanceServiceConfig holds configuration for the governance service
type GovernanceServiceConfig struct {
	DB             *pgxpool.Pool
	TemporalClient *temporalclient.Client
	ABACClient     *abacclient.Client
}

// GovernanceService provides governance and policy management capabilities
type GovernanceService struct {
	config GovernanceServiceConfig
}

// NewGovernanceService creates a new governance service instance
func NewGovernanceService(config GovernanceServiceConfig) *GovernanceService {
	return &GovernanceService{
		config: config,
	}
}

func getDB() (*pgxpool.Pool, error) {
	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" {
		return nil, fmt.Errorf("DATABASE_URL environment variable is not set")
	}
	pool, err := pgxpool.New(context.Background(), dbURL)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to db: %w", err)
	}
	return pool, nil
}

// EvaluateAccess evaluates access control for a given request
func (s *GovernanceService) EvaluateAccess(ctx context.Context, request sharedtypes.AccessEvaluationRequest) (*sharedtypes.AccessEvaluationResponse, error) {
	// Convert to ABAC request
	abacReq := abacclient.ABACRequest{
		Subject:  request.UserID,
		Action:   request.Action,
		Resource: request.Resource,
		Context:  request.Context,
	}

	// Evaluate using ABAC client
	abacResp := s.config.ABACClient.Evaluate(ctx, abacReq)

	// Log the evaluation for audit purposes
	if err := s.logAccessEvaluation(ctx, request, abacResp); err != nil {
		// Log error but don't fail the evaluation
		fmt.Printf("Warning: failed to log audit entry: %v\n", err)
	}

	return &sharedtypes.AccessEvaluationResponse{
		Allowed:  abacResp.Allowed,
		Reason:   abacResp.Reason,
		Policies: abacResp.Policies,
	}, nil
}

// GetPolicies retrieves policies for a tenant from database
func (s *GovernanceService) GetPolicies(ctx context.Context, tenantID string) ([]sharedtypes.Policy, error) {
	if s.config.DB == nil {
		return nil, fmt.Errorf("database not configured")
	}

	rows, err := s.config.DB.Query(ctx, `
		SELECT id, name, description, effect, conditions, actions, created_at, updated_at
		FROM policies
		WHERE tenant_id = $1
		ORDER BY created_at DESC
	`, tenantID)
	if err != nil {
		return nil, fmt.Errorf("failed to query policies: %w", err)
	}
	defer rows.Close()

	policies := make([]sharedtypes.Policy, 0)
	for rows.Next() {
		var policy sharedtypes.Policy
		var conditionsJSON, actionsJSON []byte
		var createdAt, updatedAt time.Time

		err := rows.Scan(&policy.ID, &policy.Name, &policy.Description, &policy.Effect, &conditionsJSON, &actionsJSON, &createdAt, &updatedAt)
		if err != nil {
			return nil, fmt.Errorf("failed to scan policy: %w", err)
		}

		if err := json.Unmarshal(conditionsJSON, &policy.Conditions); err != nil {
			policy.Conditions = make(map[string]interface{})
		}
		if err := json.Unmarshal(actionsJSON, &policy.Actions); err != nil {
			policy.Actions = []string{}
		}

		policies = append(policies, policy)
	}

	return policies, nil
}

// CreatePolicy creates a new policy
func (s *GovernanceService) CreatePolicy(ctx context.Context, policy sharedtypes.Policy) (*sharedtypes.Policy, error) {
	if err := validatePolicy(policy); err != nil {
		return nil, fmt.Errorf("invalid policy: %w", err)
	}

	if s.config.DB == nil {
		return nil, fmt.Errorf("database not configured")
	}

	if policy.ID == "" {
		policy.ID = uuid.New().String()
	}

	conditionsJSON, _ := json.Marshal(policy.Conditions)
	actionsJSON, _ := json.Marshal(policy.Actions)

	_, err := s.config.DB.Exec(ctx, `
		INSERT INTO policies (id, tenant_id, name, description, effect, conditions, actions)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
	`, policy.ID, "", policy.Name, policy.Description, policy.Effect, conditionsJSON, actionsJSON)
	if err != nil {
		return nil, fmt.Errorf("failed to create policy: %w", err)
	}

	conditions := convertConditionsToABAC(policy.Conditions)
	abacPolicy := abacclient.Policy{
		ID:          policy.ID,
		Description: policy.Description,
		Effect:      policy.Effect,
		Conditions:  conditions,
	}
	s.config.ABACClient.AddPolicy(abacPolicy)

	return &policy, nil
}

// GetAuditLog retrieves audit entries for a tenant
func (s *GovernanceService) GetAuditLog(ctx context.Context, tenantID string, limit int) ([]sharedtypes.AuditEntry, error) {
	if s.config.DB == nil {
		return nil, fmt.Errorf("database not configured")
	}

	rows, err := s.config.DB.Query(ctx, `
		SELECT id, user_id, action, resource, result, timestamp, details
		FROM audit_entries
		WHERE tenant_id = $1
		ORDER BY timestamp DESC
		LIMIT $2
	`, tenantID, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to query audit log: %w", err)
	}
	defer rows.Close()

	entries := make([]sharedtypes.AuditEntry, 0)
	for rows.Next() {
		var entry sharedtypes.AuditEntry
		var result bool
		var timestamp time.Time
		var details string

		err := rows.Scan(&entry.ID, &entry.UserID, &entry.Action, &entry.Resource, &result, &timestamp, &details)
		if err != nil {
			return nil, fmt.Errorf("failed to scan audit entry: %w", err)
		}

		if result {
			entry.Result = "allow"
		} else {
			entry.Result = "deny"
		}
		entry.Timestamp = timestamp
		entry.Reason = details

		entries = append(entries, entry)
	}

	return entries, nil
}

// Helper functions

// logAccessEvaluation logs an access evaluation to the audit log
func (s *GovernanceService) logAccessEvaluation(ctx context.Context, request sharedtypes.AccessEvaluationRequest, response abacclient.ABACResponse) error {
	if s.config.DB == nil {
		return fmt.Errorf("database not configured")
	}

	policiesJSON, _ := json.Marshal(response.Policies)

	_, err := s.config.DB.Exec(ctx, `
		INSERT INTO audit_entries (user_id, tenant_id, action, resource, result, reason, policies, timestamp)
		VALUES ($1, $2, $3, $4, $5, $6, $7, NOW())
	`, request.UserID, "", request.Action, request.Resource, response.Allowed, response.Reason, string(policiesJSON))

	return err
}

// validatePolicy validates policy structure
func validatePolicy(policy sharedtypes.Policy) error {
	if policy.Name == "" {
		return fmt.Errorf("policy name is required")
	}

	if policy.Effect != "allow" && policy.Effect != "deny" {
		return fmt.Errorf("policy effect must be 'allow' or 'deny'")
	}

	if len(policy.Actions) == 0 {
		return fmt.Errorf("at least one action is required")
	}

	return nil
}

// convertConditionsToABAC converts map conditions to ABAC client format
func convertConditionsToABAC(conditions map[string]interface{}) []abacclient.Condition {
	abacConditions := make([]abacclient.Condition, 0)

	for key, value := range conditions {
		condition := abacclient.Condition{
			Attribute: key,
			Operator:  "equals",
			Value:     value,
		}
		abacConditions = append(abacConditions, condition)
	}

	return abacConditions
}
