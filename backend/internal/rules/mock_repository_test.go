package rules

import (
	"context"
	"fmt"

	"github.com/google/uuid"
)

// MockRuleRepository is a no-op RuleRepository used by tests in this package.
// Kept here (rather than in engine_test.go) so scenario_test.go and any
// future test files can reuse it without duplicating the type.
type MockRuleRepository struct{}

func (m *MockRuleRepository) CreateRule(ctx context.Context, rule *ComplianceRule) error {
	return nil
}
func (m *MockRuleRepository) GetRule(ctx context.Context, id uuid.UUID) (*ComplianceRule, error) {
	return nil, fmt.Errorf("rule not found")
}
func (m *MockRuleRepository) ListRules(ctx context.Context, ruleType string) ([]ComplianceRule, error) {
	return []ComplianceRule{}, nil
}
func (m *MockRuleRepository) UpdateRule(ctx context.Context, rule *ComplianceRule) error {
	return nil
}
func (m *MockRuleRepository) DeleteRule(ctx context.Context, id uuid.UUID) error {
	return nil
}