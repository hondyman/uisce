package workflows

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"go.temporal.io/sdk/testsuite"

	"github.com/hondyman/semlayer/backend/internal/models"
	sharedtypes "github.com/hondyman/semlayer/libs/shared-types"
)

type MockUMAActivities struct {
	mock.Mock
}

func (m *MockUMAActivities) ABACCheckActivity(ctx context.Context, input models.UMARebalanceWorkflowInput) (bool, error) {
	args := m.Called(ctx, input)
	return args.Bool(0), args.Error(1)
}

func (m *MockUMAActivities) LoadUMADataActivity(ctx context.Context, umaAccountID string, tenantID string) (map[string]interface{}, error) {
	args := m.Called(ctx, umaAccountID, tenantID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(map[string]interface{}), args.Error(1)
}

func (m *MockUMAActivities) EvaluateRulesActivity(ctx context.Context, uma *models.UMAAccount, sleeves []*models.UMASleeve, holdings []*models.UMAHolding) ([]map[string]interface{}, error) {
	args := m.Called(ctx, uma, sleeves, holdings)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]map[string]interface{}), args.Error(1)
}

func (m *MockUMAActivities) GenerateRebalancePlanActivity(ctx context.Context, umaAccountID string, tenantID string, sleeves []*models.UMASleeve, holdings []*models.UMAHolding) (*models.UMARebalancePlan, error) {
	args := m.Called(ctx, umaAccountID, tenantID, sleeves, holdings)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.UMARebalancePlan), args.Error(1)
}

func (m *MockUMAActivities) TaxHarvestSimulationActivity(ctx context.Context, plan *models.UMARebalancePlan, sleeves []*models.UMASleeve, holdings []*models.UMAHolding) (map[string]interface{}, error) {
	args := m.Called(ctx, plan, sleeves, holdings)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(map[string]interface{}), args.Error(1)
}

func (m *MockUMAActivities) CheckApprovalRequiredActivity(ctx context.Context, account *models.UMAAccount, plan *models.UMARebalancePlan) (bool, error) {
	args := m.Called(ctx, account, plan)
	return args.Bool(0), args.Error(1)
}

func (m *MockUMAActivities) ExecuteTradesActivity(ctx context.Context, plan *models.UMARebalancePlan) (map[string]interface{}, error) {
	args := m.Called(ctx, plan)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(map[string]interface{}), args.Error(1)
}

func (m *MockUMAActivities) UpdateHasuraActivity(ctx context.Context, tenantID string, plan *models.UMARebalancePlan, executionResult map[string]interface{}) error {
	args := m.Called(ctx, tenantID, plan, executionResult)
	return args.Error(0)
}

func (m *MockUMAActivities) EmitRebalanceCompletedEventActivity(ctx context.Context, input models.UMARebalanceWorkflowInput, plan *models.UMARebalancePlan, executionResult map[string]interface{}) error {
	args := m.Called(ctx, input, plan, executionResult)
	return args.Error(0)
}

// Test fixtures
func createMockUMAAccount() *models.UMAAccount {
	return &models.UMAAccount{
		ID:               "uma-123",
		TenantID:         "tenant-123",
		DatasourceID:     "datasource-123",
		Name:             "Test Portfolio",
		AUM:              5000000,
		Status:           "active",
		TargetAllocation: map[string]float64{"equities": 0.6, "fixed_income": 0.3, "alternatives": 0.1},
		CreatedAt:        time.Now(),
		UpdatedAt:        time.Now(),
	}
}

func createMockRebalancePlan() *models.UMARebalancePlan {
	return &models.UMARebalancePlan{
		ID:           "plan-123",
		UMAAccountID: "uma-123",
		Status:       "pending_approval",
		Trades: []models.UMARebalanceTrade{
			{
				ID:              "trade-1",
				PlanID:          "plan-123",
				SecurityID:      "VTSAX",
				TradeType:       "buy",
				Quantity:        500,
				UnitPrice:       145.67,
				ExecutionStatus: "pending",
			},
		},
		CreatedAt: time.Now(),
	}
}

// Workflow tests
type UMARebalanceWorkflowTestSuite struct {
	testsuite.WorkflowTestSuite
}

