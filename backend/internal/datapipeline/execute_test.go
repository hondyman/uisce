package datapipeline

import (
	"context"
	"errors"
	"testing"

	"go.temporal.io/sdk/activity"
	"go.temporal.io/sdk/testsuite"
)

func TestWorkflowRunsActivityOnceWithoutRetry(t *testing.T) {
	var s testsuite.WorkflowTestSuite
	env := s.NewTestWorkflowEnvironment()
	calls := 0
	env.RegisterActivityWithOptions(func(_ context.Context, in RunInput) error {
		calls++
		if in.RunID != "run-1" || in.TenantID != "t1" {
			t.Errorf("input %+v", in)
		}
		return errors.New("sink failed")
	}, activity.RegisterOptions{Name: ActivityName})
	env.ExecuteWorkflow(Workflow, RunInput{TenantID: "t1", RunID: "run-1"})
	if !env.IsWorkflowCompleted() || env.GetWorkflowError() == nil {
		t.Fatal("workflow should complete with the activity's error")
	}
	if calls != 1 {
		t.Errorf("a load must not be retried automatically; ran %d times", calls)
	}
}
