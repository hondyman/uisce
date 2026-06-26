package exec

import (
	"errors"
	"math"

	"gonum.org/v1/gonum/mat"
)

// MatInv computes the inverse of a matrix using LU decomposition
func MatInv(A mat.Matrix) (*mat.Dense, error) {
	r, c := A.Dims()
	if r != c {
		return nil, errors.New("matrix must be square")
	}

	var lu mat.LU
	lu.Factorize(A)
	if lu.Det() == 0 {
		return nil, errors.New("matrix is singular")
	}

	var inv mat.Dense
	inv.Inverse(&lu)
	return &inv, nil
}

// MatMul multiplies two matrices
func MatMul(A, B mat.Matrix) *mat.Dense {
	rA, _ := A.Dims()
	_, cB := B.Dims()
	result := mat.NewDense(rA, cB, nil)
	result.Mul(A, B)
	return result
}

// MatAdd adds two matrices
func MatAdd(A, B mat.Matrix) *mat.Dense {
	r, c := A.Dims()
	result := mat.NewDense(r, c, nil)
	result.Add(A, B)
	return result
}

// MatSub subtracts matrix B from matrix A
func MatSub(A, B mat.Matrix) *mat.Dense {
	r, c := A.Dims()
	result := mat.NewDense(r, c, nil)
	result.Sub(A, B)
	return result
}

// MatScale scales a matrix by a scalar
func MatScale(alpha float64, A mat.Matrix) *mat.Dense {
	r, c := A.Dims()
	result := mat.NewDense(r, c, nil)
	result.Scale(alpha, A)
	return result
}

// VecDot computes the dot product of two vectors
func VecDot(a, b []float64) float64 {
	if len(a) != len(b) {
		panic("vector lengths must match")
	}
	sum := 0.0
	for i := range a {
		sum += a[i] * b[i]
	}
	return sum
}

// VecNorm computes the Euclidean norm of a vector
func VecNorm(v []float64) float64 {
	return math.Sqrt(VecDot(v, v))
}

// MatTrace computes the trace of a matrix
func MatTrace(A mat.Matrix) float64 {
	r, c := A.Dims()
	if r != c {
		panic("matrix must be square")
	}
	trace := 0.0
	for i := 0; i < r; i++ {
		trace += A.At(i, i)
	}
	return trace
}