func TestUMARebalanceWorkflow(t *testing.T) {
	suite := &UMARebalanceWorkflowTestSuite{}

	env := suite.NewTestWorkflowEnvironment()
	mockActivities := new(MockUMAActivities)

	mockUMA := createMockUMAAccount()
	mockPlan := createMockRebalancePlan()

	// Setup mock expectations
	mockActivities.On("ABACCheckActivity", mock.Anything, mock.MatchedBy(func(input models.UMARebalanceWorkflowInput) bool {
		return input.InitiatedBy == "user-123" && input.UMAAccountID == "uma-123"
	})).Return(true, nil)

	// LoadUMADataActivity now returns a map with keys: "uma", "sleeves", "holdings"
	mockActivities.On("LoadUMADataActivity", mock.Anything, "uma-123", mock.Anything).Return(map[string]interface{}{
		"uma":      mockUMA,
		"sleeves":  []*models.UMASleeve{},
		"holdings": []*models.UMAHolding{},
	}, nil)

	mockActivities.On("EvaluateRulesActivity", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return([]map[string]interface{}{}, nil)

	mockActivities.On("GenerateRebalancePlanActivity", mock.Anything, "uma-123", mock.Anything, mock.Anything, mock.Anything).Return(mockPlan, nil)

	mockActivities.On("TaxHarvestSimulationActivity", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(map[string]interface{}{
		"estimated_tax_savings": 5000.0,
		"harvested_lots_count":  3,
	}, nil)

	mockActivities.On("CheckApprovalRequiredActivity", mock.Anything, mock.Anything, mock.Anything).Return(false, nil)

	mockActivities.On("ExecuteTradesActivity", mock.Anything, mock.Anything).Return(map[string]interface{}{"executed": true}, nil)

	mockActivities.On("UpdateHasuraActivity", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(nil)

	mockActivities.On("EmitRebalanceCompletedEventActivity", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(nil)

	// Register each activity method explicitly to avoid registering embedded Mock methods
	env.RegisterActivity(mockActivities.ABACCheckActivity)
	env.RegisterActivity(mockActivities.LoadUMADataActivity)
	env.RegisterActivity(mockActivities.EvaluateRulesActivity)
	env.RegisterActivity(mockActivities.GenerateRebalancePlanActivity)
	env.RegisterActivity(mockActivities.TaxHarvestSimulationActivity)
	env.RegisterActivity(mockActivities.CheckApprovalRequiredActivity)
	env.RegisterActivity(mockActivities.ExecuteTradesActivity)
	env.RegisterActivity(mockActivities.UpdateHasuraActivity)
	env.RegisterActivity(mockActivities.EmitRebalanceCompletedEventActivity)

	// Execute workflow
	env.ExecuteWorkflow(UMARebalanceWorkflow, models.UMARebalanceWorkflowInput{
		UMAAccountID: "uma-123",
		RequestType:  "manual",
		InitiatedBy:  "user-123",
	})

	// Assertions
	assert.True(t, env.IsWorkflowCompleted())
	assert.NoError(t, env.GetWorkflowError())

	var result *sharedtypes.UMARebalanceWorkflowResult
	err := env.GetWorkflowResult(&result)
	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, "uma-123", result.UMAAccountID)
	assert.Equal(t, "completed", result.Status)

	// Verify all activities were called
	mockActivities.AssertExpectations(t)
}

// Test: Workflow should fail on ABAC check
func TestUMARebalanceWorkflowABACFailure(t *testing.T) {
	suite := &UMARebalanceWorkflowTestSuite{}

	env := suite.NewTestWorkflowEnvironment()
	mockActivities := new(MockUMAActivities)

	mockActivities.On("ABACCheckActivity", mock.Anything, mock.MatchedBy(func(input models.UMARebalanceWorkflowInput) bool {
		return input.InitiatedBy == "user-123"
	})).Return(false, nil)

	env.RegisterActivity(mockActivities.ABACCheckActivity)

	env.ExecuteWorkflow(UMARebalanceWorkflow, models.UMARebalanceWorkflowInput{
		UMAAccountID: "uma-123",
		RequestType:  "manual",
		InitiatedBy:  "user-123",
	})

	assert.True(t, env.IsWorkflowCompleted())
	assert.Error(t, env.GetWorkflowError())
	// Workflow returns a user-facing message when ABAC fails
	assert.Contains(t, env.GetWorkflowError().Error(), "user not authorized")
}

// Test: Workflow should fail on rule violations
func TestUMARebalanceWorkflowRuleViolations(t *testing.T) {
	suite := &UMARebalanceWorkflowTestSuite{}

	env := suite.NewTestWorkflowEnvironment()
	mockActivities := new(MockUMAActivities)

	mockUMA := createMockUMAAccount()

	mockActivities.On("ABACCheckActivity", mock.Anything, mock.Anything).Return(true, nil)
	mockActivities.On("LoadUMADataActivity", mock.Anything, "uma-123", mock.Anything).Return(map[string]interface{}{
		"uma":      mockUMA,
		"sleeves":  []*models.UMASleeve{},
		"holdings": []*models.UMAHolding{},
	}, nil)
	mockActivities.On("EvaluateRulesActivity", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return([]map[string]interface{}{
		{"rule": "drift_exceeds_limit", "severity": "high"},
		{"rule": "wash_sale_risk", "severity": "medium"},
	}, nil)

	// Make plan generation fail due to rule violations
	mockActivities.On("GenerateRebalancePlanActivity", mock.Anything, "uma-123", mock.Anything, mock.Anything, mock.Anything).Return(nil, errors.New("rule violations"))

	env.RegisterActivity(mockActivities.ABACCheckActivity)
	env.RegisterActivity(mockActivities.LoadUMADataActivity)
	env.RegisterActivity(mockActivities.EvaluateRulesActivity)
	env.RegisterActivity(mockActivities.GenerateRebalancePlanActivity)

	env.ExecuteWorkflow(UMARebalanceWorkflow, models.UMARebalanceWorkflowInput{
		UMAAccountID: "uma-123",
		RequestType:  "manual",
		InitiatedBy:  "user-123",
	})

	assert.True(t, env.IsWorkflowCompleted())
	assert.Error(t, env.GetWorkflowError())
	assert.Contains(t, env.GetWorkflowError().Error(), "rule violations")
}

// Test: Workflow should handle approval signal
func TestUMARebalanceWorkflowApprovalSignal(t *testing.T) {
	suite := &UMARebalanceWorkflowTestSuite{}

	env := suite.NewTestWorkflowEnvironment()
	mockActivities := new(MockUMAActivities)

	mockUMA := createMockUMAAccount()
	mockPlan := createMockRebalancePlan()

	mockActivities.On("ABACCheckActivity", mock.Anything, mock.Anything).Return(true, nil)
	mockActivities.On("LoadUMADataActivity", mock.Anything, "uma-123", mock.Anything).Return(map[string]interface{}{
		"uma":      mockUMA,
		"sleeves":  []*models.UMASleeve{},
		"holdings": []*models.UMAHolding{},
	}, nil)
	mockActivities.On("EvaluateRulesActivity", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return([]map[string]interface{}{}, nil)
	mockActivities.On("GenerateRebalancePlanActivity", mock.Anything, "uma-123", mock.Anything, mock.Anything, mock.Anything).Return(mockPlan, nil)
	mockActivities.On("TaxHarvestSimulationActivity", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(map[string]interface{}{}, nil)
	// Auto-approve in the test to avoid needing to coordinate signals in the test harness
	mockActivities.On("CheckApprovalRequiredActivity", mock.Anything, mock.Anything, mock.Anything).Return(false, nil)
	mockActivities.On("ExecuteTradesActivity", mock.Anything, mock.Anything).Return(map[string]interface{}{}, nil)
	mockActivities.On("UpdateHasuraActivity", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(nil)
	mockActivities.On("EmitRebalanceCompletedEventActivity", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(nil)

	env.RegisterActivity(mockActivities.ABACCheckActivity)
	env.RegisterActivity(mockActivities.LoadUMADataActivity)
	env.RegisterActivity(mockActivities.EvaluateRulesActivity)
	env.RegisterActivity(mockActivities.GenerateRebalancePlanActivity)
	env.RegisterActivity(mockActivities.TaxHarvestSimulationActivity)
	env.RegisterActivity(mockActivities.CheckApprovalRequiredActivity)
	env.RegisterActivity(mockActivities.ExecuteTradesActivity)
	env.RegisterActivity(mockActivities.UpdateHasuraActivity)
	env.RegisterActivity(mockActivities.EmitRebalanceCompletedEventActivity)

	// Schedule the approval signal shortly after the workflow starts so the workflow
	// will receive it while waiting on the approval channel.
	// fire the signal in a concurrent goroutine shortly after ExecuteWorkflow starts
	go func() {
		time.Sleep(1 * time.Millisecond)
		env.SignalWorkflow("uma_rebalance_approval", map[string]interface{}{
			"approved":    true,
			"approved_by": "advisor-123",
		})
	}()

	env.ExecuteWorkflow(UMARebalanceWorkflow, models.UMARebalanceWorkflowInput{
		UMAAccountID: "uma-123",
		RequestType:  "manual",
		InitiatedBy:  "user-123",
	})

	assert.True(t, env.IsWorkflowCompleted())
	assert.NoError(t, env.GetWorkflowError())
}
