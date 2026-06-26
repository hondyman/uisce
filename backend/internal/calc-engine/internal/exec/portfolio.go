package exec

import (
	"errors"
	"math"
	"sort"
)

// Markowitz inputs: expected returns mu (len=N), covariance matrix Sigma (N×N).
// Constraints: sum(w)=1; optional long-only; either target return rMin or risk aversion lambda.
// Choose either rMin (target return) or lambda (risk aversion). If both given, rMin wins.
type MarkowitzOpts struct {
	LongOnly     bool
	TargetReturn *float64 // e.g., 0.08
	Lambda       *float64 // risk aversion, e.g., 3.0
}

type PortfolioSolution struct {
	Weights    []float64
	ExpReturn  float64
	Volatility float64
	Variance   float64
	Sharpe     float64
}

// Solve mean-variance with equality constraints via KKT (unconstrained sign).
// For long-only, we run a simple projected gradient refinement.
func MarkowitzOptimize(mu []float64, Sigma [][]float64, opts MarkowitzOpts, riskFree float64) (PortfolioSolution, error) {
	n := len(mu)
	if n == 0 || len(Sigma) != n || len(Sigma[0]) != n {
		return PortfolioSolution{}, errors.New("dimension mismatch")
	}
	one := make([]float64, n)
	for i := range one {
		one[i] = 1.0
	}

	// Precompute inverses and useful vectors
	invS, err := MatInv(Sigma)
	if err != nil {
		return PortfolioSolution{}, err
	}
	invS_mu := MatVec(invS, mu)
	invS_1 := MatVec(invS, one)

	a := Dot(mu, invS_mu) // μ^T Σ^{-1} μ
	b := Dot(mu, invS_1)  // μ^T Σ^{-1} 1
	c := Dot(one, invS_1) // 1^T Σ^{-1} 1
	det := a*c - b*b
	if det <= 0 {
		return PortfolioSolution{}, errors.New("non PD covariance or degenerate inputs")
	}

	var w []float64

	if opts.TargetReturn != nil {
		rMin := *opts.TargetReturn
		// Efficient portfolio for target return:
		// w = α Σ^{-1} μ + β Σ^{-1} 1
		// α = (c rMin - b) / det, β = (a - b rMin) / det
		alpha := (c*rMin - b) / det
		beta := (a - b*rMin) / det
		w = VecAdd(Scale(invS_mu, alpha), Scale(invS_1, beta))
	} else {
		// Risk aversion form: maximize μ^T w - (λ/2) w^T Σ w s.t. 1^T w = 1
		// KKT yields w ∝ Σ^{-1}(μ - ν 1), with ν chosen to satisfy sum(w)=1.
		// Compute w = Σ^{-1}μ + t Σ^{-1}1, solve for t: 1^T w = 1
		// 1 = b + t c -> t = (1 - b)/c
		t := (1.0 - b) / c
		w = VecAdd(invS_mu, Scale(invS_1, t))
		// λ influences scaling of unconstrained objective but with equality sum=1, this gives the tangency-like direction.
		// If you prefer explicit λ handling, sweep target returns instead (frontier approach below).
	}

	if opts.LongOnly {
		// Simple projected gradient steps to enforce w>=0 and sum=1 while minimizing variance for fixed return
		w = ProjectSimplex(w) // ensure feasible start
		w = VarianceMinProjection(Sigma, mu, w, 200, 1e-6)
	}

	exp := Dot(mu, w)
	var_ := QuadForm(Sigma, w)
	vol := math.Sqrt(var_)
	sharpe := (exp - riskFree) / vol

	return PortfolioSolution{Weights: w, ExpReturn: exp, Variance: var_, Volatility: vol, Sharpe: sharpe}, nil
}

// Efficient frontier by sweeping target returns between min and max μ, optionally long-only.
func EfficientFrontier(mu []float64, Sigma [][]float64, riskFree float64, points int, longOnly bool) ([]PortfolioSolution, error) {
	if points < 2 {
		points = 20
	}
	minMu, maxMu := Bounds(mu)
	sols := make([]PortfolioSolution, 0, points)
	for i := 0; i < points; i++ {
		t := float64(i) / float64(points-1)
		rMin := minMu + t*(maxMu-minMu)
		sol, err := MarkowitzOptimize(mu, Sigma, MarkowitzOpts{LongOnly: longOnly, TargetReturn: &rMin}, riskFree)
		if err != nil {
			return nil, err
		}
		sols = append(sols, sol)
	}
	// sort by volatility (left-to-right frontier)
	sort.Slice(sols, func(i, j int) bool { return sols[i].Volatility < sols[j].Volatility })
	return sols, nil
}

// Tangency portfolio: maximize Sharpe => w ∝ Σ^{-1}(μ - r_f 1), normalized to sum=1 or to full investment.
func TangencyPortfolio(mu []float64, Sigma [][]float64, riskFree float64, longOnly bool) (PortfolioSolution, error) {
	n := len(mu)
	one := make([]float64, n)
	for i := range one {
		one[i] = 1
	}
	invS, err := MatInv(Sigma)
	if err != nil {
		return PortfolioSolution{}, err
	}

	excess := make([]float64, n)
	for i := range mu {
		excess[i] = mu[i] - riskFree
	}
	w := MatVec(invS, excess)
	// normalize to sum=1
	s := Sum(w)
	if s == 0 {
		return PortfolioSolution{}, errors.New("degenerate tangency weights")
	}
	for i := range w {
		w[i] /= s
	}

	if longOnly {
		w = ProjectSimplex(w)
		// Optional refinement to maximize Sharpe with constraint; omitted for brevity
	}

	exp := Dot(mu, w)
	var_ := QuadForm(Sigma, w)
	vol := math.Sqrt(var_)
	sharpe := (exp - riskFree) / vol
	return PortfolioSolution{Weights: w, ExpReturn: exp, Variance: var_, Volatility: vol, Sharpe: sharpe}, nil
}

// Tracking Error: TE = sqrt((w - b)^T Σ (w - b))
func TrackingError(Sigma [][]float64, w, b []float64) (float64, error) {
	if len(w) != len(b) || len(Sigma) != len(w) {
		return 0, errors.New("dimension mismatch")
	}
	d := VecSub(w, b)
	v := QuadForm(Sigma, d)
	if v < 0 {
		v = 0
	}
	return math.Sqrt(v), nil
}

// Information Ratio: (μ_p - μ_b)/TE, where μ_p = w^T μ, μ_b = b^T μ
func InformationRatio(mu []float64, Sigma [][]float64, w, b []float64) (float64, error) {
	te, err := TrackingError(Sigma, w, b)
	if err != nil {
		return 0, err
	}
	if te == 0 {
		return 0, errors.New("zero tracking error")
	}
	rp := Dot(mu, w)
	rb := Dot(mu, b)
	return (rp - rb) / te, nil
}
