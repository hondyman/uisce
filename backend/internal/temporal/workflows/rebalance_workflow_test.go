package workflows_test

import (
	"testing"
	"time"

	"github.com/hondyman/semlayer/backend/internal/optimizer"
	"github.com/hondyman/semlayer/backend/internal/temporal/activities"
	"github.com/hondyman/semlayer/backend/internal/temporal/workflows"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"go.temporal.io/sdk/testsuite"
)

func TestRebalanceWorkflow_FullPath(t *testing.T) {
	testSuite := &testsuite.WorkflowTestSuite{}
	env := testSuite.NewTestWorkflowEnvironment()

	// Mock activity implementations
	env.OnActivity((*activities.RebalanceActivities).CheckDriftActivity, mock.Anything, mock.Anything, mock.Anything).Return(
		&activities.CheckDriftOutput{
			DriftDetected: true,
			DriftReport: optimizer.DriftReport{
				PortfolioID:   "port-001",
				TenantID:      "tenant-001",
				DriftPercent:  0.05,
				TrackingError: 0.02,
				ComputedAt:    time.Now(),
				Exposures: []optimizer.Exposure{
					{Symbol: "SPY", CurrentWgt: 0.65, TargetWgt: 0.60, MarketValue: 650000, DriftPercent: 0.05},
				},
			},
		}, nil)

	env.OnActivity((*activities.RebalanceActivities).LoadLotsActivity, mock.Anything, mock.Anything, mock.Anything).Return(
		&activities.LoadLotsOutput{
			Lots: []optimizer.Lot{
				{LotID: "lot-001", Symbol: "AAPL", Quantity: 100, CostBasis: 15000, UnrealizedPnL: 3500, Term: "long"},
			},
		}, nil)

	env.OnActivity((*activities.RebalanceActivities).LoadPricesActivity, mock.Anything, mock.Anything, mock.Anything).Return(
		&activities.LoadPricesOutput{
			Prices: map[string]float64{"AAPL": 185.00, "SPY": 450.00},
		}, nil)

	env.OnActivity((*activities.RebalanceActivities).LoadTaxRulesActivity, mock.Anything, mock.Anything, mock.Anything).Return(
		&activities.LoadTaxRulesOutput{
			TaxRules: optimizer.TaxRules{
				ShortTermRate: 0.37,
				LongTermRate:  0.20,
				WashSaleDays:  30,
			},
		}, nil)

	env.OnActivity((*activities.RebalanceActivities).TaxAwareOptimizeActivity, mock.Anything, mock.Anything, mock.Anything).Return(
		&activities.TaxAwareOptimizeOutput{
			Plan: optimizer.Plan{
				ID:          "prop-001",
				PortfolioID: "port-001",
				TenantID:    "tenant-001",
				GeneratedAt: time.Now(),
				Trades: []optimizer.CandidateTrade{
					{Side: "SELL", Symbol: "AAPL", Qty: 10, EstValue: 1850},
				},
				TaxImpact: 500,
			},
		}, nil)

	env.OnActivity((*activities.RebalanceActivities).MonteCarloSimActivity, mock.Anything, mock.Anything, mock.Anything).Return(
		&activities.MonteCarloSimOutput{
			Summary: optimizer.MonteCarloSummary{
				MeanTaxImpact:   500,
				MedianTaxImpact: 480,
				Pct05:           300,
				Pct95:           700,
				Runs:            1000,
			},
		}, nil)

	env.OnActivity((*activities.RebalanceActivities).PolicyCheckActivity, mock.Anything, mock.Anything, mock.Anything).Return(
		&activities.PolicyCheckOutput{
			Passed:     true,
			Violations: []activities.PolicyViolation{},
		}, nil)

	env.OnActivity((*activities.RebalanceActivities).NotifyAdvisorActivity, mock.Anything, mock.Anything, mock.Anything).Return(
		&activities.NotifyAdvisorOutput{
			NotificationID: "notif-001",
			SentAt:         time.Now(),
		}, nil)

	env.OnActivity((*activities.RebalanceActivities).ExecuteTradeSagaActivity, mock.Anything, mock.Anything, mock.Anything).Return(
		&activities.ExecuteTradeSagaOutput{
			SagaID:      "saga-001",
			Status:      "completed",
			Executions:  []activities.TradeExecution{{TradeID: "trade-001", SecurityID: "AAPL", Status: "executed"}},
			CompletedAt: time.Now(),
		}, nil)

	env.OnActivity((*activities.RebalanceActivities).PersistUARActivity, mock.Anything, mock.Anything, mock.Anything).Return(
		&activities.PersistUAROutput{
			UARID:     "uar-001",
			CreatedAt: time.Now(),
		}, nil)

	// Send approval signal after a short delay
	env.RegisterDelayedCallback(func() {
		env.SignalWorkflow("AdvisorApproval", optimizer.AdvisorDecision{
			Action:    "approve",
			AdvisorID: "advisor-001",
			Rationale: "Looks good",
			Timestamp: time.Now(),
		})
	}, 100*time.Millisecond)

	// Execute workflow
	env.ExecuteWorkflow(workflows.RebalanceWorkflow, workflows.RebalanceInput{
		TenantID:    "tenant-001",
		PortfolioID: "port-001",
		AccountID:   "acct-001",
		AdvisorID:   "advisor-001",
		Threshold:   0.03,
		NumRuns:     1000,
	})

	require.True(t, env.IsWorkflowCompleted())
	require.NoError(t, env.GetWorkflowError())

	var result workflows.RebalanceOutput
	require.NoError(t, env.GetWorkflowResult(&result))
	require.Equal(t, "approve", result.Decision)
	require.NotNil(t, result.SagaResult)
	require.Equal(t, "completed", result.SagaResult.Status)
}

