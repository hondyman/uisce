package platform

import (
	"fmt"
	"strings"

	"github.com/hondyman/semlayer/backend/internal/models"
)

// PolicyService defines the interface for a minimal ABAC policy evaluation engine.
type PolicyService interface {
	// Can evaluates if a user has permission to perform an action on a resource based on a set of policies.
	Can(user models.User, action string, resource string, policies []models.Policy) (bool, error)
}

// NewPolicyService creates a new instance of the policy service.
func NewPolicyService() PolicyService {
	return &policyServiceImpl{}
}

// policyServiceImpl is the concrete implementation of the PolicyService.
type policyServiceImpl struct{}

// Can implements a simplified ABAC evaluation.
func (s *policyServiceImpl) Can(user models.User, action string, resource string, policies []models.Policy) (bool, error) {
	allowed := false

	for _, policy := range policies {
		if !s.resourceMatches(resource, policy.Resources) {
			continue
		}
		if !s.actionMatches(action, policy.Actions) {
			continue
		}

		matches, err := s.conditionsSatisfied(user, policy.Conditions)
		if err != nil {
			return false, err
		}
		if !matches {
			continue
		}

		switch strings.ToLower(policy.Effect) {
		case "deny":
			return false, fmt.Errorf("access denied by policy %s: %s", policy.ID, policy.Description)
		case "allow":
			allowed = true
		}
	}

	return allowed, nil
}

func (s *policyServiceImpl) resourceMatches(requested string, policyResources []string) bool {
	for _, pr := range policyResources {
		if pr == "*" || pr == requested {
			return true
		}
		if strings.HasSuffix(pr, "*") {
			prefix := strings.TrimSuffix(pr, "*")
			if strings.HasPrefix(requested, prefix) {
				return true
			}
		}
	}
	return false
}

func (s *policyServiceImpl) actionMatches(requested string, policyActions []string) bool {
	for _, pa := range policyActions {
		if pa == "*" || pa == requested {
			return true
		}
	}
	return false
}

func (s *policyServiceImpl) conditionsSatisfied(user models.User, conditions []models.AttributeCondition) (bool, error) {
	if len(conditions) == 0 {
		return true, nil
	}

	for _, cond := range conditions {
		values, present := s.userValuesForAttribute(user, cond.Attribute)
		if !s.evaluateCondition(values, present, cond) {
			return false, nil
		}
	}

	return true, nil
}

func (s *policyServiceImpl) userValuesForAttribute(user models.User, attribute string) ([]string, bool) {
	key := strings.TrimSpace(strings.ToLower(attribute))
	switch {
	case key == "id":
		if user.ID == "" {
			return nil, false
		}
		return []string{user.ID}, true
	case key == "email":
		if user.Email == "" {
			return nil, false
		}
		return []string{strings.ToLower(user.Email)}, true
	case key == "name":
		if user.Name == "" {
			return nil, false
		}
		return []string{user.Name}, true
	case key == "role":
		if user.Role == "" {
			return nil, false
		}
		return []string{user.Role}, true
	case key == "roles":
		roles := append([]string{}, user.Roles...)
		if len(roles) == 0 && user.Role != "" {
			roles = append(roles, user.Role)
		}
		if len(roles) == 0 {
			return nil, false
		}
		return roles, true
	case key == "permission", key == "permissions":
		if len(user.Permissions) == 0 {
			return nil, false
		}
		return append([]string{}, user.Permissions...), true
	case key == "organization":
		if user.Organization == "" {
			return nil, false
		}
		return []string{user.Organization}, true
	case key == "tenant", key == "tenant_id":
		if user.TenantID == "" {
			return nil, false
		}
		return []string{user.TenantID}, true
	case strings.HasPrefix(key, "attribute:"):
		attrKey := strings.TrimPrefix(key, "attribute:")
		if user.Attributes == nil {
			return nil, false
		}
		if value, ok := user.Attributes[attrKey]; ok && value != "" {
			return []string{value}, true
		}
		return nil, false
	case strings.HasPrefix(key, "attributes."):
		attrKey := strings.TrimPrefix(key, "attributes.")
		if user.Attributes == nil {
			return nil, false
		}
		if value, ok := user.Attributes[attrKey]; ok && value != "" {
			return []string{value}, true
		}
		return nil, false
	}
	return nil, false
}

func (s *policyServiceImpl) evaluateCondition(values []string, present bool, cond models.AttributeCondition) bool {
	operator := strings.TrimSpace(strings.ToLower(cond.Operator))
	if operator == "" {
		operator = "equals"
	}

	switch operator {
	case "equals", "in":
		if !present {
			return false
		}
		return s.anyMatch(values, cond.Values)
	case "not_equals", "not_in":
		if !present {
			return true
		}
		return !s.anyMatch(values, cond.Values)
	case "contains":
		if !present {
			return false
		}
		for _, target := range cond.Values {
			if !s.contains(values, target) {
				return false
			}
		}
		return true
	case "not_contains":
		if !present {
			return true
		}
		for _, target := range cond.Values {
			if s.contains(values, target) {
				return false
			}
		}
		return true
	case "any":
		return present && len(values) > 0
	case "empty":
		return !present || len(values) == 0
	default:
		// Unknown operators fail closed
		return false
	}
}

func (s *policyServiceImpl) anyMatch(values []string, targets []string) bool {
	for _, value := range values {
		for _, target := range targets {
			if strings.EqualFold(value, target) {
				return true
			}
		}
	}
	return false
}

func (s *policyServiceImpl) contains(values []string, target string) bool {
	for _, value := range values {
		if strings.EqualFold(value, target) {
			return true
		}
	}
	return false
}
