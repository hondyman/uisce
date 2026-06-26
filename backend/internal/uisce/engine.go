package uisce

import (
	"context"
	"time"

	"github.com/hondyman/semlayer/backend/internal/policy"
)

type Engine struct {
	PolicyManager *policy.PolicyManager
	Filters       []Filter
}

func NewEngine(pm *policy.PolicyManager, filters []Filter) *Engine {
	return &Engine{
		PolicyManager: pm,
		Filters:       filters,
	}
}

// RunDebug executes the pipeline in trace mode
func (e *Engine) RunDebug(ctx context.Context, tradeData map[string]interface{}) *TraceResult {
	tracer := NewTracer()

	// Try to get ID from data, default to "unknown"
	tradeID, ok := tradeData["id"].(string)
	if !ok {
		tradeID = "unknown"
	}

	result := &TraceResult{
		TradeID: tradeID,
		Steps:   []DebugStep{},
		Success: true,
	}

	for _, filter := range e.Filters {
		start := time.Now()

		// In a real implementation, we might look up the configuration for this filter
		// from the policy manager here, e.g.:
		// effectiveDate, _ := time.Parse("2006-01-02", tradeData["trade_date"].(string))
		// rule, _ := e.PolicyManager.GetEffectivePolicy(ctx, filter.Name(), effectiveDate)
		// filter.Configure(rule)

		err := filter.Purify(ctx, tradeData)

		tracer.RecordStep(filter.Name(), tradeData, err, time.Since(start))

		if err != nil {
			result.Success = false
			// Stop on failure? Standard behavior is usually fail-fast.
			// tracer keeps recording, but we might break here.
			// Let's break for now.
			break
		}
	}

	result.Steps = tracer.Steps
	return result
}
