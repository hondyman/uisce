package workflows

import (
	"testing"

	"github.com/apache/arrow-go/v18/arrow/memory"
	"github.com/hondyman/uisce/pkg/activities"
	"github.com/hondyman/uisce/pkg/engine/vectorized"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.temporal.io/sdk/testsuite"
)

func TestRebalanceWorkflow(t *testing.T) {
	testSuite := &testsuite.WorkflowTestSuite{}
	env := testSuite.NewTestWorkflowEnvironment()

	mem := memory.NewGoAllocator()
	engine, err := vectorized.NewDataFusionEngine(mem)
	require.NoError(t, err)

	acts := activities.NewVectorizedExecutionActivities(engine, mem)
	env.RegisterActivity(acts.VectorizedRebalanceActivity)

	input := RebalanceWorkflowInput{
		TenantID:         "tenant-sample",
		TotalOrderAmount: 200000.0,
		AccountIDs:       []int64{101, 102},
		TargetSizes:      []float64{1000000.0, 1000000.0},
		CustomFactors: [][]float64{
			{0.0},
			{0.0},
		},
	}

	env.ExecuteWorkflow(RebalanceWorkflow, input)
	require.True(t, env.IsWorkflowCompleted())
	require.NoError(t, env.GetWorkflowError())

	var result RebalanceWorkflowResult
	err = env.GetWorkflowResult(&result)
	require.NoError(t, err)

	assert.Equal(t, "COMPLETED", result.Status)
	assert.Equal(t, int64(2), result.ProcessedRows)
	assert.Equal(t, 100000.0, result.Allocations[0])
	assert.Equal(t, 100000.0, result.Allocations[1])
}
