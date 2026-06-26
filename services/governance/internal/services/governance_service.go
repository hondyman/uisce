package services

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	abacclient "github.com/hondyman/semlayer/libs/abac-client"
	hasuraclient "github.com/hondyman/semlayer/libs/hasura-client"
	sharedtypes "github.com/hondyman/semlayer/libs/shared-types"
	temporalclient "github.com/hondyman/semlayer/libs/temporal-client"
)

// GovernanceServiceConfig holds configuration for the governance service
type GovernanceServiceConfig struct {
	HasuraClient   *hasuraclient.HasuraClient
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

// GetPolicies retrieves policies for a tenant from Hasura
func (s *GovernanceService) GetPolicies(ctx context.Context, tenantID string) ([]sharedtypes.Policy, error) {
	if s.config.HasuraClient == nil {
		return nil, fmt.Errorf("Hasura client not configured")
	}

	// Query Hasura for policies
	query := `
		query GetPolicies($tenantId: String!) {
			policies(
				where: { tenant_id: { _eq: $tenantId } }
				order_by: { created_at: desc }
			) {
				id
				name
				description
				effect
				conditions
				actions
				created_at
				updated_at
			}
		}
	`

	result, err := s.config.HasuraClient.Query(query, map[string]interface{}{
		"tenantId": tenantID,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to query policies: %w", err)
	}

	// Parse response
	policies := make([]sharedtypes.Policy, 0)
	if data, ok := result["policies"].([]interface{}); ok {
		for _, item := range data {
			if policyData, ok := item.(map[string]interface{}); ok {
				policy := sharedtypes.Policy{
					ID:          getString(policyData, "id"),
					Name:        getString(policyData, "name"),
					Description: getString(policyData, "description"),
					Effect:      getString(policyData, "effect"),
				}

				// Parse conditions (stored as JSONB in Hasura)
				if conditions, ok := policyData["conditions"].(map[string]interface{}); ok {
					policy.Conditions = conditions
				} else {
					policy.Conditions = make(map[string]interface{})
				}

				// Parse actions array
				if actions, ok := policyData["actions"].([]interface{}); ok {
					policy.Actions = make([]string, 0, len(actions))
					for _, action := range actions {
						if actionStr, ok := action.(string); ok {
							policy.Actions = append(policy.Actions, actionStr)
						}
					}
				} else {
					policy.Actions = []string{}
				}

				policies = append(policies, policy)
			}
		}
	}

	return policies, nil
}

// CreatePolicy creates a new policy
func (s *GovernanceService) CreatePolicy(ctx context.Context, policy sharedtypes.Policy) (*sharedtypes.Policy, error) {
	// Validate policy structure
	if err := validatePolicy(policy); err != nil {
		return nil, fmt.Errorf("invalid policy: %w", err)
	}

	if s.config.HasuraClient == nil {
		return nil, fmt.Errorf("Hasura client not configured")
	}

	// Generate ID if not provided
	if policy.ID == "" {
		policy.ID = fmt.Sprintf("pol_%d", time.Now().UnixNano())
	}

	// Store in Hasura GraphQL
	mutation := `
		mutation CreatePolicy(
			$id: String!,
			$name: String!,
			$description: String!,
			$effect: String!,
			$conditions: jsonb!,
			$actions: jsonb!
		) {
			insert_policies_one(
				object: {
					id: $id,
					name: $name,
					description: $description,
					effect: $effect,
					conditions: $conditions,
					actions: $actions
				}
			) {
				id
				name
				created_at
			}
		}
	`

	variables := map[string]interface{}{
		"id":          policy.ID,
		"name":        policy.Name,
		"description": policy.Description,
		"effect":      policy.Effect,
		"conditions":  policy.Conditions,
		"actions":     policy.Actions,
	}

	_, err := s.config.HasuraClient.Query(mutation, variables)
	if err != nil {
		return nil, fmt.Errorf("failed to create policy in Hasura: %w", err)
	}

	// Also add to ABAC client for in-memory evaluation
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
	// Query Hasura for audit entries
	query := `
		query GetAuditLog($tenantId: String!, $limit: Int!) {
			audit_entries(
				where: { tenant_id: { _eq: $tenantId } }
				order_by: { timestamp: desc }
				limit: $limit
			) {
				id
				user_id
				action
				resource
				result
				timestamp
				details
			}
		}
	`

	result, err := s.config.HasuraClient.Query(query, map[string]interface{}{
		"tenantId": tenantID,
		"limit":    limit,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to query audit log: %w", err)
	}

	// Parse response
	entries := make([]sharedtypes.AuditEntry, 0)
	if data, ok := result["audit_entries"].([]interface{}); ok {
		for _, item := range data {
			if entryData, ok := item.(map[string]interface{}); ok {
				result := "unknown"
				if res, ok := entryData["result"].(bool); ok {
					if res {
						result = "allow"
					} else {
						result = "deny"
					}
				}

				timestamp := time.Now()
				if ts, ok := entryData["timestamp"].(string); ok {
					if parsed, err := time.Parse(time.RFC3339, ts); err == nil {
						timestamp = parsed
					}
				}

				entry := sharedtypes.AuditEntry{
					ID:        entryData["id"].(string),
					UserID:    entryData["user_id"].(string),
					Action:    entryData["action"].(string),
					Resource:  entryData["resource"].(string),
					Result:    result,
					Timestamp: timestamp,
					Reason:    entryData["details"].(string),
				}
				entries = append(entries, entry)
			}
		}
	}

	return entries, nil
}

// Helper functions

// logAccessEvaluation logs an access evaluation to the audit log
func (s *GovernanceService) logAccessEvaluation(ctx context.Context, request sharedtypes.AccessEvaluationRequest, response abacclient.ABACResponse) error {
	if s.config.HasuraClient == nil {
		return fmt.Errorf("Hasura client not configured")
	}

	mutation := `
		mutation LogAuditEntry(
			$userId: String!,
			$action: String!,
			$resource: String!,
			$result: Boolean!,
			$reason: String!,
			$policies: jsonb!
		) {
			insert_audit_entries_one(
				object: {
					user_id: $userId,
					action: $action,
					resource: $resource,
					result: $result,
					reason: $reason,
					policies: $policies,
					timestamp: "now()"
				}
			) {
				id
			}
		}
	`

	policiesJSON, _ := json.Marshal(response.Policies)
	variables := map[string]interface{}{
		"userId":   request.UserID,
		"action":   request.Action,
		"resource": request.Resource,
		"result":   response.Allowed,
		"reason":   response.Reason,
		"policies": string(policiesJSON),
	}

	_, err := s.config.HasuraClient.Query(mutation, variables)
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

// getString safely extracts string from map
func getString(m map[string]interface{}, key string) string {
	if val, ok := m[key].(string); ok {
		return val
	}
	return ""
}
