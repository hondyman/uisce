package rules

import (
	"context"
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
