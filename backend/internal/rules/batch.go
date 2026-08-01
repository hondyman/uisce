package rules

import (
	"context"
	"time"

	vm "github.com/hondyman/uisce/backend/internal/rules/vm"
)

func (e *RuleEngine) EvaluateRule(
	ctx context.Context,
	tenantID string,
	rule *RuleWithMetadata,
	input map[string]any,
) (*RuleResult, *EvalTrace) {
	start := time.Now()

	passed, trace, _ := e.Evaluate(ctx, tenantID, rule.ID, rule.Version, rule.Node, input, false)

	result := &RuleResult{
		Passed:         passed,
		Severity:      rule.Severity,
		RuleID:        rule.ID,
		RuleName:      rule.Name,
		Category:      rule.Category,
		FailureReasons: trace.FailureReasons,
		EvalTimeNs:    time.Since(start).Nanoseconds(),
	}

	if !passed {
		result.Actions = rule.Actions
		if len(trace.Fallback) > 0 {
			result.Details = append(result.Details, trace.Fallback)
		}
	}

	if passed && rule.ScoringFormula != "" {
		if score, err := e.evaluateScoringFormula(ctx, rule.ScoringFormula, input); err == nil {
			result.Score = &score
		}
	}

	return result, trace
}

func (e *RuleEngine) EvaluateBatch(
	ctx context.Context,
	tenantID string,
	rules []*RuleWithMetadata,
	input map[string]any,
) *BatchResult {
	start := time.Now()

	state := e.getState(tenantID)

	rec := vm.Project(input, state.Syms, state.Enums)
	defer vm.PutFastRecord(rec)

	stack := vm.GetStack()
	defer vm.PutStack(stack)

	results := make([]*RuleResult, len(rules))
	passedAll := true

	for i, rule := range rules {
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
			fallbackReason := ""
			if res.Unsupported != nil {
				fallbackReason = res.Unsupported.Error()
			}
			evaluator := NewConditionEvaluator()
			passed, violations, _ := EvaluateRecursiveWithDiagnostics(evaluator, rule.Node, input)
			results[i] = &RuleResult{
				Passed:         passed,
				Severity:       rule.Severity,
				RuleID:         rule.ID,
				RuleName:       rule.Name,
				Category:       rule.Category,
				Details:        []string{fallbackReason},
				Violations:     violations,
			}
			if !passed {
				passedAll = false
			}
			continue
		}

		stack.Reset()

		e.metrics.vmPathCount.Add(1)
		passed = e.vm.Run(res.Program, rec, stack)

		results[i] = &RuleResult{
			Passed:   passed,
			Severity: rule.Severity,
			RuleID:   rule.ID,
			RuleName: rule.Name,
			Category: rule.Category,
		}

		if !passed {
			passedAll = false
			results[i].Actions = rule.Actions
		} else if rule.ScoringFormula != "" {
			if score, err := e.evaluateScoringFormula(ctx, rule.ScoringFormula, input); err == nil {
				results[i].Score = &score
			}
		}
	}

	return &BatchResult{
		Results:     results,
		TotalTimeNs: time.Since(start).Nanoseconds(),
		PassedAll:   passedAll,
	}
}

func (e *RuleEngine) EvaluateSweep(
	ctx context.Context,
	tenantID string,
	rule *RuleWithMetadata,
	inputs []map[string]any,
) []*RuleResult {
	state := e.getState(tenantID)
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

	if res.Unsupported != nil || len(res.Program.Insts) == 0 {
		results := make([]*RuleResult, len(inputs))
		for i, input := range inputs {
			e.metrics.fallbacks.Add(1)
			passed, _ := e.recursive.Evaluate(*rule.Node, input)
			results[i] = &RuleResult{
				Passed:   passed,
				Severity: rule.Severity,
				RuleID:   rule.ID,
				RuleName: rule.Name,
				Category: rule.Category,
			}
		}
		return results
	}

	stack := vm.GetStack()
	defer vm.PutStack(stack)

	results := make([]*RuleResult, len(inputs))
	for i, input := range inputs {
		rec := vm.Project(input, state.Syms, state.Enums)
		stack.Reset()

		e.metrics.vmPathCount.Add(1)
		passed := e.vm.Run(res.Program, rec, stack)

		results[i] = &RuleResult{
			Passed:   passed,
			Severity: rule.Severity,
			RuleID:   rule.ID,
			RuleName: rule.Name,
			Category: rule.Category,
		}

		vm.PutFastRecord(rec)
	}

	return results
}

func (e *RuleEngine) evaluateScoringFormula(ctx context.Context, formula string, input map[string]any) (float64, error) {
	if e.env == nil {
		return 0, nil
	}
	ast, issues := e.env.Compile(formula)
	if issues != nil && issues.Err() != nil {
		return 0, issues.Err()
	}
	prg, err := e.env.Program(ast)
	if err != nil {
		return 0, err
	}
	out, _, err := prg.Eval(map[string]interface{}{
		"input": input,
	})
	if err != nil {
		return 0, err
	}
	switch v := out.Value().(type) {
	case float64:
		return v, nil
	case int:
		return float64(v), nil
	case int64:
		return float64(v), nil
	default:
		return 0, nil
	}
}
