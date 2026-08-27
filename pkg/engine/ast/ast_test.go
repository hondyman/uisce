package ast

import (
	"testing"

	"github.com/apache/arrow-go/v18/arrow"
	"github.com/apache/arrow-go/v18/arrow/array"
	"github.com/apache/arrow-go/v18/arrow/memory"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestASTNode_EvaluateVectorized(t *testing.T) {
	mem := memory.NewGoAllocator()
	schema := arrow.NewSchema([]arrow.Field{
		{Name: "nav_end", Type: arrow.PrimitiveTypes.Float64},
		{Name: "nav_start", Type: arrow.PrimitiveTypes.Float64},
		{Name: "hurdle_rate", Type: arrow.PrimitiveTypes.Float64},
	}, nil)

	builder := array.NewRecordBuilder(mem, schema)
	defer builder.Release()

	builder.Field(0).(*array.Float64Builder).AppendValues([]float64{1200000.0, 2600000.0, 510000.0}, nil)
	builder.Field(1).(*array.Float64Builder).AppendValues([]float64{1000000.0, 2500000.0, 500000.0}, nil)
	builder.Field(2).(*array.Float64Builder).AppendValues([]float64{0.08, 0.05, 0.10}, nil)

	rec := builder.NewRecord()
	defer rec.Release()

	// Expression: (nav_end - nav_start) / nav_start -> Period Return
	expr := NewBinaryOp("/",
		NewBinaryOp("-", NewVariable("nav_end"), NewVariable("nav_start")),
		NewVariable("nav_start"),
	)

	results, err := expr.EvaluateVectorized(mem, rec)
	require.NoError(t, err)
	require.Len(t, results, 3)

	assert.InDelta(t, 0.20, results[0], 0.0001)  // (1.2M - 1.0M) / 1.0M = 0.20
	assert.InDelta(t, 0.04, results[1], 0.0001)  // (2.6M - 2.5M) / 2.5M = 0.04
	assert.InDelta(t, 0.02, results[2], 0.0001)  // (510k - 500k) / 500k = 0.02
}

func TestComputeIncentiveFeeWaterfall(t *testing.T) {
	navStart := []float64{1000000.0, 2000000.0}
	navEnd := []float64{1200000.0, 2050000.0}
	hwm := []float64{1050000.0, 2100000.0}
	hurdle := []float64{0.08, 0.05} // 8%, 5%
	tYears := 1.0
	gamma := 0.20 // 20% carry

	fees := ComputeIncentiveFeeWaterfall(navStart, navEnd, hwm, hurdle, tYears, gamma)
	require.Len(t, fees, 2)

	// Account 1: hurdle = 1.0M * 1.08 = 1.08M; benchmark = max(1.05M, 1.08M) = 1.08M
	// excess = 1.2M - 1.08M = 120k; fee = 120k * 0.20 = 24k
	assert.InDelta(t, 24000.0, fees[0], 0.01)

	// Account 2: hurdle = 2.0M * 1.05 = 2.10M; benchmark = max(2.10M, 2.10M) = 2.10M
	// NAV_end = 2.05M < 2.10M -> excess <= 0 -> fee = 0
	assert.Equal(t, 0.0, fees[1])
}

func TestComputeProRataAllocationWithFactors(t *testing.T) {
	targetSizes := []float64{100000.0, 100000.0}
	factors := [][]float64{
		{0.10},  // +10% ESG bonus
		{-0.10}, // -10% penalty
	}
	totalOrder := 20000.0

	allocs, err := ComputeProRataAllocationWithFactors(targetSizes, factors, totalOrder)
	require.NoError(t, err)
	require.Len(t, allocs, 2)

	// S1_adj = 110,000; S2_adj = 90,000; sum = 200,000
	// W1 = 55%, W2 = 45%
	assert.InDelta(t, 11000.0, allocs[0], 0.01)
	assert.InDelta(t, 9000.0, allocs[1], 0.01)
}

func TestSortLotsMinTax(t *testing.T) {
	lots := []TaxLot{
		{LotID: "L1", CostBasis: 100.0, TaxTermFactor: 0.40}, // gain = (110 - 100)*0.4 = +4.0
		{LotID: "L2", CostBasis: 120.0, TaxTermFactor: 0.40}, // loss = (110 - 120)*0.4 = -4.0 (best tax loss)
		{LotID: "L3", CostBasis: 105.0, TaxTermFactor: 0.20}, // gain = (110 - 105)*0.2 = +1.0
	}

	sorted := SortLotsMinTax(lots, 110.0)
	require.Len(t, sorted, 3)

	assert.Equal(t, "L2", sorted[0].LotID) // -4.0
	assert.Equal(t, "L3", sorted[1].LotID) // +1.0
	assert.Equal(t, "L1", sorted[2].LotID) // +4.0
}

func BenchmarkASTVectorized_100k(b *testing.B) {
	mem := memory.NewGoAllocator()
	numRows := 100000

	schema := arrow.NewSchema([]arrow.Field{
		{Name: "nav_end", Type: arrow.PrimitiveTypes.Float64},
		{Name: "nav_start", Type: arrow.PrimitiveTypes.Float64},
	}, nil)

	builder := array.NewRecordBuilder(mem, schema)
	defer builder.Release()

	navEndVals := make([]float64, numRows)
	navStartVals := make([]float64, numRows)
	for i := 0; i < numRows; i++ {
		navStartVals[i] = 100000.0 + float64(i)
		navEndVals[i] = navStartVals[i] * 1.08
	}

	builder.Field(0).(*array.Float64Builder).AppendValues(navEndVals, nil)
	builder.Field(1).(*array.Float64Builder).AppendValues(navStartVals, nil)

	rec := builder.NewRecord()
	defer rec.Release()

	expr := NewBinaryOp("/",
		NewBinaryOp("-", NewVariable("nav_end"), NewVariable("nav_start")),
		NewVariable("nav_start"),
	)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := expr.EvaluateVectorized(mem, rec)
		if err != nil {
			b.Fatal(err)
		}
	}
}
