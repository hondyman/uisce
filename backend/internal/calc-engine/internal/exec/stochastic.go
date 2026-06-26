package exec

import (
	"math"
	"math/rand"
	"sort"
	"time"
)

// Geometric Brownian Motion paths: dS = μ S dt + σ S dW
// Returns matrix paths[T+1][N] with paths[0][i] = S0[i]
func GBMPaths(S0 []float64, mu, sigma []float64, T float64, steps int, seeds int) [][]float64 {
	n := len(S0)
	dt := T / float64(steps)
	out := make([][]float64, steps+1)
	for t := 0; t <= steps; t++ {
		out[t] = make([]float64, n)
	}
	copy(out[0], S0)

	r := rand.New(rand.NewSource(time.Now().UnixNano() + int64(seeds)))
	for t := 1; t <= steps; t++ {
		for i := 0; i < n; i++ {
			z := r.NormFloat64()
			drift := (mu[i] - 0.5*sigma[i]*sigma[i]) * dt
			diff := sigma[i] * math.Sqrt(dt) * z
			out[t][i] = out[t-1][i] * math.Exp(drift+diff)
		}
	}
	return out
}

// Ornstein–Uhlenbeck process: dX = θ(μ - X) dt + σ dW (mean-reverting)
func OUPath(x0, theta, mean, sigma, T float64, steps int) []float64 {
	dt := T / float64(steps)
	out := make([]float64, steps+1)
	out[0] = x0
	r := rand.New(rand.NewSource(time.Now().UnixNano()))
	for t := 1; t <= steps; t++ {
		z := r.NormFloat64()
		xPrev := out[t-1]
		out[t] = xPrev + theta*(mean-xPrev)*dt + sigma*math.Sqrt(dt)*z
	}
	return out
}

// Generic Monte Carlo driver. Simulate N scenarios, evaluate payoff, return stats.
type MCStats struct {
	Mean   float64
	Stderr float64
	Pctile map[float64]float64 // e.g., {0.01: x, 0.05: y}
}

func MonteCarlo(pricer func(r *rand.Rand) float64, sims int, quantiles []float64) MCStats {
	r := rand.New(rand.NewSource(time.Now().UnixNano()))
	vals := make([]float64, sims)
	sum := 0.0
	for i := 0; i < sims; i++ {
		v := pricer(r)
		vals[i] = v
		sum += v
	}
	mean := sum / float64(sims)
	// stderr
	var s2 float64
	for _, v := range vals {
		d := v - mean
		s2 += d * d
	}
	s2 /= float64(sims - 1)
	sort.Float64s(vals)
	pct := make(map[float64]float64, len(quantiles))
	for _, q := range quantiles {
		idx := int(math.Max(0, math.Min(float64(sims-1), q*float64(sims-1))))
		pct[q] = vals[idx]
	}
	return MCStats{Mean: mean, Stderr: math.Sqrt(s2 / float64(sims)), Pctile: pct}
}
