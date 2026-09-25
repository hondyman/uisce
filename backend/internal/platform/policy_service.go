package platform

import (
	"fmt"
	"strings"
	"unicode"

	"github.com/hondyman/uisce/backend/internal/models"
	vm "github.com/hondyman/uisce/backend/internal/rules/vm"
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

// evaluateCondition checks one attribute condition with the rule engine.
// The user's values are the field "values" (absent when the user has no such
// attribute). Matching is case-insensitive (strings.EqualFold semantics),
// and an unknown operator or an evaluation error fails closed.
func (s *policyServiceImpl) evaluateCondition(values []string, present bool, cond models.AttributeCondition) bool {
	operator := strings.TrimSpace(strings.ToLower(cond.Operator))
	if operator == "" {
		operator = "equals"
	}
	targets := foldAll(cond.Values)

	var node vm.RuleNode
	switch operator {
	case "equals", "in":
		node = attrCond("contains_any", targets)
	case "not_equals", "not_in", "not_contains":
		node = vm.RuleNode{Type: vm.NodeTypeGroup, Group: &vm.RuleGroup{Operator: "OR", Conditions: []vm.RuleNode{
			attrCond("is_null", nil),
			{Type: vm.NodeTypeGroup, Group: &vm.RuleGroup{Operator: "NOT", Conditions: []vm.RuleNode{attrCond("contains_any", targets)}}},
		}}}
	case "contains":
		node = attrCond("contains_all", targets)
	case "any":
		node = attrCond("is_not_empty", nil)
	case "empty":
		node = attrCond("is_empty", nil)
	default:
		return false
	}

	data := map[string]interface{}{}
	if present {
		data["values"] = foldAll(values)
	}
	ok, err := vm.NewAdvancedEvaluator().Evaluate(node, data)
	return err == nil && ok
}

func attrCond(op string, value interface{}) vm.RuleNode {
	return vm.RuleNode{Type: vm.NodeTypeCondition, Condition: &vm.RuleCondition{
		Field: "values", FieldPath: "values", Operator: op, Value: value,
	}}
}

// foldAll maps each string to its case-fold key: two strings have the same
// key exactly when strings.EqualFold reports them equal.
func foldAll(xs []string) []interface{} {
	out := make([]interface{}, len(xs))
	for i, x := range xs {
		out[i] = foldKey(x)
	}
	return out
}

func foldKey(s string) string {
	return strings.Map(func(r rune) rune {
		min := r
		for f := unicode.SimpleFold(r); f != r; f = unicode.SimpleFold(f) {
			if f < min {
				min = f
			}
		}
		return min
	}, s)
}
