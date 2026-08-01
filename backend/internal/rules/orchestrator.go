package rules

import (
	"context"
	"strings"
	"time"

	vm "github.com/hondyman/uisce/backend/internal/rules/vm"
)

func (e *RuleEngine) EvaluateGroup(
	ctx context.Context,
	tenantID string,
	group *RuleChain,
	input map[string]any,
) (*BatchResult, *EvalTrace) {
	switch strings.ToUpper(group.Operator) {
	case "AND", "OR":
		rules := group.Rules
		metadata := make([]*RuleWithMetadata, len(rules))
		for i := range rules {
			metadata[i] = rules[i]
		}
		batch := e.EvaluateBatch(ctx, tenantID, metadata, input)
		return batch, &EvalTrace{Revision: e.getState(tenantID).Revision}
	case "CHAIN":
		return e.evaluateChain(ctx, tenantID, group, input)
	default:
		return e.evaluateChain(ctx, tenantID, group, input)
	}
}

func (e *RuleEngine) evaluateChain(
	ctx context.Context,
	tenantID string,
	group *RuleChain,
	input map[string]any,
) (*BatchResult, *EvalTrace) {
	start := time.Now()
	state := e.getState(tenantID)

	rec := vm.Project(input, state.Syms, state.Enums)
	defer vm.PutFastRecord(rec)

	stack := vm.GetStack()
	defer vm.PutStack(stack)

	var results []*RuleResult
	for _, rule := range group.Rules {
		stack.Reset()

		key := cacheKeyFor(rule.ID, rule.Version)

		var res *vm.CompileResult
		if cached, ok := state.Cache.Load(key); ok {
			res = cached.(*vm.CompileResult)
		} else {
			e.metrics.cacheMisses.Add(1)
			newRes := CompileVM(rule.Node, state.Syms, state.Enums)
			res = &newRes
			state.Cache.Store(key, res)
		}

		var passed bool
		if res.Unsupported != nil || len(res.Program.Insts) == 0 {
			e.metrics.fallbacks.Add(1)
			e.metrics.compileErrors.Add(1)
			evaluator := NewConditionEvaluator()
			passed, _, _ = EvaluateRecursiveWithDiagnostics(evaluator, rule.Node, input)
		} else {
			e.metrics.vmPathCount.Add(1)
			passed = e.vm.Run(res.Program, rec, stack)
		}

		result := &RuleResult{
			Passed:   passed,
			Severity: rule.Severity,
			RuleID:   rule.ID,
			RuleName: rule.Name,
			Category: rule.Category,
		}

		if !passed {
			result.Actions = rule.Actions
			if res.Unsupported != nil {
				result.Details = append(result.Details, res.Unsupported.Error())
			}
			if res.Unsupported != nil || len(res.Program.Insts) == 0 {
				evaluator := NewConditionEvaluator()
				_, violations, _ := EvaluateRecursiveWithDiagnostics(evaluator, rule.Node, input)
				result.Violations = violations
			}
		} else if rule.ScoringFormula != "" {
			if score, err := e.evaluateScoringFormula(ctx, rule.ScoringFormula, input); err == nil {
				result.Score = &score
			}
		}

		results = append(results, result)

		if !passed && severityMeetsThreshold(rule.Severity, group.StopOnFirst) {
			break
		}
	}

	return &BatchResult{
		Results:     results,
		TotalTimeNs: time.Since(start).Nanoseconds(),
		PassedAll:   allPassed(results),
	}, &EvalTrace{Revision: state.Revision}
}

func severityMeetsThreshold(actual, threshold Severity) bool {
	if threshold == "" {
		return false
	}
	order := map[Severity]int{
		SeverityInfo:       0,
		SeverityWarning:    1,
		SeverityError:      2,
		SeverityHardBlock:  3,
		SeverityQuarantine: 4,
	}
	return order[actual] >= order[threshold]
}

func allPassed(results []*RuleResult) bool {
	for _, r := range results {
		if !r.Passed {
			return false
		}
	}
	return true
}
