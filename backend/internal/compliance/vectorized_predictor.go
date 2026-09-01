package compliance

import (
	"math"
	"sync"
)

// FeatureVectorBatch represents columnar arrays for SIMD/vectorized processing
type FeatureVectorBatch struct {
	AccountIDs    []string
	RuleIDs       []string
	Utilizations  []float64 // U_r (Current / Limit)
	Volatilities  []float64 // 30-day realized volatility
	Momentums     []float64 // 14-day price momentum
	ReopenCounts  []float64 // Historical breach counts
	Probabilities []float64 // Output predictions
}

type VectorizedPredictorEngine struct {
	intercept float64
	wUtil     float64
	wVol      float64
	wMom      float64
	wReopen   float64
}

func NewVectorizedPredictorEngine() *VectorizedPredictorEngine {
	return &VectorizedPredictorEngine{
		intercept: -4.50,
		wUtil:     5.20,
		wVol:      2.80,
		wMom:      1.95,
		wReopen:   0.65,
	}
}

// PredictBatch evaluates breach probabilities across 100,000 items in parallel chunks (< 5ms)
func (e *VectorizedPredictorEngine) PredictBatch(batch *FeatureVectorBatch, numWorkers int) {
	n := len(batch.Utilizations)
	if len(batch.Probabilities) < n {
		batch.Probabilities = make([]float64, n)
	}

	if numWorkers <= 0 {
		numWorkers = 4
	}

	chunkSize := (n + numWorkers - 1) / numWorkers
	var wg sync.WaitGroup

	for i := 0; i < numWorkers; i++ {
		start := i * chunkSize
		end := start + chunkSize
		if end > n {
			end = n
		}
		if start >= end {
			break
		}

		wg.Add(1)
		go func(s, eIdx int) {
			defer wg.Done()
			w0 := e.intercept
			w1 := e.wUtil
			w2 := e.wVol
			w3 := e.wMom
			w4 := e.wReopen

			for j := s; j < eIdx; j++ {
				z := w0 + (w1 * batch.Utilizations[j]) + (w2 * batch.Volatilities[j]) +
					(w3 * batch.Momentums[j]) + (w4 * batch.ReopenCounts[j])
				batch.Probabilities[j] = 1.0 / (1.0 + math.Exp(-z))
			}
		}(start, end)
	}

	wg.Wait()
}