func TestRebalanceWorkflow_NoDrift(t *testing.T) {
	testSuite := &testsuite.WorkflowTestSuite{}
	env := testSuite.NewTestWorkflowEnvironment()

	// Mock check drift returning no significant drift
	env.OnActivity((*activities.RebalanceActivities).CheckDriftActivity, mock.Anything, mock.Anything, mock.Anything).Return(
		&activities.CheckDriftOutput{
			DriftDetected: false,
			DriftReport:   optimizer.DriftReport{DriftPercent: 0.01},
		}, nil)

	env.ExecuteWorkflow(workflows.RebalanceWorkflow, workflows.RebalanceInput{
		TenantID:    "tenant-001",
		PortfolioID: "port-001",
		Threshold:   0.03,
	})

	require.True(t, env.IsWorkflowCompleted())
	require.NoError(t, env.GetWorkflowError())

	var result workflows.RebalanceOutput
	require.NoError(t, env.GetWorkflowResult(&result))
	require.Equal(t, "no_drift", result.Decision)
}

func TestRebalanceWorkflow_PolicyBlocked(t *testing.T) {
	testSuite := &testsuite.WorkflowTestSuite{}
	env := testSuite.NewTestWorkflowEnvironment()

	// Setup common mocks
	env.OnActivity((*activities.RebalanceActivities).CheckDriftActivity, mock.Anything, mock.Anything, mock.Anything).Return(
		&activities.CheckDriftOutput{DriftDetected: true, DriftReport: optimizer.DriftReport{DriftPercent: 0.05}}, nil)
	env.OnActivity((*activities.RebalanceActivities).LoadLotsActivity, mock.Anything, mock.Anything, mock.Anything).Return(
		&activities.LoadLotsOutput{Lots: []optimizer.Lot{}}, nil)
	env.OnActivity((*activities.RebalanceActivities).LoadPricesActivity, mock.Anything, mock.Anything, mock.Anything).Return(
		&activities.LoadPricesOutput{Prices: map[string]float64{}}, nil)
	env.OnActivity((*activities.RebalanceActivities).LoadTaxRulesActivity, mock.Anything, mock.Anything, mock.Anything).Return(
		&activities.LoadTaxRulesOutput{TaxRules: optimizer.TaxRules{}}, nil)
	env.OnActivity((*activities.RebalanceActivities).TaxAwareOptimizeActivity, mock.Anything, mock.Anything, mock.Anything).Return(
		&activities.TaxAwareOptimizeOutput{Plan: optimizer.Plan{ID: "prop-blocked"}}, nil)
	env.OnActivity((*activities.RebalanceActivities).MonteCarloSimActivity, mock.Anything, mock.Anything, mock.Anything).Return(
		&activities.MonteCarloSimOutput{Summary: optimizer.MonteCarloSummary{}}, nil)

	// Policy check fails
	env.OnActivity((*activities.RebalanceActivities).PolicyCheckActivity, mock.Anything, mock.Anything, mock.Anything).Return(
		&activities.PolicyCheckOutput{
			Passed: false,
			Violations: []activities.PolicyViolation{
				{PolicyID: "POL-001", PolicyName: "Concentration Limit", Severity: "error"},
			},
		}, nil)

	env.ExecuteWorkflow(workflows.RebalanceWorkflow, workflows.RebalanceInput{
		TenantID:    "tenant-001",
		PortfolioID: "port-001",
	})

	require.True(t, env.IsWorkflowCompleted())
	require.NoError(t, env.GetWorkflowError())

	var result workflows.RebalanceOutput
	require.NoError(t, env.GetWorkflowResult(&result))
	require.Equal(t, "policy_blocked", result.Decision)
}

