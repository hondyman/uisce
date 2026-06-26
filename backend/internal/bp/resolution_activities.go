package bp

import (
	"context"
	"fmt"
	"time"

	"github.com/hondyman/semlayer/backend/internal/rules"
)

type ResolutionActivities struct {
	Engine   *rules.RuleEngine
	Designer *DesignerService
}

func NewResolutionActivities(engine *rules.RuleEngine, designer *DesignerService) *ResolutionActivities {
	return &ResolutionActivities{Engine: engine, Designer: designer}
}

// LoadBlueprintActivity fetches the BP definition from DB and compiles it
func (a *ResolutionActivities) LoadBlueprintActivity(ctx context.Context, bpDefID string) (*WorkflowBlueprint, error) {
	// 1. Fetch from DesignerService
	// We reuse GetDesigner but we might want a strictly internal method that returns typed BPStep.
	// For now, we will use the internal db logic or implement a helper helper.
	// Since GetDesigner returns map[string]interface{}, it's messy to re-parse.
	// Better: Add `GetSteps` to DesignerService that returns []BPStep.

	// Assuming we added GetSteps to DesignerService (we should do that next).
	steps, err := a.Designer.GetSteps(ctx, bpDefID)
	if err != nil {
		return nil, err
	}

	// 2. Compile
	return CompileBlueprint(steps), nil
}

// ResolveApproverRoleActivity evaluates approval chain rules
func (a *ResolutionActivities) ResolveApproverRoleActivity(
	ctx context.Context,
	approvalChain interface{},
	boCtx map[string]map[string]interface{},
) (string, error) {
	// approvalChain is the ApprovalConfig JSON from the designer
	// We expect map[string]interface{}
	config, ok := approvalChain.(map[string]interface{})
	if !ok {
		// If it's *ApprovalChain struct, marshal/unmarshal or access directly.
		// For now assume generic map from JSON unmarshal
		return "", fmt.Errorf("invalid approval chain format")
	}

	// Try to extract levels or rules
	// Simplest structure: { "levels": [ { "actorRole": "...", "entryCondition": "..." } ] }
	levels, ok := config["levels"].([]interface{})
	if !ok {
		// Fallback or legacy structure support
		return "", fmt.Errorf("approval levels not found")
	}

	for _, l := range levels {
		lvl, ok := l.(map[string]interface{})
		if !ok {
			continue
		}

		cond, _ := lvl["entryCondition"].(string)
		role, _ := lvl["actorRole"].(string)

		// Evaluate condition
		// If condition is empty or "true", it matches
		if cond == "" || cond == "true" {
			return role, nil
		}

		// Use Starlark engine
		okMatch, err := a.Engine.EvaluateExpr(ctx, cond, boCtx)
		if err == nil && okMatch {
			return role, nil
		}

	}

	return "", nil
}

// ResolveBranchActivity evaluates routing rules to pick next node
func (a *ResolutionActivities) ResolveBranchActivity(
	ctx context.Context,
	routingRules interface{},
	boCtx map[string]map[string]interface{},
) (string, error) {
	// routingRules is map with "routes": [ { "condition": "...", "targetNodeId": "..." } ]
	config, ok := routingRules.(map[string]interface{})
	if !ok {
		return "", nil
	}

	routes, ok := config["routes"].([]interface{})
	if !ok {
		return "", nil
	}

	for _, r := range routes {
		rule, ok := r.(map[string]interface{})
		if !ok {
			continue
		}
		cond, _ := rule["condition"].(string)

		// targetStepKey or targetNodeId - checking what frontend sends
		// The prompt example used "targetNodeId", our blueprint usually tracks "nextNodes".
		// We'll assume the rule returns the StepKey or ID of the next target.
		target, _ := rule["targetNodeId"].(string)
		if target == "" {
			target, _ = rule["targetStepKey"].(string)
		}

		if cond == "" || cond == "true" {
			return target, nil
		}

		okMatch, err := a.Engine.EvaluateExpr(ctx, cond, boCtx)
		if err == nil && okMatch {
			return target, nil
		}

	}

	// Fallback?
	fallback, _ := config["fallbackRole"].(string) // "Role"? maybe "fallbackStep"?
	if fallback != "" {
		// This might be a role assignment, not a branch target.
		// If it's a branch activity, we expect a Step ID.
		return fallback, nil
	}

	return "", nil
}

// EvaluateDurationActivity evaluates delay expressions
func (a *ResolutionActivities) EvaluateDurationActivity(
	ctx context.Context,
	delayExpr string,
	boCtx map[string]map[string]interface{},
) (time.Duration, error) {
	if delayExpr == "" {
		return 0, nil
	}
	// Engine.EvaluateDurationExpr returns int (seconds)
	val, err := a.Engine.EvaluateDurationExpr(ctx, delayExpr, boCtx)
	if err != nil {
		return 0, err
	}
	return time.Duration(val) * time.Second, nil
}
