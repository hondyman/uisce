package exec

import (
	"math"
	"math/rand"
)

// GBMPaths generates multiple paths of Geometric Brownian Motion
func GBMPaths(S0, mu, sigma, T float64, steps int, seed int64) [][]float64 {
	if steps <= 0 {
		return nil
	}

	rng := rand.New(rand.NewSource(seed))
	dt := T / float64(steps)
	paths := make([][]float64, 1) // Single path for now
	paths[0] = make([]float64, steps+1)
	paths[0][0] = S0

	for i := 1; i <= steps; i++ {
		dW := rng.NormFloat64() * math.Sqrt(dt)
		dS := mu*dt + sigma*dW
		paths[0][i] = paths[0][i-1] * math.Exp(dS)
	}

	return paths
}

// OUPath generates a path of Ornstein-Uhlenbeck process
func OUPath(x0, theta, mean, sigma, T float64, steps int) []float64 {
	if steps <= 0 {
		return nil
	}

	rng := rand.New(rand.NewSource(42)) // Fixed seed for reproducibility
	dt := T / float64(steps)
	path := make([]float64, steps+1)
	path[0] = x0

	for i := 1; i <= steps; i++ {
		dW := rng.NormFloat64() * math.Sqrt(dt)
		dx := theta*(mean-path[i-1])*dt + sigma*dW
		path[i] = path[i-1] + dx
	}

	return path
}

// MonteCarlo performs Monte Carlo simulation
func MonteCarlo(generator func(*rand.Rand) float64, nSimulations int) *MonteCarloStats {
	if nSimulations <= 0 {
		return nil
	}

	rng := rand.New(rand.NewSource(42))
	values := make([]float64, nSimulations)

	sum := 0.0
	min := math.Inf(1)
	max := math.Inf(-1)

	for i := 0; i < nSimulations; i++ {
		val := generator(rng)
		values[i] = val
		sum += val
		if val < min {
			min = val
		}
		if val > max {
			max = val
		}
	}

	mean := sum / float64(nSimulations)

	variance := 0.0
	for _, val := range values {
		variance += (val - mean) * (val - mean)
	}
	variance /= float64(nSimulations - 1)
	std := math.Sqrt(variance)

	return &MonteCarloStats{
		Mean:   mean,
		Std:    std,
		Min:    min,
		Max:    max,
		Values: values,
		NSims:  nSimulations,
	}
}

// MonteCarloStats holds statistics from Monte Carlo simulation
type MonteCarloStats struct {
	Mean   float64
	Std    float64
	Min    float64
	Max    float64
	Values []float64
	NSims  int
}

// GBMOptionPrice prices European options using Monte Carlo with GBM
func GBMOptionPrice(S0, K, T, r, sigma float64, isCall bool, nSimulations int) float64 {
	if nSimulations <= 0 {
		return 0
	}

	rng := rand.New(rand.NewSource(42))
	sum := 0.0

	for i := 0; i < nSimulations; i++ {
		// Generate terminal stock price
		Z := rng.NormFloat64()
		ST := S0 * math.Exp((r-0.5*sigma*sigma)*T+sigma*math.Sqrt(T)*Z)

		// Calculate payoff
		var payoff float64
		if isCall {
			payoff = math.Max(ST-K, 0)
		} else {
			payoff = math.Max(K-ST, 0)
		}

		sum += payoff
	}

	return math.Exp(-r*T) * sum / float64(nSimulations)
}

// CIRPaths generates paths of Cox-Ingersoll-Ross interest rate model
func CIRPaths(r0, kappa, theta, sigma, T float64, steps int, seed int64) [][]float64 {
	if steps <= 0 {
		return nil
	}

	rng := rand.New(rand.NewSource(seed))
	dt := T / float64(steps)
	paths := make([][]float64, 1)
	paths[0] = make([]float64, steps+1)
	paths[0][0] = r0

	for i := 1; i <= steps; i++ {
		r := paths[0][i-1]
		dW := rng.NormFloat64() * math.Sqrt(dt)

		// CIR process: dr = kappa*(theta - r)*dt + sigma*sqrt(r)*dW
		drift := kappa * (theta - r) * dt
		diffusion := sigma * math.Sqrt(math.Max(r, 0)) * dW

		paths[0][i] = math.Max(r+drift+diffusion, 0) // Ensure non-negative
	}

	return paths
}

// JumpDiffusionPaths generates paths with jumps
func JumpDiffusionPaths(S0, mu, sigma, lambda, jumpMean, jumpStd, T float64, steps int, seed int64) [][]float64 {
	if steps <= 0 {
		return nil
	}

	rng := rand.New(rand.NewSource(seed))
	dt := T / float64(steps)
	paths := make([][]float64, 1)
	paths[0] = make([]float64, steps+1)
	paths[0][0] = S0

	for i := 1; i <= steps; i++ {
		dW := rng.NormFloat64() * math.Sqrt(dt)

		// Check for jump
		jump := 0.0
		if rng.Float64() < lambda*dt {
			jump = rng.NormFloat64()*jumpStd + jumpMean
		}

		dS := (mu-0.5*sigma*sigma)*dt + sigma*dW + jump
		paths[0][i] = paths[0][i-1] * math.Exp(dS)
	}

	return paths
}
