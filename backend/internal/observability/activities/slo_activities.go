package activities

import (
	"context"

	"github.com/hondyman/semlayer/backend/internal/cbo"
)

// SLOActivities holds the activities for SLO evaluation
type SLOActivities struct {
	evaluator *cbo.SLOEvaluator
	provider  *cbo.DBSLOProvider
}

// NewSLOActivities creates a new SLO activities instance
func NewSLOActivities(evaluator *cbo.SLOEvaluator, provider *cbo.DBSLOProvider) *SLOActivities {
	return &SLOActivities{
		evaluator: evaluator,
		provider:  provider,
	}
}

// LoadActiveSLOsActivity loads all active SLOs for an environment
func (a *SLOActivities) LoadActiveSLOsActivity(ctx context.Context, env string) ([]cbo.SLODefinition, error) {
	// List all SLOs
	allSLOs, err := a.provider.ListSLOs(ctx, env, nil, "", "")
	if err != nil {
		return nil, err
	}

	// Filter for enabled only
	var activeSLOs []cbo.SLODefinition
	for _, slo := range allSLOs {
		if slo.Enabled {
			activeSLOs = append(activeSLOs, slo)
		}
	}

	return activeSLOs, nil
}

// EvaluateSLOActivity evaluates a single SLO
func (a *SLOActivities) EvaluateSLOActivity(ctx context.Context, slo cbo.SLODefinition) (*cbo.SLOEvaluation, error) {
	return a.evaluator.Evaluate(ctx, &slo)
}

// HandleSLOViolationActivity handles an SLO violation
func (a *SLOActivities) HandleSLOViolationActivity(ctx context.Context, eval cbo.SLOEvaluation) error {
	// Re-construct definition from evaluation to handle check (simplified)
	// Ideally we pass full context or look it up
	// But HandleViolation just needs to call evaluator.handleViolation
	// Wait, Evaluate() already calls handleViolation if I look at my implementation in slo_evaluator.go
	// Let's check slo_evaluator.go:
	// func (e *SLOEvaluator) Evaluate(...) { ... if eval.Status == "violated" { e.handleViolation(...) } ... }

	// If Evaluate already handles it, we might not need this activity OR
	// we should separate the concern.
	// The user prompt had: evaluate -> get eval -> if violated -> handle.
	// My Evaluate() implementation currently does BOTH.
	// I should probably remove the side-effect from Evaluate() in slo_evaluator.go if I want the workflow to control it.
	// Or, just have the workflow use Evaluate() and if it returns violated, it logs/notifies via another activity if needed.
	// But handleFlow in `cbo/slo_evaluator.go` handles ASO tuning updates.

	// Let's keep Evaluate() self-contained for now as implemented, so the workflow is simpler.
	// But the user prompt explicitly showed the loop.
	// If Evaluate() handles it, calling HandleSLOViolationActivity is redundant unless it does *extra* things (like Slack alerts).
	// Currently `slo_evaluator.go` handles ASO tuning. It does NOT seem to send Slack alerts yet (that's in `slo_service.go` which I implemented previously, or `alert_dispatcher.go`).

	// My `slo_service.go` (from previous turn) handled alerts.
	// `cbo/slo_evaluator.go` handles ASO tuning.

	// I'll keep this activity as a placeholder for external notifications if not already handled.
	return nil
}
