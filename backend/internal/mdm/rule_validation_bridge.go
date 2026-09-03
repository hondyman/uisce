package mdm

import (
	"context"

	"github.com/hondyman/uisce/backend/internal/boresolver"
	"github.com/hondyman/uisce/backend/internal/rules"
)

// ValidateGoldenRecord runs a compiled compliance/data-quality rule (from
// internal/rules — the platform's single rule VM, shared with the
// compliance workflows; see internal/rules/bo_context_bridge.go) against a
// resolved MDM golden record, evaluated against Business Object field names
// rather than raw physical columns. This is the seam that lets
// SurvivorshipEngine's mastering output be gated by the same rule
// vocabulary compliance and the calc engine already use, without changing
// ResolveField's existing per-field strategy logic.
//
// bo describes the Business Object whose fields resolvedRecord's keys are
// physical columns for (e.g. the output of one or more
// SurvivorshipEngine.ResolveField calls, keyed by physical column). A nil
// rule always passes — "no validation configured for this entity type" is
// not a failure.
func ValidateGoldenRecord(ctx context.Context, engine *rules.RuleEngine, bo *boresolver.BODefinition, resolvedRecord map[string]interface{}, rule *rules.RuleNode) (bool, error) {
	if rule == nil {
		return true, nil
	}
	boCtx := rules.BuildContextFromBORow(bo, resolvedRecord)
	return engine.EvaluateNode(ctx, rule, boCtx)
}