func TestRebalanceWorkflow_Timeout(t *testing.T) {
	testSuite := &testsuite.WorkflowTestSuite{}
	env := testSuite.NewTestWorkflowEnvironment()

	// Setup mocks for successful path until waiting for approval
	env.OnActivity((*activities.RebalanceActivities).CheckDriftActivity, mock.Anything, mock.Anything, mock.Anything).Return(
		&activities.CheckDriftOutput{DriftDetected: true, DriftReport: optimizer.DriftReport{DriftPercent: 0.05}}, nil)
	env.OnActivity((*activities.RebalanceActivities).LoadLotsActivity, mock.Anything, mock.Anything, mock.Anything).Return(
		&activities.LoadLotsOutput{Lots: []optimizer.Lot{}}, nil)
	env.OnActivity((*activities.RebalanceActivities).LoadPricesActivity, mock.Anything, mock.Anything, mock.Anything).Return(
		&activities.LoadPricesOutput{Prices: map[string]float64{}}, nil)
	env.OnActivity((*activities.RebalanceActivities).LoadTaxRulesActivity, mock.Anything, mock.Anything, mock.Anything).Return(
		&activities.LoadTaxRulesOutput{TaxRules: optimizer.TaxRules{}}, nil)
	env.OnActivity((*activities.RebalanceActivities).TaxAwareOptimizeActivity, mock.Anything, mock.Anything, mock.Anything).Return(
		&activities.TaxAwareOptimizeOutput{Plan: optimizer.Plan{ID: "prop-timeout"}}, nil)
	env.OnActivity((*activities.RebalanceActivities).MonteCarloSimActivity, mock.Anything, mock.Anything, mock.Anything).Return(
		&activities.MonteCarloSimOutput{Summary: optimizer.MonteCarloSummary{}}, nil)
	env.OnActivity((*activities.RebalanceActivities).PolicyCheckActivity, mock.Anything, mock.Anything, mock.Anything).Return(
		&activities.PolicyCheckOutput{Passed: true}, nil)
	env.OnActivity((*activities.RebalanceActivities).NotifyAdvisorActivity, mock.Anything, mock.Anything, mock.Anything).Return(
		&activities.NotifyAdvisorOutput{}, nil)
	env.OnActivity((*activities.RebalanceActivities).PersistUARActivity, mock.Anything, mock.Anything, mock.Anything).Return(
		&activities.PersistUAROutput{UARID: "uar-timeout"}, nil)

	// Don't send any signal - let it timeout
	// Note: In real test, we'd need to advance time by 7 days

	env.ExecuteWorkflow(workflows.RebalanceWorkflow, workflows.RebalanceInput{
		TenantID:    "tenant-001",
		PortfolioID: "port-001",
	})

	require.True(t, env.IsWorkflowCompleted())
	require.NoError(t, env.GetWorkflowError())

	var result workflows.RebalanceOutput
	require.NoError(t, env.GetWorkflowResult(&result))
	require.Equal(t, "timeout", result.Decision)
}
