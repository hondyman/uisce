// Package abacclient provides ABAC (Attribute-Based Access Control) functionality
package abacclient

import (
	"context"
	"fmt"
)

// ABACRequest represents an access control request
type ABACRequest struct {
	Subject  string
	Action   string
	Resource string
	Context  map[string]interface{}
}

// ABACResponse represents an access control response
type ABACResponse struct {
	Allowed  bool
	Reason   string
	Policies []string
}

// Client provides ABAC policy evaluation
type Client struct {
	policies []Policy
}

// Policy represents an ABAC policy
type Policy struct {
	ID          string
	Description string
	Effect      string // "allow" or "deny"
	Conditions  []Condition
}

// Condition represents a policy condition
type Condition struct {
	Attribute string
	Operator  string
	Value     interface{}
}

// NewClient creates a new ABAC client
func NewClient() *Client {
	return &Client{
		policies: loadDefaultPolicies(),
	}
}

// AddPolicy adds a new policy to the client
func (c *Client) AddPolicy(policy Policy) {
	c.policies = append(c.policies, policy)
}

// RemovePolicy removes a policy by ID
func (c *Client) RemovePolicy(policyID string) {
	for i, policy := range c.policies {
		if policy.ID == policyID {
			c.policies = append(c.policies[:i], c.policies[i+1:]...)
			break
		}
	}
}

// ListPolicies returns all policies
func (c *Client) ListPolicies() []Policy {
	return c.policies
}

// EvaluateWithContext evaluates an access request with additional context
func (c *Client) EvaluateWithContext(ctx context.Context, req ABACRequest, additionalContext map[string]interface{}) ABACResponse {
	// Merge additional context
	mergedContext := make(map[string]interface{})
	for k, v := range req.Context {
		mergedContext[k] = v
	}
	for k, v := range additionalContext {
		mergedContext[k] = v
	}

	req.Context = mergedContext
	return c.Evaluate(ctx, req)
}

// BatchEvaluate evaluates multiple access requests in batch
func (c *Client) BatchEvaluate(ctx context.Context, requests []ABACRequest) []ABACResponse {
	responses := make([]ABACResponse, len(requests))
	for i, req := range requests {
		responses[i] = c.Evaluate(ctx, req)
	}
	return responses
}

// GetPolicy retrieves a policy by ID
func (c *Client) GetPolicy(policyID string) (Policy, error) {
	for _, policy := range c.policies {
		if policy.ID == policyID {
			return policy, nil
		}
	}
	return Policy{}, fmt.Errorf("policy with ID %s not found", policyID)
}

// UpdatePolicy updates an existing policy
func (c *Client) UpdatePolicy(policyID string, updatedPolicy Policy) error {
	for i, policy := range c.policies {
		if policy.ID == policyID {
			updatedPolicy.ID = policyID // Ensure ID remains the same
			c.policies[i] = updatedPolicy
			return nil
		}
	}
	return fmt.Errorf("policy with ID %s not found", policyID)
}

// ClearPolicies removes all policies
func (c *Client) ClearPolicies() {
	c.policies = []Policy{}
}

// CountPolicies returns the number of policies
func (c *Client) CountPolicies() int {
	return len(c.policies)
}

// EvaluateBulk evaluates multiple requests with shared context
func (c *Client) EvaluateBulk(ctx context.Context, requests []ABACRequest, sharedContext map[string]interface{}) []ABACResponse {
	responses := make([]ABACResponse, len(requests))
	for i, req := range requests {
		responses[i] = c.EvaluateWithContext(ctx, req, sharedContext)
	}
	return responses
}

// ValidateCondition validates a condition structure
func ValidateCondition(condition Condition) error {
	if condition.Attribute == "" {
		return fmt.Errorf("condition attribute cannot be empty")
	}
	if condition.Operator == "" {
		return fmt.Errorf("condition operator cannot be empty")
	}
	validOperators := []string{"equals", "contains", "in", "not_equals", "greater_than", "less_than"}
	valid := false
	for _, op := range validOperators {
		if condition.Operator == op {
			valid = true
			break
		}
	}
	if !valid {
		return fmt.Errorf("invalid operator: %s", condition.Operator)
	}
	return nil
}

