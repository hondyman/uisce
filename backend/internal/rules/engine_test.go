package rules

import (
	"context"
	"fmt"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// MockRuleRepository implements RuleRepository for testing
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

func TestRuleEngine(t *testing.T) {
	mockRepo := &MockRuleRepository{}

	// Simplified setup for CEL-only engine
	engine := NewRuleEngine(mockRepo)
	ctx := context.Background()

	t.Run("CEL Evaluation (Direct)", func(t *testing.T) {
		// Valid CEL
		res, err := engine.Evaluate(ctx, "input.val > 10", map[string]interface{}{"val": 20})
		require.NoError(t, err)
		assert.True(t, res)

		res, err = engine.Evaluate(ctx, "input.val > 10", map[string]interface{}{"val": 5})
		require.NoError(t, err)
		assert.False(t, res)
	})
}
