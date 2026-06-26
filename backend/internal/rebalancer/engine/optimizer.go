package engine

import (
	"fmt"
	"gonum.org/v1/gonum/mat"
)

// QPSolver defines the interface for a Quadratic Programming solver
// Minimize 0.5 * x'Px + q'x
// Subject to l <= Ax <= u
type QPSolver interface {
	Solve(P *mat.Dense, q *mat.VecDense, A *mat.Dense, l, u *mat.VecDense) (*mat.VecDense, error)
}

// Optimizer handles the construction of the QP problem from portfolio data
type Optimizer struct {
	Solver QPSolver
}

func NewOptimizer(solver QPSolver) *Optimizer {
	return &Optimizer{
		Solver: solver,
	}
}

// OptimizationInput contains the data needed to construct the QP matrices
type OptimizationInput struct {
	CurrentWeights []float64
	TargetWeights  []float64
	Covariance     *mat.Dense
	RiskAversion   float64
	// Constraints would go here (e.g., sector limits)
}

// OptimizePortfolio constructs the matrices and calls the solver
func (o *Optimizer) OptimizePortfolio(input OptimizationInput) ([]float64, error) {
	n := len(input.CurrentWeights)
	if n == 0 {
		return nil, fmt.Errorf("empty weights")
	}

	// 1. Construct P (Risk Matrix)
	// Objective: Minimize 0.5 * w_active' * Sigma * w_active
	// P = Sigma (Covariance Matrix)
	P := input.Covariance

	// 2. Construct q (Expected Return / Tracking Error Linear Term)
	// For pure TE minimization: q = -Sigma * w_benchmark
	// (Derived from expanding (w_p - w_b)' Sigma (w_p - w_b))
	
	// Convert target weights to VecDense
	wBenchmark := mat.NewVecDense(n, input.TargetWeights)
	
	q := mat.NewVecDense(n, nil)
	q.MulVec(P, wBenchmark)
	q.ScaleVec(-1, q) // q = -Sigma * w_b

	// 3. Construct Constraints (A, l, u)
	// Example: Fully invested constraint (sum(w) = 1)
	// A is (m x n) matrix where m is number of constraints
	
	// Row 1: Sum of weights = 1
	A_data := make([]float64, n)
	for i := range A_data {
		A_data[i] = 1.0
	}
	A := mat.NewDense(1, n, A_data)
	
	l := mat.NewVecDense(1, []float64{1.0})
	u := mat.NewVecDense(1, []float64{1.0})

	// 4. Solve
	solution, err := o.Solver.Solve(P, q, A, l, u)
	if err != nil {
		return nil, fmt.Errorf("QP solve failed: %w", err)
	}

	// Extract results
	result := make([]float64, n)
	for i := 0; i < n; i++ {
		result[i] = solution.AtVec(i)
	}

	return result, nil
}