// ClonePolicy creates a copy of a policy with a new ID
func ClonePolicy(original Policy, newID string) Policy {
	conditions := make([]Condition, len(original.Conditions))
	copy(conditions, original.Conditions)

	return Policy{
		ID:          newID,
		Description: original.Description,
		Effect:      original.Effect,
		Conditions:  conditions,
	}
}

// Evaluate evaluates an access request against policies
func (c *Client) Evaluate(ctx context.Context, req ABACRequest) ABACResponse {
	for _, policy := range c.policies {
		if c.matchesPolicy(req, policy) {
			if policy.Effect == "allow" {
				return ABACResponse{
					Allowed:  true,
					Policies: []string{policy.ID},
				}
			}
			return ABACResponse{
				Allowed:  false,
				Reason:   fmt.Sprintf("Policy %s denies access", policy.ID),
				Policies: []string{policy.ID},
			}
		}
	}

	return ABACResponse{
		Allowed: false,
		Reason:  "No matching policy found",
	}
}

// matchesPolicy checks if a request matches a policy
func (c *Client) matchesPolicy(req ABACRequest, policy Policy) bool {
	for _, condition := range policy.Conditions {
		if !c.evaluateCondition(req, condition) {
			return false
		}
	}
	return true
}

// evaluateCondition evaluates a single condition
func (c *Client) evaluateCondition(req ABACRequest, condition Condition) bool {
	var attributeValue interface{}

	// Extract attribute value from request
	switch condition.Attribute {
	case "subject":
		attributeValue = req.Subject
	case "action":
		attributeValue = req.Action
	case "resource":
		attributeValue = req.Resource
	default:
		// Check context
		if req.Context != nil {
			attributeValue = req.Context[condition.Attribute]
		}
	}

	// Evaluate based on operator
	switch condition.Operator {
	case "equals":
		return attributeValue == condition.Value
	case "not_equals":
		return attributeValue != condition.Value
	case "contains":
		if str, ok := attributeValue.(string); ok {
			if val, ok := condition.Value.(string); ok {
				return containsString(str, val)
			}
		}
	case "in":
		if arr, ok := condition.Value.([]interface{}); ok {
			for _, v := range arr {
				if attributeValue == v {
					return true
				}
			}
		}
	case "greater_than":
		if av, ok := attributeValue.(float64); ok {
			if cv, ok := condition.Value.(float64); ok {
				return av > cv
			}
		}
	case "less_than":
		if av, ok := attributeValue.(float64); ok {
			if cv, ok := condition.Value.(float64); ok {
				return av < cv
			}
		}
	}

	return false
}

// containsString checks if a string contains a substring
func containsString(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || containsString(s[1:], substr))
}

// loadDefaultPolicies loads default ABAC policies for wealth management
func loadDefaultPolicies() []Policy {
	return []Policy{
		{
			ID:          "wealth-management-admin",
			Description: "Allow admins full access to wealth management",
			Effect:      "allow",
			Conditions: []Condition{
				{Attribute: "subject", Operator: "contains", Value: "admin"},
				{Attribute: "resource", Operator: "contains", Value: "wealth"},
			},
		},
		{
			ID:          "wealth-management-trader",
			Description: "Allow traders to execute trades",
			Effect:      "allow",
			Conditions: []Condition{
				{Attribute: "subject", Operator: "contains", Value: "trader"},
				{Attribute: "action", Operator: "equals", Value: "execute"},
				{Attribute: "resource", Operator: "contains", Value: "trade"},
			},
		},
		{
			ID:          "wealth-management-readonly",
			Description: "Allow read-only access to analysts",
			Effect:      "allow",
			Conditions: []Condition{
				{Attribute: "subject", Operator: "contains", Value: "analyst"},
				{Attribute: "action", Operator: "equals", Value: "read"},
			},
		},
		{
			ID:          "deny-high-risk",
			Description: "Deny high-risk operations during market hours",
			Effect:      "deny",
			Conditions: []Condition{
				{Attribute: "action", Operator: "equals", Value: "high_risk"},
				{Attribute: "market_hours", Operator: "equals", Value: true},
			},
		},
		{
			ID:          "semantic-model-access",
			Description: "Allow semantic model calculations for authenticated users",
			Effect:      "allow",
			Conditions: []Condition{
				{Attribute: "action", Operator: "equals", Value: "calculate"},
				{Attribute: "resource", Operator: "equals", Value: "semantic_model"},
			},
		},
	}
}
