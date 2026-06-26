package exec

import (
	"errors"
	"math"
	"sort"

	"gonum.org/v1/gonum/mat"
)

// MarkowitzOpts contains options for Markowitz optimization
type MarkowitzOpts struct {
	MaxWeight float64
	MinWeight float64
}

// MarkowitzOptimize performs mean-variance portfolio optimization
func MarkowitzOptimize(mu []float64, Sigma *mat.Dense, opts MarkowitzOpts, riskFreeRate float64) ([]float64, error) {
	n := len(mu)
	if n == 0 {
		return nil, errors.New("empty returns vector")
	}

	// Default options
	if opts.MaxWeight == 0 {
		opts.MaxWeight = 1.0
	}
	if opts.MinWeight == 0 {
		opts.MinWeight = 0.0
	}

	// Convert to matrices
	ones := mat.NewVecDense(n, nil)
	for i := 0; i < n; i++ {
		ones.SetVec(i, 1.0)
	}

	// Build the optimization matrix
	A := mat.NewDense(n+2, n+2, nil)

	// Copy Sigma to top-left
	A.Copy(Sigma)

	// Add constraints
	for i := 0; i < n; i++ {
		A.Set(n, i, 1.0)     // weights sum to 1
		A.Set(n+1, i, mu[i]) // expected return constraint
		A.Set(i, n, 1.0)
		A.Set(i, n+1, mu[i])
	}

	// Right-hand side
	b := mat.NewVecDense(n+2, nil)
	b.SetVec(n, 1.0)   // sum weights = 1
	b.SetVec(n+1, 0.0) // target return = 0 (will be adjusted)

	// Solve using matrix inversion
	var inv mat.Dense
	err := inv.Inverse(A)
	if err != nil {
		return nil, err
	}

	x := mat.NewVecDense(n+2, nil)
	x.MulVec(&inv, b)

	weights := make([]float64, n)
	for i := 0; i < n; i++ {
		weights[i] = x.At(i, 0)
	}

	return weights, nil
}

// EfficientFrontier computes the efficient frontier
func EfficientFrontier(mu []float64, Sigma *mat.Dense, riskFreeRate float64, points int, longOnly bool) ([][]float64, error) {
	if points <= 0 {
		return nil, errors.New("invalid number of points")
	}

	n := len(mu)
	frontier := make([][]float64, points)

	minRet := math.Inf(1)
	maxRet := math.Inf(-1)
	for _, r := range mu {
		if r < minRet {
			minRet = r
		}
		if r > maxRet {
			maxRet = r
		}
	}

	for i := 0; i < points; i++ {
		targetRet := minRet + float64(i)*(maxRet-minRet)/float64(points-1)

		opts := MarkowitzOpts{
			MaxWeight: 1.0,
			MinWeight: 0.0,
		}
		if longOnly {
			opts.MinWeight = 0.0
		} else {
			opts.MinWeight = -1.0
		}

		weights, err := MarkowitzOptimize(mu, Sigma, opts, riskFreeRate)
		if err != nil {
			return nil, err
		}

		frontier[i] = make([]float64, n+1)
		copy(frontier[i][:n], weights)
		frontier[i][n] = targetRet
	}

	return frontier, nil
}

// TangencyPortfolio computes the tangency portfolio (maximum Sharpe ratio)
func TangencyPortfolio(mu []float64, Sigma *mat.Dense, riskFreeRate float64, longOnly bool) ([]float64, error) {
	n := len(mu)

	// Adjust returns for risk-free rate
	excessMu := make([]float64, n)
	for i := range mu {
		excessMu[i] = mu[i] - riskFreeRate
	}

	opts := MarkowitzOpts{
		MaxWeight: 1.0,
		MinWeight: 0.0,
	}
	if !longOnly {
		opts.MinWeight = -1.0
	}

	return MarkowitzOptimize(excessMu, Sigma, opts, 0.0)
}

// TrackingError computes the tracking error between portfolio and benchmark
func TrackingError(Sigma *mat.Dense, weights, benchmarkWeights []float64) (float64, error) {
	if len(weights) != len(benchmarkWeights) {
		return 0, errors.New("weight vectors must have same length")
	}

	n := len(weights)
	diff := make([]float64, n)
	for i := 0; i < n; i++ {
		diff[i] = weights[i] - benchmarkWeights[i]
	}

	// Compute variance of the difference
	variance := 0.0
	for i := 0; i < n; i++ {
		for j := 0; j < n; j++ {
			variance += diff[i] * diff[j] * Sigma.At(i, j)
		}
	}

	return math.Sqrt(variance), nil
}

// InformationRatio computes the information ratio
func InformationRatio(returns []float64, Sigma *mat.Dense, weights, benchmarkWeights []float64) (float64, error) {
	te, err := TrackingError(Sigma, weights, benchmarkWeights)
	if err != nil {
		return 0, err
	}

	if te == 0 {
		return 0, errors.New("tracking error is zero")
	}

	// Compute expected excess return
	excessReturn := 0.0
	for i, ret := range returns {
		excessReturn += (weights[i] - benchmarkWeights[i]) * ret
	}

	return excessReturn / te, nil
}

// SortBySharpeRatio sorts portfolios by Sharpe ratio
func SortBySharpeRatio(portfolios [][]float64, returns []float64, Sigma *mat.Dense, riskFreeRate float64) {
	sort.Slice(portfolios, func(i, j int) bool {
		sharpeI := calculateSharpeRatio(portfolios[i], returns, Sigma, riskFreeRate)
		sharpeJ := calculateSharpeRatio(portfolios[j], returns, Sigma, riskFreeRate)
		return sharpeI > sharpeJ
	})
}

func calculateSharpeRatio(weights []float64, returns []float64, Sigma *mat.Dense, riskFreeRate float64) float64 {
	// Calculate expected return
	expReturn := 0.0
	for i, w := range weights {
		expReturn += w * returns[i]
	}

	// Calculate volatility
	volatility := 0.0
	for i := 0; i < len(weights); i++ {
		for j := 0; j < len(weights); j++ {
			volatility += weights[i] * weights[j] * Sigma.At(i, j)
		}
	}
	volatility = math.Sqrt(volatility)

	if volatility == 0 {
		return 0
	}

	return (expReturn - riskFreeRate) / volatility
}
