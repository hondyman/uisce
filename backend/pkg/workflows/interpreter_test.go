package workflows

import (
	"context"
	"testing"

	"github.com/stretchr/testify/suite"
	"go.temporal.io/sdk/activity"
	"go.temporal.io/sdk/testsuite"
)

type InterpreterTestSuite struct {
	suite.Suite
	testsuite.WorkflowTestSuite
	env *testsuite.TestWorkflowEnvironment
}

func (s *InterpreterTestSuite) SetupTest() {
	s.env = s.NewTestWorkflowEnvironment()
}

func (s *InterpreterTestSuite) AfterTest(suiteName, testName string) {
	s.env.AssertExpectations(s.T())
}

func (s *InterpreterTestSuite) TestSimpleSequence() {
	// Define DSL
	dsl := WorkflowDefinition{
		Name:        "TestSimple",
		StartNodeID: "node1",
		Nodes: map[string]WorkflowNode{
			"node1": {
				ID:   "node1",
				Type: "ACTIVITY",
				Config: map[string]interface{}{
					"activityName": "TestActivity",
				},
				NextNodeID: stringPtr("node2"),
			},
			"node2": {
				ID:   "node2",
				Type: "END",
			},
		},
	}

	// Register Activity
	s.env.RegisterActivityWithOptions(func(ctx context.Context, config map[string]interface{}, state map[string]interface{}) (map[string]interface{}, error) {
		return map[string]interface{}{"step1": "done"}, nil
	}, activity.RegisterOptions{Name: "TestActivity"})

	s.env.ExecuteWorkflow(InterpreterWorkflow, dsl)

	s.True(s.env.IsWorkflowCompleted())
	s.NoError(s.env.GetWorkflowError())

	var result WorkflowResult
	s.env.GetWorkflowResult(&result)
	s.Equal("completed", result.Status)
	s.Equal("done", result.FinalState["step1"])
}

func (s *InterpreterTestSuite) TestBranching() {
	// Define DSL
	dsl := WorkflowDefinition{
		Name:        "TestBranching",
		StartNodeID: "start",
		GlobalState: map[string]interface{}{"value": 10},
		Nodes: map[string]WorkflowNode{
			"start": {
				ID:   "start",
				Type: "BRANCH",
				Branches: []BranchOption{
					{TargetNodeID: "pathA", Condition: "value > 5"}, // Should match
					{TargetNodeID: "pathB", Condition: "value <= 5"},
				},
			},
			"pathA": {
				ID:         "pathA",
				Type:       "ACTIVITY",
				Config:     map[string]interface{}{"activityName": "ActivityA"},
				NextNodeID: stringPtr("end"),
			},
			"pathB": {
				ID:         "pathB",
				Type:       "ACTIVITY",
				Config:     map[string]interface{}{"activityName": "ActivityB"},
				NextNodeID: stringPtr("end"),
			},
			"end": {ID: "end", Type: "END"},
		},
	}

	// Mock register activities
	s.env.RegisterActivityWithOptions(func(ctx context.Context, config map[string]interface{}, state map[string]interface{}) (map[string]interface{}, error) {
		return map[string]interface{}{"path": "A"}, nil
	}, activity.RegisterOptions{Name: "ActivityA"})

	s.env.RegisterActivityWithOptions(func(ctx context.Context, config map[string]interface{}, state map[string]interface{}) (map[string]interface{}, error) {
		return map[string]interface{}{"path": "B"}, nil
	}, activity.RegisterOptions{Name: "ActivityB"})

	// evaluateConditionLocal now really evaluates "value > 5" against
	// GlobalState{"value": 10} via internal/rules.ConditionEvaluator, so
	// pathA is taken because the condition is genuinely true, not because
	// of the old always-true stub.

	s.env.ExecuteWorkflow(InterpreterWorkflow, dsl)

	s.True(s.env.IsWorkflowCompleted())
	s.NoError(s.env.GetWorkflowError())

	var result WorkflowResult
	s.env.GetWorkflowResult(&result)
	s.Equal("A", result.FinalState["path"])
}

