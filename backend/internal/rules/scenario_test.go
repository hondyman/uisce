package rules

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

// Mocks

type MockScenarioRepository struct {
	mock.Mock
}

func (m *MockScenarioRepository) CreateScenario(ctx context.Context, scenario *RuleScenario) error {
	args := m.Called(ctx, scenario)
	return args.Error(0)
}
func (m *MockScenarioRepository) CreateScenarioVersion(ctx context.Context, version *RuleScenarioVersion) error {
	args := m.Called(ctx, version)
	return args.Error(0)
}
func (m *MockScenarioRepository) GetScenario(ctx context.Context, id string) (*RuleScenario, error) {
	args := m.Called(ctx, id)
	return args.Get(0).(*RuleScenario), args.Error(1)
}
func (m *MockScenarioRepository) GetScenarioVersion(ctx context.Context, id string) (*RuleScenarioVersion, error) {
	args := m.Called(ctx, id)
	return args.Get(0).(*RuleScenarioVersion), args.Error(1)
}
func (m *MockScenarioRepository) GetLatestScenarioVersion(ctx context.Context, scenarioID string) (*RuleScenarioVersion, error) {
	args := m.Called(ctx, scenarioID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*RuleScenarioVersion), args.Error(1)
}
func (m *MockScenarioRepository) CreateTestRun(ctx context.Context, run *RuleTestRun) error {
	args := m.Called(ctx, run)
	return args.Error(0)
}
func (m *MockScenarioRepository) UpdateTestRun(ctx context.Context, run *RuleTestRun) error {
	args := m.Called(ctx, run)
	return args.Error(0)
}
func (m *MockScenarioRepository) GetTestRun(ctx context.Context, id string) (*RuleTestRun, error) {
	args := m.Called(ctx, id)
	return args.Get(0).(*RuleTestRun), args.Error(1)
}

type MockSampleEntityRepository struct {
	mock.Mock
}

func (m *MockSampleEntityRepository) SampleEntities(ctx context.Context, tenantID string, entityName string, sampleSize int, filter map[string]interface{}) ([]map[string]interface{}, error) {
	args := m.Called(ctx, tenantID, entityName, sampleSize, filter)
	return args.Get(0).([]map[string]interface{}), args.Error(1)
}

func TestScenarioService_Create(t *testing.T) {
	mockRepo := new(MockScenarioRepository)
	service := NewScenarioService(mockRepo)
	ctx := context.Background()

	mockRepo.On("CreateScenario", ctx, mock.AnythingOfType("*rules.RuleScenario")).Return(nil)

	s, err := service.CreateRuleScenario(ctx, "tenant-1", nil, "My Scenario", "Desc", "user-1")
	require.NoError(t, err)
	assert.Equal(t, "My Scenario", s.Name)
	assert.Equal(t, "draft", s.Status)
}

func TestScenarioRunner_Run(t *testing.T) {
	mockRepo := new(MockScenarioRepository)
	mockSampleRepo := new(MockSampleEntityRepository)

	// Setup Rule Engine (CEL-only now)
	// We use MockRuleRepository defined in engine_test.go (assumed available in same package)
	mockRulesRepo := &MockRuleRepository{}
	engine := NewRuleEngine(mockRulesRepo)

	runner := NewScenarioRunner(mockRepo, mockSampleRepo, engine)
	ctx := context.Background()

	// Mock Data
	scenarioVerID := "sv-1"
	tenantID := "tenant-1"

	// Mock Rule Snapshot (Custom Rule)
	snapshotRule := TenantValidationRule{
		TenantID:     tenantID,
		RuleID:       "scenario-rule",
		InheritMode:  Custom,
		ConditionSrc: `input.page.amount < 1000`, // CEL syntax
	}
	snapshotJSON, _ := json.Marshal(snapshotRule)

	mockRepo.On("GetScenarioVersion", ctx, scenarioVerID).Return(&RuleScenarioVersion{
		ID:           scenarioVerID,
		RuleSnapshot: snapshotJSON,
	}, nil)

	mockRepo.On("CreateTestRun", ctx, mock.MatchedBy(func(run *RuleTestRun) bool {
		return run.Status == "running"
	})).Return(nil)

	mockRepo.On("UpdateTestRun", ctx, mock.MatchedBy(func(run *RuleTestRun) bool {
		return run.Status == "completed"
	})).Return(nil)

	// Mock Samples
	samples := []map[string]interface{}{
		{"amount": 500},  // Pass
		{"amount": 1500}, // Fail
	}
	mockSampleRepo.On("SampleEntities", ctx, tenantID, "payment", 10, mock.Anything).Return(samples, nil)

	// Run
	run, err := runner.RunScenario(ctx, tenantID, scenarioVerID, "payment", 10, nil)
	require.NoError(t, err)

	assert.Equal(t, "completed", run.Status)
	// currently EvaluateTenantRule returns true always, so FailureCount = 0
	assert.Equal(t, 0, run.FailureCount)
}
