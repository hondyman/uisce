package bp

import (
	"testing"
	"time"

	"github.com/hondyman/semlayer/backend/internal/rules"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/suite"
	"go.temporal.io/sdk/testsuite"
)

type BPWorkflowTestSuite struct {
	suite.Suite
	testsuite.WorkflowTestSuite
	env *testsuite.TestWorkflowEnvironment
}

func (s *BPWorkflowTestSuite) SetupTest() {
	s.env = s.NewTestWorkflowEnvironment()
}

func (s *BPWorkflowTestSuite) AfterTest(suiteName, testName string) {
	s.env.AssertExpectations(s.T())
}

func (s *BPWorkflowTestSuite) TestAdvancedBP() {
	// 1. Mock Data
	def := &BPDefinition{Name: "Advanced BP"}
	steps := []*BPStep{
		{
			Seq:               1,
			StepKey:           "step-integration",
			Type:              "integration",
			IntegrationConfig: &IntegrationConfig{Method: "POST", Endpoint: "/api/check"},
		},
		{
			Seq:               2,
			StepKey:           "step-conditional",
			Type:              "task",
			ConditionExpr:     "field('req','amt') > 100",
			ConditionExprType: "starlark",
		},
		{
			Seq:     3,
			StepKey: "step-chain",
			Type:    "approval",
			ApprovalChain: &ApprovalChain{
				Levels: []ApprovalLevel{
					{Name: "L1", ActorRole: "Man"},
					{Name: "L2", ActorRole: "Dir"},
				},
			},
		},
	}

	wfCtx := WorkflowContext{TenantID: "t1", BpKey: "adv_bp", BpVersion: 1}

	// 2. Register Activities
	activities := &BPActivities{Repo: nil, RuleEngine: &rules.RuleEngine{}}
	s.env.RegisterActivity(activities)

	// -- Mocks --
	s.env.OnActivity("RecordProcessExecutionActivity", mock.Anything, mock.Anything).Return(nil)
	s.env.OnActivity("RecordStepExecutionActivity", mock.Anything, mock.Anything).Return(nil)

	s.env.OnActivity("LoadDefinitionActivity", mock.Anything, "t1", "adv_bp", 1).Return(
		&LoadDefinitionResult{Def: def, Steps: steps}, nil,
	)

	// Step 1: Integration
	s.env.OnActivity("IntegrationActivity", mock.Anything, steps[0].IntegrationConfig, mock.Anything).Return(nil)

	// Step 2: Conditional (Mock returns false -> SKIP)
	s.env.OnActivity("EvaluateConditionActivity", mock.Anything, mock.MatchedBy(func(in ConditionEvalInput) bool {
		return in.Expr == "field('req','amt') > 100" && in.ExprType == "starlark"
	})).Return(false, nil)

	// Step 3: Approval Chain
	// L1
	s.env.OnActivity("EvaluateApprovalLevelActivity", mock.Anything, steps[2].ApprovalChain.Levels[0], mock.Anything).Return(
		&ApprovalLevelResult{ShouldEnter: true, ShouldStop: false}, nil,
	)
	s.env.RegisterDelayedCallback(func() {
		s.env.SignalWorkflow("complete_task_step-chain_lvl0", "ok")
	}, time.Millisecond*50)

	// L2
	s.env.OnActivity("EvaluateApprovalLevelActivity", mock.Anything, steps[2].ApprovalChain.Levels[1], mock.Anything).Return(
		&ApprovalLevelResult{ShouldEnter: true, ShouldStop: false}, nil,
	)
	s.env.RegisterDelayedCallback(func() {
		s.env.SignalWorkflow("complete_task_step-chain_lvl1", "ok")
	}, time.Millisecond*100)

	// 3. Execute
	s.env.ExecuteWorkflow(BPWorkflow, wfCtx)

	s.True(s.env.IsWorkflowCompleted())
	s.NoError(s.env.GetWorkflowError())
}

func TestBPWorkflowTestSuite(t *testing.T) {
	suite.Run(t, new(BPWorkflowTestSuite))
}
