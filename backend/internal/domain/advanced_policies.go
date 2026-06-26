package domain

import (
	"context"
	"encoding/json"
	"fmt"
	"regexp"
	"strings"
	"time"
)

// AdvancedPolicyEngine supports complex policy rules with conditions
type AdvancedPolicyEngine struct {
	Repo PolicyRepo
}

// PolicyCondition represents a condition in a policy rule
type PolicyCondition struct {
	Field    string      `json:"field"`
	Operator string      `json:"operator"`
	Value    interface{} `json:"value"`
}

// AdvancedPolicyRule represents an advanced policy rule with complex conditions
type AdvancedPolicyRule struct {
	ID          string            `json:"id"`
	Name        string            `json:"name"`
	Description string            `json:"description"`
	Priority    int               `json:"priority"`
	Effect      string            `json:"effect"` // "allow" or "deny"
	Conditions  []PolicyCondition `json:"conditions"`
	Actions     []string          `json:"actions"`
	Resources   []string          `json:"resources"`
	Users       []string          `json:"users"`
	Tenants     []string          `json:"tenants"`
	Enabled     bool              `json:"enabled"`
	CreatedAt   time.Time         `json:"created_at"`
	UpdatedAt   time.Time         `json:"updated_at"`
}

// EvaluatePolicy evaluates a policy rule against a request
func (ape *AdvancedPolicyEngine) EvaluatePolicy(ctx context.Context, rule AdvancedPolicyRule, req EvaluationRequest) (bool, string, error) {
	// Check if policy is enabled
	if !rule.Enabled {
		return false, "policy disabled", nil
	}

	// Check tenant match
	if !ape.matchesAny(rule.Tenants, req.TenantID) && len(rule.Tenants) > 0 {
		return false, "tenant not in allowed list", nil
	}

	// Check user match
	if !ape.matchesAny(rule.Users, req.UserID) && len(rule.Users) > 0 {
		return false, "user not in allowed list", nil
	}

	// Check resource match
	if !ape.matchesAny(rule.Resources, req.AssetID) && len(rule.Resources) > 0 {
		return false, "resource not in allowed list", nil
	}

	// Check action match
	if !ape.matchesAny(rule.Actions, string(req.Action)) && len(rule.Actions) > 0 {
		return false, "action not in allowed list", nil
	}

	// Evaluate conditions (for advanced policies with structured conditions)
	for _, condition := range rule.Conditions {
		matches, err := ape.evaluateCondition(condition, req)
		if err != nil {
			return false, fmt.Sprintf("condition evaluation error: %v", err), err
		}
		if !matches {
			return false, fmt.Sprintf("condition not met: %s", condition.Field), nil
		}
	}

	return true, "policy matches", nil
}

// evaluateCondition evaluates a single condition
func (ape *AdvancedPolicyEngine) evaluateCondition(cond PolicyCondition, req EvaluationRequest) (bool, error) {
	var fieldValue interface{}

	// Extract field value from request
	switch cond.Field {
	case "user_id":
		fieldValue = req.UserID
	case "tenant_id":
		fieldValue = req.TenantID
	case "asset_id":
		fieldValue = req.AssetID
	case "action":
		fieldValue = string(req.Action)
	case "context.time":
		if req.Context != nil {
			fieldValue = req.Context["time"]
		}
	default:
		// Check context for custom fields
		if req.Context != nil {
			fieldValue = req.Context[cond.Field]
		}
	}

	return ape.compareValues(fieldValue, cond.Operator, cond.Value)
}

// compareValues compares two values using the specified operator
func (ape *AdvancedPolicyEngine) compareValues(fieldValue interface{}, operator string, expectedValue interface{}) (bool, error) {
	switch operator {
	case "equals", "eq":
		return fmt.Sprintf("%v", fieldValue) == fmt.Sprintf("%v", expectedValue), nil
	case "not_equals", "ne":
		return fmt.Sprintf("%v", fieldValue) != fmt.Sprintf("%v", expectedValue), nil
	case "contains":
		fieldStr := fmt.Sprintf("%v", fieldValue)
		expectedStr := fmt.Sprintf("%v", expectedValue)
		return strings.Contains(fieldStr, expectedStr), nil
	case "not_contains":
		fieldStr := fmt.Sprintf("%v", fieldValue)
		expectedStr := fmt.Sprintf("%v", expectedValue)
		return !strings.Contains(fieldStr, expectedStr), nil
	case "regex":
		expectedStr := fmt.Sprintf("%v", expectedValue)
		fieldStr := fmt.Sprintf("%v", fieldValue)
		matched, err := regexp.MatchString(expectedStr, fieldStr)
		return matched, err
	case "in":
		// Check if fieldValue is in the expected array
		if expectedSlice, ok := expectedValue.([]interface{}); ok {
			fieldStr := fmt.Sprintf("%v", fieldValue)
			for _, v := range expectedSlice {
				if fieldStr == fmt.Sprintf("%v", v) {
					return true, nil
				}
			}
		}
		return false, nil
	default:
		return false, fmt.Errorf("unsupported operator: %s", operator)
	}
}

// matchesAny checks if the target matches any of the patterns (supports wildcards)
func (ape *AdvancedPolicyEngine) matchesAny(patterns []string, target string) bool {
	for _, pattern := range patterns {
		if ape.matchesPattern(pattern, target) {
			return true
		}
	}
	return false
}

// matchesPattern checks if target matches pattern (supports * wildcard)
func (ape *AdvancedPolicyEngine) matchesPattern(pattern, target string) bool {
	if pattern == "*" || pattern == target {
		return true
	}

	// Simple wildcard support
	if strings.Contains(pattern, "*") {
		regex := strings.ReplaceAll(regexp.QuoteMeta(pattern), "\\*", ".*")
		matched, _ := regexp.MatchString("^"+regex+"$", target)
		return matched
	}

	return false
}

// Example policy rules as JSON
const (
	TimeBasedAccessPolicy = `{
		"id": "time_based_access",
		"name": "Time-based Access Control",
		"description": "Allow access only during business hours",
		"priority": 10,
		"effect": "allow",
		"conditions": [
			{
				"field": "context.time",
				"operator": "regex",
				"value": "^(0[9]|1[0-7]):" 
			}
		],
		"actions": ["read", "write"],
		"enabled": true
	}`

	SensitiveDataPolicy = `{
		"id": "sensitive_data_protection",
		"name": "Sensitive Data Protection",
		"description": "Restrict access to sensitive data fields",
		"priority": 20,
		"effect": "deny",
		"resources": ["*sensitive*", "*pii*"],
		"actions": ["read", "write"],
		"users": ["contractor_*"],
		"enabled": true
	}`
)

// ParsePolicyFromJSON parses a policy from JSON string
func ParsePolicyFromJSON(jsonStr string) (*AdvancedPolicyRule, error) {
	var rule AdvancedPolicyRule
	err := json.Unmarshal([]byte(jsonStr), &rule)
	return &rule, err
}
