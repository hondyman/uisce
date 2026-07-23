package exec

import (
	"errors"
	"math"
	"sort"
)

// Basic vector ops
func Dot(a, b []float64) float64 {
	s := 0.0
	for i := range a {
		s += a[i] * b[i]
	}
	return s
}
func Sum(a []float64) float64 {
	s := 0.0
	for _, v := range a {
		s += v
	}
	return s
}
func Scale(a []float64, k float64) []float64 {
	out := make([]float64, len(a))
	for i := range a {
		out[i] = a[i] * k
	}
	return out
}
func VecAdd(a, b []float64) []float64 {
	out := make([]float64, len(a))
	for i := range a {
		out[i] = a[i] + b[i]
	}
	return out
}
func VecSub(a, b []float64) []float64 {
	out := make([]float64, len(a))
	for i := range a {
		out[i] = a[i] - b[i]
	}
	return out
}
func Bounds(a []float64) (float64, float64) {
	if len(a) == 0 {
		return 0, 0
	}
	mn, mx := a[0], a[0]
	for _, v := range a {
		if v < mn {
			mn = v
		}
		if v > mx {
			mx = v
		}
	}
	return mn, mx
}

// Matrix ops
func MatVec(A [][]float64, x []float64) []float64 {
	n := len(A)
	y := make([]float64, n)
	for i := 0; i < n; i++ {
		s := 0.0
		for j := 0; j < n; j++ {
			s += A[i][j] * x[j]
		}
		y[i] = s
	}
	return y
}

func QuadForm(S [][]float64, w []float64) float64 {
	// w^T S w
	n := len(w)
	s := 0.0
	for i := 0; i < n; i++ {
		for j := 0; j < n; j++ {
			s += w[i] * S[i][j] * w[j]
		}
	}
	return s
}

// Naive Gaussian elimination inverse (for small N, PD covariance).
func MatInv(A [][]float64) ([][]float64, error) {
	n := len(A)
	if n == 0 || len(A[0]) != n {
		return nil, errors.New("not square")
	}
	// build augmented matrix [A | I]
	M := make([][]float64, n)
	for i := 0; i < n; i++ {
		M[i] = make([]float64, 2*n)
		copy(M[i][:n], A[i])
		M[i][n+i] = 1.0
	}
	// forward elimination
	for i := 0; i < n; i++ {
		// pivot
		pivot := M[i][i]
		if pivot == 0 {
			// find non-zero pivot
			swap := -1
			for k := i + 1; k < n; k++ {
				if M[k][i] != 0 {
					swap = k
					break
				}
			}
			if swap == -1 {
				return nil, errors.New("singular")
			}
			M[i], M[swap] = M[swap], M[i]
			pivot = M[i][i]
		}
		// normalize row
		invp := 1.0 / pivot
		for j := 0; j < 2*n; j++ {
			M[i][j] *= invp
		}
		// eliminate other rows
		for k := 0; k < n; k++ {
			if k == i {
				continue
			}
			factor := M[k][i]
			if factor == 0 {
				continue
			}
			for j := 0; j < 2*n; j++ {
				M[k][j] -= factor * M[i][j]
			}
		}
	}
	// extract inverse
	inv := make([][]float64, n)
	for i := 0; i < n; i++ {
		inv[i] = make([]float64, n)
		copy(inv[i], M[i][n:])
	}
	return inv, nil
}

// Project onto the unit simplex {w>=0, sum(w)=1} using sorting-based algorithm (O(n log n)).
func ProjectSimplex(v []float64) []float64 {
	n := len(v)
	u := make([]float64, n)
	copy(u, v)
	// sort descending
	sort.Slice(u, func(i, j int) bool { return u[i] > u[j] })
	var rho int = -1
	sum := 0.0
	for j := 0; j < n; j++ {
		sum += u[j]
		t := (sum - 1.0) / float64(j+1)
		if u[j]-t > 0 {
			rho = j
		}
	}
	sum = 0
	for j := 0; j <= rho; j++ {
		sum += u[j]
	}
	theta := (sum - 1.0) / float64(rho+1)
	w := make([]float64, n)
	for i := 0; i < n; i++ {
		w[i] = v[i] - theta
		if w[i] < 0 {
			w[i] = 0
		}
	}
	return w
}

// Variance minimization with projection to simplex; simple projected gradient descent.
// Minimize 0.5 w^T Σ w subject to w in simplex and fixed expected return direction (softly enforced by early anchor).
func VarianceMinProjection(Sigma [][]float64, mu, w0 []float64, iters int, tol float64) []float64 {
	w := make([]float64, len(w0))
	copy(w, w0)
	step := 0.1
	for k := 0; k < iters; k++ {
		// gradient = Σ w
		g := MatVec(Sigma, w)
		// gradient step
		for i := range w {
			w[i] -= step * g[i]
		}
		// project to simplex
		w = ProjectSimplex(w)
		// early stop on small gradient
		ng := math.Sqrt(Dot(g, g))
		if ng < tol {
			break
		}
		// reduce step if needed
		step *= 0.99
	}
	return w
}
