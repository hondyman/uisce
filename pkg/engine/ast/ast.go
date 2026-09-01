package ast

import (
	"errors"
	"fmt"
	"math"
	"sort"

	"github.com/apache/arrow-go/v18/arrow"
	"github.com/apache/arrow-go/v18/arrow/array"
	"github.com/apache/arrow-go/v18/arrow/memory"
)

var (
	ErrNilNode         = errors.New("ast node is nil")
	ErrVariableNotFound = errors.New("variable not found in arrow schema")
	ErrTypeMismatch     = errors.New("column type mismatch: expected float64")
	ErrZeroLengthRecord = errors.New("record batch contains zero rows")
)

type ASTNodeType int

const (
	NodeVariable ASTNodeType = iota
	NodeLiteral
	NodeBinaryOp
	NodeFunction
	NodeHWM
	NodeHurdle
	NodeProRata
)

// ASTNode represents an expression tree node for financial math evaluation.
type ASTNode struct {
	Type     ASTNodeType
	Op       string      // "+", "-", "*", "/", "max", "min", "pow", "exp"
	Value    float64     // For NodeLiteral
	VarName  string      // For NodeVariable
	Left     *ASTNode
	Right    *ASTNode
	Children []*ASTNode  // For multi-argument functions
}

// NewLiteral creates a literal ASTNode.
func NewLiteral(val float64) *ASTNode {
	return &ASTNode{Type: NodeLiteral, Value: val}
}

// NewVariable creates a variable lookup ASTNode.
func NewVariable(varName string) *ASTNode {
	return &ASTNode{Type: NodeVariable, VarName: varName}
}

// NewBinaryOp creates a binary operation ASTNode.
func NewBinaryOp(op string, left, right *ASTNode) *ASTNode {
	return &ASTNode{Type: NodeBinaryOp, Op: op, Left: left, Right: right}
}

// EvaluateVectorized processes expressions directly over contiguous Arrow column arrays.
func (node *ASTNode) EvaluateVectorized(mem memory.Allocator, batch arrow.Record) ([]float64, error) {
	if node == nil {
		return nil, ErrNilNode
	}
	if batch == nil || batch.NumRows() == 0 {
		return nil, ErrZeroLengthRecord
	}

	rowCount := int(batch.NumRows())
	results := make([]float64, rowCount)

	switch node.Type {
	case NodeLiteral:
		val := node.Value
		for i := 0; i < rowCount; i++ {
			results[i] = val
		}
		return results, nil

	case NodeVariable:
		indices := batch.Schema().FieldIndices(node.VarName)
		if len(indices) == 0 {
			return nil, fmt.Errorf("%w: %s", ErrVariableNotFound, node.VarName)
		}
		col := batch.Column(indices[0])
		vec, ok := col.(*array.Float64)
		if !ok {
			return nil, fmt.Errorf("%w: %s is not Float64", ErrTypeMismatch, node.VarName)
		}

		rawValues := vec.Float64Values()
		if vec.NullN() == 0 {
			copy(results, rawValues)
		} else {
			for i := 0; i < rowCount; i++ {
				if vec.IsValid(i) {
					results[i] = rawValues[i]
				} else {
					results[i] = 0.0
				}
			}
		}
		return results, nil

	case NodeBinaryOp:
		leftVal, err := node.Left.EvaluateVectorized(mem, batch)
		if err != nil {
			return nil, err
		}
		rightVal, err := node.Right.EvaluateVectorized(mem, batch)
		if err != nil {
			return nil, err
		}

		switch node.Op {
		case "+":
			for i := 0; i < rowCount; i++ {
				results[i] = leftVal[i] + rightVal[i]
			}
		case "-":
			for i := 0; i < rowCount; i++ {
				results[i] = leftVal[i] - rightVal[i]
			}
		case "*":
			for i := 0; i < rowCount; i++ {
				results[i] = leftVal[i] * rightVal[i]
			}
		case "/":
			for i := 0; i < rowCount; i++ {
				if rightVal[i] == 0.0 {
					results[i] = 0.0
				} else {
					results[i] = leftVal[i] / rightVal[i]
				}
			}
		case "max":
			for i := 0; i < rowCount; i++ {
				results[i] = math.Max(leftVal[i], rightVal[i])
			}
		case "min":
			for i := 0; i < rowCount; i++ {
				results[i] = math.Min(leftVal[i], rightVal[i])
			}
		case "pow":
			for i := 0; i < rowCount; i++ {
				results[i] = math.Pow(leftVal[i], rightVal[i])
			}
		default:
			return nil, fmt.Errorf("unsupported binary operator: %s", node.Op)
		}
		return results, nil

	default:
		return nil, fmt.Errorf("unsupported ast node type: %v", node.Type)
	}
}

// ComputeIncentiveFeeWaterfall calculates incentive fees according to:
// Fee = max(0, (NAV_end - max(HWM, NAV_start * (1 + r_hurdle)^t)) * gamma)
func ComputeIncentiveFeeWaterfall(
	navStart, navEnd, hwm, hurdleRate []float64,
	tYears, gamma float64,
) []float64 {
	n := len(navEnd)
	fees := make([]float64, n)

	for i := 0; i < n; i++ {
		hurdleTarget := navStart[i] * math.Pow(1.0+hurdleRate[i], tYears)
		benchmark := math.Max(hwm[i], hurdleTarget)
		excess := navEnd[i] - benchmark
		if excess > 0 {
			fees[i] = excess * gamma
		} else {
			fees[i] = 0.0
		}
	}
	return fees
}

// ComputeProRataAllocationWithFactors calculates factor-adjusted allocation weights:
// W_i = (S_i * prod(1 + delta_i,k)) / sum(S_j * prod(1 + delta_j,k))
func ComputeProRataAllocationWithFactors(
	targetSizes []float64,
	factors [][]float64, // factors[i][k] for account i, factor k
	totalOrderAmount float64,
) ([]float64, error) {
	n := len(targetSizes)
	if n == 0 {
		return nil, errors.New("empty target sizes")
	}

	adjustedSizes := make([]float64, n)
	var sumAdjusted float64

	for i := 0; i < n; i++ {
		multiplier := 1.0
		if i < len(factors) {
			for _, delta := range factors[i] {
				multiplier *= (1.0 + delta)
			}
		}
		adj := targetSizes[i] * multiplier
		adjustedSizes[i] = adj
		sumAdjusted += adj
	}

	allocations := make([]float64, n)
	if sumAdjusted <= 0.0 {
		return allocations, nil
	}

	for i := 0; i < n; i++ {
		weight := adjustedSizes[i] / sumAdjusted
		allocations[i] = weight * totalOrderAmount
	}

	return allocations, nil
}

// TaxLot represents an individual tax lot for MinTax / FIFO matching.
type TaxLot struct {
	LotID         string
	AcquisitionTs int64
	Shares        float64
	CostBasis     float64
	TaxTermFactor float64 // short-term vs long-term tax multiplier
}

// SortLotsMinTax orders lots by tax gain/loss ascending: (P_current - P_cost) * tau_term
func SortLotsMinTax(lots []TaxLot, currentPrice float64) []TaxLot {
	sorted := make([]TaxLot, len(lots))
	copy(sorted, lots)

	sort.Slice(sorted, func(i, j int) bool {
		taxImpactI := (currentPrice - sorted[i].CostBasis) * sorted[i].TaxTermFactor
		taxImpactJ := (currentPrice - sorted[j].CostBasis) * sorted[j].TaxTermFactor
		return taxImpactI < taxImpactJ
	})

	return sorted
}
