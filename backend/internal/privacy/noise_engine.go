package privacy

import (
	"crypto/rand"
	"fmt"
	"math"
	"math/big"
)

type NoiseEngine struct{}

func NewNoiseEngine() *NoiseEngine {
	return &NoiseEngine{}
}

// GenerateLaplaceNoise samples from Laplace(0, b) where scale b = sensitivity / epsilon
func (e *NoiseEngine) GenerateLaplaceNoise(sensitivity, epsilon float64) (float64, error) {
	if epsilon <= 0.0 {
		return 0, fmt.Errorf("epsilon must be strictly positive, got %f", epsilon)
	}
	scale := sensitivity / epsilon

	u, err := e.cryptoUniform()
	if err != nil {
		return 0, err
	}

	var noise float64
	if u < 0 {
		noise = scale * math.Log(1.0+2.0*u)
	} else {
		noise = -scale * math.Log(1.0-2.0*u)
	}

	return noise, nil
}

// GenerateGaussianNoise samples from Normal(0, sigma^2)
func (e *NoiseEngine) GenerateGaussianNoise(sensitivity, epsilon, delta float64) (float64, error) {
	if epsilon <= 0.0 || delta <= 0.0 || delta >= 1.0 {
		return 0, fmt.Errorf("invalid Gaussian DP parameters: eps=%f, delta=%f", epsilon, delta)
	}
	sigma := math.Sqrt(2.0*math.Log(1.25/delta)) * (sensitivity / epsilon)

	u1, err := e.cryptoPositiveUniform()
	if err != nil {
		return 0, err
	}
	u2, err := e.cryptoPositiveUniform()
	if err != nil {
		return 0, err
	}

	z0 := math.Sqrt(-2.0*math.Log(u1)) * math.Cos(2.0*math.Pi*u2)
	return z0 * sigma, nil
}

func (e *NoiseEngine) cryptoUniform() (float64, error) {
	n, err := rand.Int(rand.Reader, big.NewInt(1<<53))
	if err != nil {
		return 0, err
	}
	return (float64(n.Int64()) / float64(1<<53)) - 0.5, nil
}

func (e *NoiseEngine) cryptoPositiveUniform() (float64, error) {
	n, err := rand.Int(rand.Reader, big.NewInt(1<<53-1))
	if err != nil {
		return 0, err
	}
	return (float64(n.Int64()) + 1.0) / float64(1<<53), nil
}
