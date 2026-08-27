package activities

import (
	"testing"

	"github.com/apache/arrow-go/v18/arrow/memory"
	"github.com/hondyman/uisce/pkg/engine/ast"
	"github.com/hondyman/uisce/pkg/engine/vectorized"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.temporal.io/sdk/testsuite"
)

func TestVectorizedRebalanceActivity(t *testing.T) {
	testSuite := &testsuite.WorkflowTestSuite{}
	env := testSuite.NewTestActivityEnvironment()

	mem := memory.NewGoAllocator()
	engine, err := vectorized.NewDataFusionEngine(mem)
	require.NoError(t, err)

	acts := NewVectorizedExecutionActivities(engine, mem)
	env.RegisterActivity(acts)

	req := RebalanceRequest{
		TenantID:         "tenant-test",
		TotalOrderAmount: 100000.0,
		AccountIDs:       []int64{1, 2},
		TargetSizes:      []float64{500000.0, 500000.0},
		CustomFactors: [][]float64{
			{0.0},
			{0.0},
		},
		RuleAST: ast.NewBinaryOp("*", ast.NewVariable("nav"), ast.NewLiteral(1.0)),
	}

	val, err := env.ExecuteActivity(acts.VectorizedRebalanceActivity, req)
	require.NoError(t, err)

	var resp RebalanceResponse
	err = val.Get(&resp)
	require.NoError(t, err)

	assert.True(t, resp.Success)
	assert.Equal(t, int64(2), resp.ProcessedRows)
	assert.Equal(t, 50000.0, resp.Allocations[0])
	assert.Equal(t, 50000.0, resp.Allocations[1])
}

func TestVectorizedFeeCalculationActivity(t *testing.T) {
	testSuite := &testsuite.WorkflowTestSuite{}
	env := testSuite.NewTestActivityEnvironment()

	mem := memory.NewGoAllocator()
	engine, err := vectorized.NewDataFusionEngine(mem)
	require.NoError(t, err)

	acts := NewVectorizedExecutionActivities(engine, mem)
	env.RegisterActivity(acts)

	req := FeeCalculationRequest{
		TenantID:   "tenant-test",
		NavStart:   []float64{1000000.0},
		NavEnd:     []float64{1200000.0},
		HWM:        []float64{1050000.0},
		HurdleRate: []float64{0.08},
		PeriodYear: 1.0,
		CarryRate:  0.20,
	}

	val, err := env.ExecuteActivity(acts.VectorizedFeeCalculationActivity, req)
	require.NoError(t, err)

	var resp FeeCalculationResponse
	err = val.Get(&resp)
	require.NoError(t, err)

	assert.True(t, resp.Success)
	assert.InDelta(t, 24000.0, resp.TotalFees, 0.01)
}
