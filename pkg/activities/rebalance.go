package activities

import (
	"context"
	"fmt"

	"github.com/apache/arrow-go/v18/arrow"
	"github.com/apache/arrow-go/v18/arrow/array"
	"github.com/apache/arrow-go/v18/arrow/memory"
	"github.com/hondyman/uisce/pkg/engine/ast"
	"github.com/hondyman/uisce/pkg/engine/vectorized"
	"go.temporal.io/sdk/activity"
)

// RebalanceRequest defines the input payload for vectorized batch rebalance activity.
type RebalanceRequest struct {
	TenantID        string           `json:"tenant_id"`
	TotalOrderAmount float64          `json:"total_order_amount"`
	AccountIDs      []int64          `json:"account_ids"`
	TargetSizes     []float64        `json:"target_sizes"`
	CustomFactors   [][]float64      `json:"custom_factors"`
	RuleAST         *ast.ASTNode     `json:"rule_ast,omitempty"`
}

// RebalanceResponse defines the result payload of batch rebalance calculation.
type RebalanceResponse struct {
	ProcessedRows int64     `json:"processed_rows"`
	Allocations   []float64 `json:"allocations"`
	Success       bool      `json:"success"`
}

// FeeCalculationRequest defines input for portfolio fee waterfall calculation.
type FeeCalculationRequest struct {
	TenantID   string    `json:"tenant_id"`
	NavStart   []float64 `json:"nav_start"`
	NavEnd     []float64 `json:"nav_end"`
	HWM        []float64 `json:"hwm"`
	HurdleRate []float64 `json:"hurdle_rate"`
	PeriodYear float64   `json:"period_year"`
	CarryRate  float64   `json:"carry_rate"`
}

// FeeCalculationResponse defines output for portfolio fee calculation.
type FeeCalculationResponse struct {
	CalculatedFees []float64 `json:"calculated_fees"`
	TotalFees      float64   `json:"total_fees"`
	Success        bool      `json:"success"`
}

// VectorizedExecutionActivities implements Temporal activities for off-heap math computations.
type VectorizedExecutionActivities struct {
	Engine *vectorized.DataFusionEngine
	Mem    memory.Allocator
}

// NewVectorizedExecutionActivities creates activities initialized with DataFusion engine.
func NewVectorizedExecutionActivities(engine *vectorized.DataFusionEngine, mem memory.Allocator) *VectorizedExecutionActivities {
	if mem == nil {
		mem = memory.NewGoAllocator()
	}
	return &VectorizedExecutionActivities{
		Engine: engine,
		Mem:    mem,
	}
}

// VectorizedRebalanceActivity executes factor-adjusted pro-rata allocations with Temporal heartbeats.
func (a *VectorizedExecutionActivities) VectorizedRebalanceActivity(
	ctx context.Context,
	req RebalanceRequest,
) (*RebalanceResponse, error) {
	activity.RecordHeartbeat(ctx, "Starting Vectorized Batch Rebalance")

	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	default:
	}

	allocations, err := ast.ComputeProRataAllocationWithFactors(
		req.TargetSizes,
		req.CustomFactors,
		req.TotalOrderAmount,
	)
	if err != nil {
		return nil, fmt.Errorf("pro-rata calculation failed: %w", err)
	}

	activity.RecordHeartbeat(ctx, "Completed Allocation Calculations")

	// If AST expression is supplied, evaluate it over Arrow RecordBatch
	if req.RuleAST != nil && len(req.AccountIDs) > 0 {
		schema := arrow.NewSchema([]arrow.Field{
			{Name: "account_id", Type: arrow.PrimitiveTypes.Int64},
			{Name: "nav", Type: arrow.PrimitiveTypes.Float64},
		}, nil)

		builder := array.NewRecordBuilder(a.Mem, schema)
		defer builder.Release()

		builder.Field(0).(*array.Int64Builder).AppendValues(req.AccountIDs, nil)
		builder.Field(1).(*array.Float64Builder).AppendValues(req.TargetSizes, nil)

		rec := builder.NewRecord()
		defer rec.Release()

		_, err := req.RuleAST.EvaluateVectorized(a.Mem, rec)
		if err != nil {
			return nil, fmt.Errorf("ast evaluation failed: %w", err)
		}
	}

	return &RebalanceResponse{
		ProcessedRows: int64(len(allocations)),
		Allocations:   allocations,
		Success:       true,
	}, nil
}

// VectorizedFeeCalculationActivity calculates incentive fee waterfalls over Arrow arrays.
func (a *VectorizedExecutionActivities) VectorizedFeeCalculationActivity(
	ctx context.Context,
	req FeeCalculationRequest,
) (*FeeCalculationResponse, error) {
	activity.RecordHeartbeat(ctx, "Executing Fee Waterfall Calculation")

	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	default:
	}

	fees := ast.ComputeIncentiveFeeWaterfall(
		req.NavStart,
		req.NavEnd,
		req.HWM,
		req.HurdleRate,
		req.PeriodYear,
		req.CarryRate,
	)

	var total float64
	for _, f := range fees {
		total += f
	}

	return &FeeCalculationResponse{
		CalculatedFees: fees,
		TotalFees:      total,
		Success:        true,
	}, nil
}
