package shap

import (
	"context"
	"sync"

	"github.com/hondyman/semlayer/backend/internal/ml"
)

// Minimal, clean SHAP engine implementation used for compilation and unit tests.

// ShapComputeResult is returned from SHAP computation
type ShapComputeResult struct {
	SHAPValues  map[string]float64 `json:"shap_values"`
	BaseValue   float64            `json:"base_value"`
	ComputeTime float64            `json:"compute_time_ms"`
	ModelOutput float64            `json:"model_output"`
}

// FeatureDistributions contains statistical info about features
type FeatureDistributions struct {
	Features map[string]*ml.FeatureRange `json:"features"`
}

// PythonExecutor defines the interface to compute SHAP values
type PythonExecutor interface {
	GetFeatureDistributions(ctx context.Context, modelPath string) (*FeatureDistributions, error)
	ComputeBatchSHAP(ctx context.Context, inputs []ml.PredictionInput, modelPath string) ([]ShapComputeResult, error)
	ComputeSHAP(ctx context.Context, input *ml.PredictionInput, modelPath string) (*ShapComputeResult, error)
}

// EngineConfig holds configuration for the SHAP engine
type EngineConfig struct {
	MaxInteractions int
	SHAPType        string
}

// Engine handles SHAP-based explanations
type Engine struct {
	config         *EngineConfig
	cacheMutex     sync.RWMutex
	cache          map[string]*ml.Explainability
	pythonExecutor PythonExecutor
}

// NewEngine creates a new SHAP explanation engine
func NewEngine(executor PythonExecutor, config *EngineConfig) *Engine {
	if config == nil {
		config = &EngineConfig{MaxInteractions: 10, SHAPType: "shared_kernel"}
	}
	return &Engine{config: config, cache: make(map[string]*ml.Explainability), pythonExecutor: executor}
}

// Explain generates an explanation for a single prediction
func (e *Engine) Explain(ctx context.Context, input *ml.PredictionInput, modelPath string, prediction *ml.Prediction) (*ml.Explainability, error) {
	if e.pythonExecutor != nil {
		res, err := e.pythonExecutor.ComputeSHAP(ctx, input, modelPath)
		if err != nil {
			return nil, err
		}
		exp := &ml.Explainability{
			ComputationTime: res.ComputeTime,
			BaseValue:       res.BaseValue,
			SHAPValues:      res.SHAPValues,
		}
		return exp, nil
	}

	return &ml.Explainability{ComputationTime: 0, BaseValue: 0, SHAPValues: map[string]float64{}}, nil
}

// ExplainBatch generates explanations for multiple inputs
func (e *Engine) ExplainBatch(ctx context.Context, inputs []ml.PredictionInput, modelPath string) (map[string]*ml.Explainability, error) {
	results := make(map[string]*ml.Explainability)
	for _, in := range inputs {
		exp, _ := e.Explain(ctx, &in, modelPath, nil)
		results[in.ChainID] = exp
	}
	return results, nil
}

// ClearCache purges the explanation cache
func (e *Engine) ClearCache() {
	e.cacheMutex.Lock()
	defer e.cacheMutex.Unlock()
	for k := range e.cache {
		delete(e.cache, k)
	}
}
