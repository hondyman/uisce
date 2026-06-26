package wealth

import (
	"context"
	"fmt"

	"github.com/hondyman/semlayer/backend/internal/calcengine"
)

// TaxCalcEngineAdapter provides a thin wrapper around the shared calc engine
// so wealth-specific services can request tax calculations without depending
// on calc engine details.
type TaxCalcEngineAdapter struct {
	engine calcengine.CalcEngine
}

// NewTaxCalcEngineAdapter wires a calc engine into the adapter.
func NewTaxCalcEngineAdapter(engine calcengine.CalcEngine) *TaxCalcEngineAdapter {
	return &TaxCalcEngineAdapter{engine: engine}
}

// RegisterWealthTaxMetrics is currently a placeholder that validates the calc
// engine is available before tax metrics are invoked. Real integrations would
// push metric definitions into the engine or catalog.
func RegisterWealthTaxMetrics(ctx context.Context, engine calcengine.CalcEngine, tenantID string) error {
	if engine == nil {
		return fmt.Errorf("calc engine not configured")
	}
	// Placeholder no-op registration for now.
	return nil
}

// Calculate proxies a metric request to the underlying calc engine.
func (a *TaxCalcEngineAdapter) Calculate(ctx context.Context, metric string, inputs map[string]interface{}) (*calcengine.CalcResult, error) {
	if a == nil || a.engine == nil {
		return nil, fmt.Errorf("calc engine adapter not initialized")
	}
	if inputs == nil {
		inputs = map[string]interface{}{}
	}
	return a.engine.Run(ctx, metric, inputs)
}