// TestBranchingTakesFalseBranch proves evaluateConditionLocal genuinely
// evaluates the condition rather than always matching the first branch: with
// value=2, "value > 5" is false and "value <= 5" is true, so pathB must be
// taken. Under the old always-true stub this test would fail (pathA taken).
func (s *InterpreterTestSuite) TestBranchingTakesFalseBranch() {
	dsl := WorkflowDefinition{
		Name:        "TestBranchingFalse",
		StartNodeID: "start",
		GlobalState: map[string]interface{}{"value": 2},
		Nodes: map[string]WorkflowNode{
			"start": {
				ID:   "start",
				Type: "BRANCH",
				Branches: []BranchOption{
					{TargetNodeID: "pathA", Condition: "value > 5"},
					{TargetNodeID: "pathB", Condition: "value <= 5"}, // Should match
				},
			},
			"pathA": {
				ID:         "pathA",
				Type:       "ACTIVITY",
				Config:     map[string]interface{}{"activityName": "ActivityA"},
				NextNodeID: stringPtr("end"),
			},
			"pathB": {
				ID:         "pathB",
				Type:       "ACTIVITY",
				Config:     map[string]interface{}{"activityName": "ActivityB"},
				NextNodeID: stringPtr("end"),
			},
			"end": {ID: "end", Type: "END"},
		},
	}

	s.env.RegisterActivityWithOptions(func(ctx context.Context, config map[string]interface{}, state map[string]interface{}) (map[string]interface{}, error) {
		return map[string]interface{}{"path": "A"}, nil
	}, activity.RegisterOptions{Name: "ActivityA"})

	s.env.RegisterActivityWithOptions(func(ctx context.Context, config map[string]interface{}, state map[string]interface{}) (map[string]interface{}, error) {
		return map[string]interface{}{"path": "B"}, nil
	}, activity.RegisterOptions{Name: "ActivityB"})

	s.env.ExecuteWorkflow(InterpreterWorkflow, dsl)

	s.True(s.env.IsWorkflowCompleted())
	s.NoError(s.env.GetWorkflowError())

	var result WorkflowResult
	s.env.GetWorkflowResult(&result)
	s.Equal("B", result.FinalState["path"])
}

// TestRunStoredWorkflowLoadsRealDefinition proves RunStoredWorkflow now
// fetches its DSL via ActivityLoadWorkflowDefinition (see
// workflow_definition_activities.go) instead of a hardcoded switch on
// input.WorkflowID — a mocked loader returning a definition unrelated to any
// of the old hardcoded demo IDs must still drive the interpreter correctly.
func (s *InterpreterTestSuite) TestRunStoredWorkflowLoadsRealDefinition() {
	stored := WorkflowDefinition{
		Name:        "Loaded From Storage",
		StartNodeID: "n1",
		Nodes: map[string]WorkflowNode{
			"n1": {
				ID:         "n1",
				Type:       "ACTIVITY",
				Config:     map[string]interface{}{"activityName": "ActivityA"},
				NextNodeID: stringPtr("n2"),
			},
			"n2": {ID: "n2", Type: "END"},
		},
	}

	s.env.RegisterActivityWithOptions(func(ctx context.Context, tenantID string, workflowKey string) (WorkflowDefinition, error) {
		s.Equal("some_custom_workflow", workflowKey)
		return stored, nil
	}, activity.RegisterOptions{Name: "ActivityLoadWorkflowDefinition"})

	s.env.RegisterActivityWithOptions(func(ctx context.Context, config map[string]interface{}, state map[string]interface{}) (map[string]interface{}, error) {
		return map[string]interface{}{"loaded": true}, nil
	}, activity.RegisterOptions{Name: "ActivityA"})

	s.env.ExecuteWorkflow(RunStoredWorkflow, InterpreterInput{
		WorkflowID:  "some_custom_workflow",
		InitialData: map[string]interface{}{"seed": "value"},
	})

	s.True(s.env.IsWorkflowCompleted())
	s.NoError(s.env.GetWorkflowError())

	var result WorkflowResult
	s.env.GetWorkflowResult(&result)
	s.Equal(true, result.FinalState["loaded"])
	s.Equal("value", result.FinalState["seed"])
}

func TestInterpreterTestSuite(t *testing.T) {
	suite.Run(t, new(InterpreterTestSuite))
}
